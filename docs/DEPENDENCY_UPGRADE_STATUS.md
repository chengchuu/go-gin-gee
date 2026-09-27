# Dependency upgrade status

Implementation of [the approved plan](DEPENDENCY_UPGRADE_PLAN.md) is in progress. This record distinguishes candidate selection from completed, validated upgrades.

## Baseline and environment

- Initial module baseline: Go `1.23`, toolchain `go1.23.5`.
- Available default toolchain: Go `1.26.8` on macOS amd64.
- Minimum-version validation toolchain: installed Go `1.25.12`, with `GOTOOLCHAIN=local` to prevent automatic switching.
- The first baseline test attempt failed because the sandbox could not access the default build cache. Validation was restarted with a temporary writable cache; this is an environment failure, not an application regression.
- Docker was initially unavailable. Docker Desktop 20.10.14 is now running; live database validation and final-code amd64 container checks passed as recorded below. ARM64 verification remains incomplete after a stalled emulated build.
- Public module-proxy connectivity initially failed. The configured `goproxy.cn` proxy is reachable with network permission.
- Baseline `go test ./...`, `go build ./...`, and `go vet ./...` passed on Go 1.25.12. The full test suite required loopback permission for local HTTP fixtures. The focused pre-migration persistence regressions also passed.
- The baseline race run failed on the same sandbox loopback restriction; affected suites require a permitted rerun. No clean full race result is claimed.
- Baseline `govulncheck v1.4.0` reported 16 reachable findings across the standard library, `golang.org/x/net`, and `github.com/antchfx/xpath`. Standard-library findings identify Go 1.25.13 as the fix; the toolchain pin must be advanced after verifying that release. The selected x/net target includes the reported fixes; XPath must reach at least v1.3.6. Final scanning remains required.

## Implementation checkpoint

- The Go minimum is now 1.25.0. Go 1.25.13 was downloaded and verified; the module, Docker builder, minimum-line CI job, and development documentation now use that security patch.
- CI includes explicit minimum-line and newer-toolchain jobs with automatic switching disabled, plus builds, race tests, vet, module consistency, and whitespace checks. These workflow changes have not run in hosted CI.
- GORM 2 and all three selected driver modules are installed. Application models, persistence calls, database opening, and test fixtures have been adapted; GORM v1 has been removed by module tidying.
- Model tags and hook signatures now use the GORM 2 APIs. Startup propagates database opening or migration failures instead of continuing with an unusable database.
- Focused post-migration persistence and key-value controller tests passed on Go 1.25.12. They caught a short-link visit update without a primary-key condition; it now updates the loaded row, with an additional unrelated-row regression.
- The complete Go 1.25.13 test suite, repository build, vet, and race suite passed after the GORM migration. The first full run overlapped creation of the shared test fixture and reported an import-discovery failure; the subsequent stable-worktree run passed. Final validation must be repeated after all dependency groups.
- The SQLite legacy-schema migration test passed. It uses a schema generated from the original GORM v1 models in an isolated temporary module, preserving IDs, stored values, indexes, and uniqueness through repeated migration.
- Persistence and controller fixtures now select all three engines. Server tests create random databases on loopback and fail instead of skipping when configured infrastructure is unavailable. PostgreSQL 17.6 and MySQL 8.4.6 each passed three consecutive persistence/controller race-suite runs after the counter fix described below. Hosted CI execution remains unverified.
- Gin v1.12.0 and the selected Swagger modules passed the full Go 1.25.13 suite. Regeneration with swag v1.16.6 left `swagger.json` and `swagger.yaml` unchanged; only generated Go metadata changed. The README generator pin is aligned.
- Viper v1.21.0 and pflag v1.0.10 passed the full suite. New configuration tests passed before and after upgrading, covering explicit flags, file settings, environment overrides, collections, and missing-file defaults.
- The scheduler migrated directly to `github.com/go-co-op/gocron/v2 v2.22.0` (Go 1.22). The full suite passed, including a fake-clock regression for the daily 10:00 Shanghai schedule in winter and summer.
- The six requested networking, crawling, HTTP-client, and shared-helper upgrades are installed; see the validation checkpoint below. Other utility upgrades, final security acceptance, and ARM64 container validation remain unfinished.
- `DEPENDENCY_MAINTENANCE.md` documents the repeatable process; the README links to it and states Go/CGO prerequisites.

