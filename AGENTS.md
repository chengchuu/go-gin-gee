# AGENTS.md

## Purpose

This repository is a Go monorepo with:

- one primary Gin-based HTTP service
- shared internal packages for config, persistence, and models
- a collection of standalone utility scripts under `scripts/`

Use this file as a quick orientation guide before making changes.

## Repository Layout

### Main application

- `cmd/api/main.go`
  - Thin entry point for the production API.
  - Imports generated Swagger docs and calls `internal/api.Run()`.

- `internal/api/`
  - Application bootstrap and HTTP layer.
  - Key subfolders:
    - `controllers/`: request handlers
    - `middlewares/`: auth, CORS, logging, 404 handling
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
    - `crypto/`: password hashing and JWT helpers
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

## Request Flow

For most API endpoints, the flow is:

1. Gin route matches request
2. Controller binds params or JSON
3. Controller calls repository in `internal/pkg/persistence`
4. Repository reads or writes via Gorm, or calls an external service
5. Controller returns JSON or HTML response

Common route registration lives in `internal/api/router/router.go`.

## Major Functional Areas

### Users and auth

- Routes:
  - `/api/login`
  - `/api/users`
  - `/api/users/:id`
- Main files:
  - `internal/api/controllers/auth-controller.go`
  - `internal/api/controllers/users-controller.go`
  - `internal/api/middlewares/auth.go`
  - `internal/pkg/persistence/users-repository.go`
  - `pkg/crypto/`

Flow:

- Login looks up a user by username, compares bcrypt password hashes, then returns a JWT.
- Protected endpoints use `AuthRequired()` middleware to validate the token.
- User create and update operations hash plaintext passwords before persistence.

### Tasks

- Routes:
  - `/api/tasks`
  - `/api/tasks/:id`
- Main files:
  - `internal/api/controllers/tasks-controller.go`
  - `internal/pkg/persistence/tasks-repository.go`
  - `internal/pkg/models/tasks/task.go`

Flow:

- Standard CRUD through controller -> repository -> Gorm.
- Task queries preload related `User` records.

### Alias-to-data storage

- Routes:
  - `/api/gee/get-data-by-alias`
  - `/api/gee/create-alias2data`
  - `/api/gee/count-alias2data`
- Main files:
  - `internal/api/controllers/alias2data-controller.go`
  - `internal/pkg/persistence/alias2data-repository.go`

Flow:

- `get-data-by-alias` retrieves a stored alias record and only returns it if `Public` is true.
- `count-alias2data` is stateful: it reads a record, initializes it when missing, increments a count stored in `Data`, and saves it back.

### Short links

- Routes:
  - `/api/gee/generate-short-link`
  - `/api/gee/query-short-link`
  - `/t/:key`
- Main files:
  - `internal/api/controllers/tiny-controller.go`
  - `internal/pkg/persistence/tiny-repository.go`
  - `internal/pkg/models/tiny/tiny.go`

Flow:

1. Incoming original URL is accepted by the controller.
2. Repository optionally folds `base_url` into the hash input.
3. MD5 is used to deduplicate existing links.
4. A DB row is created to obtain an auto-increment ID.
5. The numeric ID is converted into a short key.
6. Final tiny link is persisted and returned.
7. `/t/:key` resolves the key and redirects to the original URL.

Special behavior:

- Supports configured `SpecialLinks` from config.
- Supports one-time links by checking and incrementing `VisitCount`.

### Site health checks

- Route:
  - `/api/gee/check`
- Main files:
  - `internal/api/controllers/schedules-controller.go`
  - `internal/pkg/persistence/robot-repository.go`

Flow:

1. Site list is read from config.
2. Each site is checked via HTTP using `resty`.
3. An HTML report is written to `log/robot.html`.
4. A Markdown summary may be sent to a WeCom robot.

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

### Docker tag lookup

- Route:
  - `/api/gee/get-tag-name`
- Main files:
  - `internal/api/controllers/docker-controller.go`
  - `internal/pkg/persistence/docker-repository.go`

Flow:

- Calls Docker Hub over HTTP and finds a tag matching a suffix filter.

## Database Notes

- DB initialization is in `internal/pkg/db/database.go`.
- Supported drivers:
  - `sqlite`
  - `postgres`
  - `mysql`
- Auto-migrations run for:
  - `users.User`
  - `users.UserRole`
  - `tasks.Task`
  - `alias2data.Alias2data`
  - `tiny.Tiny`

If no database driver is configured, repository helpers will generally fail early through `checkDBDriver()`.

## Configuration Notes

Configuration is loaded in `internal/pkg/config/configuration.go`.

Sources:

- `--config-path` flag, defaulting to `data/config.json`
- environment variables via Viper
- fallback defaults in code

Important config fields:

- `Server.Port`
- `Server.Secret`
- `Server.Mode`
- `Database.*`
- `Data.EnableCORS`
- `Data.WeComRobotCheck`
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
- auth behavior: inspect `pkg/crypto` and `internal/api/middlewares/auth.go`
- startup or environment behavior: inspect `internal/api/api.go` and `internal/pkg/config`
- a CLI utility: work inside the relevant `scripts/<name>/main.go`
