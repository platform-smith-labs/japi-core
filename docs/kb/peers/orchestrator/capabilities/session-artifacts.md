---
type: capability
title: "Session artifacts retrieval"
tags: [artifacts, sessions, rest-api, collect-result, cross-project]
timestamp: 2026-07-07T00:00:00Z
description: "How a peer lists and reads the artifacts a session produced via the REST API"
repo: orchestrator
commit_sha: 6843154
evidence:
  - cmd/handlers/session_artifacts.go
  - cmd/db/artifacts.go
---

# Session artifacts retrieval

**What it does.** Lets a caller read the artifacts a given session produced — named,
typed outputs (e.g. a `result`) that a session emits and the orchestrator persists.
This is the "collect result" read: fetch a finished session's outputs by session UUID.

**How a peer interacts.** `GET /api/v1/sessions/{session_uuid}/artifacts`. Optional
query param `kind` narrows to one artifact kind (e.g. `kind=result`). The scope is
session-only — this endpoint returns the session's own artifacts, not project-scoped
ones. Tenant scope comes from the gateway `X-Company-UUID` header (this is an
internal service behind the ps-api gateway; callers present the gateway headers, no
separate token).

**Observable behavior.** Returns the session's matching artifacts, each already
carrying its **fully resolved content inline** (the caller does not make a second
call to dereference a blob). Each artifact reflects its **current published version**
— the endpoint returns one version per artifact (the latest), not a version history.
Results are ordered by artifact name ascending. A valid session with no matching
artifacts returns `200` with an empty list.

**Contract.** In: path `session_uuid` (UUID); optional query `kind`. Out:
`{"artifacts": [ {name, kind, content, content_type, artifact_version_uuid, status} ]}`.
- `name` — the artifact's logical name (unique per session).
- `kind` — the artifact type (e.g. `result`); the `kind` query filters on this.
- `content` — the full resolved content as a string.
- `content_type` — a media/type hint describing `content` (e.g. a MIME-like string).
- `artifact_version_uuid` — identifies the exact version returned.
- `status` — version lifecycle state; auto-published kinds read as `published`.

**Invariants.** Company-scoped: only artifacts of a session inside the caller's
company are ever returned; a session in another tenant is indistinguishable from a
non-existent one (both `404`). Versions are append-only and immutable — a re-save of
the same `(session, name)` adds a new version and the "current" pointer advances to
the newest; this endpoint always reflects the newest published version.

**Failure modes.**
- Malformed `session_uuid` → `400`.
- Unknown or cross-tenant `session_uuid` → `404` (distinct from `200 []`, which means
  "valid session, nothing to collect").
- Any other non-2xx → a genuine server error; a caller harvesting a result should
  treat it as a hard failure, not "no artifacts yet".

**Gotchas.**
- `200 {"artifacts":[]}` (empty) and `404` (unknown session) are **different** outcomes
  — do not collapse them. Empty = collect succeeded with nothing; 404 = wrong/absent
  session.
- Content is returned inline in full, so a large artifact means a large response body;
  there is no pagination or size cap exposed in this contract.
- Provenance is tracked internally per version but is **not** surfaced in this response;
  a peer needing provenance/origin metadata cannot get it from this endpoint (UNKNOWN
  whether any other contract exposes it).
- Rarely, an artifact may appear with an empty `content`, empty `content_type`, and a
  zero `artifact_version_uuid` — this reflects an artifact whose current-version pointer
  is momentarily unresolved. In the normal auto-publish path the current version is set
  immediately, so treat empty version fields as "not yet readable," not as final state.

**Business-critical data.** Backed by an `artifact` logical row (per session+name),
immutable `artifact_version` rows (each with `content_ref`, `content_type`, `status`,
provenance), and content-addressed `artifact_blob` storage keyed by content hash. The
endpoint joins the session to its `scope=session` artifacts and resolves each current
version's blob to inline content. (Tenant scoping applies as everywhere — see context.md.)
