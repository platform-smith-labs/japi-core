# KB house schema (OKF-conformant)

The frontmatter contract every KB concept file obeys. `kb-lint.sh` is implementable literally from
the field table below. See also: [layout.md](./layout.md), [capability-concept.md](./capability-concept.md),
[hygiene.md](./hygiene.md).

## Concept file = YAML frontmatter + markdown body

```yaml
---
type: capability            # REQUIRED — routing/filtering key
title: "A2A peer messaging" # REQUIRED — short human title
tags: [a2a, messaging]      # REQUIRED — list
timestamp: 2026-07-07T00:00:00Z  # REQUIRED — ISO 8601, generation time
description: "How a peer sends a message to another project and gets a reply"  # recommended
repo: orchestrator          # house: the repo this KB describes
commit_sha: 9d3b58b         # house: HEAD the content was generated against
evidence:                   # house, INTERNAL-ONLY (see below) — grounds VERIFY + drift
  - cmd/websocket/a2a_message.go
  - pkg/protocol/protocol.go
see_also:                   # OPTIONAL — NAME-based peer pointers (repo + capability), never a path
  - {repo: ps-api, capability: "Gateway request proxy", intent: "resolves the runtime name a peer needs"}
  - {repo: controller, capability: "Runtime name resolver", intent: "keys the runtime by name", descriptive: true}  # name unverified from this repo — a placeholder for kb-sync to reconcile
---
```

## Field table

| Field | Req? | Rule |
|---|---|---|
| `type` | ✅ | one of `overview \| context \| capability \| interface \| decision \| gotcha \| glossary \| note` |
| `title` | ✅ | short, human, ≤80 chars |
| `tags` | ✅ | non-empty list of kebab-case tags |
| `timestamp` | ✅ | ISO 8601, the generation time |
| `description` | rec | one line; surfaced in the generated `index.md` |
| `repo` | house | the repo alias/name the KB describes |
| `commit_sha` | house | the HEAD the content describes (drift baseline) |
| `evidence` | house | **INTERNAL-ONLY** list of file/contract paths grounding the concept |
| `provides_interfaces` / `consumes_interfaces` | interface concepts | typed edges — list of `{name, kind, contract_path?, peer?, intent}` |
| `see_also` | opt | list of `{repo, capability, intent, descriptive?}` — NAME-based cross-repo/peer pointers; **never a file path** (§0). Repo-agnostic: presumes no specific peer set. Set `descriptive: true` when `capability` is a drafter-invented placeholder **not** verified from THIS repo's own evidence/context (an explicit pin, a contract this repo emits, etc.) — kb-sync later reconciles it against the peer's real brief. Omit or `false` when the name is grounded in this repo's own context. Never read a sibling repo to "verify" a name. |

## `evidence` is internal-only — the load-bearing rule

`evidence` exists **solely** for generation-time machinery: VERIFY existence-checks named entities
against these paths, and `kb-lint.sh` drift-checks them against `commit_sha`. It is **never rendered
into the body and never shown to a consumer** — a peer repo's agent has no access to this repo's
source, so a path is dead weight to them (requirements §0, A-6). A `file:line` or source path
appearing in a concept **body** is a lint **failure**, not a warning.

## Reserved filenames

- `index.md` — GENERATED per-directory listing (by `kb-render.sh`); never hand-edited.
- `log.md` — change history, newest-first; appended per generation run.

## Links

Bundle-absolute only, e.g. `[interfaces](/self/interfaces/a2a.md)` — never repo-relative source
links.

## Lint rules derived from this schema (for `kb-lint.sh`)

1. Frontmatter parses; `type` present and in the vocabulary.
2. `title`, `tags` (non-empty), `timestamp` present.
3. `capability` and `interface` concepts: `evidence` non-empty.
4. **FAIL**: any `file:line` or source-path pointer in a concept *body*.
5. `provides_/consumes_interfaces` entries match the `{name, kind, …}` shape.
6. Bundle-absolute links resolve within `docs/kb/`.
7. Drift: for each concept, any `evidence` path changed since `commit_sha` → report stale (warning).
8. `UNKNOWN` is valid body content; report its count.
9. `see_also` (if present): every entry carries `repo` + `capability` (names) — a path or `file:line`
   in any entry → **FAIL** (the §0 no-source-pointer rule applies to frontmatter refs too).
10. `see_also` entry `descriptive` (if present) is a boolean; `true` marks the `capability` name as
    an unverified placeholder — it is **never** a lint failure. A **same-repo** `descriptive` entry is
    resolved to the sibling's real `title` by the VERIFY cross-concept pass (verify.md §C) once the
    bundle exists; only **cross-repo** placeholders persist for kb-sync to reconcile.
