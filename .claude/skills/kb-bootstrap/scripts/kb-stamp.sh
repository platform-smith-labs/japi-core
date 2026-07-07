#!/usr/bin/env bash
# kb-stamp.sh <repo-root> [concept.md ...] — normalize the `timestamp:` and `commit_sha:` frontmatter
# of KB concept files to REAL deterministic values (generation-time stamp), replacing any
# model-invented placeholder from DRAFT. Rewrites ONLY those two scalar lines; never touches bodies.
# NOT byte-idempotent across runs by design (timestamp = wall clock) — belongs to the GENERATION
# layer, run ONCE per run BEFORE kb-lint. Never invoked from kb-render (which stays byte-identical).
# No LLM. (Phase 2 / FR-4)
set -euo pipefail
ROOT="${1:-.}"; shift || true
KB="$ROOT/docs/kb"
[ -d "$KB" ] || { echo "kb-stamp: no $KB (nothing to stamp)" >&2; exit 0; }

TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
SHA="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo UNKNOWN)"

# Target set: explicit args (the concepts regenerated this run — minimal churn), else every concept.
if [ "$#" -gt 0 ]; then
  FILES=("$@")
else
  mapfile -t FILES < <(find "$KB/self" -type f -name '*.md' ! -name 'index.md' ! -path '*/extract/*' ! -path '*/eval/*' 2>/dev/null | LC_ALL=C sort)
fi

stamp_one() {
  local f="$1" tmp; tmp="$(mktemp)"
  awk -v ts="$TS" -v sha="$SHA" '
    BEGIN{ d=0; seen_ts=0; seen_sha=0 }
    /^---[[:space:]]*$/ {
      d++
      if (d==2) {                                  # closing fence: insert any missing fields first
        if(!seen_ts)  print "timestamp: " ts
        if(!seen_sha) print "commit_sha: " sha
        print; next
      }
      print; next
    }
    d==1 && /^timestamp:/  { print "timestamp: " ts;  seen_ts=1;  next }
    d==1 && /^commit_sha:/ { print "commit_sha: " sha; seen_sha=1; next }
    { print }
  ' "$f" > "$tmp" && mv "$tmp" "$f"
}

for f in "${FILES[@]}"; do [ -f "$f" ] && stamp_one "$f"; done
echo "kb-stamp: stamped ${#FILES[@]} concept(s) @ $TS ($SHA)" >&2
