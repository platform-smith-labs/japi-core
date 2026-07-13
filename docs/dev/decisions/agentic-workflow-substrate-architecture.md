# Decision: Agentic Playbook Substrate — declared-file event plane + conductor-pod control loop + metronome (observed V1)

**Date**: 2026-07-13
**Status**: Accepted
**Context**: Synced from the design of record in the **solution monorepo**
(`docs/work/work-2607051522-substrate-design-of-record/` — research `0013` + addenda 1–5, and the
2026-07-13 in-place rewrite of the solution copy of this doc). The solution copy is authoritative;
this edition is finetuned for **japi-core**. Supersedes the earlier `ps_task`-MCP + guarantee-ticker
spine, which was deleted from V1.

## Decision — the whole workspace flow

A **PLAYBOOK** is a customer-authored `.claude/` agent-config bundle spanning a workspace. A run
rides an **explicitly-created `conversation`** whose **participant roster is declared upfront**.
Each pod's state of record is its **local append-only jsonl** (single-writer discipline); a runtime
**watcher pushes the whole file** on change; the orchestrator **idempotently ingests** lines into a
generic event table. One pod — the **primary / initial conversation member — is the conductor**: a
stateless-re-fold LLM loop that reads the table via a **cursored read MCP tool** and directs
workers over **A2A** (narrative + wake hints + proceed signals). The only deterministic wake is a
**metronome tick**. Coordination is **checkpoint barriers over the declared roster** — nothing
arbitrates in V1. Runs are **observed, not unattended**: the human is the guarantee layer.

```
worker pod                          orchestrator                     conductor pod (primary)
──────────                          ────────────                     ───────────────────────
agent appends work.jsonl ─┐
runtime watcher (tokio)   ├─ whole file ──▶ ingest: split lines,  ─▶ event table ──┐
  mtime+len poll ─────────┘   over WS        drop torn tail,          (append-only) │ cursored
                                             insert missing rows                    │ read MCP
metronome ─────────── tick (A2A) ────────────────────────────────▶ conductor: stateless re-fold,
                                                                    reads table + staleness view,
worker ◀───────── A2A "proceed" (narrative / wake hints) ────────── decides, signals workers
```

## This repo's role — none directly (shared framework; awareness only)

No S1 component lands in japi-core. The orchestrator's new pieces ride existing japi-core
patterns: typed handlers + querier for the ingest write path and the read MCP's cursored queries,
and standard body-parse limits on any HTTP-facing surface.

If the S1 work surfaces a framework-level need (e.g., a reusable cursored-list/seq-pagination
helper for the event table's `since_seq` reads), it arrives as a relay from the S1 prime
(`work-2607130316-playbook-file-sync-plane` @ orchestrator) — do not pre-build speculatively.

## Rules (the flow's laws — full statements in the solution copy)

1. **Two planes** — the event table is truth (read via the MCP); A2A is narrative + wake hint. The
   conductor **never acts on A2A content — only on the re-folded table**.
2. **Whole-file push, self-healing** — redundancy IS the reliability mechanism. Three conditions:
   re-send all watched files on WS reconnect; slow periodic re-send (~5–10 min); drop the
   unterminated tail line at parse. No checkpoints, no ACKs, no per-message delivery protocol.
3. **Run anchor = conversation + declared roster** — barrier = monotonic fold over the roster
   **as-of run creation** (never inferred from arriving events). **Class-1 fence**: watched-file
   rows are telemetry/barrier input, **never** claim/lease arbitration.
4. **Conductor = the primary pod; stateless re-fold** — re-derives "what next" from a cursored
   table read each wake; restartable via the B2 primary-claim CAS. The metronome exists because an
   idle agent has no clock (a silently-dead worker emits nothing).
5. **V1 = observed** — deferred to the automation boundary: claim/lease CAS, ticker re-drive +
   bounds, conductor respawn (A6), enforced stop switch (B5), session caps (B6), budget governor,
   deterministic verify floor.
6. **Verify = in-flow disk-conformance** — a verify task freshly reads produced files against the
   task objective; fail → flip back → re-run, bounded → escalate. This + the observing human is
   V1's entire fake-done posture.
7. **Sessions / tenancy / escalation / relay content** — B2 primary pointer (CAS); A4 fail-closed
   single-workspace A2A; `escalated` = a pod jsonl event → event table → attention surface; relay
   documents travel as A2A bodies and the **receiving playbook writes them to `relays/inbound/`
   before acting** (skill convention).
8. **Security posture: deliberately uncapped V1** (maintainer, 2026-07-13) — no path allowlists,
   size caps, rate limits, or redaction; caps are introduced as and when necessity is observed.

## Exceptions

1. **Unattended runs** — re-introduce claim/lease CAS, the guarantee ticker, A6 respawn, enforced
   B5/B6, the budget governor, and the verify floor before any run executes unwatched.
2. **Contended dispatch** — if workers ever compete for tasks, Class-1 arbitration returns
   server-side; the file plane never carries it.
3. **Large watched files** — whole-file push assumes tens-of-KB files; bigger files switch to a
   pull/delta path feeding the same ingest function.

## See Also

- Design of record: solution monorepo, `docs/work/work-2607051522-substrate-design-of-record/`
  (research `0013` + addenda 1–5; `ticket-register.md`)
- [playbook-terminology](./playbook-terminology.md)
- [append-only-work-event-log](./append-only-work-event-log.md)
