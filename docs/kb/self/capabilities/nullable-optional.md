---
type: capability
title: "Nullable optional type"
tags: [nullable, optional, option-monad, handler-context, generics]
timestamp: 2026-07-07T02:32:18Z
description: "Generic type-safe optional (Nullable[T]) used for request-scoped context fields instead of *T pointers"
repo: japi-core
commit_sha: 910ed6a
evidence:
  - handler/nullable.go
  - handler/nullable_test.go
see_also:
  - {repo: japi-core, capability: "Typed handler framework", intent: "HandlerContext fields (params, body, auth) are Nullable-typed"}
  - {repo: japi-core, capability: "Typed middleware pipeline", intent: "middleware populates the Nullable ctx fields handlers read"}
---

# Nullable optional type

**What it does.** `handler.Nullable[T]` is a generic wrapper for a value that may or may not be
present — a type-safe optional used instead of `*T` pointers for request-scoped data on
`HandlerContext` (parsed params, request body, auth fields such as the user UUID and request ID).
Middleware populates these fields; handlers read them and must decide how to treat absence.

**How a peer interacts.** A Go caller (in this repo or a consuming service) constructs one of:
- `handler.NewNullable(v)` — a present value holding `v`.
- `handler.Nil[T]()` — an empty (absent) value.

and reads it through the method set below. There is no endpoint or wire contract — this is a
compile-time Go type.

**Contract (method set + return semantics).**
- `HasValue() bool` — true iff a value is present.
- `Value() (T, error)` — present: `(v, nil)`; **empty: `(zero, error)`** (an internal-server-error
  API error, "nullable value is not present").
- `TryValue() (T, bool)` — present: `(v, true)`; empty: `(zero, false)`. Never errors, never panics.
- `ValueOr(default T) T` — present: `v`; empty: the caller-supplied `default`.
- `ValueOrDefault() T` — present: `v`; empty: the zero value of `T`.

**Observable behavior.**
- Presence is tracked by an explicit internal flag, independent of the wrapped value. Storing a zero
  value (`NewNullable(0)`, `NewNullable("")`, `NewNullable(false)`, or even a nil pointer) yields a
  **present** Nullable — a zero value is not treated as absent. This distinguishes "field set to
  zero" from "field never set".
- **JSON:** `Nullable[T]` exposes no exported fields and defines no custom JSON encoding, so standard
  Go JSON marshalling of a `Nullable` value produces `{}` regardless of contents. <!-- lint-ok: marshal — JSON marshalling behavior is the peer-facing point --> It therefore does
  **not** round-trip through JSON. This is why it is used only for request-scoped `HandlerContext`
  fields (populated in-process by middleware), and **not** for API request/response model structs —
  models use `*T` / `*time.Time` for nullable fields so they marshal correctly. <!-- lint-ok: marshal — JSON marshalling behavior IS the peer-facing gotcha here -->


**Invariants.**
- Immutable value type: unexported fields, no setters, value receivers — a `Nullable` cannot be
  mutated after construction; passing it by value cannot alter the original.
- Presence is fixed at construction (`NewNullable` → present, `Nil` → absent).
- Read methods never mutate state.

**Failure modes.** The only failure surface is `Value()` on an empty Nullable, which returns a
non-nil error (and the zero value) rather than panicking. A peer that calls `Value()` **must** check
the error; the absence is reported explicitly, not by a sentinel value.

**Gotchas.**
- Ignoring the `Value()` error yields the zero value silently — it can look like a legitimate zero.
  For genuinely optional fields prefer `TryValue()`, `ValueOr()`, or a `HasValue()` guard; reserve
  `Value()` for fields a required middleware is expected to have populated.
- Do not use `Nullable[T]` in JSON-serialized model structs — it marshals to `{}`. Use `*T` there. <!-- lint-ok: marshal — the marshalling failure is the gotcha being warned about -->

- A stored nil pointer or zero value still reports `HasValue() == true`; absence means "constructed
  via `Nil`", not "wrapped value is zero/nil".

**See also / peers.** japi-core "Handler framework" (owns `HandlerContext`, whose optional fields are
Nullable-typed); japi-core "Typed middleware" (populates those Nullable fields before the handler
runs).
