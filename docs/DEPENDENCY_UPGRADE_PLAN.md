# Dependency upgrade plan

## Status and confirmed scope

Planning only. This document does not implement upgrades or authorize publishing, deployment, or changes to external systems. Begin implementation only after a separate instruction to proceed.

The confirmed scope is to:

- Establish Go 1.25 as the minimum supported Go version.
- Migrate directly from `github.com/jinzhu/gorm` to `gorm.io/gorm` and its driver modules, without a GORM v1 compatibility layer.
- Upgrade the remaining dependencies in separately validated stages.
- Accept dependency API changes while preserving intended application behavior and support for SQLite, PostgreSQL, and MySQL.
- Establish a repeatable process for future dependency upgrades, not a one-time Gin/GORM exception.

Exact dependency versions, toolchain patch versions, and database test versions remain implementation-time decisions. Do not silently raise the Go minimum above 1.25; request approval if a selected dependency requires it.

## Current repository baseline

- `go.mod` declares Go `1.23` and toolchain `go1.23.5`.
- Direct dependencies include Gin `v1.8.1` and GORM v1 `v1.9.16`.
- The API and standalone commands under `scripts/` share one module graph.
- `Dockerfile` uses `golang:1.23-bookworm`, enables CGO, and installs native build dependencies for SQLite support.
- `.github/workflows/docker-publish.yml` runs repository-wide tests and builds images for `linux/amd64` and `linux/arm64`. Pull-request builds do not publish; pushes to `release/v1` publish images.
- Database initialization in `internal/pkg/db/database.go` supports three database engines and runs automatic migrations for the key-value and short-link models.

Recheck this baseline before implementation. Dependency age alone does not establish a vulnerability, and no target-version compatibility or upgrade validation is claimed by this plan.

## Boundaries

- Preserve routes, HTTP statuses, response bodies, validation behavior, authentication checks, configuration precedence, logging lifecycle, and CLI contracts unless a specific behavior change is approved.
- Preserve the existing controller-to-repository-to-model structure.
- Do not add obsolete dependency compatibility wrappers for unreleased consumers.
- Do not combine upgrades with unrelated refactoring, new features, runtime-image redesign, or removal of supported database engines.
- Do not assume development databases, fixtures, or bundled example data are disposable.
- Do not alter deployment mounts, publication triggers, credentials, or external repositories as part of dependency maintenance.
- Defer dependency bots and automatic merging to a separate decision. Upgrading dependencies does not authorize automatic publication or deployment.

## Stage 1: Inventory and establish validation evidence

1. Inventory direct dependencies and relevant indirect dependencies across the API, shared packages, tests, and scripts.
2. Record each proposed upgrade's current and target versions, purpose, affected packages, minimum Go requirement, upstream release or migration notes, maintenance status, and known security findings.
3. Classify each change as a routine same-module upgrade, a coordinated dependency-family upgrade, a migration or replacement, or a toolchain change. Treat pre-v1 modules and behavior-sensitive libraries as higher risk even without a major-version change.
4. Select exact stable versions compatible with Go 1.25. Explain deliberate deferrals instead of assuming that every dependency must move to its latest release.
5. Run baseline tests, builds, vet, race tests, and vulnerability checks. Record toolchain versions and pre-existing failures separately from upgrade regressions.
6. Identify missing behavioral coverage and add focused regression tests before changing the corresponding dependency where practical.
7. Prepare disposable database fixtures and a database-version matrix for SQLite, PostgreSQL, and MySQL. Prevent tests from connecting to developer or production databases accidentally.

Exit criteria: a reviewed target inventory, reproducible baseline results, and a concrete validation matrix. Record environmental blockers rather than treating skipped checks as passes.

## Stage 2: Align the Go baseline

1. Update the module's Go minimum to 1.25 and select an exact toolchain patch.
2. Align CI and the Docker builder with the chosen toolchain policy. Preserve CGO support, existing native dependencies, and both target architectures.
3. Test with an explicit Go 1.25 toolchain and automatic toolchain switching disabled so a newer downloaded toolchain cannot mask a raised minimum requirement.
4. If the primary development toolchain is newer than 1.25, retain a separate Go 1.25 compatibility job and validate the newer toolchain too.
5. Update relevant development documentation without implying that arbitrary future Go releases have been verified.

Exit criteria: the existing application and utility commands pass the baseline checks on Go 1.25, and container builds succeed without unrelated dependency migrations. Keep required toolchain-related module changes small and explain them.

## Stage 3: Migrate directly to GORM 2

Treat the ORM and its database drivers as one coordinated migration. GORM 2 uses `gorm.io/gorm`; updating the old module alone does not perform this migration. Its release numbering must not be inferred from the product name.

1. Replace GORM v1 imports with `gorm.io/gorm` and select compatible versions of `gorm.io/driver/sqlite`, `gorm.io/driver/postgres`, and `gorm.io/driver/mysql`.
2. Adapt database opening, configuration, logger setup, connection-pool access, error handling, and cleanup to the selected APIs.
3. Audit persistence methods for transaction boundaries, not-found handling, expressions, affected-row interpretation, update semantics, zero values, primary keys, and concurrency behavior. Adapt only behavior affected by the migration.
4. Review model tags, table and column names, indexes, constraints, generated IDs, and automatic migrations against the existing schema and intended behavior.
5. Update tests and fixtures directly. Remove obsolete direct GORM v1 requirements; investigate any remaining transitive use instead of ignoring it.
6. Verify both fresh database initialization and migration of representative existing schemas on all three engines. Use disposable copies for any retained database files.
7. Compare schemas and persisted values before and after migration. Do not accept unexplained destructive changes or silent schema drift.

