# PowerShell file.ps1
# Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass; powershell -NoProfile -ExecutionPolicy Bypass -File "C:\Web\web\go-gin-gee\scripts\batch-copy-files.ps1"
# Windows Git Bash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-copy-files.ps1"
# VSCode REGEXP: ^.*?"(?:GET|POST|PUT|DELETE|HEAD|OPTIONS|PATCH)\s+//?assets?([^"\s]+)\s+HTTP/[\d.]+" .*$
# Force UTF-8 console encoding
[Console]::InputEncoding  = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$sourceRoot = "C:\Web\web\archives\asset_frozen"
$destination = "C:\Web\web\archives\asset"

# IMPORTANT: read file list with UTF-8
$fileList = Get-Content "C:\Web\web\go-gin-gee\scripts\dedupe-decode\out1.secret.txt" -Encoding UTF8

foreach ($filePath in $fileList) {
    $filePath = $filePath.Trim()
    if ([string]::IsNullOrWhiteSpace($filePath)) { continue }

    # Normalize leading slash for Windows join behavior
    $relativePath = $filePath.TrimStart('\','/')

    # Build full source/target file paths
    $sourceFile = Join-Path $sourceRoot $relativePath
    $targetPath = Join-Path $destination $relativePath
    $targetDir  = Split-Path -Parent $targetPath

    # Write-Output "sourceFile: $sourceFile"
    # Write-Output "targetPath: $targetPath"
    # Write-Output "targetDir : $targetDir"

    # Check source file exists (use full source path)
    if (!(Test-Path -LiteralPath $sourceFile -PathType Leaf)) {
        Write-Warning "Source file not found, skipping: $sourceFile"
        continue
    }

    # Ensure target directory exists
    if (!(Test-Path -LiteralPath $targetDir -PathType Container)) {
        New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    }

    # Copy source -> target
    Copy-Item -LiteralPath $sourceFile -Destination $targetPath -Force
}

Write-Host "Task Complete!" -ForegroundColor Green
