# AGENTS.md

## Purpose

This repository is a Go monorepo with:

- one primary Gin-based HTTP service
- shared internal packages for config, persistence, and models
- a collection of standalone utility scripts under `scripts/`

Use this file as a quick orientation guide before making changes.

Treat this project as a fresh design with no existing users or released versions.
Do not add migration logic or backward-compatibility layers. Each new deployment
starts with an entirely new SQLite database; never reset a database automatically
or use an existing application database for validation.

## Repository Layout

### Main application

- `cmd/api/main.go`
  - Thin entry point for the production API.
  - Imports generated Swagger docs and calls `internal/api.Run()`.

- `internal/api/`
  - Application bootstrap and HTTP layer.
  - Key subfolders:
    - `controllers/`: request handlers
    - `middlewares/`: CORS, logging, and 404 handling
    - `router/`: route registration and Gin setup

- `internal/pkg/`
  - Core application internals.
  - Key subfolders:
    - `config/`: configuration loading from flags, env, and config file
    - `db/`: Gorm connection setup and auto-migration
    - `models/`: DB models and API-related structs
    - `persistence/`: repository layer and external integrations

- `pkg/`
  - Reusable shared helpers used across the app.
  - Includes:
    - `http-err/`: standard JSON error response helper
    - `logger/`: project logger
    - `helpers/`: misc utility helpers

### Runtime assets and docs

- `assets/data/`
  - Example/default data files such as config, SQLite DB, and HTML template.
- `docs/`
  - Generated Swagger artifacts.
- `Dockerfile`
  - Multi-stage build for packaging the API.

### Utility scripts

- `scripts/`
  - Independent CLI tools, each usually with its own `main.go`.
  - These are not part of the normal API request path.

## Primary Entry Points

### API startup flow

1. `cmd/api/main.go`
2. `internal/api/api.go`
3. `internal/pkg/config.Setup()`
4. `internal/pkg/db.SetupDB()`
5. `internal/api/router.Setup()`
6. Gin server starts on configured port

### Important startup behavior

- Logger is initialized first.
- Process timezone is set to `UTC` in `internal/api/api.go`.
- If `config.Data.Sites` is populated, a scheduled health check is started before the HTTP server begins serving traffic.

### Access-log lifecycle

`internal/api.Run()` opens `./log/api.log` with create/write/append flags before scheduled health
checks or router setup. Missing directories use `0755`; new files use `0666` subject to umask.
Existing entries and permissions are preserved. Directory/open failures stop startup. The API
passes the writer to `router.Setup`, restores the previous Gin writer and closes the file when
the lifecycle returns, and returns errors to `cmd/api` for nonzero process exit. No new graceful
shutdown behavior is provided; process termination releases descriptors through the OS.

The file retains the existing Gin request format. Application stdout/stderr and recovery output
are unchanged. `log/robot.html` keeps its existing replacement behavior. Webmazey also builds this
application; its mounts and deployment policy are not changed by the logging fix.

The separate server repository's `websg` Compose configuration mounts `/web/log-webgee:/web/log`
on `go-gin-gee`, not Nginx `webgee`. Activation requires a verified append-safe API image, writable
host storage for its runtime identity, approved retention, and separately authorized API-only
recreation. The pinned production image is not updated automatically. A bind mount does not
recover old container-local logs, and Docker log limits do not bound `api.log`.

## Request Flow

For most API endpoints, the flow is:

1. Gin route matches request
2. Controller binds params or JSON
3. Controller calls repository in `internal/pkg/persistence`
4. Repository reads or writes via Gorm, or calls an external service
5. Controller returns JSON or HTML response

Common route registration lives in `internal/api/router/router.go`.

## Major Functional Areas

### Key-Value Store

- Routes:
  - `POST /api/gee/kv/get`
  - `POST /api/gee/kv/set`
  - `POST /api/gee/kv/increment`
- Main files:
  - `internal/api/controllers/kv-controller.go`
  - `internal/pkg/persistence/kv-repository.go`
  - `internal/pkg/models/kv/`

Flow:

- Keys are provided in JSON request bodies, never paths or query strings.
- `set` uses upsert semantics.
- `increment` uses atomic database arithmetic and a dedicated counter table.
- The key-value endpoints follow the project API response convention.

### Short links

- Routes:
  - `/api/gee/generate-short-link`
  - `/api/gee/query-short-link`
  - `/t/:link_key`
