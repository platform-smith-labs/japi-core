# Go Module Path v3 Fix — Implementation Plan

**Work Item**: work-0003
**Created**: 2026-03-30
**Status**: Ready for Implementation

---

## Documentation Chain

- **Manifest**: [../manifest.md](../manifest.md)
- **Research**: [../research/0001-module-path-v3-research.md](../research/0001-module-path-v3-research.md)
- **Requirements**: [../requirements/0001-module-path-v3-req.md](../requirements/0001-module-path-v3-req.md)

---

## Overview

japi-core is tagged at v3.x.x but `go.mod` declares the module path without the required `/v3` suffix. Go's module system requires this suffix for any major version ≥ 2. This is a **purely mechanical fix** — no API, behaviour, or logic changes. The entire fix is:

1. Add `/v3` to the module path in `go.mod`
2. Add `/v3` to all internal self-imports in Go source files (23 lines across 15 files)
3. Add `retract` directives for the broken v3.0.0 and v3.1.0 tags
4. Update `README.md` and `CLAUDE.md` import examples
5. Commit as one atomic change and tag `v3.2.0`

**Files changed:** `go.mod`, `go.sum`, 15 `.go` files, `README.md`, `CLAUDE.md`
**No source logic changes whatsoever.**

---

## Current State

```
module github.com/platform-smith-labs/japi-core   ← WRONG

go 1.25.0
toolchain go1.25.8
```

Latest tag: `v3.1.0` (broken — consumers cannot import as v3)

---

## What We Are NOT Doing

- No API changes
- No test logic changes
- No deletion of existing `v3.0.0` or `v3.1.0` remote tags (module proxy immutability)
- No `v3/` subdirectory (major branch strategy, not major subdirectory strategy)
- No changes to v1 or v2 branches

---

## Phase 1 — Fix Module Path and Re-tag as v3.2.0

**Single atomic commit covering all changes.**
**Commit message:** `fix!: add /v3 suffix to module path for Go major version compliance`

### Pre-flight Verification

Confirm current broken state:

```bash
head -1 go.mod
# Expected: module github.com/platform-smith-labs/japi-core

grep -r '"github.com/platform-smith-labs/japi-core/' --include="*.go" . | wc -l
# Expected: ~23 lines (all missing /v3)
```

---

### Step 1: Update go.mod module path

```bash
sed -i '' \
  's|^module github.com/platform-smith-labs/japi-core$|module github.com/platform-smith-labs/japi-core/v3|' \
  go.mod
```

Verify:
```bash
head -1 go.mod
# Must show: module github.com/platform-smith-labs/japi-core/v3
```

---

### Step 2: Update all internal import paths in Go source files

A single `sed` pass covers all 15 files and 23 import lines. The pattern only matches quoted import strings — it cannot match the unquoted `go.mod` module declaration line.

```bash
find . -name "*.go" -not -path "*/vendor/*" \
  -exec sed -i '' \
    's|"github\.com/platform-smith-labs/japi-core/|"github.com/platform-smith-labs/japi-core/v3/|g' \
    {} +
```

**Files this touches (confirmed by research):**

Production (12 files):
- `router/chi.go` — imports `japi-core/core`
- `handler/adapter.go` — imports `japi-core/core`
- `handler/nullable.go` — imports `japi-core/core`
- `swagger/routes.go` — imports `japi-core/handler`
- `swagger/generator.go` — imports `japi-core/handler`
- `middleware/typed/request.go` — imports `japi-core/core`, `japi-core/handler`
- `middleware/typed/response.go` — imports `japi-core/core`, `japi-core/handler`
- `middleware/typed/auth.go` — imports `japi-core/core`, `japi-core/handler`, `japi-core/jwt`
- `middleware/typed/json.go` — imports `japi-core/core`, `japi-core/handler`
- `middleware/typed/csv.go` — imports `japi-core/core`, `japi-core/handler`
- `middleware/typed/logging.go` — imports `japi-core/handler`
- `middleware/typed/request_id.go` — imports `japi-core/handler`, `japi-core/middleware/http` (aliased)

Test (3 files):
- `router/chi_test.go` — imports `japi-core/router`
- `handler/context_test.go` — imports `japi-core/core`
- `middleware/typed/request_id_test.go` — imports `japi-core/handler`, `japi-core/middleware/http`

Verify zero bare imports remain:
```bash
grep -r '"github.com/platform-smith-labs/japi-core/' --include="*.go" . | grep -v '/v3/'
# Must return empty (0 lines)
```

---

### Step 3: Add retract directives to go.mod

