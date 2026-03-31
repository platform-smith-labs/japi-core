---
name: Go Module Path v3 Fix Requirements
description: Requirements for adding /v3 suffix to module path and re-tagging as v3.2.0
type: project
---

# Go Module Path v3 Fix — Requirements

**Work Item**: work-0003
**Date**: 2026-03-30
**Research Reference**: `docs/work/work-0003/research/0001-module-path-v3-research.md`

---

## 1. Overview and Objectives

japi-core is tagged at v3.x.x but its `go.mod` module path does not carry the `/v3` suffix required by Go's module system for major version ≥ 2. This makes the module unimportable by any downstream consumer using the v3 path. The fix is mechanical and well-understood: update `go.mod`, update all internal imports, add `retract` directives for the broken tags, and release as v3.2.0.

**Primary objectives:**
- Make japi-core importable as `github.com/platform-smith-labs/japi-core/v3` by downstream consumers.
- Retract the broken v3.0.0 and v3.1.0 tags so `go get` warns consumers away from them.
- Leave the project in a fully green build and test state.
- **No functional changes** — this is a pure module path correction.

**Out of scope:**
- Any API changes to japi-core.
- Migrating consumers (platform-smith-api or others) — that is a separate work item.
- Changes to v1 or v2 branches.

---

## 2. Functional Requirements

### REQ-1: Update go.mod Module Path

**What must happen:**
- Line 1 of `go.mod` changed from:
  `module github.com/platform-smith-labs/japi-core`
  to:
  `module github.com/platform-smith-labs/japi-core/v3`

**Acceptance condition:** `head -1 go.mod` outputs `module github.com/platform-smith-labs/japi-core/v3`.

---

### REQ-2: Update All Internal Import Paths in Go Source Files

**What must happen:**
All 15 Go files that contain internal self-imports must have their import paths updated from `github.com/platform-smith-labs/japi-core/<pkg>` to `github.com/platform-smith-labs/japi-core/v3/<pkg>`.

**Complete file list (from research):**

Production files (12):
- `router/chi.go`
- `handler/adapter.go`, `handler/nullable.go`
- `swagger/routes.go`, `swagger/generator.go`
- `middleware/typed/request.go`, `middleware/typed/response.go`
- `middleware/typed/auth.go`, `middleware/typed/json.go`
- `middleware/typed/csv.go`, `middleware/typed/logging.go`
- `middleware/typed/request_id.go`

Test files (3):
- `router/chi_test.go`
- `handler/context_test.go`
- `middleware/typed/request_id_test.go`

**Acceptance condition:** `grep -r '"github.com/platform-smith-labs/japi-core/' --include="*.go" .` returns only lines containing `/v3/` (zero lines without it).

---

### REQ-3: Add retract Directives for Broken Tags

**What must happen:**
`go.mod` must contain a `retract` block for the two tags that carried the wrong module path:

```
retract (
    v3.1.0 // module path missing /v3 suffix — broken for consumers
    v3.0.0 // module path missing /v3 suffix — broken for consumers
)
```

**Rationale:** Go's `retract` directive causes `go get` to display a warning when a consumer tries to use a retracted version. It signals that v3.0.0 and v3.1.0 are broken without requiring tag deletion (which would violate module proxy immutability).

**Acceptance condition:** `grep -A3 "retract" go.mod` shows both v3.0.0 and v3.1.0 retracted.

---

### REQ-4: Update Documentation Import Examples

**What must happen:**
- `README.md`: All `go get` lines and import path examples updated to use `/v3`.
- `CLAUDE.md`: Any `go get` reference updated.

**Acceptance condition:** `grep -r 'platform-smith-labs/japi-core[^/v]' README.md CLAUDE.md` returns no lines (i.e., all bare references include `/v3`).

---

### REQ-5: go build, go test, go vet Must Pass

After all changes:
- `go build ./...` exits 0.
- `go vet ./...` exits 0.
- `go test ./...` exits 0.
- `go test -race ./...` exits 0.
- `go mod tidy` produces no diff.

---

### REQ-6: Release as v3.2.0

**What must happen:**
- A single atomic commit containing all changes in REQ-1 through REQ-4.
- Annotated git tag `v3.2.0` on that commit.
- Both `main` branch and `v3.2.0` tag pushed to origin.
- The existing `v3.0.0` and `v3.1.0` tags are **not deleted** from remote (module proxy immutability).

**Rationale for v3.2.0 over v3.1.1:** Although this is a fix, it is a highly visible, consumer-impacting change. v3.2.0 communicates a deliberate, substantive release. v3.1.1 could be misread as a trivial patch.

**Acceptance condition:** `git tag --sort=-v:refname | head -3` shows `v3.2.0` as the latest tag.

---

## 3. Non-Functional Requirements

### NFR-1: Single Atomic Commit

All changes (go.mod, imports, docs) must be in one commit. This ensures the module path fix is atomic — a partial commit with only go.mod updated but imports not yet updated would leave the repo in a broken state.

### NFR-2: No Functional Changes

No behaviour changes to any API. No test logic changes. If a test file requires an import path update, only the import line changes — never the test logic.

### NFR-3: go.sum Integrity

After `go mod tidy`, `go.sum` must match `go.mod`. The `retract` addition may cause `go mod tidy` to add a self-referential entry — this is normal and expected.

---

## 4. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| AC-1 | `go.mod` line 1 is `module github.com/platform-smith-labs/japi-core/v3` | `head -1 go.mod` |
| AC-2 | Zero Go files contain bare (non-v3) internal import paths | `grep -r '"github.com/platform-smith-labs/japi-core/' --include="*.go" .` shows only `/v3/` lines |
| AC-3 | `go.mod` retracts v3.0.0 and v3.1.0 | `grep retract go.mod` |
| AC-4 | `go build ./...` exits 0 | Build |
| AC-5 | `go test ./...` exits 0 | Test |
| AC-6 | `go test -race ./...` exits 0 | Race test |
| AC-7 | `go vet ./...` exits 0 | Vet |
| AC-8 | `go mod tidy` produces no diff | `git diff --exit-code go.sum` |
| AC-9 | README.md and CLAUDE.md use `/v3` in all import/go get examples | `grep` check |
| AC-10 | Tag `v3.2.0` exists on remote | `git ls-remote --tags origin` |

---

## 5. Constraints and Assumptions

- Do **not** delete `v3.0.0` or `v3.1.0` from remote — module proxy immutability.
- The `sed` one-liner is safe to use because the quoted import pattern cannot match the unquoted go.mod module line.
- No v1 or v2 branch changes are in scope.
- `go mod tidy` may add a self-module entry to `go.sum` for the retract directives — this is expected behaviour.
