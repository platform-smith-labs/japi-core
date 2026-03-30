# Work Item: work-0001 - CORS AllowedMethods Fix & Override API

**Status**: ✅ Completed
**Created**: 2026-03-30
**Last Updated**: 2026-03-30
**Estimated Effort**: Small (2 phases, 3 files)
**Owner**: TBD
**Priority**: High (production bug causing PATCH requests to fail)
**Estimated Effort**: TBD - to be determined during planning

## Original Request

Playwright debugging confirmed the root cause: workspace edit (PATCH) fails due to CORS.

Access to fetch at 'http://localhost:9004/api/v1/workspaces/{id}' from origin
'http://localhost:8080' has been blocked by CORS policy: Response to preflight request
doesn't pass access control check: No 'Access-Control-Allow-Origin' header is present
on the requested resource.

Network trace: GET → 200 ✅, POST → 201 ✅, PATCH → net::ERR_FAILED ❌

Root cause: japi-core/router/chi.go defines AllowedMethods without "PATCH":
AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
//                                                  ^^^ PATCH missing

Do a deep analysis and give me a fix for this. Also create an option that enables a app using japi-core to pass an explicit list. The idea is that if the app does not call this override function, then we use the default list.

## Description

`router/chi.go` hardcodes `AllowedMethods` without PATCH or HEAD in both router constructors, causing CORS preflight failures for all browser-initiated PATCH requests. Fix adds PATCH and HEAD to the default list and introduces a `CORSConfig` struct with `NewChiRouterWithCORSConfig()` and `DefaultCORSConfig()` so consuming apps can override the allowed methods list without forking the library. All existing call sites remain unmodified.

## Workflow Progress

- [x] Research
- [x] Requirements
- [x] Planning
- [x] Implementation
- [x] Validation
- [ ] Deployment

## Artifacts

### Research
- [0001: Initial Research](./research/0001-cors-methods-research.md) (auto-created)
- Add more with `/research --work work-0001 "topic"`

### Requirements
- [0001: Initial Requirements](./requirements/0001-cors-methods-req.md) (auto-created)
- Add more with `/new_req --work work-0001 "topic"`

### Issues
- Add with `/new_issue --work work-0001 "issue description"`

### Plans
- [Master Plan](./plans/master.md)

### Implementation
- [Implementation Status](./implementation/status.md) (when started)

## Journal Sessions

## Key Decisions

## Dependencies

## Change Log

- 2026-03-30: Work item created
- 2026-03-30: Research completed → `./research/0001-cors-methods-research.md`
- 2026-03-30: Requirements documented → `./requirements/0001-cors-methods-req.md`
- 2026-03-30: Status → 📝 Requirements
- 2026-03-30: Research + requirements updated — override mechanism changed from CORSConfig struct to functional options pattern
- 2026-03-30: Implementation plan created → `./plans/master.md`
- 2026-03-30: Status → 🎨 Planning
- 2026-03-30: Implementation completed — Phase 1 (router/chi.go refactor + tests) + Phase 2 (README update)
- 2026-03-30: Status → ✅ Completed — all 4 tests pass, all 10 acceptance criteria met
