---
name: wishlist
description: Capture a deferred, cross-cutting improvement into the docs/wishlist/ structure as a self-contained, numbered item that a future Claude session can read and bootstrap into a parent work item (/work + /conduct) or a plain /work, OR report implementation status of wishlist items. Use the capture mode when the user says "add to wishlist", "capture this for later", "park this", "note this as future work", or describes an acknowledged-but-deferred improvement that no current parent/work item owns. Use the status mode (`--status` / `status`) when the user asks how a wishlist item is progressing, whether it's been picked up, or wants a synopsis of its implementation. Do NOT use capture for items already owned by a parent work item (those stay in that item's deferred-items.md) or for actionable now-work (use /work or /conduct directly).
---

# Wishlist

The wishlist (`docs/wishlist/`) holds platform-wide improvements that are
acknowledged, deliberately deferred, and not yet owned by any work item.
Each item is a **self-contained numbered folder** so a fresh Claude session can
open it and run `/work` (+ `/conduct` for cross-repo) with no other context.

## Command modes

```bash
/wishlist "<description>"          # Default: CAPTURE a new wishlist item
/wishlist --status                 # STATUS: dashboard of all wishlist items
/wishlist --status NNNN            # STATUS: deep synopsis of one item + its linked epic/work
/wishlist status [NNNN]            # alias for --status
```

