# go-gin-gee

Gee is a project that provides several services for everyday work. The project is based on Gin [1], and follows the ProjectLayout [3] structure. In addition, some daily scripts in the folder `scripts`, which can be used by the command `run` directly.

<!-- omit from toc -->
## Table of Contents

- [Script Examples](#script-examples)
- [API Examples](#api-examples)
  - [Key-Value Store](#key-value-store)
- [Build](#build)
- [Deployment](#deployment)
  - [Supervisor](#supervisor)
- [Docker](#docker)
  - [Quick Start](#quick-start)
  - [Build Image](#build-image)
  - [Run Container](#run-container)
- [Documentation](#documentation)
- [Contributing](#contributing)
  - [Local Development Setup](#local-development-setup)
  - [Details](#details)
- [References](#references)

## Script Examples

1\. Change Git name and email for different projects.

macOS Bash or zsh:

```bash
bash scripts/batch-set-git-identity.sh --path="/Users/X/Web" --username="YOUR_NAME" --useremail="YOUR_NAME@email.com"
```

Windows 10 Git Bash:

```bash
bash scripts/batch-set-git-identity.sh --path="C:/Web" --username="YOUR_NAME" --useremail="YOUR_NAME@email.com"
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

### Key-Value Store

The key-value API exposes exactly three JSON endpoints:

| Method | Path | Authorization | Behavior |
| :-- | :-- | :-- | :-- |
| POST | `/api/gee/kv/get` | Required only for private entries | Read an entry using a key from the JSON body |
| POST | `/api/gee/kv/set` | `X-API-Key` required | Create or replace an entry (upsert) |
| POST | `/api/gee/kv/increment` | `X-API-Key` required | Atomically increment a numeric counter |

Configure accepted keys in `Data.KVAPIKeys`. Keys must match `^[a-z0-9][a-z0-9._:-]{0,127}$`; they are never read from paths, query strings, or headers. Entry visibility is `public` or `private` and defaults to `private`. An unauthorized read of a private entry returns the same `404` response as an unknown key.

All responses contain exactly `code`, `message`, and `data`. Application codes complement, rather than replace, meaningful HTTP statuses:

| Code | Message | HTTP status |
| --: | :-- | --: |
| 0 | success | 200 or 201 |
| 40001 | invalid request | 400 |
| 40002 | invalid key | 400 |
| 40003 | invalid value | 400 |
| 40101 | API key required | 401 |
| 40301 | access denied | 403 |
| 40401 | key not found | 404 |
| 40901 | incompatible value type | 409 |
| 50001 | internal server error | 500 |

Get a public value:

```bash
curl --request POST '${BASE_URL}/api/gee/kv/get' \
  --header 'Content-Type: application/json' \
  --data '{"key":"site.title"}'
```

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "key": "site.title",
    "value": "Gee Service",
    "content_type": "text/plain",
    "visibility": "public",
    "created_at": "2026-07-26T03:00:00Z",
    "updated_at": "2026-07-26T03:00:00Z"
  }
}
```

Create or replace a value:

```bash
curl --request POST '${BASE_URL}/api/gee/kv/set' \
  --header 'Content-Type: application/json' \
  --header "X-API-Key: ${GEE_KV_API_KEY}" \
  --data '{"key":"site.title","value":"Gee Service","content_type":"text/plain","visibility":"private"}'
```

Creation returns HTTP `201`; replacement returns HTTP `200`. The response data includes `created: true` or `created: false`.

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "key": "site.title",
    "value": "Gee Service",
    "content_type": "text/plain",
    "visibility": "private",
    "created": true
  }
}
```

Increment a counter:

```bash
curl --request POST '${BASE_URL}/api/gee/kv/increment' \
  --header 'Content-Type: application/json' \
  --header "X-API-Key: ${GEE_KV_API_KEY}" \
  --data '{"key":"page.views","delta":1}'
```

An omitted `delta` defaults to `1`; explicit zero, positive, and negative deltas are supported. Missing counters start at zero, and increments use database-side arithmetic. A key already used for a non-counter entry returns HTTP `409`.

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "key": "page.views",
    "value": 101,
    "delta": 1
  }
}
```

Errors contain no internal details and normally use `data: null`:

```json
{
  "code": 40401,
  "message": "key not found",
  "data": null
}
```

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

## Deployment

Environment Variables:

- `${WEBHOOK_ID}`: Discord Webhook ID.
- `${WEBHOOK_TOKEN}`: Discord Webhook Token.
- `${BASE_URL}`: The Base URL for this Service.

Config-file only private webhook API fields:

- `Data.EnableWebhookAPI`: Set to `on` to enable `POST /api/gee/webhook-message`.
- `Data.WebhookAPIKeys`: API keys accepted by the private webhook API via `X-Webhook-API-Key`.

Key-value API configuration:

- `Data.KVAPIKeys`: API keys accepted by key-value write operations and private reads via `X-API-Key`.

API-key checks apply only where handlers explicitly implement them. The webhook endpoint uses its webhook-specific header and key list; the key-value endpoints use `X-API-Key` and `Data.KVAPIKeys`.

Upgrading does not automatically drop legacy user tables. After backing up the database and verifying the deployment, operators may optionally remove obsolete tables such as `gee_user` and `gee_user_role` manually.

### Supervisor

```text
[program:api]
directory=/web/go-gin-gee
command=/web/go-gin-gee/dist/api-linux-amd64 --config-path="/web/go-gin-gee/data/config.json"
autostart=true
autorestart=true
environment=WEBHOOK_ID="WEBHOOK_ID",WEBHOOK_TOKEN="WEBHOOK_TOKEN",BASE_URL="https://example.com/path"
```

## Docker

### Quick Start

```bash
GEE_VERSION="v$(date +"%Y%m%d%H%M%S")" && \
GEE_TAG="go-gin-gee:${GEE_VERSION}" && \
docker build -t "${GEE_TAG}" . && \
docker run --name "go-gin-gee-${GEE_VERSION}" -p 3000:3000 "${GEE_TAG}"
```

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

`${RUN_FLAG}` is optional, default is `-r`("RUN"). `${WEBHOOK_ID}` and `${WEBHOOK_TOKEN}` are optional. If you don't want to send the message to Discord, just remove them. `${BASE_URL}` is required. It's the Base URL for this Service.

```bash
bash ./scripts/docker-build.sh ${RUN_FLAG} \
  "WEBHOOK_ID=${WEBHOOK_ID}" \
  "WEBHOOK_TOKEN=${WEBHOOK_TOKEN}" \
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
  "WEBHOOK_ID=WEBHOOK_ID" \
  "WEBHOOK_TOKEN=WEBHOOK_TOKEN" \
  "BASE_URL=https://example.com/path"
```

### Run Container

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
  "WEBHOOK_ID=${WEBHOOK_ID}" \
  "WEBHOOK_TOKEN=${WEBHOOK_TOKEN}" \
  "BASE_URL=${BASE_URL}"
```

Example:

```bash
bash ./scripts/docker-run.sh "docker.io/mazeyqian/go-gin-gee:v20230615221222-api" \
  "WEBHOOK_ID=WEBHOOK_ID" \
  "WEBHOOK_TOKEN=WEBHOOK_TOKEN" \
  "BASE_URL=https://example.com/path"
```

## Documentation

Download [swag](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@v1.8.12
```

Generate:

```bash
swag init --dir cmd/api,internal/api/controllers --parseDependency --output docs
```

Make sure your GO Path is on the PATH environment variable `export PATH=$(go env GOPATH)/bin:$PATH` if the following error occurs `command not found: swag`.

Run and visit: <http://localhost:3000/docs/index.html>

## Contributing

### Local Development Setup

```bash
git clone https://github.com/chengchuu/go-gin-gee.git
go mod download
go run scripts/init/main.go && \
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

To reset to Go default proxy settings:

```bash
go env -w GOPROXY=https://proxy.golang.org,direct
```

It's necessary to run the command `go run scripts/init/main.go` when serving the project first.

Serve:

```bash
go run cmd/api/main.go --config-path="data/config.dev.json"
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