## Six-dependency validation checkpoint

The requested versions are installed in `go.mod` and `go.sum`:

| Module | Version |
| --- | --- |
| `golang.org/x/net` | `v0.58.0` |
| `github.com/antchfx/xpath` | `v1.3.8` |
| `github.com/go-resty/resty/v2` | `v2.17.2` |
| `github.com/gocolly/colly/v2` | `v2.3.0` |
| `github.com/chengchuu/gurl` | `v1.2.0` |
| `github.com/chengchuu/asiatz` | `v1.2.0` |

The module retains Go 1.25 minimum support. Related transitive updates include Colly's parsing dependencies and the required `golang.org/x/*` modules; unrelated direct dependencies were not upgraded in this group.

The full test suite, full race suite, repository build, vet, `go mod tidy -diff`, and `git diff --check` passed on Go 1.25.13. The Windows amd64 link-checker cross-build passed; this is compile coverage, not Windows runtime validation. Added regressions cover local crawling, duplicate-link handling, missing pages, excluded links, and short-link hash compatibility. Existing redirect, webhook, and Shanghai scheduling tests also passed.

The initial post-upgrade `govulncheck v1.4.0` run reported one reachable finding: `GO-2026-5676` in `github.com/quic-go/quic-go v0.59.0`. The subsequent review upgraded this indirect dependency to the fixed `v0.59.1`. Rescanning on Go 1.25.13 reports zero reachable vulnerabilities, with one finding in imported packages and five in required modules without identified calls to the vulnerable symbols. This result does not establish that the entire dependency graph is vulnerability-free.

## Candidate inventory

These exact candidates were resolved through Go module metadata. Listed Go requirements are module declarations, not proof that the complete selected graph builds. Upstream change review, transitive graph review, and validation are required before accepting each group.

- ORM migration: replace `github.com/jinzhu/gorm v1.9.16` with `gorm.io/gorm v1.31.2` (Go 1.18), `gorm.io/driver/sqlite v1.6.0` (Go 1.20), `gorm.io/driver/mysql v1.6.0` (Go 1.18), and `gorm.io/driver/postgres v1.6.3` (Go 1.25.0). Affects database setup, models, persistence, and database-backed tests.
- HTTP framework: `github.com/gin-gonic/gin v1.8.1` to `v1.12.0` (Go 1.25.0). Affects the API, middleware, handlers, binding, and HTTP tests.
- Swagger family: `github.com/swaggo/files` from `v0.0.0-20220728132757-551d4a08d97a` to `v1.0.1` (Go 1.16); `github.com/swaggo/gin-swagger v1.5.3` to `v1.6.1` (Go 1.23.0); `github.com/swaggo/swag v1.8.1` to `v1.16.6` (Go 1.18). Review generated documentation compatibility together.
- Configuration: `github.com/spf13/viper v1.13.0` to `v1.21.0` (Go 1.23.0), and `github.com/spf13/pflag v1.0.5` to `v1.0.10` (Go 1.12). Preserve flags, environment handling, defaults, and config-file precedence.
- Scheduling: replace `github.com/go-co-op/gocron v1.17.0` directly with stable `github.com/go-co-op/gocron/v2 v2.22.0` (Go 1.22). Preserve scheduling time and background operation; do not retain the superseded v1 API.
- HTTP client: `github.com/go-resty/resty/v2 v2.7.0` to `v2.17.2` (Go 1.23.0). Preserve redirect and webhook behavior. Defer `resty.dev/v3`: its current latest version is `v3.0.0-rc.4`, not a stable release.
- Crawling: `github.com/gocolly/colly/v2 v2.1.0` to `v2.3.0` (Go 1.24.0). Validate affected standalone commands.
- Shell utilities: `github.com/bitfield/script v0.20.2` to `v0.25.1` (Go 1.25.0). Review pre-v1 changes and command side effects.
- Shared helpers: `github.com/samber/lo v1.33.0` to `v1.53.0` (Go 1.18).
- Timezone helper: `github.com/chengchuu/asiatz v1.1.4` to `v1.2.0` (Go 1.19).
- URL helper: `github.com/chengchuu/gurl v1.1.2` to `v1.2.0` (Go 1.19). Preserve short-link hash and URL behavior.
- Spreadsheet utility: `github.com/szyhf/go-excel v1.5.3` to `v1.6.1` (Go 1.24).
- Short-key conversion: `github.com/takuoki/clmconv v1.1.0` to `v1.2.0` (Go 1.24). Preserve existing ID-to-key mappings.
- Parsing: retain `github.com/tdewolff/parse/v2 v2.8.16` provisionally; the same-module update query returned no newer version. Review maintenance and security findings before final acceptance.
- Networking: `golang.org/x/net v0.34.0` to candidate `v0.58.0` (Go 1.25.0). Defer `v0.59.0`, which requires Go 1.26.0.
- System support: `golang.org/x/sys v0.29.0` to candidate `v0.47.0` (Go 1.25.0). Defer `v0.48.0`, which requires Go 1.26.0.

