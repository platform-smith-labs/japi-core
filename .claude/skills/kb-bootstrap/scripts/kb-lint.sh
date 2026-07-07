#!/usr/bin/env bash
# kb-lint.sh <repo-root> — validate a docs/kb bundle against the house schema + §0 hygiene.
# FAIL (exit 1): schema violations, body source-pointer leaks. WARN: hygiene/brevity/drift.
# Deterministic; no LLM. (Phase 2 / FR-4.1; rules from references/schema.md + references/hygiene.md)
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_kb_lib.sh
. "$HERE/_kb_lib.sh"

ROOT="${1:-.}"
KB="$ROOT/docs/kb"
VOCAB="overview context capability interface decision gotcha glossary note"
FAILS=0; WARNS=0; UNKNOWNS=0
fail() { echo "FAIL  $1" >&2; FAILS=$((FAILS+1)); }
warn() { echo "warn  $1" >&2; WARNS=$((WARNS+1)); }

[ -d "$KB" ] || { echo "kb-lint: no $KB" >&2; exit 1; }

# Concept files = self narrative + collection concepts; skip index.md, log.md, extract/, eval/, kb-config.
mapfile -t FILES < <(find "$KB/self" -type f -name '*.md' ! -name 'index.md' ! -path '*/extract/*' ! -path '*/eval/*' 2>/dev/null | LC_ALL=C sort)
[ "${#FILES[@]}" -eq 0 ] && { echo "kb-lint: no concept files under $KB/self" >&2; exit 0; }

HEAD_SHA="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || true)"

# Optional kb-config brevity_exempt globs — concept paths whose line-count WARN (check 8) is skipped
# ONLY; every other check (incl. all §0 FAILs) still applies. Mirrors kb-extract.sh's exclude reader.
# Globs match the repo-relative concept path (e.g. docs/kb/self/interfaces/schema-*.md); a shell
# `case` '*' spans '/', so '**/schema-*.md' works too. Layout-agnostic: authored per-repo, nothing baked.
BREV_EXEMPT=()
CFG="$ROOT/docs/kb/kb-config.yaml"
[ -f "$CFG" ] && mapfile -t BREV_EXEMPT < <(awk '/^brevity_exempt:/{e=1;next} e&&/^[[:space:]]+-/{sub(/^[[:space:]]+-[[:space:]]*/,"");gsub(/^["'\'']+|["'\'']+$/,"");print;next} e&&/^[^[:space:]]/{e=0}' "$CFG")
brevity_exempt() { local p="$1" g; for g in "${BREV_EXEMPT[@]:-}"; do [ -n "$g" ] || continue; case "$p" in ${g}) return 0;; esac; done; return 1; }

