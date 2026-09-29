# PowerShell
# Convert English characters in file or directory names to uppercase or lowercase for a target path.
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-convert-filename-case.ps1" -Path "E:\VIDEO"
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-convert-filename-case.ps1" -Path "E:\VIDEO" -Mode Lower
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-convert-filename-case.ps1" -Path "E:\VIDEO" -Mode Lower -FileType Video
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-convert-filename-case.ps1" -Path "E:\VIDEO" -Mode Lower -IncludeExtension
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -ExecutionPolicy Bypass -File "scripts/batch-convert-filename-case.ps1" -Path "/path/to/files"
#
# Preview only
# pwsh -NoProfile -ExecutionPolicy Bypass -File "scripts/batch-convert-filename-case.ps1" -Path "/path/to/files" -WhatIf

[CmdletBinding(SupportsShouldProcess = $true)]
param(
  [Parameter(Mandatory = $true)]
  [string]$Path,

  [ValidateSet("Upper", "Lower")]
  [string]$Mode = "Upper",

  [ValidateSet("File", "Directory")]
  [string]$TargetType = "File",

  [ValidateSet("All", "Video", "Image")]
  [string]$FileType = "All",

  [switch]$IncludeExtension,

  [switch]$Recurse
)

[Console]::InputEncoding  = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

function Convert-EnglishCase {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Value,

    [Parameter(Mandatory = $true)]
    [ValidateSet("Upper", "Lower")]
    [string]$Mode
  )

  if ($Mode -eq "Upper") {
    return [regex]::Replace($Value, '[a-z]', {
      param($match)
      return $match.Value.ToUpperInvariant()
    })
  }

  return [regex]::Replace($Value, '[A-Z]', {
    param($match)
    return $match.Value.ToLowerInvariant()
  })
}

function New-TemporaryFileName {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Directory
  )

  do {
    $name = "__tmp_uppercase_{0}.tmp" -f ([guid]::NewGuid().ToString('N'))
    $fullPath = Join-Path $Directory $name
  } while (Test-Path -LiteralPath $fullPath)

  return $name
}

function Get-PresetExtensions {
  param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("All", "Video", "Image")]
    [string]$FileType
  )

  if ($FileType -eq "Video") {
    return @("mp4", "mov", "mkv", "avi", "wmv", "flv", "webm", "m4v", "mpeg", "mpg", "3gp", "ts", "m2ts", "mts", "ogv")
  }

  if ($FileType -eq "Image") {
    return @("jpg", "jpeg", "png", "gif", "webp", "bmp", "tif", "tiff", "heic", "heif", "svg", "avif")
  }

  return @()
}

function Convert-FileNameCase {
  param(
    [Parameter(Mandatory = $true)]
    [System.IO.FileInfo]$Item,

    [Parameter(Mandatory = $true)]
    [ValidateSet("Upper", "Lower")]
    [string]$Mode,

    [switch]$IncludeExtension
  )

  if ($IncludeExtension) {
    return Convert-EnglishCase -Value $Item.Name -Mode $Mode
  }

  return ("{0}{1}" -f (Convert-EnglishCase -Value $Item.BaseName -Mode $Mode), $Item.Extension)
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
if ($TargetType -eq "File") {
  $items = Get-ChildItem -LiteralPath $root -File -Recurse:$Recurse |
    Sort-Object FullName
  $presetExtensions = Get-PresetExtensions -FileType $FileType
  if ($presetExtensions.Count -gt 0) {
    $items = $items |
      Where-Object { $presetExtensions -contains $_.Extension.TrimStart(".").ToLowerInvariant() }
  }
} else {
  $items = Get-ChildItem -LiteralPath $root -Directory -Recurse:$Recurse |
    Sort-Object FullName -Descending
}

$renamePlan = @()
foreach ($item in $items) {
  $newName = if ($TargetType -eq "File") {
    Convert-FileNameCase -Item $item -Mode $Mode -IncludeExtension:$IncludeExtension
  } else {
    Convert-EnglishCase -Value $item.Name -Mode $Mode
  }
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
  Write-Error "Rename collision detected after case conversion. No files were renamed."
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

# Pass 1: move every file to a temporary unique name. This makes case-only
# renames reliable on case-insensitive filesystems.
if ($TargetType -eq "File") {
  foreach ($item in $renamePlan) {
    if ($PSCmdlet.ShouldProcess($item.Item.FullName, "Rename to temporary name $($item.TempName)")) {
      Rename-Item -LiteralPath $item.Item.FullName -NewName $item.TempName
    }
  }

  # Pass 2: move temporary names to final names.
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
  Write-Host ("Preview Complete! Would rename {0} {1}(s) to {2}case." -f $renamePlan.Count, $TargetType.ToLowerInvariant(), $Mode.ToLowerInvariant()) -ForegroundColor Yellow
} else {
  Write-Host ("Task Complete! Renamed {0} {1}(s) to {2}case." -f $renamePlan.Count, $TargetType.ToLowerInvariant(), $Mode.ToLowerInvariant()) -ForegroundColor Green
}
