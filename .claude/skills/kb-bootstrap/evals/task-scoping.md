# Task-scoping eval — the decisive acceptance test (§4.2)

The KB's whole purpose: can a peer agent scope a real task from THIS repo's KB ALONE (no source)?
This eval is **generic** — it derives its own scenario from the target repo's own KB, so it gates
completion for whatever repo the skill just ran on, assuming nothing about what other repos exist.

It is the **decisive acceptance gate**: run it after LINT passes and before the KB is accepted/committed.

## Procedure (derive the scenario from THIS repo's KB — no external input)

1. **Derive the scenario in a fresh KB-only context (no source).** Steps 1–3 MUST be performed by an
   agent in a **separate context that has read ONLY this repo's `docs/kb/`** — never the orchestrator
   context that ran the pipeline, which has already read the source and would bias the scenario toward
   seams a peer could not see from the KB alone. Open `docs/kb/self/capabilities/` (+ `interfaces/`,
   `gotchas/`, `overview.md`, `context.md`). Use nothing outside `docs/kb/` — no repo source, no other
   repo. (This derivation context is distinct from the step-4 answer agent.)

2. **Pick the seam.** Choose the scenario subject:
   - Prefer a **pair of capabilities that chain** — one produces something a second consumes, or one
     must be called before the other (an ordering/hand-off seam). Pick the pair a peer is most likely
     to combine.
   - If no two capabilities chain (a thin/single-capability repo), fall back to the **single most
     peer-relevant capability** plus one interface it exposes.
   Record which capabilities/interfaces (by slug) you chose and why they are the most peer-relevant.

3. **Pose the task a peer would actually scope.** Write, in one short paragraph, a concrete task an
   agent in a *different* repo would need to accomplish that **crosses the seam** you picked — i.e. it
   cannot be answered by one capability alone; it forces the hand-off between them. Phrase it as the
   peer would ("My repo needs to <goal>. Which of this repo's capabilities/interfaces must I call, in
   what order, and what must I watch out for?"). Do NOT name internal mechanics — a peer wouldn't know
   them.

4. **Run a fresh KB-only agent.** Give a new agent, in a separate context, ONLY this repo's `docs/kb/`
   (no source on disk, no other repo) and the task from step 3. Capture its verbatim answer.

5. **Ground-truth check (source-side verification of the answer).** Enumerate the specific claims the
   KB-only agent relied on in its step-4 answer — the named surfaces, the calling order, the field
   names / wire shapes, the gotchas/invariants — and verify each against THIS repo's **actual source**.
   Give **status-code, error-code, and existence-signal** claims explicit scrutiny: any claim about what
   code a call returns (2xx vs 4xx vs 5xx) or what error a branch raises — and **especially** any claim
   that two distinct inputs are **indistinguishable** ("X and Y both return `<code>`") — must be confirmed
   against the source on **both** branches, since a KB can plausibly flatten two branches the source
   actually distinguishes.
   Unlike the KB-only scoping agent, this pass **is legitimately allowed to read this repo's source**:
   the source is the ground-truth oracle. It asks not "could the peer scope the task?" (that is step 4)
   but "is what the peer consumed from the KB actually TRUE?". Prefer a fresh context reading the source
   directly against the enumerated claims. Any claim the source **contradicts** — a mislabeled wire
   field, a wrong enforcement locus, an incorrect order/precondition — is a **KB defect**: record it in
   the eval record's `gaps` as a `ground-truth` finding, route it back to DRAFT/VERIFY, regenerate,
   re-lint, and re-run this eval. (A claim the source merely does not cover is not a ground-truth
   failure — that is a coverage matter for the step-4 criteria.)

## Pass criteria

The KB-only agent, from the KB alone, must:
- **name the concrete surface** — the specific capabilities/interfaces and their routes/entry-point
  names/contracts needed for the task;
- **make the behavior clear** — the observable behavior and calling order (sync/async, submit-then-poll,
  events, etc.) as required to complete the task;
- **surface the relevant gotchas/invariants** — the traps a peer would hit (ordering, id ownership,
  scoping, idempotence, error/edge behavior) that apply to this task;
- **close the cross-capability hand-off** — correctly connect the two chained capabilities across the
  seam (what the first yields, what the second needs, in what order), so the peer could actually chain
  them; and
- **NOT need to invent, guess, or ask for this repo's source** — every step is answerable from the KB.
- **be ground-truth-true (step 5)** — every claim the agent relied on that step 5 checked against the
  source must hold; a claim the source contradicts is a KB defect and **FAILS the eval even when every
  scoping criterion above is met**.

A single unmet criterion — a missing scoping criterion OR a step-5 ground-truth defect — is a **FAIL**:
the gap is a real hole in the KB, not a test artifact. Route the gap back to DRAFT/VERIFY, regenerate,
re-lint, and re-run this eval.

## Emit the eval record

Write the outcome as a machine-readable record under `docs/kb/self/eval/` (see
[../references/layout.md](../references/layout.md) — the `eval/` dir is generated and excluded from the
concept/lint/render sweep like `extract/`). One record per run, filename
`task-scoping-<HEAD-short-sha>.md`:

```
---
type: eval-record
eval: task-scoping
timestamp: 2026-07-07T00:00:00Z   # run time, ISO 8601
commit_sha: 9d3b58b               # HEAD the KB was generated against
result: pass                      # pass | fail
seam:                             # the derived scenario subject
  capabilities: [<slug-a>, <slug-b>]   # the chained pair (or single + interface)
  rationale: "why this is the most peer-relevant seam"
task: >-
  The verbatim peer task posed in step 3.
gaps:                             # scoping criteria unmet AND step-5 ground-truth defects (empty on pass)
  - kind: scoping                 # scoping | ground-truth
    detail: "what the peer could not answer from the KB, and which criterion it maps to"
  - kind: ground-truth            # a claim the peer consumed that the source contradicts (step 5)
    detail: "the false claim, the source truth, and the concept slug to fix"
---

## Agent answer (verbatim)

<the KB-only agent's full answer>

## Judgement

<per-criterion pass/fail with a one-line justification each>

## Ground-truth check (step 5)

<the specific claims the KB-only agent relied on, each marked TRUE / CONTRADICTED against the source,
with the source truth for any CONTRADICTED claim and the concept slug it must be routed back to>
```

`result: pass` with an empty `gaps` list is the gate. Do not accept/commit the KB while the latest
`task-scoping-*` record is `fail`.
