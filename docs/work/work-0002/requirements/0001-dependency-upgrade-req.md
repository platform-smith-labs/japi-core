---
name: Go Dependency Upgrade Requirements
description: Requirements for upgrading go.mod dependencies to latest versions
type: project
---

# Go Dependency Upgrade Requirements

Work Item: work-0002
Date: 2026-03-30
Research Reference: `docs/work/work-0002/research/0001-dependency-upgrade-research.md`

---

## 1. Overview and Objectives

japi-core currently carries three security vulnerabilities in transitive dependencies (`golang.org/x/crypto`, `golang.org/x/net`) and several direct dependencies that have fallen behind their latest stable releases. This requirements document defines what must be done, in what order, and to what standard of verification, to bring all dependencies to their appropriate latest versions without introducing regressions.

**Primary objectives:**

- Eliminate all known CVEs affecting the dependency tree immediately.
- Advance all direct dependencies that have safe, non-breaking upgrades available.
- Define a safe, gated upgrade strategy for the two dependencies requiring additional care (pgx, lib/pq).
- Leave the project in a fully green test and build state after every phase.

**Toolchain note (verified 2026-03-30):** Build machine runs **Go 1.25.1**, satisfying pgx v5.9.1's Go 1.25 requirement. The `go.mod` directive will be updated to `go 1.25.0` as part of the pgx upgrade.

**lib/pq usage (verified 2026-03-30):** `lib/pq` appears only as a blank import `_ "github.com/lib/pq"` in `db/query_context_test.go`. It is not used in production code. No deprecated symbol migration is needed.

**Out of scope:**

- Changes to the public API of japi-core.
- Adding or removing dependencies.

---

## 2. Functional Requirements

### REQ-1: Go Toolchain Upgrade to go1.25.8 (Highest Priority)

**What must happen:**
- Go toolchain upgraded from go1.25.1 to **go1.25.8**.
- `go.mod` `toolchain` directive set to `go1.25.8`.

**Rationale (verified by `govulncheck ./...` on 2026-03-30):**
The current go1.25.1 toolchain has **12 active vulnerabilities** in the standard library that japi-core's code directly calls:
- `crypto/tls`: GO-2026-4340, GO-2026-4337, GO-2025-4008 (handshake/session/ALPN issues)
- `crypto/x509`: GO-2025-4175, GO-2025-4155, GO-2025-4013, GO-2025-4007 (cert validation issues)
- `net/url`: GO-2026-4601, GO-2026-4341, GO-2025-4010 (URL parsing issues)
- `encoding/asn1`: GO-2025-4011 (DER parsing memory exhaustion)
- `encoding/pem`: GO-2025-4009 (quadratic complexity)

All are fixed by go1.25.8. This is a patch release — no API changes.

Additionally, `x/crypto` (GO-2025-4116/4134/4135) and `x/net` (GO-2026-4440/4441) have real vulnerabilities in their `ssh` and `html` sub-packages respectively, but govulncheck confirms japi-core's code does not call the vulnerable paths. Upgrading them as part of Phase 2 is good hygiene but they are not actively exploitable in japi-core.

**Acceptance condition:** `govulncheck ./...` Symbol Results section is empty (0 active vulnerabilities) after toolchain upgrade.

---

### REQ-2: Safe Direct Dependency Upgrades

**What must happen:**
All of the following direct dependencies must be upgraded to the versions listed. Research confirms none introduce breaking API changes relative to japi-core's declared minimum Go version of 1.24.

| Dependency | From | To |
|---|---|---|
| `github.com/go-chi/chi/v5` | v5.2.3 | v5.2.5 |
| `github.com/go-playground/validator/v10` | v10.28.0 | v10.30.1 |
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | v5.3.1 |
| `github.com/swaggo/swag` | v1.16.4 | v1.16.6 |
| `github.com/go-openapi/spec` | v0.21.0 | v0.22.4 |
| `google.golang.org/protobuf` | v1.36.8 | v1.36.11 |
| `github.com/go-openapi/jsonpointer` | v0.21.0 | v0.22.5 |

Additionally, all `golang.org/x/*` packages that do not yet carry security fixes (e.g. `x/text`, `x/sys`, `x/sync`) should be brought to their latest stable versions as part of this phase to reduce future drift.

**Acceptance condition:** `go build ./...` and `go test ./...` both pass cleanly after applying all upgrades in this group.

---

### REQ-3: pgx Upgrade to v5.9.1

pgx v5 is japi-core's primary database driver. Go 1.25.1 is available on the build machine, satisfying v5.9.1's toolchain requirement. The upgrade proceeds directly to v5.9.1 in a single step.

