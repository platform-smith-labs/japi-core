# Go Dependency Upgrade — Implementation Plan

**Work Item**: work-0002
**Created**: 2026-03-30
**Status**: Ready for Implementation

---

## Documentation Chain

- **Manifest**: [../manifest.md](../manifest.md)
- **Research**: [../research/0001-dependency-upgrade-research.md](../research/0001-dependency-upgrade-research.md)
- **Requirements**: [../requirements/0001-dependency-upgrade-req.md](../requirements/0001-dependency-upgrade-req.md)

---

## Overview

Bring all `go.mod` dependencies to their latest stable versions. Five CVEs are present in the current transitive graph (`x/crypto` and `x/net`). The upgrade is split into four atomic, independently-revertable commits following the phased approach from the requirements.

**Files changed:** `go.mod`, `go.sum`, `CHANGELOG.md` (one entry in Phase 3b only).
**No source code changes required.**

---

## Current State

```
go 1.24.0

# Direct dependencies with available upgrades:
github.com/go-chi/chi/v5             v5.2.3   → v5.2.5
github.com/go-openapi/spec           v0.21.0  → v0.22.4
github.com/go-playground/validator/v10 v10.28.0 → v10.30.1
github.com/golang-jwt/jwt/v5         v5.3.0   → v5.3.1
github.com/jackc/pgx/v5              v5.7.6   → v5.9.1
github.com/lib/pq                    v1.10.0  → v1.12.1
github.com/swaggo/swag               v1.16.4  → v1.16.6

# Indirect — security fixes:
golang.org/x/crypto     v0.42.0 → v0.49.0   (3 CVEs: HIGH + 2 MEDIUM)
golang.org/x/net        v0.43.0 → v0.52.0   (2 CVEs)

# Indirect — drift reduction:
golang.org/x/text       v0.29.0 → v0.35.0
golang.org/x/sys        v0.36.0 → v0.42.0
golang.org/x/sync       v0.17.0 → v0.20.0
golang.org/x/tools      v0.36.0 → v0.43.0
golang.org/x/mod        v0.27.0 → v0.34.0
google.golang.org/protobuf v1.36.8 → v1.36.11
github.com/go-openapi/jsonpointer v0.21.0 → v0.22.5
```

**Verified facts:**
- Build machine: Go 1.25.1 — satisfies pgx v5.9.1's Go 1.25 requirement.
- `lib/pq` usage: blank import only in `db/query_context_test.go` — no production import, no deprecated symbols in use.
- `prometheus/client_golang` is already at latest (v1.23.2) — no action needed.

---

## What We Are NOT Doing

- Upgrading `georgysavva/scany/v2`, `go-chi/cors`, `gocarina/gocsv`, `google/uuid` — no material upgrades flagged.
- Changing any japi-core public API or source code.
- Adding or removing dependencies.

---

## Phase 1 — Go Toolchain Upgrade to go1.25.8

**Scope:** Upgrade the Go toolchain from go1.25.1 to go1.25.8.
**Commit message:** `fix(toolchain): upgrade Go to go1.25.8 to resolve 12 stdlib vulnerabilities`

**Why:** `govulncheck ./...` found **12 active vulnerabilities**, all in the Go standard library (`crypto/tls`, `crypto/x509`, `net/url`, `encoding/asn1`, `encoding/pem`). The fix is a toolchain patch upgrade — no dependency changes needed. go1.25.8 resolves all 12. See `research/0001-dependency-upgrade-research.md` for the full govulncheck output.

Additionally, upgrading to go1.25.8 satisfies pgx v5.9.1's Go 1.25 requirement (Phase 3a), so Phase 1 and Phase 3a's toolchain concern are resolved together.

### Pre-flight

```bash
# Record baseline
~/go/bin/govulncheck ./...
# Expected: 12 active stdlib vulnerabilities
```

### Steps

Update `go.mod` toolchain directive:

```bash
go get toolchain@go1.25.8
go mod tidy
```

Or manually update the `toolchain` line in `go.mod`:
```
go 1.25.0
toolchain go1.25.8
```

### Verification

```bash
go version                 # confirm go1.25.8
go build ./...
go vet ./...
go test ./...
go test -race ./...
~/go/bin/govulncheck ./... # must show 0 active vulnerabilities
```

### Success Criteria

- [ ] `go version` reports go1.25.8
- [ ] `go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `go test -race ./...` exits 0
- [ ] `govulncheck ./...` reports **0 active vulnerabilities** (Symbol Results section empty)
- [ ] `git diff --exit-code go.sum` is clean after `go mod tidy`

**Go/no-go gate:** All criteria met before proceeding to Phase 2.

---

## Phase 2 — Safe Direct Dependency Upgrades

**Scope:** chi, validator, jwt, swag, go-openapi/spec, protobuf, go-openapi/jsonpointer.
**Commit message:** `chore(deps): upgrade safe direct dependencies`

**Research confirms:** None of these introduce breaking API changes. All changes are additive or pure bug/security fixes.

### Steps

```bash
go get \
  github.com/go-chi/chi/v5@v5.2.5 \
  github.com/go-playground/validator/v10@v10.30.1 \
  github.com/golang-jwt/jwt/v5@v5.3.1 \
  github.com/swaggo/swag@v1.16.6 \
  github.com/go-openapi/spec@v0.22.4 \
  google.golang.org/protobuf@v1.36.11 \
  github.com/go-openapi/jsonpointer@v0.22.5

go mod tidy
```

### Verification

```bash
go build ./...
go vet ./...
go test ./...
go test -race ./...
govulncheck ./...   # confirm no new findings introduced
```

### Success Criteria

- [ ] `go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `go test -race ./...` exits 0
- [ ] `go list -m github.com/go-chi/chi/v5` shows v5.2.5
- [ ] `go list -m github.com/go-playground/validator/v10` shows v10.30.1
- [ ] `go list -m github.com/golang-jwt/jwt/v5` shows v5.3.1
- [ ] `go list -m github.com/swaggo/swag` shows v1.16.6
- [ ] `git diff --exit-code go.sum` is clean after `go mod tidy`

---

## Phase 3a — pgx v5.9.1 (+ Go directive bump to 1.25)

**Commit message:** `chore(deps): upgrade pgx to v5.9.1, bump go directive to 1.25`

**Research confirms:** No public API removals or renames across v5.7.6 → v5.9.1. The `go` directive bump from 1.24.0 to 1.25.0 happens automatically when `go get` resolves pgx's requirements. Go 1.25.1 is available on the build machine.

Notable improvements included in this upgrade:
- SCRAM-SHA-256-PLUS channel binding authentication
- TSVector type support
- LRU statement cache performance improvement
- DoS/OOM security fixes (malformed server message protection)
- Removal of internal `x/crypto` dependency

### Steps

```bash
go get github.com/jackc/pgx/v5@v5.9.1
go mod tidy
```

### Verification

```bash
head -3 go.mod            # must show "go 1.25.0" or later
go build ./...
go vet ./...
go test ./...
go test -race ./...
go list -m github.com/jackc/pgx/v5
```

### Success Criteria

- [ ] `go.mod` `go` directive is `1.25.0` (or the version `go get` sets)
- [ ] `go list -m github.com/jackc/pgx/v5` shows v5.9.1
- [ ] `go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0 (all `db/` tests pass)
- [ ] `go test -race ./...` exits 0
- [ ] `git diff --exit-code go.sum` is clean after `go mod tidy`

**Note on indirect promotions:** After `go mod tidy`, check `go.mod` for any package that moved from `// indirect` to direct. If any unrecognised package was promoted, investigate before committing (NFR-5).

---