- Main files:
  - `internal/api/controllers/link-controller.go`
  - `internal/pkg/persistence/link-repository.go`
  - `internal/pkg/models/link/link.go`

Flow:

1. Incoming original URL is accepted by the controller.
2. The controller validates the JSON `common_api_key` field against `Data.CommonAPIKeys` and derives the permanent `DirectRedirect` policy. Missing or invalid keys require a warning.
3. SHA-256 of a fixed JSON structure deduplicates the exact original URL, effective base URL, one-time setting, and redirect policy. Keep public `base_url` overrides.
4. A DB row is created to obtain an auto-increment ID inside a transaction.
5. The numeric ID is converted into a short key.
6. If the generated key matches `SpecialLinks`, convert the new candidate into that direct, reusable special link and retry the original request with a fresh ID in the same transaction. Use a loop without an arbitrary retry limit. No underscore suffix is added.
7. The final `data` response value is computed at runtime from the base URL and short key.
8. `/t/:link_key` resolves the stored policy and redirects directly or through the Go-configured warning page. Resolution consumes one-time visits before the warning page is shown.

Special behavior:

- Configured `SpecialLinks` resolve first, work before any database record exists, and always redirect directly. The first configured match wins. Database resolution is the fallback.
- Fixed aliases use `fixed:<key>` in `dedup_hash`; ordinary links retain SHA-256 fingerprints. Different aliases may share a destination.
- Special-link records are inserted only on generated-key collisions. Conversion replaces the candidate fingerprint with `fixed:<key>` and sets the configured destination, direct redirection, and `OneTime: false`. Existing deduplicated records are never converted. All inserts and updates roll back together on failure.
- There is no startup provisioning or snapshot verification. Configuration overrides stored destinations; removing an alias can expose its previously stored destination, and adding one can shadow a generated link.
- Supports one-time links by checking and incrementing `VisitCount`.
- Lookup reads only the `link_key` query parameter. Creation binds `link.CreateRequest`, not the persistence entity; clients cannot set policy, keys, or visit counts.
- `data` contains the generated URL, not persisted model state. `tiny_link` is a deprecated, identical response alias; callers should use `data`. The response preserves `errors` and does not include a `link` field.
- `LinkRepository` is accessed through `GetLinkRepository()` and logs with `[Link]`.
- The `link.Link` model uses `gee_link`, with unique indexes `uk_link_dedup_hash` and `uk_link_key`. This is a new-database schema, not a migration from the old MD5 model. Never reset a user's database as validation.
- `ResolveLink` returns a destination and policy. `Data.LinkRedirectPageURL` is the sole warning URL source; request headers are ignored. Missing/invalid configuration fails closed with 503. Preserve `Cache-Control: no-store` on redirect responses.
- `EnableCORS` is independent of warning configuration. Webmazey sets it to `off`, with Nginx owning API CORS and Go handling OPTIONS. Standalone Go can enable its own CORS middleware.
- Common keys authorize creation only. Removing a key does not revoke existing direct links. No key is persisted, logged, returned, or included in a fingerprint or generated URL. Keep secrets out of public frontend bundles and request-body logs.

### Site health checks

- Route:
  - `/api/gee/check`
  - `/api/gee/webhook-message`
- Main files:
  - `internal/api/controllers/schedules-controller.go`
  - `internal/pkg/persistence/robot-repository.go`

Flow:

1. Site list is read from config.
2. Each site is checked via HTTP using `resty`.
3. An HTML report is written to `log/robot.html`.
4. A summary message may be sent to a Discord webhook when `WEBHOOK_ID` and `WEBHOOK_TOKEN` are configured.
5. `/api/gee/webhook-message` reuses the Discord sender and is disabled unless `Data.EnableWebhookAPI` is `on` with a matching `X-Webhook-API-Key`.

### API-key access control

- The private webhook API uses config-file-based keys from `Data.WebhookAPIKeys`.
- Common feature keys use the separate `Data.CommonAPIKeys` list and the JSON `common_api_key` field. Currently only short-link creation uses them; they do not authorize Webhook or KV operations.
- `POST /api/gee/webhook-message` validates the `X-Webhook-API-Key` header in `internal/api/controllers/webhook-controller.go`.
- Other routes are not protected by this API-key check unless their handlers explicitly implement it.

### Agent/server utilities

