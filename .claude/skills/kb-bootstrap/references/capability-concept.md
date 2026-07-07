# The capability concept — the core deliverable

A `capabilities/*.md` file is the point of the KB: a **clinical, business-logic-only summary** of one
functionality, written so an agent in a *different* repo can scope a task that touches it — **without
reading this repo's source**. Brevity is clarity. Soft cap ~120 lines; shorter is better.

See [hygiene.md](./hygiene.md) for the omit-rules and [schema.md](./schema.md) for frontmatter.

## Fixed section set (a capability body)

1. **What it does** — the business logic, in a sentence or two. Not how it's built.
2. **How a peer interacts** — the entry point: endpoint / RPC / message kind / tool name a peer calls.
3. **Observable behavior** — what happens, what the caller gets back, ordering/timing that matters. If
   readiness/completion is **async**, give the concrete signal a peer polls — read endpoint/RPC +
   status field + terminal value (or a `see_also` pointer to the repo that owns that read); bare prose
   ("poll until ready") is insufficient.
4. **Contract** — inputs / outputs / errors, named by role (reference, never a pasted schema). If an
   output identifier is consumed by another capability in a *different form* for the same entity (this
   returns a UUID; a peer keys that entity by name), state the **bridge** — the route/capability that
   resolves one to the other — or name that peer in `see_also`. Mark a field list `key fields:` when
   it is not the confirmed-complete wire shape (open vs closed); never let a partial list read as
   exhaustive. When an error is raised by a delegated upstream check rather than this repo, say so, so
   a peer knows whose error shape it will receive.
5. **Invariants** — what always holds (idempotency, ordering, at-least-once, auth scope). For a
   precondition or check a peer relies on, state its **enforcement locus** — enforced *here* vs
   delegated to an upstream/downstream dependency this repo forwards to — whenever the repo doesn't
   enforce it itself. The locus decides where a peer's call fails and which error shape it must parse; a
   repo that merely forwards a requirement must say the check lives elsewhere (name the owner), not
   imply it enforces it. (Relevant to any repo that delegates enforcement — gateway / proxy / wrapper.)
6. **Failure modes** — what can go wrong and what the peer observes when it does.
7. **Gotchas** — the traps that would bite an integrator.
8. **Business-critical data** — only the tables/columns the main logic depends on and why (omit
   ubiquitous joins; see hygiene).
9. **See also / peers** — when the flow continues in another capability or repo (an identity hand-off,
   or the owner of an async-readiness read), name it by **repo + capability**, and mirror it into the
   `see_also` frontmatter so the edge is machine-usable. Names only — never a path. (Omit when there
   are no peers.)

Omit any section that is genuinely empty — don't pad. Use `UNKNOWN` for a fact you can't ground.

## Library / compile-time-consumer repos

Some repos are consumed at **compile time via `import` / a direct API call**, not over a wire (a
shared framework or helper library). Such a repo can still be **rich** in capabilities — this is a
matter of *framing*, orthogonal to the sparse-roster rule (a library may have many capabilities). For
these, the runtime-service framing above bends:

- **How a peer interacts** — a **package + exported function/type call** (an `import`/API call), not
  an endpoint / RPC / message kind.
- **Integration seam** — a **call-ordering / field-population dependency** (call A before B; populate
  field X or B misbehaves), not a runtime identity hand-off. It is verified as an **ordering
  hand-off**: does a concept state the required call order / prerequisite field population?
- **Async readiness** — usually **N/A**: a synchronous call returns its result. Omit the section;
  do **not** invent a poll signal to satisfy the readiness expectation.
- **Business-critical data** — usually **omitted**: a library owns no tenant/customer tables.
- **Framework vocabulary** — for an HTTP / serialization / codec framework, `marshal` / `serialize`
  / `deserialize` etc. are the library's **expected peer-facing vocabulary**, not internal-mechanic
  leaks. Use the `lint-ok` marker (see [hygiene.md](./hygiene.md)) liberally on those lines here —
  it is the norm for this repo shape, not the sparing exception.

## Consumer / frontend repo

Some repos **consume** other repos' contracts without exposing any of their own — a React SPA, a
dashboard, a thin client. Such a repo exposes **no wire API a peer calls and no importable library**;
it only reads other repos' HTTP contracts and renders/acts on them. Its peer audience is a **backend
repo**, and the peer-relevant content is *what this repo consumes from you, and what breaks in its UI
if you change it*. For these the runtime-service framing inverts:

- **How a peer interacts** — **Backend contracts consumed**: the endpoints this repo *calls* (method +
  path) and the request/response fields it depends on. Mark a non-exhaustive field list `key fields:`.
- **Integration seam** — a **produced-by-peer → consumed-here** dependency: which upstream field this
  repo reads and keys off, and any name→id bridge it performs itself — not a runtime hand-off it emits.
- **Async readiness** — **what the UI polls**: real and peer-relevant here (a change to the peer's
  poll/status contract breaks this repo), unlike the library framing which omits it. Give the read +
  status field + terminal value it waits on.
- **Observable behavior** — what the UI *does* with the response (cache/invalidate, optimistic update,
  redirect, stream), not what it returns to a caller (it has none).
- **Business-critical data** — usually **omitted**: a frontend owns no tenant/customer tables.
- **Enforcement locus** — a client-side guard is **advisory UX only, NEVER authorization**. State that
  the peer must enforce server-side; a disabled button or hidden route is not a security boundary.

## GOOD example (concrete, tight, behavior-first)

```markdown
---
type: capability
title: "A2A peer messaging"
tags: [a2a, messaging, cross-project]
timestamp: 2026-07-07T00:00:00Z
repo: orchestrator
commit_sha: 9d3b58b
evidence: [cmd/websocket/a2a_message.go, pkg/protocol/protocol.go]
---

# A2A peer messaging

**What it does.** Lets one project's session send a message to another project in the same
workspace and receive a reply — the cross-project "ask a peer" channel.

**How a peer interacts.** Send the `a2a_send` tool call with `to_project` (the peer's UUID) and a
`body`; set `in_reply_to` to thread a reply.

**Observable behavior.** The message is persisted before delivery, then delivered to the target's
live session; if the target has no live session the message stays pending and is delivered when it
next connects. A reply arrives as a later `a2a_send` back to the sender.

**Contract.** In: `{to_project, body, in_reply_to?}`. Out: an ack that the message was durably
accepted (not that it was read). Errors: unknown/foreign-workspace target is rejected.

**Invariants.** Persist-before-deliver (no lost messages on a down peer); a message is never
delivered across workspace boundaries.

**Failure modes.** Target down → pending, not an error; the sender is not blocked. Foreign target →
rejected at send.

**Gotchas.** Delivery ack ≠ read receipt. Waiting for a reply is the caller's job (replies land as a
later turn), not a synchronous return.

**Business-critical data.** Messages persist to the A2A conversation store keyed by target project;
`in_reply_to` threads them. (Tenant scoping applies as everywhere — see context.md.)
```

## BAD example (everything this KB rejects)

```markdown
# A2A messaging

The `A2AMessage` struct is unmarshalled from JSON in `a2a_message.go:78` and mapped to a
`deliveryRecord` DTO via `toDelivery()`, which serializes it back to protobuf before the
`conversations` table INSERT (joined to `projects` and `companies` on every query) at line 104...
```

Why it's bad: narrates internal mechanics (unmarshal, DTO mapping, serialization), cites `file:line`
in the body, restates the join chain and the ubiquitous tenant join, and tells a peer **nothing**
about how to *use* the capability. It is longer and less useful.