## Phase 3b — lib/pq v1.12.1 (+ CHANGELOG entry)

**Commit message:** `chore(deps): upgrade lib/pq to v1.12.1, requires PostgreSQL 14+`

**Research confirms:** `lib/pq` is a blank import in `db/query_context_test.go` only. No deprecated symbols are used. The only consumer-visible change is the PostgreSQL 14+ minimum requirement introduced in v1.11.0.

### Steps

```bash
go get github.com/lib/pq@v1.12.1
go mod tidy
```

Then add to `CHANGELOG.md` (create the file if it does not exist):

```markdown
## [Unreleased]

### Changed
- Upgraded `github.com/lib/pq` to v1.12.1. **PostgreSQL 14 or later is now required** for consumers that register the `lib/pq` driver for `database/sql` in their test suites. This does not affect japi-core's primary database interface (pgx/v5).
```

### Verification

```bash
go build ./...
go vet ./...
go test ./...
go list -m github.com/lib/pq   # must show v1.12.1
grep -r '"github.com/lib/pq"' . --include="*.go"
# must show ONLY db/query_context_test.go
git diff --exit-code go.sum
```

### Success Criteria

- [ ] `go list -m github.com/lib/pq` shows v1.12.1
- [ ] `go build ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] PostgreSQL 14+ constraint documented in `CHANGELOG.md`
- [ ] No direct `lib/pq` import exists outside `db/query_context_test.go`
- [ ] `git diff --exit-code go.sum` is clean after `go mod tidy`

---

## Overall Acceptance Criteria

| # | Criterion | Phase |
|---|---|---|
| AC-1 | `govulncheck` Symbol Results section is empty (0 active vulns) | Phase 1 |
| AC-2 | All safe direct deps at target versions (REQ-2 table) | Phase 2 |
| AC-3 | pgx at v5.9.1, `go.mod` declares `go 1.25.0` | Phase 3a |
| AC-4 | lib/pq at v1.12.1, no direct import outside test file | Phase 3b |
| AC-5 | `go test ./...` exits 0 after every phase | All phases |
| AC-6 | `go test -race ./...` exits 0 after every phase | All phases |
| AC-7 | `go build ./...` exits 0 after every phase | All phases |
| AC-8 | `go vet ./...` exits 0 after every phase | All phases |
| AC-9 | `go mod tidy` produces no diff after each commit | All phases |
| AC-10 | Each phase is a separate atomic commit | All phases |
| AC-11 | PostgreSQL 14+ constraint in CHANGELOG | Phase 3b |

---

## Recommended Agents for Implementation

**Primary:**
- **dependency-manager** — Run the `go get` commands, verify `go.mod`/`go.sum` correctness, and check for indirect-to-direct promotions after each phase.

**Phase 1 gate:**
- **security-engineer** — Interpret `govulncheck` output before and after Phase 1; confirm all 5 CVEs are resolved and no new findings introduced by Phase 2+.

**Quality gate (all phases):**
- **code-reviewer** — Before each phase commit: confirm only `go.mod`/`go.sum` changed, no source files, no unreviewed indirect promotions.

**Skills:**
- `/commit` — Create each phase's conventional commit with the exact message specified above.

---

## Progress Tracking

| Phase | Status | Commit |
|-------|--------|--------|
| Phase 1: Toolchain upgrade | ⏳ Not Started | `fix(toolchain): upgrade Go to go1.25.8 to resolve 12 stdlib vulnerabilities` |
| Phase 2: Safe direct deps | ⏳ Not Started | `chore(deps): upgrade safe direct dependencies` |
| Phase 3a: pgx v5.9.1 | ⏳ Not Started | `chore(deps): upgrade pgx to v5.9.1, bump go directive to 1.25` |
| Phase 3b: lib/pq v1.12.1 | ⏳ Not Started | `chore(deps): upgrade lib/pq to v1.12.1, requires PostgreSQL 14+` |
