# pwsh -NoProfile -File scripts/powershell/test/test-batch-remove-empty-directories.ps1
$ErrorActionPreference = 'Stop'
$scriptPath = Join-Path (Split-Path -Parent $PSScriptRoot) 'batch-remove-empty-directories.ps1'
$engine = (Get-Process -Id $PID).Path
$root = Join-Path ([System.IO.Path]::GetTempPath()) ('empty-directory-test-' + [guid]::NewGuid().ToString('N'))

function New-FixtureDirectory {
  param([string]$RelativePath)
  return [System.IO.Directory]::CreateDirectory((Join-Path $root $RelativePath)).FullName
}

function Assert-Directory {
  param([string]$Path, [bool]$Exists = $true)
  if ([System.IO.Directory]::Exists($Path) -ne $Exists) { throw "Unexpected directory state: $Path" }
}

function Invoke-Cleanup {
  param([int]$Status, [string[]]$Arguments)
  $ErrorActionPreference = 'Continue'
  $PSNativeCommandUseErrorActionPreference = $false
  $output = & $engine -NoProfile -ExecutionPolicy Bypass -File $scriptPath @Arguments 2>&1
  if ($LASTEXITCODE -ne $Status) { throw "Expected status $Status, got ${LASTEXITCODE}: $output" }
  return ($output | Out-String)
}