Relevant indirect dependencies must be reassessed after each group, especially database drivers, validators, serialization libraries, `golang.org/x/crypto`, and `golang.org/x/text`. Do not upgrade the entire transitive graph indiscriminately.

## Regression coverage added before migration

`internal/pkg/persistence/database_regression_test.go` adds disposable SQLite coverage for:

- Generated IDs and timestamps.
- Repeated migrations preserving rows and unique keys.
- Transaction rollback and empty-value updates.
- Short-link deduplication, stored keys, visits, one-time expiration, and special-link precedence.

These tests passed against GORM v1 before migration and GORM 2 after adaptation. The GORM 2 suites also passed on PostgreSQL 17.6 and MySQL 8.4.6, including the representative legacy-schema fixtures described above. Existing controller tests cover additional key-value concurrency and API behavior.

## Live database validation and counter fix

The first MySQL run exposed a reproducible failure in `TestIncrementKVConcurrentDistinctCounters`. InnoDB's deadlock report showed conflicting gap locks caused by updating missing counter rows before inserting them. The focused test reproduced the failure twice.

Counter increments now use a single database upsert with atomic arithmetic, retaining the existing transaction and key-type checks. The obsolete counter-insert retry was removed. The concurrency regression also checks that all 40 counters are persisted with value `1`.

After the fix, three consecutive race-enabled persistence/controller suite runs passed on each of MySQL 8.4.6 and PostgreSQL 17.6. The full SQLite-default normal and race suites, repository build, vet, module consistency, and whitespace checks passed on Go 1.25.13. Both disposable server containers and their temporary storage were removed after validation; no shared database was used.

The final-source Linux amd64 image built successfully as `gee-validation:amd64` (image ID `510c66438f8d`). Its isolated smoke checks passed: API health, Swagger HTTP 200, SQLite-backed short-link creation, and successful lookup after a container restart. Access-log lines increased from 3 to 5 across the restart rather than being truncated. The test container and its disposable database/log data were removed.

The ARM64 build downloaded its dependencies and prepared runtime assets, but the emulated API compilation produced no further output for more than four hours. The build was canceled; no final ARM64 image or runtime smoke result exists. The cause is unconfirmed, so this is not evidence of an application compilation error. Further VM-wide process inspection was blocked by approval review because it would expose a broader process surface. Retry on a native ARM64 builder or after updating the emulation environment; neither requires weakening application tests. No image was published, and no validation containers remain.

## Remaining stage gates

1. Complete baseline validation, upstream review, and the database test matrix.
2. Align the module, CI, and container toolchain with Go 1.25 minimum support.
3. Migrate GORM and validate fresh and existing schemas on all three engines.
4. Upgrade and validate the remaining dependency groups independently.
5. Add missing CI gates and reusable maintenance documentation.
6. Audit every completion criterion in the plan against actual results.

No publishing, deployment, or shared-database migration has been performed.