- Routes:
  - `/server/mock`
  - `/server/agent/record`
- Main files:
  - `internal/api/controllers/agent-controller.go`
  - `internal/pkg/persistence/agent-repository.go`

Flow:

- `/server/mock` echoes a provided mock response structure.
- `/server/agent/record` decodes URL-encoded JSON and writes a pretty-printed file into the configured agent records directory.

### Retired APIs

- `GET /api/gee/get-tag-name` is retired. It no longer calls Docker Hub or returns `tagName`.
- `controllers.RetiredAPI` in `internal/api/controllers/retired-controller.go` returns HTTP 410 with `Cache-Control: no-store` and the existing `http_err.APIResponse` envelope: `code: 41001` (`CodeAPIRetired`), `message: "This API has been retired."`, and `data: null`.
- To retire another endpoint, map its existing method and path directly to `controllers.RetiredAPI` in the router and remove its unused implementation. The shared handler terminates the handler chain and needs no database or outbound HTTP requests.
- Keep retirement explicit per route; do not replace unknown-route handling with a retirement response.

## Database Notes

- DB initialization is in `internal/pkg/db/database.go`.
- Supported drivers:
  - `sqlite`
  - `postgres`
  - `mysql`
- Auto-migrations run for:
  - `kv.Entry`
  - `kv.Counter`
  - `link.Link`

Database setup creates the fresh schema and never deletes an existing database.

If no database driver is configured, repository helpers will generally fail early through `checkDBDriver()`.

## Configuration Notes

Configuration is loaded in `internal/pkg/config/configuration.go`.

Sources:

- `--config-path` flag, defaulting to `data/config.json`
- environment variables via Viper
- fallback defaults in code

Important config fields:

- `Server.Port`
- `Server.Mode`
- `Database.*`
- `Data.EnableCORS`
- `Data.WebhookID`
- `Data.WebhookToken`
- `Data.EnableWebhookAPI`
- `Data.WebhookAPIKeys`
- `Data.CommonAPIKeys`
- `Data.LinkRedirectPageURL`
- `Data.KVAPIKeys`
- `Data.BaseURL`
- `Data.AgentRecordsPath`
- `Data.Sites`
- `Data.SpecialLinks`

## Files Worth Checking Before Edits

When changing behavior, start here:

- routing: `internal/api/router/router.go`
- startup: `internal/api/api.go`
- config: `internal/pkg/config/configuration.go`
- DB wiring: `internal/pkg/db/database.go`
- shared persistence helpers: `internal/pkg/persistence/common.go`

## API Response Convention

New JSON APIs must return a consistent response envelope:

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

Rules:

* The top-level fields are always `code`, `message`, and `data`.
* Successful responses use application code `0`.
* Failed responses use a documented non-zero application code.
* HTTP status codes must continue to represent the actual HTTP result.
* `data` contains the actual result using its natural JSON type.
* Object results use `{ ... }`.
* Collection results use `[ ... ]`; an empty collection uses `[]`.
* Scalar results use their actual string, number, or boolean type.
* Responses with no result use `null`.
* Failed responses normally use `data: null`.
* Do not use `{}` as a generic placeholder for missing data.
* Do not duplicate response values in deprecated top-level fields.
* Do not expose internal errors, SQL, API keys, stack traces, configuration values, or filesystem paths.
* JSON APIs following this envelope should not use `204 No Content`; use a suitable success status with `data: null` when no result is returned.

This convention applies to the key-value API and future newly added JSON APIs. Existing APIs are not retroactively migrated unless a task explicitly requests it.

## Working Guidelines

- Treat `cmd/api` as the service entry point.
- Treat `scripts/` commands as separate tools unless the task is explicitly about those utilities.
- Prefer following the existing controller -> repository -> model structure for API changes.
- Keep external side effects in repositories or dedicated service-style code, not directly in router setup.
- Be aware that some runtime files are expected under `data/`, while source examples currently live under `assets/data/`.

## Quick Mental Model

If you are changing:

- an endpoint: start in `internal/api/router` and `internal/api/controllers`
- DB-backed behavior: continue into `internal/pkg/persistence` and `internal/pkg/models`
- API-key access control: inspect `internal/api/controllers/webhook-controller.go` and `internal/pkg/config`
- startup or environment behavior: inspect `internal/api/api.go` and `internal/pkg/config`
- a CLI utility: work inside the relevant `scripts/<name>/main.go`
