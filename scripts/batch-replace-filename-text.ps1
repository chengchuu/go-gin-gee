# PowerShell
# Replace literal text combinations in file or directory names for a target path.
#
# Example:
# my_file_name.md -> my-file-name.md
# -Replace="_=-"
# -Replace="*=-" -Replace="abc=xyz"
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-replace-filename-text.ps1" -Path "E:\NOTES" -Replace="_=-"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -ExecutionPolicy Bypass -File "scripts/batch-replace-filename-text.ps1" -Path "/path/to/files" -Replace="_=-"
#
# Preview only
# pwsh -NoProfile -ExecutionPolicy Bypass -File "scripts/batch-replace-filename-text.ps1" -Path "/path/to/files" -Replace="_=-" -WhatIf

[CmdletBinding(SupportsShouldProcess = $true, PositionalBinding = $false)]
param(
  [Parameter(Mandatory = $true)]
  [string]$Path,

  [ValidateSet("File", "Directory")]
  [string]$TargetType = "File",

  [switch]$Recurse,

  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]]$RemainingArguments
)

[Console]::InputEncoding  = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

function New-TemporaryFileName {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Directory
  )

  do {
    $name = "__tmp_replace_{0}.tmp" -f ([guid]::NewGuid().ToString('N'))
    $fullPath = Join-Path $Directory $name
  } while (Test-Path -LiteralPath $fullPath)

  return $name
}

function Convert-ReplacementTextToRule {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Value
  )

  $separatorIndex = $Value.IndexOf("=")
  if ($separatorIndex -lt 1) {
    throw "Invalid replacement value '$Value'. Expected format: from=to"
  }

  return [pscustomobject]@{
    From = $Value.Substring(0, $separatorIndex)
    To   = $Value.Substring($separatorIndex + 1)
  }
}

function Get-ReplacementRules {
  param(
    [string[]]$Arguments
  )

  $rules = @()
  for ($i = 0; $i -lt $Arguments.Count; $i++) {
    $argument = $Arguments[$i]

    if ($argument -match '^-Replace[:=](.*)$') {
      $rules += Convert-ReplacementTextToRule -Value $Matches[1]
      continue
    }

    if ($argument -ieq "-Replace") {
      if ($i + 1 -ge $Arguments.Count) {
        throw "Missing replacement value after -Replace. Expected format: -Replace=""from=to"""
      }

      if ($Arguments[$i + 1].StartsWith("-")) {
        throw "Missing replacement value after -Replace. Expected format: -Replace=""from=to"""
      }

      $rules += Convert-ReplacementTextToRule -Value $Arguments[$i + 1]
      $i++
      continue
    }

    throw "Unknown argument: $argument"
  }

  if ($rules.Count -eq 0) {
    throw "At least one replacement is required. Example: -Replace=""_=-"""
  }

  return $rules
}

function Convert-NameWithReplacementRules {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Name,

    [Parameter(Mandatory = $true)]
    [object[]]$Rules
  )

  $newName = $Name
  foreach ($rule in $Rules) {
    $newName = $newName.Replace($rule.From, $rule.To)
  }

  return $newName
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

$replacementRules = Get-ReplacementRules -Arguments $RemainingArguments

$root = (Resolve-Path -LiteralPath $Path).Path
if ($TargetType -eq "File") {
  $items = Get-ChildItem -LiteralPath $root -File -Recurse:$Recurse |
    Sort-Object FullName
} else {
  $items = Get-ChildItem -LiteralPath $root -Directory -Recurse:$Recurse |
    Sort-Object FullName -Descending
}

$renamePlan = @()
foreach ($item in $items) {
  $newName = Convert-NameWithReplacementRules -Name $item.Name -Rules $replacementRules
  if ($newName -ceq $item.Name) {
    continue
  }

  $parentPath = if ($TargetType -eq "File") { $item.DirectoryName } else { $item.Parent.FullName }
  $targetPath = Join-Path $parentPath $newName
  $renamePlan += [pscustomobject]@{
    Item       = $item
    ParentPath = $parentPath
    TempName   = New-TemporaryFileName -Directory $parentPath
    NewName    = $newName
    TargetPath = $targetPath
  }
}

if ($renamePlan.Count -eq 0) {
  Write-Host "No file names need changes." -ForegroundColor Yellow
  return
}

$collisions = $renamePlan |
  Group-Object { "{0}`0{1}" -f $_.ParentPath, $_.NewName.ToUpperInvariant() } |
  Where-Object { $_.Count -gt 1 }

if ($collisions.Count -gt 0) {
  Write-Error "Rename collision detected after applying replacements. No files were renamed."
  foreach ($collision in $collisions) {
    Write-Error ("Target collision: {0}" -f $collision.Group[0].TargetPath)
    foreach ($item in $collision.Group) {
      Write-Error ("  Source: {0}" -f $item.Item.FullName)
    }
  }
  exit 1
}

foreach ($item in $renamePlan) {
  $existingTarget = Get-Item -LiteralPath $item.TargetPath -ErrorAction SilentlyContinue
  if ($null -ne $existingTarget -and !(Test-SameFilesystemPath -Left $existingTarget.FullName -Right $item.Item.FullName)) {
    Write-Error ("Target already exists, skipping all renames: {0}" -f $item.TargetPath)
    exit 1
  }
}

if ($TargetType -eq "File") {
  foreach ($item in $renamePlan) {
    if ($PSCmdlet.ShouldProcess($item.Item.FullName, "Rename to temporary name $($item.TempName)")) {
      Rename-Item -LiteralPath $item.Item.FullName -NewName $item.TempName
    }
  }

  foreach ($item in $renamePlan) {
    $tempPath = Join-Path $item.ParentPath $item.TempName
    if ($PSCmdlet.ShouldProcess($tempPath, "Rename to $($item.NewName)")) {
      Rename-Item -LiteralPath $tempPath -NewName $item.NewName
      Write-Host ("Renamed: {0} -> {1}" -f $item.Item.Name, $item.NewName)
    }
  }
} else {
  foreach ($item in $renamePlan) {
    if ($PSCmdlet.ShouldProcess($item.Item.FullName, "Rename to temporary name $($item.TempName)")) {
      Rename-Item -LiteralPath $item.Item.FullName -NewName $item.TempName
    }

    $tempPath = Join-Path $item.ParentPath $item.TempName
    if ($PSCmdlet.ShouldProcess($tempPath, "Rename to $($item.NewName)")) {
      Rename-Item -LiteralPath $tempPath -NewName $item.NewName
      Write-Host ("Renamed: {0} -> {1}" -f $item.Item.Name, $item.NewName)
    }
  }
}

if ($WhatIfPreference) {
  Write-Host ("Preview Complete! Would rename {0} {1}(s)." -f $renamePlan.Count, $TargetType.ToLowerInvariant()) -ForegroundColor Yellow
} else {
  Write-Host ("Task Complete! Renamed {0} {1}(s)." -f $renamePlan.Count, $TargetType.ToLowerInvariant()) -ForegroundColor Green
}
