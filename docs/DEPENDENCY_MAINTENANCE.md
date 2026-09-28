# Dependency maintenance

Use this workflow for dependency updates across the API, shared packages, and standalone commands. All of them share the root Go module.

## Toolchain policy

The minimum supported Go version is 1.25. The selected minimum-line validation toolchain is Go 1.25.13; CI also checks Go 1.26.8. Disable automatic toolchain switching during validation with `GOTOOLCHAIN=local` so an implicit download cannot conceal a raised minimum requirement.

Keep the module declaration, selected toolchain, CI versions, and Docker builder consistent. A dependency that requires Go 1.26 or later needs either a compatible earlier release or explicit approval to raise the minimum. A developer having a newer Go installation does not establish minimum-version compatibility.

## Prepare an upgrade

1. Record the current worktree state and preserve unrelated changes.
2. Identify the dependency's use across application code, scripts, tests, and generators.
3. Select an exact stable target version. Read upstream release and migration notes, inspect its Go requirement, and check maintenance and security status.
4. Classify the update as routine, a coordinated dependency-family update, a migration, or a toolchain change. Review major-version replacements explicitly; a same-module latest query does not discover every successor module.
5. Capture baseline validation results before editing. Distinguish existing failures from upgrade regressions.
6. Identify affected contracts and add regression coverage before changing behavior-sensitive dependencies where practical.

Prioritize confirmed vulnerabilities and unsupported components. Do not infer vulnerability status from version age alone.

## Implement one coherent group

- Pin selected versions instead of performing a blanket upgrade of the module graph.
- Adapt only affected APIs and behavior. Do not combine upgrades with unrelated cleanup.
- Review all `go.mod` and `go.sum` changes, including indirect dependencies selected by the module graph.
- Preserve API responses, validation, configuration precedence, logging, persistence semantics, and CLI contracts unless a separate behavior change is approved.
- Update source-owned generators and regenerate artifacts only when required.
- Record an explicit reason for each deferred update, especially when preserving the Go minimum limits version selection.

## Validate

Run these checks with the selected toolchains and CGO enabled:

```bash
go test ./...
go build ./...
go test -race ./...
go vet ./...
go mod tidy
git diff --check
```

Inspect the module-file diff after tidying. Once the intended module changes are settled, another tidy must leave them unchanged.

Run a pinned compatible version of `govulncheck` and record its version, toolchain, findings, and triage. A clean scan is evidence about known vulnerabilities, not a guarantee of security.

Also require focused tests for affected packages, non-publishing Docker builds for both configured Linux architectures, and isolated container startup smoke tests. Confirm results rather than assuming a CI configuration proves they passed.

For ORM and driver changes, test SQLite, PostgreSQL, and MySQL with recorded database versions. Check fresh initialization, migration of representative existing schemas, repeated migration, uniqueness, IDs, timestamps, transaction commit and rollback, upserts, zero-value updates, and concurrent writes. SQLite tests alone do not establish the other engines' compatibility.

### Database test selection

Database-backed persistence and key-value controller tests default to disposable SQLite files. For server-backed tests, select `GEE_TEST_DB_ENGINE=postgres` or `GEE_TEST_DB_ENGINE=mysql`, set `GEE_TEST_DB_PORT` to the local test server port, and set `GEE_TEST_DB_PASSWORD` to its test-only password. PostgreSQL uses the `postgres` account; MySQL uses `root`. CI provisions PostgreSQL 17.6 and MySQL 8.4.6 for this matrix.

The fixture connects only to `127.0.0.1`, creates a randomly named `gee_test_...` database per test, and drops only that database during cleanup. It does not read application configuration or accept an existing database name. The test account needs database creation privileges. Use an isolated test server, never a tunnel or port forward to a shared environment.

Run the server-backed suites with:

```bash
go test -race -v ./internal/pkg/persistence ./internal/api/controllers
```

The verbose output records the database engine and runtime version. Selecting a server engine without valid settings or a reachable server fails; it does not skip. The legacy SQLite schema fixture was generated with GORM v1.9.16 and the original models. Server variants translate the legacy dialect types and are representative schemas, not captured production dumps.

Use disposable databases and controlled HTTP endpoints. Never use live credentials, send real webhooks, or run test migrations on shared databases. Report unavailable engines or infrastructure as blocked validation, not as successful or silently skipped checks.

## Record acceptance evidence

Each upgrade record must include:

- Scope, old and new versions, target rationale, and upstream references.
- Go requirements and the resulting transitive dependency changes.
- Any intended behavior changes and their approval.
- Tests, builds, race checks, vet, vulnerability results, and container/database evidence.
- Pre-existing failures, skipped checks, unresolved blockers, and deferral reasons.
- Rollback instructions and any separately required publication or deployment approval.

Do not mark an upgrade complete while a required validation gate lacks evidence.

## Roll back safely

Revert a coherent code-only upgrade as a unit, including its module files, adaptations, and relevant generated artifacts, then rerun validation. Preserve unrelated work.

Before testing schema changes on valuable data, obtain a backup and verify restoration. Reverting an ORM dependency or container image does not undo automatic schema migrations. Database restoration, publishing, and deployment require separate authorization.

## Review periodically

Review dependencies periodically and triage confirmed security findings promptly. Automation may propose updates after a separate decision, but must not bypass review, validation, or migration approval. Preserve the existing non-publishing pull-request checks and publication controls.

See the [upgrade plan](DEPENDENCY_UPGRADE_PLAN.md) and [implementation status](DEPENDENCY_UPGRADE_STATUS.md) for the current modernization effort and its outstanding validation.
