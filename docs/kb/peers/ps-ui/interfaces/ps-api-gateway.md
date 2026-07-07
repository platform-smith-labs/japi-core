---
type: interface
title: "ps-api gateway (consumed HTTP interface)"
tags: [interface, http, ps-api, consumed]
timestamp: 2026-07-07T06:27:35Z
description: "The single backend HTTP interface ps-ui consumes, and its envelope conventions"
repo: ps-ui
commit_sha: 1f5f197
evidence: [src/lib/api-client.ts]
consumes_interfaces:
  - {name: "ps-api REST gateway", kind: rest, peer: ps-api, intent: "the only backend ps-ui calls; proxies to orchestrator"}
---

# ps-api gateway (consumed HTTP interface)

**What it is.** The one HTTP interface ps-ui consumes: the **ps-api gateway** at `VITE_API_URL`
(default `http://localhost:9004/api`). Every capability's endpoints are paths under this base
(versioned `/v1/...`). ps-ui provides no interface of its own — this concept exists to name the
consumed contract envelope; the specific endpoints live in each `capabilities/` concept.

**Envelope conventions ps-ui relies on** (a peer changing the gateway must preserve these):

- **Auth**: `Authorization: Bearer <jwt>` header on every authenticated call.
- **Versioning**: paths are under `/v1/`.
- **Error shape**: flat `{message}` or nested `{error:{code,message}}`; nested `code:401` on an active
  session means "token expired" (drives auto-logout).
- **Empty success bodies**: a 2xx may return no body.
- **Content type**: JSON request/response (`Content-Type: application/json`); SSE for the coding-session
  event stream (see the coding-sessions capability).

**Integration seam (call ordering).** A session must exist before its SSE event stream is opened, and a
runtime must be launched/ready before a session attaches to it — ps-ui sequences these; the gateway
must keep the corresponding create-before-read ordering valid.

**See also.** ps-api — owner of this gateway; orchestrator — the true owner of platform state ps-api
proxies to.