**What must happen:**
- `github.com/jackc/pgx/v5` upgraded from v5.7.6 to v5.9.1.
- `go.mod` `go` directive updated from `1.24.0` to `1.25.0` (required by pgx v5.9.1; `go get` will set this automatically).
- No pgx public API changes across this range — upgrade is non-breaking from japi-core's perspective.
- Notable improvements included: SCRAM-SHA-256-PLUS channel binding, PostgreSQL 18 OAuth support, TSVector type, LRU statement cache, DoS/OOM security fixes, removal of `x/crypto` internal dependency.

**Acceptance condition:** All existing database-layer tests pass. `go.mod` declares `go 1.25.0`.

---

### REQ-4: lib/pq Upgrade to v1.12.1

**Verified facts (2026-03-30):**
- `lib/pq` appears only as `_ "github.com/lib/pq"` (blank import) in `db/query_context_test.go`. No production code imports it directly.
- No deprecated symbols (`NullTime`, `CopyIn`, `CopyInToSchema`) are used anywhere in japi-core.
- Go 1.21 minimum (v1.11.0 requirement) is already satisfied by Go 1.25.

**What must happen:**
- `github.com/lib/pq` upgraded from v1.10.0 to v1.12.1.
- Document the PostgreSQL 14+ constraint introduced by v1.11.0 in CHANGELOG or release notes so downstream consumers running tests with `database/sql` + lib/pq driver are aware.

**Acceptance condition:** `go build ./...` and `go test ./...` pass. PostgreSQL 14+ constraint is documented.

---

### REQ-5: All Tests Must Pass After Each Phase

Each upgrade phase must leave the test suite in a fully passing state before the next phase begins. Tests must not be skipped, commented out, or marked as expected-to-fail in order to satisfy this requirement. Specifically:

- `go test ./...` must exit 0.
- `go test -race ./...` must exit 0 (no data races introduced).
- No test files may be modified to accommodate an upgrade unless the modification reflects a deliberate API migration (which must be documented in the commit message).

---

### REQ-6: go build Must Succeed

After every phase, `go build ./...` must complete without errors or warnings. This includes all packages, examples, and test utilities within the module. The `go vet ./...` linter must also pass cleanly.

---

## 3. Non-Functional Requirements

### NFR-1: Phased Rollout — No Big-Bang Upgrades

All upgrades must be applied in discrete, reviewable phases as defined in Section 6. Each phase must be committed and verified independently. A single commit that upgrades all dependencies simultaneously is not acceptable; it makes regression bisection impossible.

### NFR-2: Rollback Readiness

Each phase commit must be self-contained such that a `git revert` of that single commit fully reverts the upgrade. There must be no cross-phase dependencies within a single commit. If a phase introduces a regression that cannot be resolved within the phase, it must be reverted rather than masked.

### NFR-3: go.sum Integrity

After every upgrade, `go mod tidy` must be run and the resulting `go.sum` changes must be committed together with the `go.mod` changes. A `go.sum` file that does not match `go.mod` is not acceptable.

### NFR-4: Go Toolchain

The build machine runs Go 1.25.1. The `go.mod` `go` directive will be updated from `1.24.0` to `1.25.0` as part of the pgx v5.9.1 upgrade (Phase 3). All other phases keep the directive at `1.24.0`. No phase may require a toolchain version beyond 1.25.

### NFR-5: No Indirect-to-Direct Promotions Without Review

Running `go mod tidy` may change some indirect dependencies from indirect to direct if the upgrade resolves transitive chains differently. Any such promotion must be reviewed. If an indirect dependency is promoted to direct and japi-core does not intentionally use it, investigate whether a transitive import has been inadvertently introduced.

### NFR-6: Security Scanning Baseline

Before starting Phase 1, run a baseline vulnerability scan (`govulncheck ./...` or equivalent) and record the findings. After Phase 1, re-run and confirm all identified CVEs are resolved. This baseline-and-verify step is mandatory.

---

## 4. Acceptance Criteria

The dependency upgrade work item is complete when all of the following are true:

| # | Criterion | Verification Method |
|---|---|---|
| AC-1 | `govulncheck ./...` Symbol Results section is empty (0 active vulnerabilities) | Run govulncheck after Phase 1 |
| AC-2 | All safe direct dependencies listed in REQ-2 are at their target versions | `go list -m all` output |
| AC-3 | `github.com/jackc/pgx/v5` is at v5.9.1 and `go.mod` declares `go 1.25.0` | `go list -m github.com/jackc/pgx/v5` + `head -3 go.mod` |
| AC-4 | `github.com/lib/pq` is at v1.12.1 and no direct import exists in source | `go list -m github.com/lib/pq` + grep |
| AC-5 | `go test ./...` exits 0 | CI run |
| AC-6 | `go test -race ./...` exits 0 | CI run |
| AC-7 | `go build ./...` exits 0 | CI run |
| AC-8 | `go vet ./...` exits 0 | CI run |
| AC-9 | `go mod tidy` produces no diff after upgrades are committed | `git diff --exit-code go.sum` |
| AC-10 | Each upgrade phase is in a separate, atomic commit | `git log --oneline` |
| AC-11 | PostgreSQL 14+ constraint is documented | CHANGELOG or release notes |
| AC-12 | PostgreSQL 14+ constraint for lib/pq is documented in CHANGELOG or release notes | Review CHANGELOG |

