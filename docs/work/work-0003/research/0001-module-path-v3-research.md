---
name: Go Module Path v3 Fix Research
description: Deep analysis of /v3 suffix requirement, all affected files, tagging strategy
type: project
---

# Go Module Path v3 Fix — Research

**Work Item**: work-0003
**Date**: 2026-03-30

---

## Executive Summary

`github.com/platform-smith-labs/japi-core` is tagged at **v3.1.0** but `go.mod` declares the module path without the required `/v3` suffix. Go's module system mandates this suffix for any major version ≥ 2. Consumers attempting `go get github.com/platform-smith-labs/japi-core/v3` receive a build error.

**Three coordinated changes are required:**
1. Update `go.mod` module declaration (1 line)
2. Update all internal self-imports in Go source files (23 import lines across 15 files)
3. Retract v3.0.0 and v3.1.0; tag a corrected **v3.2.0**

---

## 1. Current State

### go.mod (confirmed by reading file)

```
module github.com/platform-smith-labs/japi-core   ← WRONG: missing /v3

go 1.25.0
toolchain go1.25.8
```

- No `replace` directives present.
- `go.sum` has 175 lines covering external dependencies only; no self-references.

### Git Tags

```
v3.1.0   ← latest (wrong module path)
v3.0.0   ← (wrong module path)
v2.0.0   ← (predates this issue)
```

Commits since v3.0.0: 8 commits including toolchain upgrade, dep upgrades, CORS API, Claude workflow.

---

## 2. Go Major Version Module Rule

