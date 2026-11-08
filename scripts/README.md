# Scripts

Bash utilities are in `scripts/bash/`, with their tests in `scripts/bash/test/`.
PowerShell utilities are in `scripts/powershell/`, with their regression tests in
`scripts/powershell/test/`. Run the examples below from the repository root.

- [Fetch Complete Remote Branch History](#fetch-complete-remote-branch-history)
- [Filename Batch Rename Helpers](#filename-batch-rename-helpers)
  - [Other Filename Helpers](#other-filename-helpers)
  - [Pad Numeric Filename Prefixes](#pad-numeric-filename-prefixes)
- [Remove Orphan Lyrics](#remove-orphan-lyrics)
- [Extract Missing Files from Nginx Errors](#extract-missing-files-from-nginx-errors)
- [Hanle Nginx Log](#hanle-nginx-log)
- [Check Blog](#check-blog)
- [Change Git Name and Email for Different Projects](#change-git-name-and-email-for-different-projects)
- [`git pull` All Projects in a Folder](#git-pull-all-projects-in-a-folder)
- [Consolidate Designated Files/Folders and Execute Customized ESLint Commands](#consolidate-designated-filesfolders-and-execute-customized-eslint-commands)
- [Calculate Days Between Two Dates (datediff)](#calculate-days-between-two-dates-datediff)

## Fetch Complete Remote Branch History

Fetch every branch from a repository's existing `origin`, including historical
file contents omitted by a partial clone:

```bash
bash scripts/bash/fetch-complete-branches.sh --path "/Users/Example"
```

Requires Git 2.36+ and Bash 3.2+. This can download substantial data. The script
refreshes `origin/*`, expands shallow history when needed, and downloads missing
objects in batches rather than repeating a full fetch. It verifies all locally
referenced objects before removing origin's partial-clone settings. Future ordinary
fetches track all origin branches without filtering. Local branches and working
files are preserved; branches are not checked out or merged. Deleted remote refs
are not pruned. Tags are not explicitly fetched, and Git LFS payloads and submodule
repositories are excluded. Run without concurrent Git operations.

If fetching or verification fails, the script exits nonzero and retains the
partial-clone settings. Downloaded objects and successful ref updates may remain.
Missing objects reachable only from other remotes or local history are requested
from origin as well; the command fails if origin cannot supply them. Invalid
command usage returns `2`.

## Filename Batch Rename Helpers

### Other Filename Helpers

Windows 10 PowerShell:

```powershell
$PathC = "C:\Directory\Path"
$TargetTypeC = "Directory"
$TargetTypeC = "File"

powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-convert-filename-case.ps1" -Path $PathC -Recurse -TargetType $TargetTypeC -Mode Upper
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-replace-filename-text.ps1" -Path $PathC -Recurse -TargetType $TargetTypeC `
  -Replace " =-" -Replace "S-=S_" -Replace "-V=_V" -Replace "-202=_202"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-pad-filenames.ps1" -Path $PathC -Separator S -Length 3
```

```powershell
$PathB = "C:\Directory\Path"
$TargetTypeB = "File"
$TargetTypeB = "Directory"

powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-convert-filename-case.ps1" -Path $PathB -Recurse -TargetType $TargetTypeB
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-replace-filename-text.ps1" -Path $PathB -Recurse -TargetType $TargetTypeB `
  -Replace "_=-" `
  -Replace " ="

powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-replace-filename-text.ps1" -Path $PathB -Recurse -TargetType $TargetTypeB -Replace "__=_"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-replace-filename-text.ps1" -Path $PathB -Recurse -TargetType $TargetTypeB -Replace "PLACEHOLDER="

$EnDashRuleB = '_{0}_=_' -f [char]0x2013
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-replace-filename-text.ps1" -Path $PathB -Recurse -TargetType $TargetTypeB -Replace $EnDashRuleB

powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-format-date-filenames.ps1" -Path $PathB -Recurse -TargetType $TargetTypeB
```

Normalize immediate child directory names:

```powershell
$PathA = "C:\Directory\Path"
$TargetTypeA = "Directory"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-convert-filename-case.ps1" -Path $PathA -TargetType $TargetTypeA -Mode Upper
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-replace-filename-text.ps1" -Path $PathA -TargetType $TargetTypeA -Replace "_=-" -Replace " =-"
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-format-date-filenames.ps1" -Path $PathA -TargetType $TargetTypeA
```

macOS:

```bash
pwsh -NoProfile -ExecutionPolicy Bypass -File "./scripts/powershell/batch-convert-filename-case.ps1" -Path "/Users/Path" -Recurse
pwsh -NoProfile -ExecutionPolicy Bypass -File "./scripts/powershell/batch-replace-filename-text.ps1" -Path "/Users/Path" -Replace "-=_" -Recurse
```

### Pad Numeric Filename Prefixes

Use `batch-pad-filenames.ps1` to left-pad ASCII digits before the first literal,
case-sensitive separator. For example, `1S_A.mp3` becomes `001S_A.mp3` with
`-Separator S -Length 3`. Non-numeric prefixes are skipped. Longer prefixes,
extensions, and the remaining filename are preserved. Only files are renamed.

Windows PowerShell preview:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-pad-filenames.ps1" -Path "E:\Music" -Separator S -Length 3 -Recurse -WhatIf
```

macOS preview:

```bash
pwsh -NoProfile -File "./scripts/powershell/batch-pad-filenames.ps1" -Path "/path/to/music" -Separator S -Length 3 -Recurse -WhatIf
```

Remove `-WhatIf` to rename files. `-PadChar` defaults to `0` and accepts one
filename-safe character. Omit `-Recurse` to process immediate files only.
Before any rename, the script checks the entire batch for duplicate targets
(case-insensitively) and existing targets. A conflict reports the paths and exits
with status `1`, including during previews. Runtime failures also exit with `1`;
renames completed before a runtime failure are not rolled back.

## Remove Orphan Lyrics

`batch-remove-orphan-lrc.ps1` permanently deletes LRC files without a matching
audio filename in the same directory. Matching ignores case and the final
extension, but preserves spaces and punctuation. Audio contents are not inspected.
Supported extensions: `mp3`, `ogg`, `oga`, `opus`, `flac`, `wav`, `wave`, `m4a`,
`aac`, `wma`, `aif`, `aiff`, `aifc`, `ape`, `alac`, `wv`, `dsf`, and `dff`.

Preview in Windows PowerShell:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\scripts\powershell\batch-remove-orphan-lrc.ps1" -Path "E:\Musics" -Recurse -WhatIf
```

Preview on macOS:

```bash
pwsh -NoProfile -File "./scripts/powershell/batch-remove-orphan-lrc.ps1" -Path "/path/to/music" -Recurse -WhatIf
```

Remove `-WhatIf` to permanently delete the listed files, bypassing Trash/Recycle Bin.
Omit `-Recurse` to inspect only immediate files. Hidden files are included.
Each directory is matched independently; audio in another directory does not keep
an LRC file. Directory links are not traversed, a linked root is rejected, and LRC
symbolic links are left untouched. Audio files and directories are never deleted.

The full scan finishes before deletion begins. Each deleted path and a final count
are logged. Success (including no orphans) returns `0`; scanning or deletion errors
return `1`. Deletion is not atomic: runtime failures can leave partial progress.
Do not change the directory contents while cleanup is running.

## Extract Missing Files from Nginx Errors

Extract unique filesystem paths from exact `open() ... failed (2: No such file or directory)` errors:

```bash
go run ./scripts/extract-nginx-missing-files -in nginx-error.log -out missing-files.log
```

```bash
cat nginx-error.log | go run ./scripts/extract-nginx-missing-files
```

Naturally sort paths and include entries from an inclusive local timestamp through the command start time:

```bash
go run ./scripts/extract-nginx-missing-files \
  -in nginx-error.log \
  -out missing-files.log \
  -sort \
  -since="2026/09/09 09:00:00"
```

## Hanle Nginx Log

VSCode Regex:

```regex
^.*?"(?:GET|POST|PUT|DELETE|HEAD|OPTIONS|PATCH)\s+//?assets?([^"\s]+)\s+HTTP/[\d.]+" .*$
```

dedupe-decode:

```bash
go run scripts/dedupe-decode/main.go -in log/in-asset.log -out log/out-asset.log
```

batch-copy-files:

```bash
# Windows GitBash
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "scripts\powershell\batch-copy-files.ps1"
```

## Check Blog

```bash
go run ./scripts/check-web-links \
  -url="https://blog.mazey.net/" \
  -reportPath="log/MAZEY_LINKS_09.log"
```

## Change Git Name and Email for Different Projects

macOS Bash or zsh:

```bash
bash scripts/bash/batch-set-git-identity.sh --path="/Users/X/Web" --username="<your-name>" --useremail="<your-email>"
```

Windows 10 Git Bash:

```bash
bash scripts/bash/batch-set-git-identity.sh --path="C:/Web" --username="<your-name>" --useremail="<your-email>"
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.0.0) | [简体中文](http://blog.mazey.net/2956.html)

## `git pull` All Projects in a Folder

```bash
go run scripts/batch-git-pull/main.go -path="/Users/X/Web"
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.1.0) | [简体中文](http://blog.mazey.net/3035.html)

## Consolidate Designated Files/Folders and Execute Customized ESLint Commands

```bash
go run scripts/eslint-files/main.go -files="file1.js,file2.js" -esConf="custom.eslintrc.js" -esCom="--fix"
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.4.0) | [简体中文](http://blog.mazey.net/4207.html)

## Calculate Days Between Two Dates (datediff)

A small helper to calculate the number of days between two calendar dates. The script expects dates in `YYYY-MM-DD` format and returns the exclusive number of full 24‑hour days between the two midnights (i.e., it does not count both endpoints).

Script location:
- [`scripts/typescript/src/calc-datediff.ts`](./typescript/src/calc-datediff.ts)

Usage examples:

- Using flags:
```bash
node scripts/typescript/src/calc-datediff.ts --start 2025-10-01 --end 2025-10-31
```

- Using positional arguments:
```bash
node scripts/typescript/src/calc-datediff.ts 2025-10-01 2025-10-31
```

Example output:
```
Days between 2025-10-01 and 2025-10-31: 30
```

See the [TypeScript calculator setup](./typescript/README.md#calculate-annualized-return-and-elapsed-days)
for dependency installation and the annualized return calculator.
