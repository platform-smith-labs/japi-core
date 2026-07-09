---
type: capability
title: "LLM node (runtime-less)"
tags: [workflow-node, llm, anthropic, structured-output, conductor]
timestamp: 2026-07-09T10:49:10Z
description: "The llm Conductor node — one Anthropic call to summarize/structure/classify workflow data, no container"
repo: ps-workflow
commit_sha: b1f4682
evidence:
  - internal/workers/nodes/llm.go
  - internal/workers/nodes/llm_client.go
  - internal/workers/nodes/sendprompt_schema.go
  - internal/secrets/resolver.go
see_also:
  - {repo: ps-workflow, capability: "Coding-agent session nodes", intent: "container-based alternative that runs a coding agent", descriptive: false}
  - {repo: db-migration, capability: "Integration credential secret store", intent: "owns the anthropic provider + claude_api_key auth type this node resolves", descriptive: true}
---

# LLM node (runtime-less)

**What it does.** A workflow node that makes one plain LLM call over data already in the
workflow — summarize, restructure, or classify a value — and returns the model's text (optionally as
a validated structured object). It is the lightweight, container-less sibling of the agent-session
node: no runtime, no git, no container is launched.

**How a peer interacts.** Add a Conductor task of type `llm` to a workflow definition, carrying its
inputs under the `_ps` annotation block. The node runs synchronously and terminates — it never parks
or waits.

**Contract.** Inputs (under `_ps`): `prompt` (required) and `response_schema` (optional — a JSON
object that may carry a `required: [strings]` list). Outputs: `message` (the reply text). When
`response_schema` is supplied, three more outputs appear: `schema_valid` (bool), and either
`response` (the parsed object, when valid) or `schema_error` (a reason string, when invalid). Without
`response_schema`, only `message` is returned.

**Observable behavior.** On success the node completes with `message`. A downstream node can branch on
`schema_valid` or read fields out of `response` (e.g. to drive a SWITCH). A schema-validation failure
does **not** fail the node — it completes with `schema_valid=false` and a `schema_error`, so the
caller must inspect those flags rather than assuming the node failing means bad output.

**Invariants.** The call resolves a **company-scoped** Anthropic credential (no per-user context
needed). Provider is `anthropic` and auth type is `claude_api_key`, reading the credential field
`api_key` — deliberately the runtime-less LLM key, distinct from the coding-agent (`claude_code`)
credential that launches agent sessions; one does not grant the other.

**Failure modes.** The node reports FAILED if `prompt` is missing, or if the model call errors
(credential not configured, model API returns an error, transport failure). "Credential not
configured" surfaces as a call error from the secret resolver, not a fabricated answer.

**Gotchas.**
- **Dual live gate → NOT_LIVE.** The node only calls the model when an env flag (`LLM_NODE_LIVE`) is
  on **and** a caller is wired. If either is missing it returns NOT_LIVE — an honest "not enabled on
  this stack," never a made-up reply. Both must be satisfied before the node does real work.
- **Structural validation only.** `response_schema` validation in this version checks only that the
  reply is a JSON object and that every name in `required` is present. There is **no** JSON-Schema
  type or enum checking — a field of the wrong type still passes as long as the key exists.
- **Not the agent node.** This node cannot run code, tests, git, or a multi-step coding task. For any
  work needing a container/coding agent, use the agent-session node instead.

**See also / peers.**
- ps-workflow — agent-session node (`run-agent-session`): the container-based alternative that runs a
  coding agent; use it when a runtime, git, or code execution is needed.
- db-migration / orchestrator — the integration-credential / secret store that owns the seeded
  `anthropic` provider and `claude_api_key` auth type this node resolves against.
