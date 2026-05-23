# Windows GitBash
# powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\batch-rename-files.ps1"
$path   = "E:\VIDEO"
$prefix = "VID"
$start  = 1
$exts   = 'mp4','mov','mkv','avi','wmv','flv','webm','m4v','mpeg','mpg','3gp','ts'

$files = Get-ChildItem -LiteralPath $path -File |
  Where-Object { $exts -contains $_.Extension.TrimStart('.').ToLower() } |
  Sort-Object LastWriteTime, Name

# Pass 1: temp unique names
$map = @()
$n = 0
foreach ($f in $files) {
  $tmp = "__tmp_rename_{0}_{1}{2}" -f ([guid]::NewGuid().ToString('N')), $n, $f.Extension
  Rename-Item -LiteralPath $f.FullName -NewName $tmp
  $map += [pscustomobject]@{ TempName = $tmp; Ext = $f.Extension }
  $n++
}

# Pass 2: final names VID0001... (preserve original extension casing)
$i = $start
foreach ($m in $map) {
  $new = "{0}{1:D4}{2}" -f $prefix, $i, $m.Ext
  Rename-Item -LiteralPath (Join-Path $path $m.TempName) -NewName $new
  $i++
}