Required persistence regressions include:

- Key-value reads, missing keys, upserts, atomic increments, and concurrent requests.
- Transaction commit and rollback, error paths, and affected-row behavior where the application relies on it.
- Short-link creation, deduplication, generated keys, resolution, visit counting, one-time behavior, and configured special links.
- Repeated startup migrations without unexpected schema changes or data loss.

Exit criteria: no application or test code depends on GORM v1; intended persistence and API behavior passes on SQLite, PostgreSQL, and MySQL; schema effects are documented and reviewed. SQLite-only success does not satisfy this stage.

## Stage 4: Upgrade remaining dependency groups

Upgrade one dependency or a necessary compatible group at a time. Suggested review groups are:

- HTTP framework and integrations: Gin, middleware dependencies, validators, and Swagger integration.
- Configuration and scheduling: Viper, flags, scheduling libraries, and their relevant indirect dependencies.
- HTTP clients and utilities: Resty, crawling libraries, parsing libraries, and command-specific dependencies.
- Shared Go ecosystem modules and remaining indirect dependencies, guided by the selected graph and security findings.

These are review groups, not a requirement to upgrade unrelated modules together. Adjust the order when verified dependency constraints require it.

For every group:

1. Verify target versions and upstream changes against the inventory.
2. Apply the smallest coherent upgrade and necessary API adaptations.
3. Inspect the complete module graph and `go.mod`/`go.sum` diff, including transitive changes. Avoid a blanket repository-wide upgrade.
4. Regenerate source-owned artifacts only when required, using the corresponding supported generator.
5. Run focused regressions and the repository-wide validation gates.
6. Record versions, rationale, behavior changes, results, limitations, and rollback instructions before beginning the next group.

Exit criteria: every dependency in the inventory has an explicit upgrade or deferral decision, and each completed group has independent validation evidence.

## Validation gates

The following checks are planned for implementation; creating this document does not run them:

- Focused regression tests for each changed behavior boundary.
- `go test ./...` and `go build ./...` to cover both the API and standalone commands.
- `go test -race ./...` on a supported CGO-enabled test environment.
- `go vet ./...`.
- `govulncheck ./...` using a recorded compatible tool version, with findings triaged against the baseline. A clean scan does not prove the absence of vulnerabilities.
- Module consistency checks, with unexpected module-file changes resolved before acceptance.
- `git diff --check`.
- Non-publishing Docker builds for `linux/amd64` and `linux/arm64`, plus isolated container startup smoke tests with disposable configuration and storage.
- Database integration tests using recorded versions of all three supported engines; these jobs must not silently skip unavailable engines.

HTTP regressions should cover routing, redirects, request binding, validation, response envelopes, authentication boundaries, CORS, recovery, and append-safe access logging. Utility regressions should preserve argument parsing, output, file behavior, and exit statuses for affected commands.

External integrations must use controlled test endpoints or mocks. Do not send real webhooks, run scheduled checks against unrelated services, or use live credentials during validation.

Extend CI with the missing gates as implementation proceeds. Keep pull-request checks non-publishing and retain existing publication controls. Report local checks separately from CI and container results.

## Rollback and data safety

- Keep each stage independently reviewable and retain the previous dependency and toolchain baseline.
- For code-only failures, revert the coherent stage, including module files and associated adaptations, then rerun its checks.
- Before exercising schema changes on valuable data, require a backup and a tested restoration path. A dependency or image rollback does not necessarily reverse automatic migrations.
- Do not run new migrations against shared or production databases within this task.
- Obtain separate authorization for publication, deployment, database restoration, or any destructive action.

## Repeatable maintenance workflow

Document the process in a concise maintenance guide or existing contributor documentation during implementation. Use the same checklist for future upgrades:

- Scope, current versions, exact targets, and upstream evidence.
- Go minimum and toolchain compatibility.
- Affected application contracts and data risks.
- Necessary dependency grouping and transitive changes.
- Regression coverage and validation results, including skipped or blocked checks.
- Rollback procedure and any separate deployment approval.

Review dependencies periodically and handle confirmed security findings promptly. Automation may propose updates later, but should not replace version review, validation, or explicit migration decisions.

## Completion criteria

- [ ] Go 1.25 is the verified minimum across module configuration, CI, and documented development requirements.
- [ ] The selected container toolchain and both target architectures are validated.
- [ ] GORM 2 and all three driver integrations replace application and test use of GORM v1.
- [ ] Fresh-schema and existing-schema tests pass on SQLite, PostgreSQL, and MySQL.
- [ ] Remaining dependencies have recorded upgrade or deferral decisions.
- [ ] Intended API, persistence, logging, and affected CLI behavior is preserved.
- [ ] Validation evidence distinguishes passes, pre-existing failures, and unresolved blockers.
- [ ] A reusable maintenance checklist and rollback guidance are documented.
- [ ] No unapproved publishing, deployment, destructive migration, or unrelated refactoring occurs.

## References

- [Repository guidance](../AGENTS.md)
- [Module configuration](../go.mod)
- [Container build](../Dockerfile)
- [CI workflow](../.github/workflows/docker-publish.yml)
- [GORM 2 migration notes](https://gorm.io/docs/v2_release_note.html)
- [Go module version selection](https://go.dev/ref/mod#minimal-version-selection)
- [Go vulnerability management](https://go.dev/doc/security/vuln/)
