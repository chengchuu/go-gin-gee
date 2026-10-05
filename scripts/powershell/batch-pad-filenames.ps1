# Left-pad numeric filename prefixes before a literal, case-sensitive separator.
# pwsh -NoProfile -File scripts/powershell/batch-pad-filenames.ps1 -Path ./files -Separator S -Length 3 -WhatIf
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts\powershell\batch-pad-filenames.ps1 -Path E:\Music -Separator S -Length 3

[CmdletBinding(SupportsShouldProcess = $true)]
param(
  [Parameter(Mandatory = $true)]
  [string]$Path,

  [Parameter(Mandatory = $true)]
  [ValidateNotNullOrEmpty()]
  [string]$Separator,

  [Parameter(Mandatory = $true)]
  [ValidateRange(1, 2147483647)]
  [int]$Length,

  [ValidateScript({ $_ -notmatch '[<>:"/\\|?*\x00-\x1f]' })]
  [char]$PadChar = '0',

  [switch]$Recurse
)

$ErrorActionPreference = 'Stop'

try {
  if (!(Test-Path -LiteralPath $Path -PathType Container)) {
    throw "Path not found or not a directory: $Path"
  }

  $files = @(Get-ChildItem -LiteralPath $Path -File -Recurse:$Recurse | Sort-Object FullName)
  $plan = @(
    foreach ($file in $files) {
      $index = $file.BaseName.IndexOf($Separator, [System.StringComparison]::Ordinal)
      if ($index -le 0) { continue }
      $prefix = $file.BaseName.Substring(0, $index)
      if ($prefix -notmatch '\A[0-9]+\z' -or $prefix.Length -ge $Length) { continue }

      $newName = $prefix.PadLeft($Length, $PadChar) + $file.Name.Substring($index)
      [pscustomobject]@{
        Source = $file.FullName
        OriginalName = $file.Name
        NewName = $newName
        Target = Join-Path $file.DirectoryName $newName
      }
    }
  )

  # Preflight the entire batch before changing any files, including recursive results.
  $targets = [System.Collections.Generic.Dictionary[string,string]]::new([System.StringComparer]::OrdinalIgnoreCase)
  foreach ($item in $plan) {
    if ($targets.ContainsKey($item.Target)) {
      throw "Target collision: '$($targets[$item.Target])' and '$($item.Source)' -> '$($item.Target)'. No files were renamed."
    }
    $targets.Add($item.Target, $item.Source)
    if (Test-Path -LiteralPath $item.Target) {
      throw "Target already exists: '$($item.Source)' -> '$($item.Target)'. No files were renamed."
    }
  }

  if ($plan.Count -eq 0) {
    Write-Host 'No file names need changes.'
    return
  }

  $renamed = 0
  foreach ($item in $plan) {
    if ($PSCmdlet.ShouldProcess($item.Source, "Rename to $($item.NewName)")) {
      Rename-Item -LiteralPath $item.Source -NewName $item.NewName
      $renamed++
      Write-Host ("Renamed: {0} -> {1}" -f $item.OriginalName, $item.NewName)
    }
  }

  if ($WhatIfPreference) {
    Write-Host ("Preview Complete! Would rename {0} file(s)." -f $plan.Count)
  } else {
    Write-Host ("Task Complete! Renamed {0} file(s)." -f $renamed)
  }
} catch {
  Write-Error -Message $_.Exception.Message -ErrorAction Continue
  exit 1
}
