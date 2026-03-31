# Session Journal

**Date**: 2026-03-30
**Session ID**: 20260330-0001
**Type**: Maintenance / Feature
**Work Items**: work-0001 (CORS Fix), work-0002 (Dependency Upgrade)

---

## Overview

Two work items completed in this session:

1. **work-0001** — CORS AllowedMethods fix and new functional options API for `NewChiRouter`.
2. **work-0002** — Full dependency audit and upgrade: Go toolchain → go1.25.8 (resolving 12 stdlib CVEs), safe direct deps, pgx v5.9.1, lib/pq v1.12.1.

Both work items went through the full `/work` pipeline: research → requirements → plan → implementation → commit → push. Released as **v3.1.0**.

---

## Session Goals

- Complete work-0001: verify tests, update README, mark complete.
- Complete work-0002: implement phased dependency upgrades, verify with govulncheck, commit atomically.
- Push all changes including Claude workflow files, docs, and tag v3.1.0.

---

## User Requests (Chronological)

1. Verify work-0001 router tests and complete Phase 2 (README).
2. Implement work-0002 dependency upgrade via `/implement_plan docs/work/work-0002/plans/master.md`.
3. Run final verification: `govulncheck ./...`, `go test -race ./...`, check `go.mod` directives.
4. Commit all changes (work-0001 + work-0002 + docs), push to remote, push tag `v3.1.0`.
5. `/journal` — this document.

---

## Technical Work

### work-0001: CORS Functional Options API

**Files changed**: `router/chi.go`, `router/chi_test.go`, `router/export_test.go`, `README.md`

**What was done**:
- Refactored `NewChiRouter` and `NewChiRouterWithCORS` to delegate to a shared internal `newChiRouter` constructor.
- Added `RouterOption` functional options type and six `With*` constructors:
  - `WithAllowedOrigins`, `WithAllowedMethods`, `WithAllowedHeaders`
  - `WithExposedHeaders`, `WithAllowCredentials`, `WithMaxAge`
- Added `NewChiRouterWithOptions(opts ...RouterOption)` as a new public API entry point.
- **Bug fix**: Default `AllowedMethods` now includes `PATCH` and `HEAD` (was previously missing both).
- Added tests covering default config and options override behaviour.
- Updated README with options table and usage examples.

**Commit**: `79ea7db feat(router): add CORS functional options API and fix AllowedMethods`

---

### work-0002: Dependency Upgrade — Four Atomic Commits

#### Phase 1 — Go Toolchain → go1.25.8

**Baseline govulncheck**: 12 active vulnerabilities in the Go standard library (go1.25.1):
- `crypto/tls`: GO-2026-4340, GO-2026-4337, GO-2025-4008
- `crypto/x509`: GO-2025-4175, GO-2025-4155, GO-2025-4013, GO-2025-4007
- `net/url`: GO-2026-4601, GO-2026-4341, GO-2025-4010
- `encoding/asn1`: GO-2025-4011
- `encoding/pem`: GO-2025-4009

**Fix**: Added `toolchain go1.25.8` to `go.mod`.

```bash
go get toolchain@go1.25.8
go mod tidy
```

**Post-Phase govulncheck**: `No vulnerabilities found.` (0 active)

**Commit**: `566b1aa fix(toolchain): upgrade Go to go1.25.8 to resolve 12 stdlib vulnerabilities`

---

#### Phase 2 — Safe Direct Dependency Upgrades

| Package | From | To |
|---------|------|----|
| `go-chi/chi/v5` | v5.2.3 | v5.2.5 |
| `go-playground/validator/v10` | v10.28.0 | v10.30.1 |
| `golang-jwt/jwt/v5` | v5.3.0 | v5.3.1 |
| `swaggo/swag` | v1.16.4 | v1.16.6 |
| `go-openapi/spec` | v0.21.0 | v0.22.4 |
| `go-openapi/jsonpointer` | v0.21.0 | v0.22.5 |
| `google.golang.org/protobuf` | v1.36.8 | v1.36.11 |
| `golang.org/x/*` | various | latest |

**Notable indirect promotion**: `prometheus/client_golang` moved from `// indirect` to direct — legitimate, as `metrics/prometheus.go` directly imports it. Was miscategorised before.

**Notable structural change**: `go-openapi/swag` v0.22+ split into sub-packages (`conv`, `jsonname`, `jsonutils`, `loading`, `stringutils`, `typeutils`, `yamlutils`). All remain `// indirect`.

**Commit**: `56a37a3 chore(deps): upgrade safe direct dependencies`

---

#### Phase 3a — pgx v5.9.1 + go directive to 1.25

