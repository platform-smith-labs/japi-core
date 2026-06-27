---
name: wishlist
description: Capture a deferred, cross-cutting improvement into the docs/wishlist/ structure as a self-contained, numbered item that a future Claude session can read and bootstrap into an /epic or /work. Use when the user says "add to wishlist", "capture this for later", "park this", "note this as future work", or describes an acknowledged-but-deferred improvement that no current epic/work item owns. Do NOT use for items already owned by an epic (those stay in the epic's deferred-items.md) or for actionable now-work (use /work or /epic directly).
---

# Wishlist capture

The wishlist (`docs/wishlist/`) holds platform-wide improvements that are
acknowledged, deliberately deferred, and not yet owned by any epic or work item.
Each item is a **self-contained numbered folder** so a fresh Claude session can
open it and run `/epic` or `/work` with no other context.

## Hierarchy (where the wishlist sits)

The workflow has three tiers, **each parent optional**:

```
wishlist item  →  epic (0..N, one per milestone)  →  work item(s) (1..N per epic)
```

A wishlist item is the **top, optional** tier: it may be picked up into **0..N epics** over time
(incrementally — one epic per milestone for a multi-milestone item). An epic need not come from a
wishlist; a work item need not come from an epic. When a wishlist item *is* picked up, the link is
**bidirectional** and the `/epic` / `/work` commands maintain the wishlist side (see step 7 +
`/epic`'s "Wishlist Linkage"). Upward status flows work → epic → wishlist, each hop only if that
parent exists.

## Monorepo root resolution (repo-context-free)

The wishlist lives at the **monorepo root** under `docs/wishlist/`. This skill may be invoked from
the root or from any child repo, so resolve the root first: if `./docs/wishlist/` exists, CWD is
the root (use `.`); else if `../docs/wishlist/` exists, CWD is a child repo (root is `..`). All
`docs/wishlist/...` paths below are relative to the resolved root. Never hardcode a repo name.

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
7. **Do not scaffold** the epic/work item — that happens when the item is picked
   up. When it is picked up (by `/epic` or `/work`), the linkage MUST be made
   **bidirectional**, and those commands own the wishlist side of it:
   - the item's `## Tracking (epics / work items)` table gets a row per epic/work
     item + status (this is the wishlist→epic back-link);
   - the registry row in `docs/wishlist/README.md` moves **Open → Picked up** with
     the epic/work ID;
   - the scaffolded epic/work manifest carries a `**Wishlist**: NNNN[ — milestone Mx]`
     header field pointing back here.
   **Multi-milestone items** (a roadmap of M0→Mn delivered one epic per milestone)
   stay a **single** wishlist item: one registry row, and one `## Tracking` row per
   milestone/epic — never a new wishlist item per milestone.

## File:line discipline

Every code reference must be `path:line` relative to the monorepo root, with a note
to re-grep symbols if lines drift. Prefer linking symbols + a stable anchor over
bare line numbers where the file churns.

## README template

```markdown
# Wishlist NNNN — <Title>

**Status**: 🎯 Open (not yet scaffolded)
**Suggested workflow**: /epic | /work — <repos involved + why>
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

## Tracking (epics / work items)
<Filled in by /epic or /work when the item is picked up — the wishlist→epic back-link.
 Omit (or leave the header with "_not yet scaffolded_") while Status is Open.
 For a multi-milestone item, one row per milestone/epic.>

| Milestone | Epic | Work item(s) | Status |
|-----------|------|--------------|--------|
| <Mx or "—"> | <epic-NNNN or "_not yet scaffolded_"> | <repo work-NNNN, …> | <🎯 / 🔄 / ✅ / 🔴> |

## Where to start
<Reading order of the supporting files + the exact /epic or /work command to run.>

## Related
<Links to other wishlist items, epics, decisions, prior analyses.>
```

## Quality bar

- A future session reading only the folder can scaffold the epic/work item.
- No "TODO: investigate" gaps that the capturing session could have filled now.
- Decisions and especially rejections are explicit, with rationale.
- Every claim that maps to code carries a file:line.
- The index row is added and the date is real.
