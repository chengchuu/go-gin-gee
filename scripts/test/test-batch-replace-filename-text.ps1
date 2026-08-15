# PowerShell
# Regression checks for batch-replace-filename-text.ps1.
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\test\test-batch-replace-filename-text.ps1"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -File "scripts/test/test-batch-replace-filename-text.ps1"

[Console]::InputEncoding  = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ErrorActionPreference = 'Stop'

function Assert-DirectoryHasName {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Directory,

    [Parameter(Mandatory = $true)]
    [string]$Name
  )

  $names = Get-ChildItem -LiteralPath $Directory -Force | ForEach-Object { $_.Name }
  if (!($names -ccontains $Name)) {
    throw "Expected directory '$Directory' to contain entry named '$Name'"
  }
}

function Assert-DirectoryDoesNotHaveName {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Directory,

    [Parameter(Mandatory = $true)]
    [string]$Name
  )

  $names = Get-ChildItem -LiteralPath $Directory -Force | ForEach-Object { $_.Name }
  if ($names -ccontains $Name) {
    throw "Unexpected directory entry '$Name' exists in '$Directory'"
  }
}

function Assert-Throws {
  param(
    [Parameter(Mandatory = $true)]
    [scriptblock]$ScriptBlock,

    [Parameter(Mandatory = $true)]
    [string]$Message
  )

  try {
    & $ScriptBlock
  } catch {
    return
  }

  throw $Message
}

$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) "batch-replace-filename-text.ps1"
$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("replace-filename-test-{0}" -f ([guid]::NewGuid().ToString('N')))
$childDir = Join-Path $testRoot "child"
$dirModeRoot = Join-Path $testRoot "dir_mode"
$dirModeChild = Join-Path $dirModeRoot "child_dir"
$hyphenRoot = Join-Path $testRoot "hyphen"

try {
  New-Item -ItemType Directory -Path $testRoot | Out-Null
  New-Item -ItemType Directory -Path $childDir | Out-Null
  New-Item -ItemType Directory -Path $dirModeChild | Out-Null

  New-Item -ItemType File -Path (Join-Path $testRoot "my_file_name.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "abc*topic.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "already-ok.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $childDir "child_file_name.md") | Out-Null

  & $scriptPath -Path $testRoot -Replace="_=-" -Replace="*=-" -Replace="abc=xyz"

  Assert-DirectoryHasName -Directory $testRoot -Name "my-file-name.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "xyz-topic.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "already-ok.md"
  Assert-DirectoryHasName -Directory $childDir -Name "child_file_name.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "my_file_name.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "abc*topic.md"

  & $scriptPath -Path $testRoot -Replace "_=-" -Recurse

  Assert-DirectoryHasName -Directory $childDir -Name "child-file-name.md"
  Assert-DirectoryDoesNotHaveName -Directory $childDir -Name "child_file_name.md"

  New-Item -ItemType File -Path (Join-Path $testRoot "preview_file.md") | Out-Null
  & $scriptPath -Path $testRoot -Replace "_=-" -WhatIf

  Assert-DirectoryHasName -Directory $testRoot -Name "preview_file.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "preview-file.md"

  & $scriptPath -Path $testRoot -Replace "_=-" -TargetType Directory -Recurse

  $renamedDirModeRoot = Join-Path $testRoot "dir-mode"
  Assert-DirectoryHasName -Directory $testRoot -Name "dir-mode"
  Assert-DirectoryHasName -Directory $renamedDirModeRoot -Name "child-dir"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "dir_mode"

  New-Item -ItemType Directory -Path $hyphenRoot | Out-Null
  New-Item -ItemType File -Path (Join-Path $hyphenRoot "project-topic.md") | Out-Null
  & $scriptPath -Path $hyphenRoot -Replace "-=_"

  Assert-DirectoryHasName -Directory $hyphenRoot -Name "project_topic.md"
  Assert-DirectoryDoesNotHaveName -Directory $hyphenRoot -Name "project-topic.md"

  Assert-Throws -ScriptBlock {
    & $scriptPath -Path $testRoot -Replace -TargetType File
  } -Message "Expected missing replacement value to throw."

  Write-Host "All regression checks passed." -ForegroundColor Green
} finally {
  if (Test-Path -LiteralPath $testRoot) {
    Remove-Item -LiteralPath $testRoot -Recurse -Force
  }
}
