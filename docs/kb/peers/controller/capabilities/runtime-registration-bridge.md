---
type: capability
title: "Runtime registration bridge"
tags: [registration, handshake, instance-uuid, builder, websocket, cross-repo]
timestamp: 2026-07-07T00:00:00Z
description: "How a runtime's handshake is bound for routing and forwarded upstream enriched with the pre-minted instance/project/environment UUIDs"
repo: controller
commit_sha: 3412b7d
evidence:
  - src/websocket/server.rs
  - src/websocket/registry.rs
  - src/orchestrator/executor.rs
---

# Runtime registration bridge

**What it does.** When a runtime container comes up and connects downstream, it
sends a one-time `registration` handshake. The controller binds that live
connection so it can route later commands to the runtime, then forwards an
*enriched* registration upstream to the orchestrator so the orchestrator can
correlate the runtime to the `runtime_instance` row it pre-created at spawn.

**How a peer interacts.** A runtime sends a `registration` message as its first
frame: `{name, version, platform, instance_uuid}`. The orchestrator receives the
forwarded, enriched registration as an upstream message and reads its identity
fields to advance the launch lifecycle. Neither peer calls the controller
directly for this — it is the automatic bridge step of the handshake.

**Observable behavior.**
- The runtime's connection is bound in the controller's routing registry *before*
  the registration is forwarded, so the controller can immediately route work
  (e.g. `setup_devcontainer`, `build_image`, `spawn_runtime`) to it.
- The forwarded registration carries `instance_uuid`, `project_uuid`, and
  `environment_uuid` injected from the spawn metadata the controller stored when
  it launched the runtime.
- **UUID reconciliation:** the `instance_uuid` the runtime echoes is reconciled
  against the orchestrator-minted value captured at spawn. Precedence is
  **prefer the pre-minted value**. If the runtime echoes a different value, the
  pre-minted one is kept and forwarded (mismatch logged as a warning). Only when
  no pre-minted value exists is the runtime-echoed value accepted. The forwarded
  registration always carries the *reconciled* UUID — this is the field the
  orchestrator joins on to find its pre-created `runtime_instance` row.
- **Builder and Product both forward.** A builder pod's registration is bound and
  forwarded upstream exactly like a product runtime's. The internal role
  (Builder/Product) is observability-only and does **not** gate forwarding.

**Contract.** In (from runtime): `registration` message, data
`{name, version, platform, instance_uuid}`. Out (to orchestrator): the same
registration envelope with `instance_uuid` (reconciled), `project_uuid`, and
`environment_uuid` merged into its data block. Builder/product discrimination is
the orchestrator's job — it keys on the `-builder` suffix in the runtime `name`,
not on any controller-supplied role field.

**Invariants.**
- Bind-before-forward: the connection is registered for routing before its
  registration is sent upstream.
- The forwarded `instance_uuid` is the reconciled value (pre-minted wins), never
  blindly the runtime-echoed value — otherwise the orchestrator cannot correlate
  its pre-minted row and the launch stalls before setup.
- Both roles forward; role never suppresses a registration.
- The registry is **in-memory only** (per controller process) — spawn metadata
  and bindings do not survive a controller restart; there is no database here.

**Failure modes.**
- **Registration arrives before spawn metadata was stored** (race): the runtime
  is treated as Product with empty enrichment and the registration is *still*
  forwarded — never silently dropped or misclassified as a builder.
- **Mismatched echoed UUID:** kept-pre-minted, warned, launch proceeds correctly.
- If no pre-minted UUID exists at all (un-migrated runtime), the echoed value is
  accepted as the correlation key.

**Gotchas.**
- A peer must not rely on any controller-set role flag to tell builder from
  product — that signal is the `-builder` name suffix, decided upstream.
- After a spawn that failed and is retried once (e.g. port eviction), the
  controller re-stores the runtime's `project_uuid`/`environment_uuid` before the
  retry connects. Without this the re-created instance would register with NULL
  project context, so any project-scoped step (e.g. git token minting) could
  never fire. A peer debugging a runtime that registered with no project should
  suspect a spawn-retry path that skipped this re-store.
- The registration is a *one-time* first frame; subsequent frames on the same
  connection are runtime events/responses, not re-registrations.

**Business-critical data.** The bridge depends on spawn-time metadata held only
in the controller's in-memory registry: the orchestrator-minted `instance_uuid`
plus `project_uuid` and `environment_uuid`. These are the correlation keys the
orchestrator needs to link a live runtime back to its pre-created instance and
its owning project/environment. No persistent store is involved on the controller
side. (instance_uuid vs runtime_name distinction: see context.md.)
