$sourceRoot = "C:\SourceParentFolder"  # The common root folder of your files
$destination = "D:\TargetFolder"       # Where you want them to go
$fileList = Get-Content "C:\path\to\files_to_copy.txt"

foreach ($filePath in $fileList) {
    # Calculate the relative path to preserve subdirectories
    $relativePath = $filePath.Replace($sourceRoot, "")
    $targetPath = Join-Path $destination $relativePath
    $targetDir = Split-Path $targetPath

    # Create the folder structure if it doesn't exist
    if (!(Test-Path $targetDir)) { New-Item -ItemType Directory -Path $targetDir -Force }

    # Copy the file
    Copy-Item -Path $filePath -Destination $targetPath -Force
}

Write-Host "Task Complete!" -ForegroundColor Green
