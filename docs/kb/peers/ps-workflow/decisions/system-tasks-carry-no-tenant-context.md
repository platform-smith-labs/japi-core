---
type: decision
title: "Engine-executed system tasks carry no _ps/tenant context field"
tags: [decision, multi-tenancy, system-tasks, task-catalog, conductor]
timestamp: 2026-07-09T10:49:10Z
description: "System tasks (HTTP/WAIT/INLINE/JSON_JQ_TRANSFORM/SET_VARIABLE) run in the engine with no worker, so they declare no company/user context field; isolation is service-layer"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - docs/dev/decisions/system-tasks-carry-no-tenant-context.md
  - internal/taskcatalog/catalog.go
  - cmd/handlers/workflow_task_catalog.go
see_also:
  - {repo: ps-workflow, capability: "Workflow task catalog", intent: "the catalog whose system entries omit context fields"}
---

# Engine-executed system tasks carry no _ps/tenant context field

**Consequence for a peer.** The per-task tenant-context field (`_ps.company_uuid` / `_ps.user_uuid`)
applies only to **worker-backed platform tasks** — a ps-workflow worker reads and asserts it. The
five **engine-executed system tasks** (`HTTP`, `WAIT`, `INLINE`, `JSON_JQ_TRANSFORM`,
`SET_VARIABLE`) run inside Conductor with no worker to assert anything, so their task-catalog entries
declare **no** context field. A peer reading the task catalog must not expect every task to carry a
`company_uuid` field; a system entry legitimately has none. Tenant isolation for those tasks comes
from the service layer (workflow-name namespacing + execution tagging + authz on every engine call),
not per node, and they inherit it from the execution they run in. Free-form system inputs are flagged
by an `allows_additional_inputs` marker rather than fabricated context fields.
