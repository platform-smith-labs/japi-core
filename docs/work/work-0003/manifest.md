# Work Item: work-0003 - Go Module Path v3 Suffix Fix

**Status**: ✅ Completed
**Created**: 2026-03-30
**Last Updated**: 2026-03-30
**Owner**: TBD
**Priority**: High (blocks consumers from importing v3)
**Estimated Effort**: TBD

## Original Request

when I try to use japi-core v3.0 I get this build error and suggested fix
 The proper fix: update japi-core's module path to /v3.

  Go's module system rule is clear: a module tagged v3.x.x that has a go.mod file must declare module .../v3 in that file. The /v3 suffix must be in both the module declaration AND all import paths. This is not optional — it's how Go resolves
  major version upgrades correctly.

  What needs to change

  1. In japi-core (you do this first, before platform-smith-api can consume it):

  // go.mod — change module path
  module github.com/platform-smith-labs/japi-core/v3   // was: .../japi-core

  Every internal import in japi-core's own source files that references itself:
  // Before
  import "github.com/platform-smith-labs/japi-core/handler"
  import "github.com/platform-smith-labs/japi-core/middleware/typed"

  // After
  import "github.com/platform-smith-labs/japi-core/v3/handler"
  import "github.com/platform-smith-labs/japi-core/v3/middleware/typed"

  Re-tag as v3.1.0 (or v3.1.1 if you want a new release after the path fix).
do a deep analysis and figure out a proper fix

## Description

japi-core is tagged at v3.x.x but its go.mod declares `module github.com/platform-smith-labs/japi-core` without the `/v3` suffix. Go's module system requires that any module at major version ≥ 2 include the major version suffix in its module path. This causes a build error for consumers attempting to import japi-core as a v3 module. The fix requires updating go.mod, all internal self-imports, and re-tagging.

## Workflow Progress

- [x] Research
- [x] Requirements
- [x] Planning
- [x] Implementation
- [x] Validation
- [ ] Deployment

## Artifacts

### Research
- [0001: Module Path v3 Fix Research](./research/0001-module-path-v3-research.md) (2026-03-30)

### Requirements
- [0001: Module Path v3 Fix Requirements](./requirements/0001-module-path-v3-req.md) (2026-03-30)

### Issues

### Plans
- [Master Plan](./plans/master.md) (2026-03-30)

### Implementation

## Journal Sessions

## Key Decisions

## Dependencies

## Change Log

- 2026-03-30: Work item created
- 2026-03-30: Research completed → `./research/0001-module-path-v3-research.md`
- 2026-03-30: Requirements documented → `./requirements/0001-module-path-v3-req.md`
- 2026-03-30: Status → 📝 Requirements
- 2026-03-30: Implementation plan created → `./plans/master.md`
- 2026-03-30: Status → 🎨 Planning
- 2026-03-30: Implementation completed — commit bc3e96a, tagged v3.2.0
- 2026-03-30: Status → ✅ Completed
