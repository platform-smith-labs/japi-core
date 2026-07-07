# Decision: Repo isolation — KB-first cross-repo knowledge, no sibling filesystem access

**Date**: 2026-07-07
**Status**: Accepted
**Context**: In the PlatformSmith product each repo runs in its own container and has **no filesystem access to sibling repos** (the `solution` container itself will clone only the solution repo — `repos/` stays empty; the conductor folds child state via A2A messages + DB, not by reading child working trees). To validate that our `/work`-suite skills actually work under that constraint, we emulate it now in the monorepo: an AI agent may only read/edit **its own repo**, and all cross-repo knowledge flows through the locally-folded knowledge base (`docs/kb/peers/`) and the A2A relay channel. This is the consuming half of the KB model established in `work-2607051522-substrate-design-of-record/research/0012-relay-synced-git-owned-kb-final-model.md` — the fanout that folds every peer's brief into each repo's `docs/kb/peers/<repo>/` is what makes the ban survivable.

## Decision

An AI agent's filesystem universe is **its own repo**. It must never read, grep, or edit another repo's working tree. Cross-repo knowledge comes from the local folded KB (`docs/kb/peers/<repo>/`); the only live cross-repo channel is an A2A **relay**, reserved for system-critical facts and KB gaps. Cross-repo **edits** are never permitted.

## Rules

### 1. No sibling-repo filesystem access

An agent treats **its own repo** as the whole world. Any path **outside this repo** (another repo's working tree — a sibling directory in the dev monorepo, or simply not present in a prod container) is off-limits for `Read`, `Grep`, `Glob`, and `Edit`. `docs/kb/peers/**` inside this repo is **not** a cross-repo read — it is this repo's own folded copy.

```bash
# CORRECT: learn about a peer from your own folded KB (paths are within THIS repo)
Read docs/kb/index.md
Read docs/kb/peers/<peer>/capabilities/<capability>.md

# WRONG: reach into another repo's source (not yours — and in prod, not even on disk)
Read <path-to-another-repo>/cmd/websocket/handler.go    # another repo's tree — forbidden
Grep -r "SomeSymbol" <path-to-another-repo>             # another repo's tree — forbidden
```

### 2. KB-first research, then relay

Initial cross-repo research is answered from `docs/kb/peers/<repo>/`, starting at `docs/kb/index.md`. Only escalate to a relay when the KB is insufficient.

```
1. Read docs/kb/peers/<repo>/  (folded brief — the default source)
2. If a fact is (a) system-critical AND unclear, or (b) a KB gap / marked UNKNOWN,
   or (c) contradicted by observed behavior → emit a relay to that repo.
3. Do NOT relay for routine confirmation of facts the KB already states.
```

### 3. Relay is for system-critical / unclear only

The **A2A ask-a-peer channel** — a live message to the peer repo's agent over the platform's A2A transport (the authoritative live tier of the KB model) — is used sparingly: to affirm a decision-critical fact the KB leaves unclear, to report a contradiction, or to ask a question the folded brief cannot answer. It is **not** a local script: `scripts/rlog.sh` / `relays.jsonl` is the *conductor's* parent→child delivery ledger (single-writer = the conductor), never a worker's outbound channel. Trust the git-owned, PR-reviewed KB by default; when no live peer is reachable, stop and ask the human.

### 4. Cross-repo edits are never allowed

Repo ownership / single-writer / PR boundaries are absolute. An agent never edits another repo's files — not to "fix" a peer, not to sync, nothing. Changes another repo needs are requested via relay.

### 5. Escape hatch = ask the human

If an agent believes a cross-repo read is genuinely unavoidable, it **stops and asks the human for explicit approval** rather than reading. The need itself is a signal that the KB has a gap — note it so the next `kb-sync` can close it. (In the dev monorepo the sibling source still exists, which is exactly why now is the time to surface such gaps.)

## Rationale

- **Emulate prod to harden the skills + KB now.** Behaving as if sibling source is absent surfaces KB gaps while the source is still on disk to fix them against. The ban is a forcing function, not a blind-trust bet.
- **Consistency with the substrate design of record.** `0012` settled that knowledge crosses repo boundaries over the relay/A2A transport and lands via each repo's own commits — no platform injection, no sibling reads.
- **Enforcement is prose-level by choice.** We accept the ~5% agent-error slack; the goal is to prove the skills are *designed* for container isolation, not to build an airtight sandbox. A `PreToolUse` deny hook is a deliberate future step, not part of this decision.

## Exceptions

1. **The conductor / epic orchestration plane** (`/conduct`, `/epic`, `scripts/conduct-board.sh`) — in the dev monorepo it reads child work logs off disk as a stand-in for the future A2A+DB mechanism. This is script/tooling behavior, not an AI agent doing cross-repo research, and it disappears in prod (where `repos/` is empty). Out of scope for the ban.
2. **Explicit human approval** (Rule 5) — a human may grant a one-off cross-repo read.

## Enforcement

- Skill prose: the `/work`-suite commands (`research`, `research_codebase`, `work`, `planv0`, `implement_plan`, `new_req`, `new_issue`) and each repo's `CLAUDE.md` carry the isolation rule.
- Reviewed in code review; a cross-repo `Read`/`Edit` in a work-item transcript is a defect.

## See Also

- [Relay-synced git-owned KB final model](../../work/work-2607051522-substrate-design-of-record/research/0012-relay-synced-git-owned-kb-final-model.md)
- [Parent/child work items and conduct](./parent-child-work-items-and-conduct.md)
- `scripts/kb-copy-peer.sh` / `scripts/kb-fanout-peers.sh` — the fold that makes the ban survivable
