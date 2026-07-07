# kb-config.yaml — the repo-owned steering file

How a maintainer durably steers generation **without** hand-editing generated files that the next run
would clobber. Lives at `docs/kb/kb-config.yaml`, is hand-edited, and **survives regeneration** (the
skill never overwrites it). Read by every stage (EXTRACT, DRAFT, VERIFY). Absent → sensible defaults.

## Schema

```yaml
# docs/kb/kb-config.yaml — all keys optional
exclude:                      # paths the KB ignores entirely (globs)
  - "vendor/**"
  - "**/*_test.go"
  - "docs/**"

brevity_exempt:               # concept paths whose line-count WARN is skipped (globs; every OTHER lint check still applies)
  - "docs/kb/self/interfaces/schema-*.md"

capability_hints:             # maintainer-named functionalities + seed pointers to steer EXTRACT/DRAFT
  - name: "A2A peer messaging"
    seed: ["cmd/websocket/a2a_message.go"]
  - name: "Container lifecycle"
    seed: ["cmd/internal/launch/"]

notes:                        # durable guidance injected into DRAFT prompts (free text)
  - "Sessions are ephemeral; never describe them as persistent."
  - "The A2A reply is asynchronous — always say so."

pins:                         # facts the maintainer asserts to break ambiguous ties (a strong prior, NOT an override of readable code)
  - "The canonical peer transport is A2A over the orchestrator, not direct HTTP."
```

## Key semantics

| Key | Effect |
|---|---|
| `exclude` | globs removed from EXTRACT's inventory and from DRAFT's view — noise/vendor/test paths |
| `brevity_exempt` | globs (repo-relative concept paths) whose **line-count brevity WARN only** (`kb-lint.sh` check 8) is suppressed — for reference-shaped concepts where length tracks the contract, not verbosity (a complete schema reference: tables, columns, FKs, enum value sets). Every other check still fires, including the §0 no-source-pointer / no-fabrication **FAILs**. A shell `case` glob's `*` spans `/`, so `**/schema-*.md` also matches |
| `capability_hints` | names + seeds bias which capabilities DRAFT writes and where it starts looking; does not cap discovery (DRAFT may find more) |
| `notes` | free-text steering injected verbatim into DRAFT prompts — corrections and framing, **not** ground truth that overrides readable code (a note the code negates is a **PIN CONFLICT**, same as a pin) |
| `pins` | maintainer-asserted facts used as a **tie-breaker for ambiguous inference**; DRAFT/VERIFY treat a pin as a strong prior, **not** ground truth that overrides readable code. A pin the code contradicts is a **PIN CONFLICT** VERIFY surfaces to the maintainer, not a fact to copy |

## Defaults when absent

- No exclusions (but EXTRACT still skips `.git/` and obvious build dirs by built-in heuristic).
- DRAFT discovers capabilities itself from the code + EXTRACT fact sheet.
- No extra notes/pins.

## Rules

- **Never generated or overwritten** by the skill — it is the one hand-authored file inside `docs/kb/`.
- Keep it small; it is steering, not content. Long guidance belongs in the repo's own docs, referenced
  from a `note`.
- `kb-lint.sh` validates it parses and that `capability_hints[].seed` / `exclude` globs are
  well-formed; it does not require the file to exist.
