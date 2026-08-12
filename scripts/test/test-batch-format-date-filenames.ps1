# PowerShell
# Regression checks for batch-format-date-filenames.ps1.
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\test\test-batch-format-date-filenames.ps1"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -File "scripts/test/test-batch-format-date-filenames.ps1"

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

$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) "batch-format-date-filenames.ps1"
$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("date-filename-test-{0}" -f ([guid]::NewGuid().ToString('N')))
$childDir = Join-Path $testRoot "child"
$dirModeRoot = Join-Path $testRoot "24-0401_notes"
$dirModeChild = Join-Path $dirModeRoot "23-0102-child"

try {
  New-Item -ItemType Directory -Path $testRoot | Out-Null
  New-Item -ItemType Directory -Path $childDir | Out-Null
  New-Item -ItemType Directory -Path $dirModeChild | Out-Null

  New-Item -ItemType File -Path (Join-Path $testRoot "26-0802-project-topic.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "25-0330_Video.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "20-0818_TypeScript_Project.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "26-0230-invalid.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "20260802-already-formatted.md") | Out-Null
  New-Item -ItemType File -Path (Join-Path $childDir "25-0105-child-note.md") | Out-Null

  & $scriptPath -Path $testRoot

  Assert-DirectoryHasName -Directory $testRoot -Name "20260802-project-topic.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "20250330_Video.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "20200818_TypeScript_Project.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "26-0230-invalid.md"
  Assert-DirectoryHasName -Directory $testRoot -Name "20260802-already-formatted.md"
  Assert-DirectoryHasName -Directory $childDir -Name "25-0105-child-note.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "26-0802-project-topic.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "25-0330_Video.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "20-0818_TypeScript_Project.md"

  & $scriptPath -Path $testRoot -Recurse

  Assert-DirectoryHasName -Directory $childDir -Name "20250105-child-note.md"
  Assert-DirectoryDoesNotHaveName -Directory $childDir -Name "25-0105-child-note.md"

  New-Item -ItemType File -Path (Join-Path $testRoot "27-1201-preview.md") | Out-Null
  & $scriptPath -Path $testRoot -WhatIf

  Assert-DirectoryHasName -Directory $testRoot -Name "27-1201-preview.md"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "20271201-preview.md"

  & $scriptPath -Path $testRoot -TargetType Directory -Recurse

  $renamedDirModeRoot = Join-Path $testRoot "20240401_notes"
  Assert-DirectoryHasName -Directory $testRoot -Name "20240401_notes"
  Assert-DirectoryHasName -Directory $renamedDirModeRoot -Name "20230102-child"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "24-0401_notes"

  Write-Host "All regression checks passed." -ForegroundColor Green
} finally {
  if (Test-Path -LiteralPath $testRoot) {
    Remove-Item -LiteralPath $testRoot -Recurse -Force
  }
}
