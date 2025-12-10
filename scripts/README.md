# Scripts

- [Scripts](#scripts)
  - [Change Git Name and Email for Different Projects](#change-git-name-and-email-for-different-projects)
  - [`git pull` All Projects in a Folder](#git-pull-all-projects-in-a-folder)
  - [Consolidate Designated Files/Folders and Execute Customized ESLint Commands](#consolidate-designated-filesfolders-and-execute-customized-eslint-commands)
  - [Convert TypeDoc Comments to Markdown](#convert-typedoc-comments-to-markdown)
  - [Convert Markdown to TypeDoc Comments](#convert-markdown-to-typedoc-comments)
  - [Transfer Apple Note Table to Markdown Table](#transfer-apple-note-table-to-markdown-table)
  - [Calculate Days Between Two Dates (datediff)](#calculate-days-between-two-dates-datediff)

## Change Git Name and Email for Different Projects

```bash
go run scripts/change-git-user/main.go -path="/Users/X/Web" -username="YOUR_NAME" -useremail="YOUR_NAME@email.com"
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

## Convert TypeDoc Comments to Markdown

```bash
go run scripts/convert-typedoc-to-markdown/main.go
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.2.0) | [简体中文](http://blog.mazey.net/3494.html#%E6%B3%A8%E9%87%8A%E8%BD%AC_Markdown)

## Convert Markdown to TypeDoc Comments

```bash
go run scripts/convert-markdown-to-typedoc/main.go
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.3.0) | [简体中文](http://blog.mazey.net/3494.html#Markdown_%E8%BD%AC%E6%B3%A8%E9%87%8A)

## Transfer Apple Note Table to Markdown Table

```bash
go run scripts/transfer-notes-to-md-table/main.go
```

## Calculate Days Between Two Dates (datediff)

A small helper to calculate the number of days between two calendar dates. The script expects dates in `YYYY-MM-DD` format and returns the exclusive number of full 24‑hour days between the two midnights (i.e., it does not count both endpoints).

Script location:
- `scripts/datediff/main.go`

Usage examples:

- Using flags:
```bash
go run scripts/datediff/main.go -start 2025-10-01 -end 2025-10-31
```

- Using positional arguments:
```bash
go run scripts/datediff/main.go 2025-10-01 2025-10-31
```

Example output:
```
Days between 2025-10-01 and 2025-10-31: 30
```