```bash
go get github.com/jackc/pgx/v5@v5.9.1
# Output: go: upgraded go 1.24.0 => 1.25.0
```

`go.mod` now declares `go 1.25.0`. All `db/` tests passed.

**Commit**: `9204fe0 chore(deps): upgrade pgx to v5.9.1, bump go directive to 1.25`

---

#### Phase 3b — lib/pq v1.12.1 + CHANGELOG

```bash
go get github.com/lib/pq@v1.12.1
```

- `lib/pq` is used only as `_ "github.com/lib/pq"` in `db/query_context_test.go` — confirmed no production import, no deprecated symbol usage.
- Created `CHANGELOG.md` documenting the PostgreSQL 14+ minimum requirement for consumers using lib/pq in test suites.

**Commit**: `ca31f44 chore(deps): upgrade lib/pq to v1.12.1, requires PostgreSQL 14+`

---

### Final Verification

```
$ ~/go/bin/govulncheck ./...
No vulnerabilities found.

$ go test -race ./...
ok  github.com/platform-smith-labs/japi-core/db
ok  github.com/platform-smith-labs/japi-core/handler
ok  github.com/platform-smith-labs/japi-core/metrics
ok  github.com/platform-smith-labs/japi-core/middleware/http
ok  github.com/platform-smith-labs/japi-core/middleware/typed
ok  github.com/platform-smith-labs/japi-core/router

$ head -4 go.mod
go 1.25.0
toolchain go1.25.8
```

---

### Docs and Workflow Commit

**Commit**: `f2a372b chore: add Claude Code workflow and work item documentation`
- `.claude/` — 80+ agent definitions and workflow settings
- `CLAUDE.md` — project conventions, FP guidelines, API design rules
- `docs/work/work-0001/` and `docs/work/work-0002/` — full artifact chains (research, requirements, plans, implementation status)

---

## Issues Encountered and Resolved

### 1. Flaky test: `TestAdapterContextTimeout/returns_timeout_error_from_handler`

- **Symptom**: Failed once in full suite run (`expected 504, got 200`), passed consistently in isolation and in all subsequent runs.
- **Root cause**: Pre-existing timing sensitivity — the test uses a 1ms context timeout with a 2ms sleep, which is fragile under load.
- **Resolution**: Confirmed not caused by toolchain upgrade (pre-dates it). Documented and moved on. Not in scope to fix.

### 2. macOS linker warning during race tests

- **Symptom**: `ld: warning: '...metrics.test...': has malformed LC_DYSYMTAB`
- **Root cause**: macOS system linker quirk, unrelated to Go code.
- **Resolution**: Warning only — all tests pass.

---

## Files Created / Modified

| File | Action | Notes |
|------|--------|-------|
| `go.mod` | Modified | toolchain go1.25.8, go 1.25.0, all dep upgrades |
| `go.sum` | Modified | Updated checksums |
| `router/chi.go` | Modified | Functional options refactor + AllowedMethods fix |
| `router/chi_test.go` | Created | CORS options tests |
| `router/export_test.go` | Created | Test exports |
| `README.md` | Modified | CORS options documentation |
| `CHANGELOG.md` | Created | PostgreSQL 14+ constraint notice |
| `CLAUDE.md` | Created | Project instructions for Claude Code |
| `.claude/` | Created | Workflow configuration |
| `docs/work/work-0001/` | Created | Full work item artifacts |
| `docs/work/work-0002/` | Created | Full work item artifacts |
| `docs/journal/` | Created | This journal |

---

## Git Log (Final State)

```
f2a372b chore: add Claude Code workflow and work item documentation
79ea7db feat(router): add CORS functional options API and fix AllowedMethods
ca31f44 chore(deps): upgrade lib/pq to v1.12.1, requires PostgreSQL 14+
9204fe0 chore(deps): upgrade pgx to v5.9.1, bump go directive to 1.25
56a37a3 chore(deps): upgrade safe direct dependencies
566b1aa fix(toolchain): upgrade Go to go1.25.8 to resolve 12 stdlib vulnerabilities
ece12a0 feat!: change Nullable.Value() to return (T, error) [BREAKING CHANGE]
```

**Tag**: `v3.1.0` on `f2a372b`, pushed to origin.

---

## Next Steps

- Monitor for any downstream consumer issues from the `go` directive bump (1.24 → 1.25) — consumers on Go 1.24 will need to upgrade their toolchain.
- The `TestAdapterContextTimeout` flaky test is worth revisiting: replacing the `time.Sleep(2ms)` approach with a properly expired context would eliminate the timing dependency.
- `govulncheck` should be added to CI so future stdlib vulnerabilities surface automatically without manual scans.
