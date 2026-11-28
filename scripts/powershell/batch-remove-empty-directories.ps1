# Preview empty directories. Add -Delete to remove them permanently.
# pwsh -NoProfile -File scripts/powershell/batch-remove-empty-directories.ps1 -Path ./files -Recurse

[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = 'Medium')]
param(
  [Parameter(Mandatory = $true)]
  [string]$Path,

  [switch]$Recurse,
  [switch]$IncludePlaceholderFiles,
  [switch]$IncludeEmptySubtrees,
  [switch]$Delete
)

$ErrorActionPreference = 'Stop'

function Get-RealDirectory {
  param([string]$LiteralPath)

  $directory = Get-Item -LiteralPath $LiteralPath -Force
  if ($directory -isnot [System.IO.DirectoryInfo] -or
      ($directory.Attributes -band [System.IO.FileAttributes]::ReparsePoint)) {
    throw "Expected a real filesystem directory at '$LiteralPath'."
  }
  return $directory
}

try {
  if ($IncludeEmptySubtrees -and !$Recurse) {
    throw '-IncludeEmptySubtrees requires -Recurse.'
  }

  $root = Get-RealDirectory $Path
  $rootNode = [pscustomobject]@{ Path = $root.FullName; Parent = $null; Eligible = $false; Blocked = $false }
  $pending = [System.Collections.Generic.Queue[object]]::new()
  $scanned = [System.Collections.Generic.List[object]]::new()
  $pending.Enqueue($rootNode)

  # Scan every candidate before making changes. The root is only a traversal entry.
  while ($pending.Count -gt 0) {
    $node = $pending.Dequeue()
    $isRoot = $null -eq $node.Parent
    $null = Get-RealDirectory $node.Path
    $entries = @(Get-ChildItem -LiteralPath $node.Path -Force)
    foreach ($entry in $entries) {
      if ($entry.Attributes -band [System.IO.FileAttributes]::ReparsePoint) {
        $node.Eligible = $false
      } elseif ($entry -is [System.IO.DirectoryInfo]) {
        if ($Recurse -or $isRoot) {
          $pending.Enqueue([pscustomobject]@{
            Path = $entry.FullName
            Parent = $node
            Eligible = $true
            Blocked = $false
            Placeholders = [System.Collections.Generic.List[string]]::new()
          })
        }
        if (!$IncludeEmptySubtrees) { $node.Eligible = $false }
      } elseif (!$isRoot -and $IncludePlaceholderFiles -and
                $entry -is [System.IO.FileInfo] -and @('.gitkeep', '.DS_Store') -ccontains $entry.Name) {
        $node.Placeholders.Add($entry.Name)
      } else {
        $node.Eligible = $false
      }
    }
    if (!$isRoot) { $scanned.Add($node) }
  }

  # Reversing breadth-first traversal visits the deepest directories first. Any ineligible child
  # keeps its parent out of the plan, even if other children can be removed.
  $plan = [System.Collections.Generic.List[object]]::new()
  for ($index = $scanned.Count - 1; $index -ge 0; $index--) {
    $node = $scanned[$index]
    if ($node.Eligible) {
      $plan.Add($node)
    } else {
      $node.Parent.Eligible = $false
    }
  }

  $removed = 0
  foreach ($node in $plan) {
    $action = 'Permanently delete directory'
    if ($node.Placeholders.Count -gt 0) {
      $action += ' and its placeholders (' + ($node.Placeholders -join ', ') + ')'
    }
    if (!$Delete -or $WhatIfPreference) {
      foreach ($name in $node.Placeholders) {
        $filePath = [System.IO.Path]::Combine($node.Path, $name)
        if ($Delete) {
          $null = $PSCmdlet.ShouldProcess($filePath, 'Permanently delete placeholder file')
        } else {
          Write-Host ('Would delete file: {0}' -f $filePath)
        }
      }
      if ($Delete) {
        $null = $PSCmdlet.ShouldProcess($node.Path, 'Permanently delete directory')
      } else {
        Write-Host ('Would delete directory: {0}' -f $node.Path)
      }
      continue
    }
    if ($node.Blocked -or !$PSCmdlet.ShouldProcess($node.Path, $action)) {
      $node.Parent.Blocked = $true
      Write-Host ('Skipped directory: {0}' -f $node.Path)
      continue
    }

    # Check ancestors from the root down before touching anything at this path.
    $ancestors = [System.Collections.Generic.Stack[string]]::new()
    for ($ancestor = $node; $null -ne $ancestor; $ancestor = $ancestor.Parent) {
      $ancestors.Push($ancestor.Path)
    }
    foreach ($directoryPath in $ancestors) { $null = Get-RealDirectory $directoryPath }
    $current = @(Get-ChildItem -LiteralPath $node.Path -Force)
    foreach ($entry in $current) {
      if ($entry -isnot [System.IO.FileInfo] -or
          ($entry.Attributes -band [System.IO.FileAttributes]::ReparsePoint) -or
          !$node.Placeholders.Contains($entry.Name)) {
        throw "Directory contents changed before deletion at '$($node.Path)'."
      }
    }

    foreach ($entry in $current) {
      [System.IO.File]::Delete($entry.FullName)
      Write-Host ('Deleted file: {0}' -f $entry.FullName)
    }
    [System.IO.Directory]::Delete($node.Path, $false)
    $removed++
    Write-Host ('Deleted directory: {0}' -f $node.Path)
  }

  if (!$Delete -or $WhatIfPreference) {
    Write-Host ('Preview complete. Would delete {0} directory(s).' -f $plan.Count)
  } else {
    Write-Host ('Cleanup complete. Deleted {0} directory(s).' -f $removed)
  }
} catch {
  Write-Error -Message $_.Exception.Message -ErrorAction Continue
  exit 1
}
