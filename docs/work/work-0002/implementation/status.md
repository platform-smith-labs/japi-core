# Implementation Status: work-0002

**Status**: ✅ Completed
**Date**: 2026-03-30

## Phases

| Phase | Commit | Status |
|-------|--------|--------|
| Phase 1: Go toolchain → go1.25.8 | `566b1aa` | ✅ Complete |
| Phase 2: Safe direct deps | `56a37a3` | ✅ Complete |
| Phase 3a: pgx v5.9.1 + go 1.25 | `9204fe0` | ✅ Complete |
| Phase 3b: lib/pq v1.12.1 + CHANGELOG | `ca31f44` | ✅ Complete |

## Key Decisions

- **Phase 1 scope changed from x/crypto/x/net to toolchain upgrade**: govulncheck confirmed all 12 active vulnerabilities were stdlib issues in go1.25.1, fixed by go1.25.8. x/crypto and x/net vulnerabilities exist but are not called by japi-core code.
- **pgx single-step upgrade**: Go 1.25.1 was available on the build machine, allowing direct upgrade to v5.9.1 without intermediate v5.8.0 step.
- **prometheus/client_golang indirect→direct promotion**: Legitimate — `metrics/prometheus.go` directly imports it. Previous `// indirect` marking was incorrect.
- **go-openapi/swag refactoring**: v0.22+ split the `swag` monorepo into sub-packages (`conv`, `jsonname`, `jsonutils`, etc.). All remain `// indirect`.

## Acceptance Criteria Verification

| AC | Criterion | Result |
|----|-----------|--------|
| AC-1 | govulncheck Symbol Results empty | ✅ 0 active vulnerabilities |
| AC-2 | Safe direct deps at target versions | ✅ All at target |
| AC-3 | pgx v5.9.1, go.mod declares go 1.25.0 | ✅ Confirmed |
| AC-4 | lib/pq v1.12.1, no direct import outside test | ✅ Confirmed |
| AC-5 | go test ./... exits 0 | ✅ All phases |
| AC-6 | go test -race ./... exits 0 | ✅ All phases |
| AC-7 | go build ./... exits 0 | ✅ All phases |
| AC-8 | go vet ./... exits 0 | ✅ All phases |
| AC-9 | go mod tidy produces no diff | ✅ All phases |
| AC-10 | Each phase in separate atomic commit | ✅ 4 commits |
| AC-11 | PostgreSQL 14+ constraint documented | ✅ CHANGELOG.md |