- **No `--status`/`status` token →** capture mode (the original behavior, below).
- **`--status` or `status` present →** status mode ([jump to Status mode](#status-mode---status)).
  An optional `NNNN` (the 4-digit item number, e.g. `0003`) scopes it to one item.

## Hierarchy (where the wishlist sits)

The workflow has three tiers, **each parent optional** (the epic entity is retired — a standalone
**parent work item + `/conduct`** plays that role; see
`docs/dev/decisions/parent-child-work-items-and-conduct.md`):

```
wishlist item  →  parent work item (0..N, one per milestone)  →  child work item(s) (1..N per parent)
```

A wishlist item is the **top, optional** tier: it may be picked up into **0..N parent work items**
over time (incrementally — one **sibling parent** per milestone for a multi-milestone item, advanced
by `/conduct <parent> next`). A parent need not come from a wishlist; a work item need not have a
parent. When a wishlist item *is* picked up, the link is **bidirectional** and the `/conduct` /
`/work` commands maintain the wishlist side (see step 7 + `/conduct`'s wishlist notes). Upward
status flows child → parent → wishlist, each hop only if that parent exists. **Legacy items picked
up into epics** (`docs/epics/`) keep their epic links unchanged — frozen, never migrated.

## Monorepo root resolution (repo-context-free)

The wishlist lives at the **monorepo root** under `docs/wishlist/`. This skill may be invoked from
the root or from any child repo, so resolve the root first: if `./docs/wishlist/` exists, CWD is
the root (use `.`); else if `../docs/wishlist/` exists, CWD is a child repo (root is `..`). All
`docs/wishlist/...` paths below are relative to the resolved root. Never hardcode a repo name.

---

# Capture mode (default)

## Layout (the established pattern — follow it)

```
docs/wishlist/
  README.md                      # index: preamble + Open table + Picked-up table
  NNNN_slug/                     # one folder per item, zero-padded 4-digit, kebab slug
    README.md                    # REQUIRED — the bootstrap brief (see template)
    findings.md                  # OPTIONAL — measured evidence / investigation data
    feature-specs.md             # OPTIONAL — per-feature breakdown (layer, how, where)
    references.md                # OPTIONAL — authoritative file:line map + schemas
    <anything else>.md           # add as many as the item needs; keep each focused
```

Simple items may be a single `README.md`. Rich items (multi-repo, measured
evidence, many sub-features) should split into focused files — one concern per
file, linked from the README. Bias toward **precise, concise, robust**: enough that
the next session needs no live investigation to start.

## Procedure

1. **Read `docs/wishlist/README.md`** to find the next number (highest `NNNN` + 1,
   zero-padded to 4 digits) and to match the current preamble/table format.
2. **Pick a slug**: short, kebab-case, descriptive (e.g.
   `launch-timeline-progress-ux`). Folder = `NNNN_slug`.
3. **Write `NNNN_slug/README.md`** from the template below. This is mandatory and
   must stand alone.
4. **Add supporting files** as the item warrants (findings/feature-specs/
   references/…). Put measured numbers, raw captures, and file:line maps here, not
   in the README. Cross-link with relative paths.
5. **Capture decisions faithfully** — especially the user's rejections. If the user
   said "no, do it this way instead", record both the rejected idea AND the
   replacement AND the one-line rationale, so it is never re-proposed.
6. **Update `docs/wishlist/README.md`**: add a row to the **Open** table
   (`[NNNN](./NNNN_slug/) | Item | Why it matters | Suggested workflow | Origin |
   Added`). Use the real current date (check `currentDate`/`env`).
7. **Do not scaffold** the work item — that happens when the item is picked up.
   When it is picked up (by `/work`, with children via `/conduct scaffold`), the
   linkage MUST be made **bidirectional**, and those commands own the wishlist
   side of it:
   - the item's `## Tracking (parents / work items)` table gets a row per parent/
     work item + status (this is the wishlist→work back-link);
   - the registry row in `docs/wishlist/README.md` moves **Open → Picked up** with
     the work ID;
   - the scaffolded parent's `created` event carries `wishlist=NNNN` so its
     generated manifest's `**Wishlist**` header points back here.
   **Multi-milestone items** (a roadmap of M0→Mn delivered one parent per
   milestone — sequenced sibling parents, advanced by `/conduct <parent> next`)
   stay a **single** wishlist item: one registry row, and one `## Tracking` row per
   milestone/parent — never a new wishlist item per milestone.

## File:line discipline

Every code reference must be `path:line` relative to the monorepo root, with a note
to re-grep symbols if lines drift. Prefer linking symbols + a stable anchor over
bare line numbers where the file churns.

## README template

```markdown
# Wishlist NNNN — <Title>

**Status**: 🎯 Open (not yet scaffolded)
**Suggested workflow**: /work [+ /conduct for cross-repo] — <repos involved + why>
**Primary repos**: <repo(s)>
**Origin**: <session / incident / date>
**Added**: YYYY-MM-DD

## One-paragraph brief
<What it is, why it matters, the guiding constraint. Self-contained.>

## Scope — accepted
<Table or list of accepted features/changes. Link feature-specs.md if rich.>

## Out of scope — explicitly rejected (do not re-propose)
<Each rejected idea + the user's rationale + the replacement, if any.>

## Suggested acceptance criteria
<Numbered, testable.>

## Tracking (parents / work items)
<Filled in by /work + /conduct when the item is picked up — the wishlist→work back-link.
 Omit (or leave the header with "_not yet scaffolded_") while Status is Open.
 For a multi-milestone item, one row per milestone/parent. Legacy rows may name epics.>

| Milestone | Parent work item | Child work item(s) | Status |
|-----------|------------------|--------------------|--------|
| <Mx or "—"> | <work-… or "_not yet scaffolded_"> | <repo work-…, …> | <🎯 / 🔄 / ✅ / 🔴 / 🚨> |

## Where to start
<Reading order of the supporting files + the exact /work (and /conduct scaffold) commands to run.>

## Related
<Links to other wishlist items, parent work items, decisions, prior analyses.>
```

## Quality bar

- A future session reading only the folder can scaffold the epic/work item.
- No "TODO: investigate" gaps that the capturing session could have filled now.
- Decisions and especially rejections are explicit, with rationale.
- Every claim that maps to code carries a file:line.
- The index row is added and the date is real.

---

# Status mode (`--status`)

Read-only. Produces a synopsis of wishlist items by combining each item's own
documentation with the documentation of any epic/work item it has been picked up
into. **This mode never writes** — it only reads and reports. Mirrors
`/epic status`, but its job is to answer "where does this wishlist item stand, and
how far has its implementation gotten?"

## Resolving the picked-up epic/work ID

An item is **Open** until it is scaffolded; once picked up, its ID lives in two
places (read both; the README is authoritative, the index is the quick map):

1. The **Picked up** table row in `docs/wishlist/README.md`
   (`# | Item | Owner | Epic/Work ID | Moved`).
2. The item's own `NNNN_slug/README.md` — its `**Status**:` line and `## Related`
   section, where the new `epic-NNNN…`/`work-NNNN…` ID is linked when picked up.

Resolve a work/epic ID to a directory by glob, never arithmetic:

1. **Work IDs** (incl. parent work items): `docs/work/{id}*/manifest.md` or, for
   child-repo work, `{repo}/docs/work/{id}*/manifest.md`.
2. **Legacy epic IDs**: exact `docs/epics/{id}/manifest.md`, else
   `docs/epics/{id}-*/manifest.md` (frozen items picked up before the
   parent/child model).
3. One match → use it. Zero → report the ID as **linked but directory not found**
   (stale link). Multiple → list them.

## Procedure — `/wishlist --status` (no number): all-items dashboard

1. Read `docs/wishlist/README.md`. Parse the **Open** and **Picked up** tables.
2. For each item folder `NNNN_slug/`, read its `README.md` `**Status**:` line.
3. For each **picked-up** item, resolve its epic/work ID (above) and read that
   manifest's `Status` (and `Last Synced` if an epic) to get live progress.
4. Present a single dashboard:

```
Wishlist status — {N} items ({O} open, {P} picked up)

| #    | Item                              | Wishlist status         | Picked up → | Impl status        |
|------|-----------------------------------|-------------------------|-------------|--------------------|
| 0001 | Reconnect-safe launch delivery    | 🎯 Open                 | —           | —                  |
| 0003 | Cross-pod agent coordination      | ✅ Picked up            | epic-0072   | 🔄 In Implementation |

Recommended:
  1. {next action — e.g. "0002 is the oldest open item; scaffold with /epic"}
  2. {…}
```

Keep "Item" to a short label (don't dump the full why-it-matters cell).

## Procedure — `/wishlist --status NNNN`: single-item deep synopsis

1. Resolve the folder: `docs/wishlist/NNNN_*/`. Zero → "Wishlist item NNNN not
   found." Multiple → list and stop.
2. Read **every** `.md` in the folder (`README.md` plus any `findings.md`,
   `feature-specs.md`, `references.md`, …). Summarize: the one-paragraph brief,
   accepted scope, explicitly-rejected items (always surface these), and suggested
   acceptance criteria.
3. Determine pick-up state from the README `**Status**:` line + the index tables.
4. **If Open** — report it as not-yet-scaffolded, restate the suggested workflow
   and the exact `/work` (+ `/conduct`) command from "Where to start", and stop.
5. **If picked up** — resolve the linked ID and traverse its docs:
   - **Parent work item**: read its generated `manifest.md` — `Status`, the
     children board between the BOARD anchors (`**Barrier Phase**` + per-repo
     Work Item / Phase / Status table, derived by `scripts/conduct-board.sh`),
     and the Change Log. For each child row, read that work item's manifest for
     its current phase/status (resolve IDs as above). Surface 🚨 Escalated
     children prominently — they block completion until a human decides.
   - **Legacy epic**: read `manifest.md` — `Status`, the **Tracked Repos** table,
     the **Change Log**, and any **Sub-Epic Roadmap**; traverse tracked work
     items the same way.
   - **Plain work item**: read `manifest.md` — status, current phase, latest
     change-log entry, and `implementation/status.md` if present.
6. Cross-check the wishlist item's **acceptance criteria** against what the epic/
   work docs say is done, and call out anything still open or any
   explicitly-rejected idea that appears to have crept back in.
7. Present the synopsis:

```
Wishlist 0003 — Cross-pod agent coordination protocol
Wishlist status: ✅ Picked up → epic-0072-… (🔄 In Implementation)

Brief: {1–2 sentences}

Implementation progress (epic-0072):
| Repo         | Work Item  | Phase             | Status        |
|--------------|-----------|-------------------|---------------|
| orchestrator | work-0118 | 🔄 In Implementation | 3/5 phases   |
| runtime      | work-0119 | 📝 Requirements    | Not started   |

Acceptance criteria:
  ✅ {met criterion} — {where, file:line or work/phase}
  ⬜ {open criterion}
  ⚠️ {rejected idea that resurfaced, if any}

Recommended next:
  1. {action}
```

## Status-mode quality bar

- Read the item's whole folder, not just its README — supporting files hold the
  acceptance criteria and rejections you must check against.
- Always traverse into the linked epic/work docs for picked-up items; never infer
  implementation status from the wishlist text alone.
- Surface stale links (ID present but directory missing) rather than silently
  skipping them.
- Re-surface explicitly-rejected ideas if the implementation appears to include
  them — that is a regression worth flagging.
- This mode is strictly read-only: do not edit the wishlist, the index, or any
  epic/work doc.