try {
  $tree = New-FixtureDirectory 'tree [literal]'
  $empty = New-FixtureDirectory 'tree [literal]/empty'
  $child = New-FixtureDirectory 'tree [literal]/chain/middle/child'
  $middle = Split-Path -Parent $child
  $chain = Split-Path -Parent $middle
  $placeholderChild = New-FixtureDirectory 'tree [literal]/combined/child'
  $combined = Split-Path -Parent $placeholderChild
  $placeholder = New-FixtureDirectory 'tree [literal]/placeholder-only'
  $hidden = New-FixtureDirectory 'tree [literal]/keep-hidden'
  $mixed = New-FixtureDirectory 'tree [literal]/keep-mixed'
  $mixedEmpty = New-FixtureDirectory 'tree [literal]/keep-mixed/empty'
  [System.IO.File]::WriteAllText((Join-Path $tree '.gitkeep'), 'root')
  [System.IO.File]::WriteAllText((Join-Path $combined '.gitkeep'), '')
  [System.IO.File]::WriteAllText((Join-Path $placeholderChild '.DS_Store'), '')
  [System.IO.File]::WriteAllText((Join-Path $placeholder '.gitkeep'), '')
  [System.IO.File]::WriteAllText((Join-Path $hidden '.gitkeep'), '')
  [System.IO.File]::WriteAllText((Join-Path $hidden '.keep-me'), 'hidden')
  [System.IO.File]::WriteAllText((Join-Path $mixed 'keep.txt'), 'keep')

  $preview = Invoke-Cleanup 0 @('-Path', $tree)
  if (!$preview.Contains("Would delete directory: $empty") -or $preview.Contains($child)) {
    throw "Unexpected nonrecursive preview: $preview"
  }
  Assert-Directory $empty
  $preview = Invoke-Cleanup 0 @('-Path', $tree, '-Recurse')
  if (!$preview.Contains("Would delete directory: $child") -or
      ($preview -split '\r?\n') -ccontains "Would delete directory: $middle") {
    throw "Unexpected recursive preview: $preview"
  }
  $null = Invoke-Cleanup 1 @('-Path', $tree, '-IncludeEmptySubtrees', '-Delete')
  Assert-Directory $empty
  $preview = Invoke-Cleanup 0 @('-Path', $tree, '-Recurse', '-IncludeEmptySubtrees')
  $lines = @($preview -split '\r?\n')
  $childIndex = [array]::IndexOf($lines, "Would delete directory: $child")
  $middleIndex = [array]::IndexOf($lines, "Would delete directory: $middle")
  $chainIndex = [array]::IndexOf($lines, "Would delete directory: $chain")
  $emptyIndex = [array]::IndexOf($lines, "Would delete directory: $empty")
  if ($childIndex -lt 0 -or $middleIndex -le $childIndex -or $chainIndex -le $middleIndex -or
      $emptyIndex -le $middleIndex -or $preview.Contains($combined)) {
    throw "Unexpected subtree preview: $preview"
  }

  $null = Invoke-Cleanup 0 @('-Path', $tree, '-IncludePlaceholderFiles', '-Delete')
  Assert-Directory $placeholder $false
  Assert-Directory $combined
  Assert-Directory $placeholderChild
  Assert-Directory $empty $false
  Assert-Directory $child

  $options = @('-Path', $tree, '-Recurse', '-IncludePlaceholderFiles', '-IncludeEmptySubtrees')
  $preview = Invoke-Cleanup 0 ($options + @('-Delete', '-WhatIf'))
  foreach ($file in @((Join-Path $combined '.gitkeep'), (Join-Path $placeholderChild '.DS_Store'))) {
    if (!$preview.Contains($file) -or ![System.IO.File]::Exists($file)) { throw "Incorrect placeholder preview: $file" }
  }
  Assert-Directory $chain
  $null = Invoke-Cleanup 0 ($options + '-Delete')
  foreach ($path in @($chain, $combined, $mixedEmpty)) { Assert-Directory $path $false }
  foreach ($path in @($tree, $hidden, $mixed)) { Assert-Directory $path }
  if ([System.IO.File]::ReadAllText((Join-Path $tree '.gitkeep')) -cne 'root' -or
      [System.IO.File]::ReadAllText((Join-Path $hidden '.keep-me')) -cne 'hidden' -or
      ![System.IO.File]::Exists((Join-Path $hidden '.gitkeep')) -or
      [System.IO.File]::ReadAllText((Join-Path $mixed 'keep.txt')) -cne 'keep') {
    throw 'Cleanup changed noncandidate contents.'
  }
  $emptyRoot = New-FixtureDirectory 'empty-root'
  $null = Invoke-Cleanup 0 @('-Path', $emptyRoot, '-Delete')
  Assert-Directory $emptyRoot
  $null = Invoke-Cleanup 1 @('-Path', (Join-Path $root 'missing'), '-Delete')
  $null = Invoke-Cleanup 1 @('-Path', (Join-Path $mixed 'keep.txt'), '-Delete')

  if ($env:OS -ne 'Windows_NT') {
    $outside = New-FixtureDirectory 'outside/empty'
    $links = New-FixtureDirectory 'tree [literal]/keep-links'
    $symlink = Join-Path $links 'linked-directory'
    $null = [System.IO.Directory]::CreateSymbolicLink($symlink, (Split-Path -Parent $outside))
    $null = [System.IO.File]::CreateSymbolicLink((Join-Path $links '.gitkeep'), (Join-Path $tree '.gitkeep'))
    $null = Invoke-Cleanup 0 ($options + '-Delete')
    Assert-Directory $outside
    Assert-Directory $links
    Assert-Directory $symlink
    $null = Invoke-Cleanup 1 @('-Path', $symlink, '-Recurse', '-Delete')

    if ((& id -u) -ne '0') {
      $failure = New-FixtureDirectory 'scan-failure'
      $blocked = New-FixtureDirectory 'scan-failure/blocked'
      $safe = New-FixtureDirectory 'scan-failure/must-stay'
      try {
        & chmod 000 $blocked
        if ($LASTEXITCODE -ne 0) { throw 'Could not restrict test fixture permissions.' }
        $null = Invoke-Cleanup 1 @('-Path', $failure, '-Recurse', '-Delete')
        Assert-Directory $safe
      } finally {
        & chmod 700 $blocked
      }

      $deleteFailure = New-FixtureDirectory 'delete-failure'
      $mustStay = New-FixtureDirectory 'delete-failure/must-stay'
      try {
        & chmod 500 $deleteFailure
        if ($LASTEXITCODE -ne 0) { throw 'Could not restrict deletion permissions.' }
        $null = Invoke-Cleanup 1 @('-Path', $deleteFailure, '-Delete')
        Assert-Directory $mustStay
      } finally {
        & chmod 700 $deleteFailure
      }
    }
  }
  Write-Host 'All empty directory regression checks passed.'
} finally {
  if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
