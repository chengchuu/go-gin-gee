# PowerShell
# Regression checks for batch-replace-underscore-filenames.ps1.
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\test-batch-replace-underscore-filenames.ps1"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -File "scripts/test-batch-replace-underscore-filenames.ps1"

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

$scriptPath = Join-Path $PSScriptRoot "batch-replace-underscore-filenames.ps1"
$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("underscore-filename-test-{0}" -f ([guid]::NewGuid().ToString('N')))
$childDir = Join-Path $testRoot "child"

try {
  New-Item -ItemType Directory -Path $testRoot | Out-Null
  New-Item -ItemType Directory -Path $childDir | Out-Null

  New-Item -ItemType File -Path (Join-Path $testRoot "my_file_name.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "already-ok.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $childDir "child_file_name.md") | Out-Null

  & $scriptPath -Path $testRoot

  Assert-DirectoryHasName -Directory $testRoot -Name "my-file-name.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "already-ok.md"
  Assert-DirectoryHasName -Directory $childDir -Name "child_file_name.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "my_file_name.md"

  & $scriptPath -Path $testRoot -Recurse

  Assert-DirectoryHasName -Directory $childDir -Name "child-file-name.md"
  Assert-DirectoryDoesNotHaveName -Directory $childDir -Name "child_file_name.md"

  New-Item -ItemType File -Path (Join-Path $testRoot "preview_file.md") | Out-Null
  & $scriptPath -Path $testRoot -WhatIf

  Assert-DirectoryHasName -Directory $testRoot -Name "preview_file.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "preview-file.md"

  Write-Host "All regression checks passed." -ForegroundColor Green
} finally {
  if (Test-Path -LiteralPath $testRoot) {
    Remove-Item -LiteralPath $testRoot -Recurse -Force
  }
}
