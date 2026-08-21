# PowerShell
# Regression checks for batch-convert-filename-case.ps1.
#
# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\test-batch-convert-filename-case.ps1"
#
# PowerShell 7/macOS/Linux
# pwsh -NoProfile -File "scripts/test-batch-convert-filename-case.ps1"

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

$scriptPath = Join-Path $PSScriptRoot "batch-convert-filename-case.ps1"
$testRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("filename-case-test-{0}" -f ([guid]::NewGuid().ToString('N')))
$childDir = Join-Path $testRoot "child"

try {
  New-Item -ItemType Directory -Path $testRoot | Out-Null
  New-Item -ItemType Directory -Path $childDir | Out-Null

  New-Item -ItemType File -Path (Join-Path $testRoot "abc-123.txt") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "NAME-OK.TXT") | Out-Null
  New-Item -ItemType File -Path (Join-Path $testRoot "xtm.dvd-halfcd2.mkv") | Out-Null
  New-Item -ItemType File -Path (Join-Path $childDir "nested.txt") | Out-Null

  & $scriptPath -Path $testRoot -Mode Upper

  Assert-Exists (Join-Path $testRoot "ABC-123.TXT")
  Assert-Exists (Join-Path $testRoot "NAME-OK.TXT")
  Assert-Exists (Join-Path $testRoot "XTM.DVD-HALFCD2.MKV")
  Assert-Exists (Join-Path $childDir "nested.txt")
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "abc-123.txt"

  & $scriptPath -Path $testRoot -Mode Upper -Recurse

  Assert-Exists (Join-Path $childDir "NESTED.TXT")
  Assert-DirectoryDoesNotHaveName -Directory $childDir -Name "nested.txt"

  & $scriptPath -Path $testRoot -Mode Upper -Recurse

  New-Item -ItemType File -Path (Join-Path $testRoot "MIXED-Case.MKV") | Out-Null
  & $scriptPath -Path $testRoot -Mode Lower

  Assert-DirectoryHasName -Directory $testRoot -Name "mixed-case.mkv"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "MIXED-Case.MKV"

  New-Item -ItemType File -Path (Join-Path $testRoot "preview.txt") | Out-Null
  & $scriptPath -Path $testRoot -Mode Upper -WhatIf

  Assert-DirectoryHasName -Directory $testRoot -Name "preview.txt"
  Assert-DirectoryDoesNotHaveName -Directory $testRoot -Name "PREVIEW.TXT"

  Write-Host "All regression checks passed." -ForegroundColor Green
} finally {
  if (Test-Path -LiteralPath $testRoot) {
    Remove-Item -LiteralPath $testRoot -Recurse -Force
  }
}
