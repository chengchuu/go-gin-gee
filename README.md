<!-- omit from toc -->
# go-gin-gee

Gee is a project that provides several services for everyday work. The project is based on Gin [1], and follows the ProjectLayout [3] structure. In addition, some daily scripts in the folder `scripts` depend on Script [4], which can be used by the command `run` directly.

<!-- omit from toc -->
## Table of Contents

- [Script Examples](#script-examples)
- [API Examples](#api-examples)
- [Build](#build)
- [Deploy](#deploy)
  - [Supervisor](#supervisor)
- [Docker](#docker)
  - [Build Image](#build-image)
  - [Run](#run)
- [Documentation](#documentation)
- [Contributing](#contributing)
  - [Quick Start](#quick-start)
  - [Details](#details)
- [References](#references)

## Script Examples

1\. Change Git name and email for different projects.

```bash
go run scripts/change-git-user/main.go -path="/Users/X/Web" -username="YOUR_NAME" -useremail="YOUR_NAME@email.com"
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.0.0) | [简体中文](http://blog.mazey.net/2956.html)

2\. `git pull` all projects in a folder.

```bash
go run scripts/batch-git-pull/main.go -path="/Users/X/Web"
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.1.0) | [简体中文](http://blog.mazey.net/3035.html)

3\. Consolidate designated files/folders and execute customized ESLint commands.

```bash
go run scripts/eslint-files/main.go -files="file1.js,file2.js" -esConf="custom.eslintrc.js" -esCom="--fix"
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.4.0) | [简体中文](http://blog.mazey.net/4207.html)

4\. Convert TypeDoc comments to Markdown.

```bash
go run scripts/convert-typedoc-to-markdown/main.go
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.2.0) | [简体中文](http://blog.mazey.net/3494.html#%E6%B3%A8%E9%87%8A%E8%BD%AC_Markdown)

5\. Convert Markdown to TypeDoc comments.

```bash
go run scripts/convert-markdown-to-typedoc/main.go
```

Usage: [English](https://github.com/chengchuu/go-gin-gee/releases/tag/v1.3.0) | [简体中文](http://blog.mazey.net/3494.html#Markdown_%E8%BD%AC%E6%B3%A8%E9%87%8A)

6\. Transfer Apple note table to Markdown table.

```bash
go run scripts/transfer-notes-to-md-table/main.go
```

More in folder [`scripts`](./scripts/README.md).

## API Examples

The base URL for this API is an environment variate `${BASE_URL}`, such as `https://example.com/path`.

<!-- omit from toc -->
### Generate Short Link

Description:

Generate the short link for the original link.

Path: `/api/gee/generate-short-link`

Method: POST

Params:

| Params    | Type     | Description   | Required |
| :-------- | :------- | :------------ | :------- |
| ori_link  | string   | Original Link | Yes      |

Example:

```bash
curl --location --request POST '${BASE_URL}/api/gee/generate-short-link' \
--header 'Content-Type: application/json' \
--data-raw '{
  "ori_link": "http://blog.mazey.net/tiny?ts=654321-221467-f22c24-493220-228e97-d90c73"
}'
```

Returns:

| Params    | Type     | Description | Required |
| :-------- | :--------| :---------- | :------- |
| tiny_link | string   | Short Link  | Yes      |

Example:

Success: Status Code 201

```json
{
  "tiny_link": "${BASE_URL}/t/b"
}
```

Failure: Status Code 400

```json
{
  "code": 400
}
```

## Build

Default:

```bash
go build cmd/api/main.go
```

Linux:

It's usually helpful to run the command `chmod u+x script-name-linux-amd64` if the permission error happens.

```bash
GOOS=linux GOARCH=amd64 go build -o dist/api-linux-amd64 cmd/api/main.go
```

macOS:

```bash
GOOS=darwin GOARCH=amd64 go build -o dist/api-mac-darwin-amd64 cmd/api/main.go
```

Windows:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/api-windows-amd64 cmd/api/main.go
```

## Deploy

Environment Variates:

- `${WECOM_ROBOT_CHECK}`: WeCom Robot Key.
- `${BASE_URL}`: The Base URL for this Service.

### Supervisor

```text
[program:api]
directory=/web/go-gin-gee
command=/web/go-gin-gee/dist/api-linux-amd64 --config-path="/web/go-gin-gee/data/config.json"
autostart=true
autorestart=true
environment=WECOM_ROBOT_CHECK="b2lsjd46-7146-4nv2-8767-86cb0cncjdbe",BASE_URL="https://example.com/path"
```

## Docker

### Build Image

Run `bash ./scripts/docker-build.sh -h` to see the help message.

```text
Usage: docker-build.sh [OPTIONS] [ENV_VARS...]
Build and run a Docker container for the go-gin-gee API.

Options:
  -r, --run     Run the Docker container after building (default)
  -b, --build   Build the Docker image but do not run it
  -h, --help    Print this help message and exit

Environment variables:
  Any additional arguments passed to the script will be passed as environment variables to the Docker container.
```

Usage:

`${RUN_FLAG}` is optional, default is `-r`("RUN"). `${WECOM_ROBOT_CHECK}` is optional. If you don't want to send the message to WeCom Robot, just remove it. `${BASE_URL}` is required. It's the Base URL for this Service.

```bash
bash ./scripts/docker-build.sh ${RUN_FLAG} \
  "WECOM_ROBOT_CHECK=${WECOM_ROBOT_CHECK}" \
  "BASE_URL=${BASE_URL}"
```

Examples:

Example 1: Build and Push

```bash
bash ./scripts/docker-build.sh -b
```

Example 2: Build and Run

```bash
bash ./scripts/docker-build.sh -r \
  "WECOM_ROBOT_CHECK=b2lsjd46-7146-4nv2-8767-86cb0cncjdbe" \
  "BASE_URL=https://example.com/path"
```

### Run

Run `bash ./scripts/docker-run.sh -h` to see the help message.

```text
Usage: docker-run.sh [OPTIONS] IMAGE_TAG [ENV_VARS...]
Run a Docker container from the specified IMAGE_TAG with the specified environment variables.

Options:
  -h, --help    Print this help message and exit

Environment variables:
  Any additional arguments passed to the script will be passed as environment variables to the Docker container.

Note:
  The first argument (IMAGE_TAG) must be the tag name of the Docker image to run.
```

Find the latest image tag name: [Tags](https://hub.docker.com/repository/docker/mazeyqian/go-gin-gee/tags?page=1&ordering=last_updated)

Usage:

```bash
bash ./scripts/docker-run.sh "${DOCKER_HUB_REPOSITORY_TAGNAME}" \
  "WECOM_ROBOT_CHECK=${WECOM_ROBOT_CHECK}" \
  "BASE_URL=${BASE_URL}"
```

Example:

```bash
bash ./scripts/docker-run.sh "docker.io/mazeyqian/go-gin-gee:v20230615221222-api" \
  "WECOM_ROBOT_CHECK=b2lsjd46-7146-4nv2-8767-86cb0cncjdbe" \
  "BASE_URL=https://example.com/path"
```

## Documentation

Download [swag](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@v1.8.12
```

Generate:

```bash
swag init --dir cmd/api --parseDependency --output docs
```

Make sure your GO Path is on the PATH environment variable `export PATH=$(go env GOPATH)/bin:$PATH` if the following error occurs `command not found: swag`.

Run and visit: <http://localhost:3000/docs/index.html>

## Contributing

### Quick Start

```bash
git clone https://github.com/chengchuu/go-gin-gee.git
go mod download
go run scripts/init/main.go
go run cmd/api/main.go --config-path="data/config.dev.json"
```

### Details

Download Project:

```bash
git clone https://github.com/chengchuu/go-gin-gee.git
```

Download modules:

```bash
go mod download
```

If `i/o timeout`, run the command to replace the proxy:

```bash
go env -w GOPROXY=https://goproxy.cn
```

To disable the proxy completely and download modules directly:

```bash
go env -w GOPROXY=direct
```

To reset to Go's default proxy settings:

```bash
go env -w GOPROXY=https://proxy.golang.org,direct
```

It's necessary to run the command `go run scripts/init/main.go` when serving the project first.

Serve:

```bash
go run cmd/api/main.go --config-path="data/config.dev.json"
```

Restart:

```bash
go run scripts/restart/main.go
```

Visit: <http://127.0.0.1:3000/api/ping>.

```text
pong/v1.0.0/2022-09-29 04:52:43
```

## References

1. [Gin Web Framework](https://github.com/gin-gonic/gin)
2. [lo - Iterate over slices, maps, channels...](https://pkg.go.dev/github.com/samber/lo)
3. [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
4. [script](https://pkg.go.dev/github.com/bitfield/script)
5. [go-rest-template](https://github.com/antonioalfa22/go-rest-template)
