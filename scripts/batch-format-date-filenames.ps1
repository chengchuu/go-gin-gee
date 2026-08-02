# PowerShell
# Convert leading YY-MMDD date strings in file names to YYYYMMDD.
#
# Example:
# 26-0802-project-topic.md -> 20260802-project-topic.md
# 25-0330_Video.md -> 20250330_Video.md
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-format-date-filenames.ps1" -Path "E:\NOTES"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -ExecutionPolicy Bypass -File "scripts/batch-format-date-filenames.ps1" -Path "/path/to/files"
#
# Preview only
# pwsh -NoProfile -ExecutionPolicy Bypass -File "scripts/batch-format-date-filenames.ps1" -Path "/path/to/files" -WhatIf

[CmdletBinding(SupportsShouldProcess = $true)]
param(
  [Parameter(Mandatory = $true)]
  [string]$Path,

  [int]$Century = 2000,

  [switch]$Recurse
)

[Console]::InputEncoding  = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

function Convert-LeadingDateString {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Name,

    [Parameter(Mandatory = $true)]
    [int]$Century
  )

  $match = [regex]::Match($Name, '^(\d{2})-(\d{2})(\d{2})([-_].+)$')
  if (!$match.Success) {
    return $Name
  }

  $year = $Century + [int]$match.Groups[1].Value
  $month = [int]$match.Groups[2].Value
  $day = [int]$match.Groups[3].Value

  try {
    $null = [datetime]::new($year, $month, $day)
  } catch {
    Write-Warning ("Invalid date in file name, skipping: {0}" -f $Name)
    return $Name
  }

  return "{0:D4}{1}{2}{3}" -f $year, $match.Groups[2].Value, $match.Groups[3].Value, $match.Groups[4].Value
}

function New-TemporaryFileName {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Directory
  )

  do {
    $name = "__tmp_dateformat_{0}.tmp" -f ([guid]::NewGuid().ToString('N'))
    $fullPath = Join-Path $Directory $name
  } while (Test-Path -LiteralPath $fullPath)

  return $name
}

function Test-SameFilesystemPath {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Left,

    [Parameter(Mandatory = $true)]
    [string]$Right
  )

  if ($IsWindows -or $env:OS -eq "Windows_NT") {
    return [string]::Equals($Left, $Right, [System.StringComparison]::OrdinalIgnoreCase)
  }

  if ([string]::Equals($Left, $Right, [System.StringComparison]::Ordinal)) {
    return $true
  }

  if (![string]::Equals($Left, $Right, [System.StringComparison]::OrdinalIgnoreCase)) {
    return $false
  }

  $leftIdentity = if ($IsMacOS) {
    & stat -f "%d:%i" -- $Left 2>$null
  } else {
    & stat -c "%d:%i" -- $Left 2>$null
  }

  $rightIdentity = if ($IsMacOS) {
    & stat -f "%d:%i" -- $Right 2>$null
  } else {
    & stat -c "%d:%i" -- $Right 2>$null
  }

  return ($LASTEXITCODE -eq 0 -and $leftIdentity -eq $rightIdentity)
}

if (!(Test-Path -LiteralPath $Path -PathType Container)) {
  throw "Path not found or not a directory: $Path"
}

$root = (Resolve-Path -LiteralPath $Path).Path
$files = Get-ChildItem -LiteralPath $root -File -Recurse:$Recurse |
  Sort-Object FullName

$renamePlan = @()
foreach ($file in $files) {
  $newName = Convert-LeadingDateString -Name $file.Name -Century $Century
  if ($newName -ceq $file.Name) {
    continue
  }

  $targetPath = Join-Path $file.DirectoryName $newName
  $renamePlan += [pscustomobject]@{
    File       = $file
    TempName   = New-TemporaryFileName -Directory $file.DirectoryName
    NewName    = $newName
    TargetPath = $targetPath
  }
}

if ($renamePlan.Count -eq 0) {
  Write-Host "No file names need changes." -ForegroundColor Yellow
  return
}

$collisions = $renamePlan |
  Group-Object { "{0}`0{1}" -f $_.File.DirectoryName, $_.NewName.ToUpperInvariant() } |
  Where-Object { $_.Count -gt 1 }

if ($collisions.Count -gt 0) {
  Write-Error "Rename collision detected after date formatting. No files were renamed."
  foreach ($collision in $collisions) {
    Write-Error ("Target collision: {0}" -f $collision.Group[0].TargetPath)
    foreach ($item in $collision.Group) {
      Write-Error ("  Source: {0}" -f $item.File.FullName)
    }
  }
  exit 1
}

foreach ($item in $renamePlan) {
  $existingTarget = Get-Item -LiteralPath $item.TargetPath -ErrorAction SilentlyContinue
  if ($null -ne $existingTarget -and !(Test-SameFilesystemPath -Left $existingTarget.FullName -Right $item.File.FullName)) {
    Write-Error ("Target already exists, skipping all renames: {0}" -f $item.TargetPath)
    exit 1
  }
}

foreach ($item in $renamePlan) {
  if ($PSCmdlet.ShouldProcess($item.File.FullName, "Rename to temporary name $($item.TempName)")) {
    Rename-Item -LiteralPath $item.File.FullName -NewName $item.TempName
  }
}

foreach ($item in $renamePlan) {
  $tempPath = Join-Path $item.File.DirectoryName $item.TempName
  if ($PSCmdlet.ShouldProcess($tempPath, "Rename to $($item.NewName)")) {
    Rename-Item -LiteralPath $tempPath -NewName $item.NewName
  }
}

if ($WhatIfPreference) {
  Write-Host ("Preview Complete! Would rename {0} file(s)." -f $renamePlan.Count) -ForegroundColor Yellow
} else {
  Write-Host ("Task Complete! Renamed {0} file(s)." -f $renamePlan.Count) -ForegroundColor Green
}