Per the [Go module reference](https://go.dev/ref/mod#major-version-suffixes):

> If a module is at major version v2 or higher, the module path must end with a major version suffix like `/v2`.

| Version range | Required module path |
|---------------|---------------------|
| v0.x / v1.x | `module example.com/foo` |
| v2.x | `module example.com/foo/v2` |
| v3.x | `module example.com/foo/v3` |

### Required Change to go.mod

```
# Before (WRONG):
module github.com/platform-smith-labs/japi-core

# After (CORRECT):
module github.com/platform-smith-labs/japi-core/v3
```

### Required Change to Internal Imports

Every package within japi-core that imports another japi-core package must insert `/v3`:

```go
// Before:
"github.com/platform-smith-labs/japi-core/core"
"github.com/platform-smith-labs/japi-core/handler"
"github.com/platform-smith-labs/japi-core/jwt"
"github.com/platform-smith-labs/japi-core/middleware/http"
"github.com/platform-smith-labs/japi-core/middleware/typed"
"github.com/platform-smith-labs/japi-core/router"

// After:
"github.com/platform-smith-labs/japi-core/v3/core"
"github.com/platform-smith-labs/japi-core/v3/handler"
"github.com/platform-smith-labs/japi-core/v3/jwt"
"github.com/platform-smith-labs/japi-core/v3/middleware/http"
"github.com/platform-smith-labs/japi-core/v3/middleware/typed"
"github.com/platform-smith-labs/japi-core/v3/router"
```

### Does `go mod tidy` Help?

No. `go mod tidy` only manages dependency resolution and `go.sum`. The module path and import path fixes must be made manually (or via `sed`).

---

## 3. Complete File Inventory

### 3a. Go Source Files — Production (12 files, 20 import lines)

| # | File | Import Lines to Update |
|---|------|----------------------|
| 1 | `router/chi.go` | `japi-core/core` |
| 2 | `handler/adapter.go` | `japi-core/core` |
| 3 | `handler/nullable.go` | `japi-core/core` |
| 4 | `swagger/routes.go` | `japi-core/handler` |
| 5 | `swagger/generator.go` | `japi-core/handler` |
| 6 | `middleware/typed/request.go` | `japi-core/core`, `japi-core/handler` |
| 7 | `middleware/typed/response.go` | `japi-core/core`, `japi-core/handler` |
| 8 | `middleware/typed/auth.go` | `japi-core/core`, `japi-core/handler`, `japi-core/jwt` |
| 9 | `middleware/typed/json.go` | `japi-core/core`, `japi-core/handler` |
| 10 | `middleware/typed/csv.go` | `japi-core/core`, `japi-core/handler` |
| 11 | `middleware/typed/logging.go` | `japi-core/handler` |
| 12 | `middleware/typed/request_id.go` | `japi-core/handler`, `japi-core/middleware/http` (aliased) |

### 3b. Test Files (3 files, 3 import lines)

| # | File | Import Lines to Update |
|---|------|----------------------|
| 13 | `router/chi_test.go` | `japi-core/router` |
| 14 | `handler/context_test.go` | `japi-core/core` |
| 15 | `middleware/typed/request_id_test.go` | `japi-core/handler`, `japi-core/middleware/http` (aliased) |

**Total: 15 Go files, 23 import lines.**

### 3c. Files Confirmed to NOT Require Changes

The following packages have no internal self-imports and need no changes:

- `jwt/jwt.go`, `middleware/http/*.go`, `middleware/validation/*.go`
- `core/*.go`, `db/*.go`, `metrics/*.go`
- `handler/registry_test.go`, `handler/types.go`, `handler/nullable_test.go`
- `router/export_test.go`

### 3d. Documentation Files (Non-Breaking, Should Be Updated)

| File | Nature |
|------|--------|
| `README.md` | ~20 occurrences of `go get` and import examples |
| `CLAUDE.md` | 1 `go get` mention |

---

## 4. Internal Dependency Graph (Confirms No Missed Files)

```
core          ← no internal imports (leaf)
jwt           ← no internal imports (leaf)
middleware/http ← no internal imports (leaf)
db            ← no internal imports (leaf)
metrics       ← no internal imports (leaf)
handler       ← imports core
router        ← imports core
swagger       ← imports handler
middleware/typed ← imports core, handler, jwt, middleware/http
```

Unidirectional, no cycles. The `sed` one-liner covers all files exhaustively.

---

## 5. Tagging Strategy

### Option A — Delete and re-create tags (NOT recommended)

Deleting published tags from `proxy.golang.org` violates Go module immutability. Published modules are cached permanently. Version numbers cannot be re-used for different content.

### Option B — New tag v3.2.0 (RECOMMENDED)

1. Apply the fix, commit it
2. Tag `v3.2.0`
3. Push

### Option C — Retract broken versions (belt-and-suspenders, pair with B)

Add `retract` directives to `go.mod`:

```
retract (
    v3.1.0 // module path missing /v3 suffix — broken for consumers
    v3.0.0 // module path missing /v3 suffix — broken for consumers
)
```

**Recommended**: Option B + Option C. `retract` surfaces a warning in `go get` output when a consumer tries to use the broken versions.

---

## 6. Consumer Impact

Consumers upgrading to v3 must:

1. Update `go.mod`:
   ```
   go get github.com/platform-smith-labs/japi-core/v3@latest
   ```

2. Update all import paths (mechanical find-and-replace):
   ```go
   // Before:
   import "github.com/platform-smith-labs/japi-core/handler"
   // After:
   import "github.com/platform-smith-labs/japi-core/v3/handler"
   ```

This is a one-time, mechanical migration. The Go team's standard major version upgrade pattern.

---

## 7. Exact Execution Commands

### Step 1: Update go.mod module path

```bash
sed -i '' 's|^module github.com/platform-smith-labs/japi-core$|module github.com/platform-smith-labs/japi-core/v3|' go.mod
```

### Step 2: Update all internal imports in Go files

```bash
find . -name "*.go" -not -path "*/vendor/*" \
  -exec sed -i '' \
    's|"github\.com/platform-smith-labs/japi-core/|"github.com/platform-smith-labs/japi-core/v3/|g' {} +
```

Safe: the pattern matches only quoted import strings. The `go.mod` module declaration line is unquoted and is not affected.

### Step 3: Add retract directives to go.mod

Manually add before end of file:
```
retract (
    v3.1.0 // module path missing /v3 suffix — broken for consumers
    v3.0.0 // module path missing /v3 suffix — broken for consumers
)
```

### Step 4: Update README.md and CLAUDE.md

```bash
# Update import path examples in docs:
find . -name "*.md" -not -path "*/docs/work/*" -not -path "*/docs/journal/*" \
  -exec sed -i '' \
    's|github\.com/platform-smith-labs/japi-core/|github.com/platform-smith-labs/japi-core/v3/|g' {} +

# Fix bare go get (no trailing slash):
sed -i '' \
  's|go get github\.com/platform-smith-labs/japi-core$|go get github.com/platform-smith-labs/japi-core/v3|g' \
  README.md CLAUDE.md
```

### Step 5: go mod tidy + verify

```bash
go mod tidy
go build ./...
go vet ./...
go test ./...
```

### Step 6: Commit and tag

```bash
git add go.mod go.sum README.md CLAUDE.md \
  router/chi.go router/chi_test.go \
  handler/adapter.go handler/nullable.go handler/context_test.go \
  swagger/routes.go swagger/generator.go \
  middleware/typed/auth.go middleware/typed/request_id.go middleware/typed/request_id_test.go \
  middleware/typed/request.go middleware/typed/response.go \
  middleware/typed/json.go middleware/typed/csv.go middleware/typed/logging.go

git commit -m "fix!: add /v3 suffix to module path for Go major version compliance

Go's module system requires major version >= 2 to include the major version
suffix in the module path. Updates go.mod declaration, all 15 Go files with
internal self-imports (23 import lines total), and documentation.

Retracts v3.0.0 and v3.1.0 which had the incorrect module path and were
not consumable by downstream modules.

Consumers must update their imports:
  Before: github.com/platform-smith-labs/japi-core/handler
  After:  github.com/platform-smith-labs/japi-core/v3/handler"

git tag v3.2.0
git push origin main v3.2.0
```

---

## 8. Change Count Summary

| Category | Count |
|----------|-------|
| `go.mod` lines changed | 1 |
| Production `.go` files | 12 |
| Test `_test.go` files | 3 |
| **Total Go files** | **15** |
| Total import lines updated | 23 |
| Documentation files updated | 2 (README.md, CLAUDE.md) |
| `retract` entries added | 2 |
| New tag | v3.2.0 |
