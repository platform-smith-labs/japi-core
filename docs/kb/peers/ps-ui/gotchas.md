---
type: gotcha
title: "Cross-cutting integrator traps"
tags: [gotchas, integration]
timestamp: 2026-07-07T06:27:35Z
description: "Traps a backend peer hits when reasoning about ps-ui that aren't tied to one capability"
repo: ps-ui
commit_sha: 1f5f197
evidence: [src/lib/api-client.ts, src/routes/_auth.tsx, src/stores/active-workspace-store.ts]
---

# Cross-cutting integrator traps

- **ps-ui's guards are advisory, never authorization.** The `_auth` route guard and the
  `$workspaceUuid` membership guard exist for UX only and can fail-open (e.g. against a stale query
  cache). Authorization is enforced server-side. Never rely on the UI having blocked a call — assume
  every request can reach your endpoint and enforce there.

- **Error bodies must be one of two shapes.** ps-ui parses `{"message":"..."}` OR
  `{"error":{"code":N,"message":"..."}}`. Only the nested form with `code: 401` on an active session
  triggers the auto-logout. Any other shape degrades to a generic `Request failed: <status>` with no
  usable message.

- **A 2xx may carry an empty body.** ps-ui tolerates an empty body on success (e.g. a 201 create that
  returns no JSON). Returning `null`/empty on a 2xx is fine; returning malformed JSON is not.

- **Resources are keyed by UUID on the wire but shown by name.** ps-ui never displays raw UUIDs.
  Some routes key by **name** (coding sessions by `sessionName`; the workspace-agents roster and
  launched-runtime attach by runtime **name**). A response that returns only a UUID with no
  human name, or that changes a name's uniqueness guarantees, can break name-keyed navigation.

- **Contract read-shape must stay stable across a resource's lifecycle.** ps-ui caches responses in
  React Query and re-fetches on invalidation; a field present on create but absent on a later list read
  (or vice-versa) surfaces as an inconsistent UI, not an error.