Manually edit `go.mod` to add the `retract` block. The final `go.mod` preamble should look like:

```
module github.com/platform-smith-labs/japi-core/v3

go 1.25.0

toolchain go1.25.8

retract (
	v3.1.0 // module path missing /v3 suffix — broken for consumers
	v3.0.0 // module path missing /v3 suffix — broken for consumers
)
```

The `retract` block goes after the `toolchain` line and before the `require` blocks.

---

### Step 4: Update documentation

Update import and `go get` examples in `README.md` and `CLAUDE.md`:

```bash
# Update import path examples in docs (excluding work/ and journal/ internal docs):
find . -name "*.md" \
  -not -path "*/docs/work/*" \
  -not -path "*/docs/journal/*" \
  -exec sed -i '' \
    's|github\.com/platform-smith-labs/japi-core/|github.com/platform-smith-labs/japi-core/v3/|g' \
    {} +

# Fix bare go get lines (no trailing slash — a separate pattern):
sed -i '' \
  's|go get github\.com/platform-smith-labs/japi-core$|go get github.com/platform-smith-labs/japi-core/v3|g' \
  README.md CLAUDE.md
```

---

### Step 5: go mod tidy

```bash
go mod tidy
```

`go mod tidy` may add a self-referential entry to `go.sum` for the `retract` directives — this is expected and correct.

---

### Step 6: Verify

```bash
go build ./...     # Must exit 0
go vet ./...       # Must exit 0
go test ./...      # Must exit 0
go test -race ./...  # Must exit 0
```

Post-fix `govulncheck` (optional sanity check — should remain at 0):
```bash
~/go/bin/govulncheck ./...
```

---

### Step 7: Commit

```bash
git add go.mod go.sum \
  router/chi.go router/chi_test.go \
  handler/adapter.go handler/nullable.go handler/context_test.go \
  swagger/routes.go swagger/generator.go \
  middleware/typed/auth.go middleware/typed/request_id.go \
  middleware/typed/request_id_test.go \
  middleware/typed/request.go middleware/typed/response.go \
  middleware/typed/json.go middleware/typed/csv.go \
  middleware/typed/logging.go \
  README.md CLAUDE.md

git commit -m "fix!: add /v3 suffix to module path for Go major version compliance

Go's module system requires major version >= 2 to include the major version
suffix in the module path (go.dev/ref/mod#major-version-suffixes).

Changes:
- go.mod: module path → github.com/platform-smith-labs/japi-core/v3
- 15 Go files: internal imports updated (23 import lines total)
- go.mod: retract v3.0.0 and v3.1.0 (had wrong module path)
- README.md, CLAUDE.md: import/go get examples updated

Consumers must update their import paths:
  Before: github.com/platform-smith-labs/japi-core/handler
  After:  github.com/platform-smith-labs/japi-core/v3/handler"
```

---

### Step 8: Tag and push

```bash
git tag -a v3.2.0 -m "Release v3.2.0: fix Go module path /v3 suffix"
git push origin main
git push origin v3.2.0
```

Do NOT delete v3.0.0 or v3.1.0 from remote.

---

## Success Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| AC-1 | go.mod line 1 is `.../japi-core/v3` | `head -1 go.mod` |
| AC-2 | Zero bare internal imports remain | `grep -r '"...japi-core/' --include="*.go" . \| grep -v '/v3/'` returns empty |
| AC-3 | go.mod retracts v3.0.0 and v3.1.0 | `grep retract go.mod` |
| AC-4 | `go build ./...` exits 0 | Build |
| AC-5 | `go test ./...` exits 0 | Test |
| AC-6 | `go test -race ./...` exits 0 | Race test |
| AC-7 | `go vet ./...` exits 0 | Vet |
| AC-8 | `go mod tidy` produces no diff | `git diff --exit-code go.sum` |
| AC-9 | README.md uses `/v3` in all examples | `grep` check |
| AC-10 | Tag `v3.2.0` on remote | `git ls-remote --tags origin \| grep v3.2.0` |

---

## Recommended Agents for Implementation

- **dependency-manager** — Run the `sed` commands, verify `go.mod`/`go.sum` correctness after tidy.
- **code-reviewer** — Before committing: confirm only import paths changed (no logic changes), retract directives are syntactically correct.
- `/commit` — Create the conventional commit with the exact message specified above.

---

## Progress Tracking

| Phase | Status | Commit |
|-------|--------|--------|
| Phase 1: Module path fix + v3.2.0 | ⏳ Not Started | `fix!: add /v3 suffix to module path for Go major version compliance` |