---

## 5. Constraints and Assumptions

**Constraints:**

- The `go.mod` `go` directive will move from `1.24.0` to `1.25.0` as part of the pgx upgrade. This is the only directive change in scope.
- PostgreSQL 14+ is assumed for test environments (lib/pq v1.11.0 hard requirement). This constraint must be documented for downstream consumers.
- `github.com/prometheus/client_golang` is already at its latest version (v1.23.2) and requires no action.
- `github.com/georgysavva/scany/v2`, `github.com/go-chi/cors`, `github.com/gocarina/gocsv`, and `github.com/google/uuid` were not flagged in research as having material upgrades available; they are out of scope.

**Assumptions:**

- **Verified:** japi-core does not import `lib/pq` directly — only a blank import in `db/query_context_test.go`.
- **Verified:** No deprecated lib/pq symbols (`NullTime`, `CopyIn`) are used anywhere in japi-core.
- No downstream consumers depend on the pre-fix (buggy) `RouteHeaders` double-invocation behaviour fixed in chi v5.2.5.
- The test suite accurately reflects production behaviour. If tests pass but production behaviour regresses, that is a test coverage gap, not a scope item for this work.
- CI infrastructure is available to run `go test -race ./...` as part of verification.

---

## 6. Upgrade Strategy (Phased Approach)

### Phase 1 — Security-Critical Patches (Immediate)

**Scope:** `golang.org/x/crypto`, `golang.org/x/net`, and any other `golang.org/x/*` packages.

**Steps:**
1. Run baseline `govulncheck ./...` and record output.
2. Run `go get golang.org/x/crypto@v0.49.0 golang.org/x/net@v0.52.0` plus latest for remaining `x/` packages.
3. Run `go mod tidy`.
4. Run `go build ./...`, `go vet ./...`, `go test ./...`, `go test -race ./...`.
5. Re-run `govulncheck ./...` and confirm CVEs are resolved.
6. Commit as a single atomic commit: `fix(deps): upgrade x/crypto and x/net to resolve CVEs`.

**Go/no-go gate:** All tests pass, all CVEs resolved.

---

### Phase 2 — Safe Direct Dependency Upgrades

**Scope:** chi, validator, jwt, swag, go-openapi/spec, protobuf, go-openapi/jsonpointer.

**Steps:**
1. Run `go get` for each package at its target version (see REQ-2 table).
2. Run `go mod tidy`.
3. Run `go build ./...`, `go vet ./...`, `go test ./...`, `go test -race ./...`.
4. Commit as a single atomic commit: `chore(deps): upgrade safe direct dependencies`.

**Go/no-go gate:** All tests pass. No new `govulncheck` findings introduced.

---

### Phase 3 — pgx v5.9.1 and lib/pq v1.12.1 (Separate Commits)

These two packages must be committed separately from each other.

**Phase 3a — pgx v5.9.1:**
1. Run `go get github.com/jackc/pgx/v5@v5.9.1` (will automatically update `go` directive to `1.25.0`).
2. Run `go mod tidy`.
3. Run `go build ./...`, `go vet ./...`, `go test ./...`, `go test -race ./...`.
4. Confirm `go.mod` declares `go 1.25.0`.
5. Commit: `chore(deps): upgrade pgx to v5.9.1, bump go directive to 1.25`.

**Phase 3b — lib/pq v1.12.1:**
1. Run `go get github.com/lib/pq@v1.12.1`.
2. Run `go mod tidy`.
3. Run `go build ./...`, `go vet ./...`, `go test ./...`.
4. Update CHANGELOG or release notes to document the PostgreSQL 14+ constraint for consumers using `database/sql` + lib/pq driver in tests.
5. Commit: `chore(deps): upgrade lib/pq to v1.12.1, requires PostgreSQL 14+`.

**Go/no-go gate (Phase 3):** All tests pass. PostgreSQL constraint documented.

---

## 7. References

- Research document: `docs/work/work-0002/research/0001-dependency-upgrade-research.md`
- Work item: `work-0002`
- Current `go.mod` direct dependencies baseline recorded in research summary table
- CVE references: CVE-2025-22869 (x/crypto HIGH), plus four additional CVEs across x/crypto and x/net