for f in "${FILES[@]}"; do
  rel="${f#"$ROOT"/}"

  # 1. frontmatter present
  if ! kb_has_fm "$f"; then fail "$rel: no frontmatter"; continue; fi

  # 2. type present + in vocabulary
  type="$(kb_fm_scalar "$f" type)"
  if [ -z "$type" ]; then fail "$rel: missing 'type'"
  elif ! printf '%s ' $VOCAB | grep -qwF "$type"; then fail "$rel: type '$type' not in vocabulary"; fi

  # 3. required scalars
  for k in title timestamp; do
    [ -n "$(kb_fm_scalar "$f" "$k")" ] || fail "$rel: missing '$k'"
  done
  [ "$(kb_fm_list_count "$f" tags)" -gt 0 ] 2>/dev/null || warn "$rel: empty/missing 'tags'"

  # 4. capability/interface: evidence non-empty
  case "$type" in
    capability|interface)
      [ "$(kb_fm_list_count "$f" evidence)" -gt 0 ] 2>/dev/null || fail "$rel: $type concept has empty 'evidence' (grounding required)"
      ;;
  esac

  body="$(kb_body "$f")"

  # 5. FAIL: source pointers in body — file:line, or markdown links to code files
  if printf '%s\n' "$body" | grep -qE '[[:alnum:]_./-]+\.(go|rs|ts|tsx|js|jsx|py|sql|proto|graphql|java|rb):[0-9]+'; then
    fail "$rel: body contains a file:line source pointer (evidence belongs in frontmatter only — §0)"
  fi
  if printf '%s\n' "$body" | grep -qE '\]\([^)]*\.(go|rs|ts|tsx|js|jsx|py|sql|proto|graphql|java|rb)([):#])'; then
    fail "$rel: body links to a source file (no consumer-facing source pointers — §0)"
  fi

  # 5b. FAIL: see_also frontmatter must be NAME-based (repo + capability) — no path / file:line
  # (schema.md rule 9 — §0 extends to frontmatter refs).
  saw="$(kb_fm_block "$f" | awk '/^see_also:/{e=1;next} e&&/^[^[:space:]]/{e=0} e{print}')"
  if [ -n "$saw" ] && printf '%s\n' "$saw" | grep -qiE '\.(go|rs|ts|tsx|js|jsx|py|sql|proto|graphql|java|rb)([[:space:]"'\'':)]|$)|(^|[[:space:]])path:'; then
    fail "$rel: see_also has a file path/source pointer — must be name-based (repo + capability) only (§0 rule 9)"
  fi

  # 6. WARN: internal-mechanic terms in body.
  # (word boundaries \b are non-POSIX / unsupported on busybox grep — use plain substring alternation)
  # The match is context-blind, so it can flag a term used in a different sense — e.g. the concurrency
  # sense of "serialize" ("operations don't serialize"), not marshalling. A reviewed false positive is
  # exempted per-line with an inline `<!-- lint-ok: <term> — <why> -->` marker on that source line
  # (§0 hygiene: a warn means "justify or cut"; the marker records the justification). We warn only if
  # a flagged line lacks the marker: first grep selects candidate lines, second reports any un-marked one.
  if printf '%s\n' "$body" \
       | grep -iE 'unmarshal|marshal|serialize|deserialize|[^a-z]dto[^a-z]|struct mapping|dependency injection' \
       | grep -qiv 'lint-ok'; then
    warn "$rel: body has internal-mechanic term(s) — justify (inline '<!-- lint-ok: … -->') or cut (§0 hygiene)"
  fi

  # 7. WARN: capability/interface with too-thin body (likely a bare reference, no behavior)
  case "$type" in
    capability|interface)
      nlines="$(printf '%s\n' "$body" | grep -cE '[^[:space:]]')"
      [ "$nlines" -ge 5 ] || warn "$rel: $type body very thin ($nlines lines) — behavioral prose required (F5)"
      ;;
  esac

  # 8. WARN: brevity budget (capability 120, others 150 lines). Skippable per-file via kb-config
  #    brevity_exempt globs — a COMPLETE schema-reference interface is long in proportion to the
  #    contract, not verbose. ONLY this line-count WARN is scoped; §0 hard rules (checks 5/5b) stay enforced.
  if ! brevity_exempt "$rel"; then
    total="$(wc -l < "$f")"; cap=150; [ "$type" = capability ] && cap=120
    [ "$total" -le "$cap" ] || warn "$rel: $total lines > $cap soft cap (BREVITY IS CLARITY, NFR-1)"
  fi

  # 9. UNKNOWN count
  u="$(printf '%s\n' "$body" | grep -cw 'UNKNOWN' || true)"; UNKNOWNS=$((UNKNOWNS + u))

  # 10. drift: any evidence path changed since commit_sha?
  sha="$(kb_fm_scalar "$f" commit_sha)"
  # only drift-check if the stamped sha is a real commit in this clone (avoids spurious drift on
  # a foreign/shallow sha, where `git diff` returns 128 not 1)
  if [ -n "$sha" ] && [ -n "$HEAD_SHA" ] && [ "$sha" != "$HEAD_SHA" ] \
     && git -C "$ROOT" rev-parse -q --verify "$sha^{commit}" >/dev/null 2>&1; then
    while IFS= read -r p; do
      [ -n "$p" ] || continue
      if ! git -C "$ROOT" diff --quiet "$sha" HEAD -- "$p" 2>/dev/null; then
        warn "$rel: evidence '$p' changed since commit_sha $sha → regenerate (drift)"; break
      fi
    done < <(kb_fm_list_items "$f" evidence)
  fi

  # 11. bundle-absolute links resolve
  while IFS= read -r lnk; do
    [ -z "$lnk" ] && continue
    tgt="${lnk%%#*}"
    [ -e "$KB$tgt" ] || warn "$rel: bundle link '$lnk' does not resolve under docs/kb"
  done < <(printf '%s\n' "$body" | grep -oE '\]\(/[^)]+\)' | sed -E 's/^\]\(//; s/\)$//')
done

# Guard: a single-file concept directly under self/ whose stem is not a recognized singular concern
# or collection name is likely a stray/misnamed file. kb-render surfaces it under "## Self", but
# flag it so it gets promoted to a collection dir or renamed (layout growth rule). WARN only.
KNOWN_SELF="overview context glossary interfaces gotchas decisions"
while IFS= read -r f; do
  n="$(basename "$f")"; n="${n%.md}"
  printf '%s ' $KNOWN_SELF | grep -qwF "$n" \
    || warn "${f#"$ROOT"/}: unrecognized single-file self concept '$n.md' — promote to a collection dir or rename (layout growth rule)"
done < <(find "$KB/self" -maxdepth 1 -type f -name '*.md' ! -name 'index.md' 2>/dev/null | LC_ALL=C sort)

echo "kb-lint: ${#FILES[@]} concept(s), $FAILS fail, $WARNS warn, $UNKNOWNS UNKNOWN marker(s)" >&2
[ "$FAILS" -eq 0 ]
