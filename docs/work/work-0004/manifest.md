# Work Item: work-0004 - Generic Service Injection for Handler Registry

**Status**: ✅ Completed
**Created**: 2026-04-01
**Last Updated**: 2026-04-01
**Owner**: Syam Krishnan
**Priority**: High
**Estimated Effort**: TBD - to be determined during planning

## Original Request

Add generic service injection to the handler registry so that applications consuming japi-core can inject arbitrary typed dependencies (HTTP clients, cache clients, message queues, etc.) into handlers without resorting to global mutable state. Initial prompt sourced from `platform-smith-api/docs/work/work-0006/japi-core-di-prompt.md`.

Additional directive: Do a deep analysis and validate claims in the prompt, evaluate whether there is a better solution that is more generic and extensible.

## Description

`HandlerContext` currently only exposes `DB *sql.DB` and `Logger *slog.Logger` as injectable dependencies. The dependency injection pipeline is sealed: `RegisterWithRouter` -> `Adapt()` -> `AdaptHandler()` -> `HandlerContext{}` with fixed fields. Applications needing additional dependencies (e.g., HTTP clients) are forced into global mutable state patterns with init functions, violating FP principles.

## Workflow Progress

- [x] Research
- [x] Requirements
- [x] Planning
- [x] Implementation
- [x] Validation
- [ ] Deployment

## Artifacts

### Research
- [0001: Generic Service Injection Deep Analysis](./research/0001-generic-service-injection-research.md) (auto-created)
- [0002: Typed Field vs map[string]any Design Analysis](./research/0002-typed-field-vs-map-design-research.md)
- Add more with `/research --work work-0004 "topic"`

### Requirements
- [0001: Generic Service Injection Requirements](./requirements/0001-generic-service-injection-req.md) (auto-created)
- Add more with `/new_req --work work-0004 "topic"`

### Issues
- Add with `/new_issue --work work-0004 "issue description"`

### Plans
- [Master Plan](./plans/master.md) — 3-phase single-file plan

### Implementation
- [Implementation Status](./implementation/status.md) (when started)

## Journal Sessions

## Key Decisions

## Dependencies

## Change Log

- 2026-04-01: Work item created
- 2026-04-01: Research completed with deep analysis and alternative evaluation
- 2026-04-01: Requirements documented
- 2026-04-01: Implementation plan created (3 phases, single-file)
- 2026-04-01: All 3 phases implemented — Services field, functional options, tests, README
