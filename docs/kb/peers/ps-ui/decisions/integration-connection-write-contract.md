---
type: decision
title: "Integration-connection write contract (create + assign)"
tags: [decision, integrations, contract]
timestamp: 2026-07-07T06:27:35Z
description: "The exact create/assign shape ps-ui sends for integration connections"
repo: ps-ui
commit_sha: 1f5f197
evidence: [docs/dev/decisions/integration-connection-write-contract.md, src/api/integrations.ts]
---

# Integration-connection write contract (create + assign)

**Consequence for a peer.** When ps-ui creates an integration connection it sends a **non-empty
`display_name`**, and when mapping it to a workspace it assigns with a **single
`integration_connection_uuid`** — NOT a `connection_uuids` array. This assign contract is **distinct
from the git-assign contract** (which uses a different key). A backend peer must not unify the two
assign shapes: the integration side is single-UUID, the git side is a set. Diverging here silently
breaks credential-create from the UI.
