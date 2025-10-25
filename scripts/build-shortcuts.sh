#!/bin/bash
# shortcuts

## scripts

### batch-git-pull

GOOS=darwin GOARCH=amd64 go build -o dist/batch-git-pull-mac-darwin-amd64 scripts/batch-git-pull/main.go
GOOS=linux GOARCH=amd64 go build -o dist/batch-git-pull-linux-amd64 scripts/batch-git-pull/main.go
GOOS=windows GOARCH=amd64 go build -o dist/batch-git-pull-windows-amd64-v1.exe scripts/batch-git-pull/main.go

### change-git-user

GOOS=darwin GOARCH=amd64 go build -o dist/change-git-user-mac-darwin-amd64 scripts/change-git-user/main.go
GOOS=linux GOARCH=amd64 go build -o dist/change-git-user-linux-amd64 scripts/change-git-user/main.go
GOOS=windows GOARCH=amd64 go build -o dist/change-git-user-windows-amd64-v4.exe scripts/change-git-user/main.go

### transfer-files-to-json

GOOS=darwin GOARCH=amd64 go build -o dist/transfer-files-to-json-mac-darwin-amd64 scripts/transfer-files-to-json/main.go
GOOS=linux GOARCH=amd64 go build -o dist/transfer-files-to-json-linux-amd64 scripts/transfer-files-to-json/main.go

### convert-files-to-json

GOOS=darwin GOARCH=amd64 go build -o dist/convert-files-to-json-mac-darwin-amd64 scripts/convert-files-to-json/main.go

### convert-typedoc-to-markdown

GOOS=darwin GOARCH=amd64 go build -o dist/convert-typedoc-to-markdown-mac-darwin-amd64 scripts/convert-typedoc-to-markdown/main.go
GOOS=linux GOARCH=amd64 go build -o dist/convert-typedoc-to-markdown-linux-amd64 scripts/convert-typedoc-to-markdown/main.go
GOOS=windows GOARCH=amd64 go build -o dist/convert-typedoc-to-markdown-windows-amd64.exe scripts/convert-typedoc-to-markdown/main.go

### convert-markdown-to-typedoc

GOOS=darwin GOARCH=amd64 go build -o dist/convert-markdown-to-typedoc-mac-darwin-amd64 scripts/convert-markdown-to-typedoc/main.go
GOOS=linux GOARCH=amd64 go build -o dist/convert-markdown-to-typedoc-linux-amd64 scripts/convert-markdown-to-typedoc/main.go
GOOS=windows GOARCH=amd64 go build -o dist/convert-markdown-to-typedoc-windows-amd64.exe scripts/convert-markdown-to-typedoc/main.go

### eslint-files

GOOS=darwin GOARCH=amd64 go build -o dist/eslint-files-mac-darwin-amd64 scripts/eslint-files/main.go
GOOS=linux GOARCH=amd64 go build -o dist/eslint-files-linux-amd64 scripts/eslint-files/main.go

## Check Memory Usage 

# https://linuxhint.com/check_memory_usage_process_linux/
ps -o pid,user,%mem,command ax | sort -b -k3 -r

## Supervisor

brew install supervisor

"
To restart supervisor after an upgrade:
  brew services restart supervisor
Or, if you don't want/need a background service you can just run:
  /usr/local/opt/supervisor/bin/supervisord -c /usr/local/etc/supervisord.conf --nodaemon

/usr/local/opt/supervisor/bin/supervisord -v

cat /usr/local/etc/supervisord.conf

[include]
files = /usr/local/etc/supervisor.d/*.ini

/Users/mazey/Desktop/supervisor/b-startup.ini
"

## Nginx

"
where nginx
brew services restart supervisor
brew services list
nginx -t
"
