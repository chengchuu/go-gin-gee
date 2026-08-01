# PowerShell
# Regression checks for batch-uppercase-filenames.ps1.
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\test-batch-uppercase-filenames.ps1"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -File "scripts/test-batch-uppercase-filenames.ps1"

[Console]::InputEncoding  = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ErrorActionPreference = 'Stop'

function Assert-Exists {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  if (!(Test-Path -LiteralPath $Path -PathType Leaf)) {
    throw "Expected file does not exist: $Path"
  }
}

function Assert-NotExists {
  param(
    [Parameter(Mandatory = $true)]
    [string]$Path
  )

  if (Test-Path -LiteralPath $Path) {
    throw "Unexpected file exists: $Path"
  }
}

$scriptPath = Join-Path $PSScriptRoot "batch-uppercase-filenames.ps1"
$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("uppercase-file-test-{0}" -f ([guid]::NewGuid().ToString('N')))
$childDir = Join-Path $testRoot "child"

try {
  New-Item -ItemType Directory -Path $testRoot | Out-Null
  New-Item -ItemType Directory -Path $childDir | Out-Null

  New-Item -ItemType File -Path (Join-Path $testRoot "abc-123.txt") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "NAME-OK.TXT") | Out-Null
  New-Item -ItemType File -Path (Join-Path $childDir "nested.txt") | Out-Null

  & $scriptPath -Path $testRoot

  Assert-Exists (Join-Path $testRoot "ABC-123.TXT")
  Assert-Exists (Join-Path $testRoot "NAME-OK.TXT")
  Assert-Exists (Join-Path $childDir "nested.txt")
  Assert-NotExists (Join-Path $testRoot "abc-123.txt")

  & $scriptPath -Path $testRoot -Recurse

  Assert-Exists (Join-Path $childDir "NESTED.TXT")
  Assert-NotExists (Join-Path $childDir "nested.txt")

  & $scriptPath -Path $testRoot -Recurse

  New-Item -ItemType File -Path (Join-Path $testRoot "preview.txt") | Out-Null
  & $scriptPath -Path $testRoot -WhatIf

  Assert-Exists (Join-Path $testRoot "preview.txt")
  Assert-NotExists (Join-Path $testRoot "PREVIEW.TXT")

  Write-Host "All regression checks passed." -ForegroundColor Green
} finally {
  if (Test-Path -LiteralPath $testRoot) {
    Remove-Item -LiteralPath $testRoot -Recurse -Force
  }
}
