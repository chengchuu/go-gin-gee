$sourceRoot = "C:\SourceParentFolder"  # The common root folder of your files
$destination = "D:\TargetFolder"       # Where you want them to go
$fileList = Get-Content "C:\path\to\files_to_copy.txt"

foreach ($filePath in $fileList) {
    $filePath = $filePath.Trim()
    if ([string]::IsNullOrWhiteSpace($filePath)) { continue }

    # Check if source file exists before copying
    if (!(Test-Path -LiteralPath $filePath -PathType Leaf)) {
        Write-Warning "Source file not found, skipping: $filePath"
        continue
    }

    # Calculate the relative path to preserve subdirectories
    $relativePath = $filePath.Replace($sourceRoot, "")
    $targetPath = Join-Path $destination $relativePath
    $targetDir = Split-Path $targetPath

    # Create the folder structure if it doesn't exist
    if (!(Test-Path -LiteralPath $targetDir -PathType Container)) {
        New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
    }

    # Copy the file
    Copy-Item -LiteralPath $filePath -Destination $targetPath -Force
}

Write-Host "Task Complete!" -ForegroundColor Green
