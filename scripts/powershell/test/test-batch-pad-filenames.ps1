# pwsh -NoProfile -File scripts/powershell/test/test-batch-pad-filenames.ps1
$ErrorActionPreference = 'Stop'
$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) 'batch-pad-filenames.ps1'
$engine = (Get-Process -Id $PID).Path
$root = Join-Path ([System.IO.Path]::GetTempPath()) ('pad-filenames-test-' + [guid]::NewGuid().ToString('N'))

function New-Fixture {
  param([string]$Directory, [string[]]$Names)
  $null = New-Item -ItemType Directory -Path $Directory -Force
  foreach ($name in $Names) {
    [System.IO.File]::WriteAllText((Join-Path $Directory $name), $name)
  }
}

function Assert-Names {
  param([string]$Directory, [string[]]$Expected)
  $actual = @(Get-ChildItem -LiteralPath $Directory -File | ForEach-Object { $_.Name })
  if (@(Compare-Object $actual $Expected -CaseSensitive).Count) {
    throw "Unexpected filenames in ${Directory}: $actual"
  }
}

function Invoke-Padding {
  param([int]$ExpectedStatus, [string[]]$Arguments)
  # Windows PowerShell surfaces native stderr as error records; inspect the process status instead.
  $ErrorActionPreference = 'Continue'
  $output = & $engine -NoProfile -ExecutionPolicy Bypass -File $scriptPath @Arguments 2>&1
  $ErrorActionPreference = 'Stop'
  if ($LASTEXITCODE -ne $ExpectedStatus) {
    throw "Expected status $ExpectedStatus, got ${LASTEXITCODE}: $output"
  }
}

try {
  $normal = Join-Path $root 'normal'
  $names = @('1S_A.mp3', '12S_A.mp3', '123S_A.mp3', '1234S_A.mp3', 'ABS_A.mp3', 'S_A.mp3', '13s_A.mp3', '12_A.mp3', '1.S', '7S_[A].MP3')
  New-Fixture $normal $names
  $child = Join-Path $normal '2S_directory'
  New-Fixture $child @('2S_A.mp3')
  Invoke-Padding 0 @('-Path', $normal, '-Separator', 'S', '-Length', '3', '-WhatIf', '-Recurse')
  Assert-Names $normal $names
  Assert-Names $child @('2S_A.mp3')
  Invoke-Padding 0 @('-Path', $normal, '-Separator', 'S', '-Length', '3')
  Assert-Names $normal @('001S_A.mp3', '012S_A.mp3', '123S_A.mp3', '1234S_A.mp3', 'ABS_A.mp3', 'S_A.mp3', '13s_A.mp3', '12_A.mp3', '1.S', '007S_[A].MP3')
  Assert-Names $child @('2S_A.mp3')
  Invoke-Padding 0 @('-Path', $normal, '-Separator', 'S', '-Length', '3', '-Recurse')
  Assert-Names $child @('002S_A.mp3')
  if ([System.IO.File]::ReadAllText((Join-Path $normal '001S_A.mp3')) -cne '1S_A.mp3') { throw 'File content changed' }
  Invoke-Padding 0 @('-Path', $normal, '-Separator', 'S', '-Length', '3', '-Recurse')

  $literal = Join-Path $root 'literal'
  New-Fixture $literal @('01[S]_A.mp3', '2[S]_B')
  Invoke-Padding 0 @('-Path', $literal, '-Separator', '[S]', '-Length', '3', '-PadChar', '9')
  Assert-Names $literal @('901[S]_A.mp3', '992[S]_B')

  $numeric = Join-Path $root 'numeric'
  $nonAscii = ([char]0xFF11).ToString() + 'S_A.mp3'
  $long = '123456789012345678901234567890S_A.mp3'
  New-Fixture $numeric @($nonAscii, $long, '2S_moreS.mp3', '01S_A.mp3')
  Invoke-Padding 0 @('-Path', $numeric, '-Separator', 'S', '-Length', '3')
  Assert-Names $numeric @($nonAscii, $long, '002S_moreS.mp3', '001S_A.mp3')

  $collision = Join-Path $root 'collision'
  New-Fixture $collision @('5S_safe.mp3')
  $nested = Join-Path $collision 'nested'
  New-Fixture $nested @('1S_A.mp3', '01S_A.mp3')
  Invoke-Padding 1 @('-Path', $collision, '-Separator', 'S', '-Length', '3', '-Recurse')
  Assert-Names $collision @('5S_safe.mp3')
  Assert-Names $nested @('1S_A.mp3', '01S_A.mp3')
  Invoke-Padding 1 @('-Path', $collision, '-Separator', 'S', '-Length', '3', '-Recurse', '-WhatIf')

  $existing = Join-Path $root 'existing'
  New-Fixture $existing @('1S_A.mp3', '001S_A.mp3', '2S_B.mp3')
  Invoke-Padding 1 @('-Path', $existing, '-Separator', 'S', '-Length', '3')
  Assert-Names $existing @('1S_A.mp3', '001S_A.mp3', '2S_B.mp3')

  $separate = Join-Path $root 'separate'
  foreach ($name in @('a', 'b')) { New-Fixture (Join-Path $separate $name) @('1S_A.mp3') }
  Invoke-Padding 0 @('-Path', $separate, '-Separator', 'S', '-Length', '3', '-Recurse')
  foreach ($name in @('a', 'b')) { Assert-Names (Join-Path $separate $name) @('001S_A.mp3') }

  $empty = Join-Path $root 'empty'
  $null = New-Item -ItemType Directory -Path $empty
  Invoke-Padding 0 @('-Path', $empty, '-Separator', 'S', '-Length', '3')
  Invoke-Padding 1 @('-Path', (Join-Path $root 'missing'), '-Separator', 'S', '-Length', '3')
  Invoke-Padding 1 @('-Path', $empty, '-Separator', 'S', '-Length', '0')
  Invoke-Padding 1 @('-Path', $empty, '-Separator', 'S', '-Length', '3', '-PadChar', '/')
  Write-Host 'All filename padding regression checks passed.'
} finally {
  if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
