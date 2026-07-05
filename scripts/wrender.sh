#!/usr/bin/env bash
# wrender.sh — fold a work item's event log (work.jsonl) into manifest.md.
#
# Usage:
#   scripts/wrender.sh <work-dir>
#
# Contract (see docs/dev/decisions/append-only-work-event-log.md):
#   * PURE PROJECTION. Deterministic: identical work.jsonl → byte-identical
#     manifest.md. No LLM, no judgment, no network.
#   * The manifest is a GENERATED VIEW. Never hand-edit it; to change state,
#     append an event with scripts/wlog.sh, then re-run this.
#   * The renderer only PLACES prose the LLM already authored (event `note`
#     fields); it never invents narrative. Judgment is captured at append time.
#
# Sections rendered (all folded from the log):
#   Header  — Status (last status_changed, or 🚨 if a later `escalated` event),
#             Epic Phase Done (last phase_done),
#             Owner/Epic/Parent/Wishlist/Priority/Effort (latest created|meta_changed),
#             Created (first event), Last Updated (last event).
#   Artifacts       — artifact_added events.
#   Open Relays     — relay_sent/relay_received minus relay_resolved (by slug+dir).
#   Upstream Msgs   — relay_received events (full history).
#   Change Log      — every work.jsonl event, in seq order, with its note.
#
# TWO input logs (writer-partitioned — see
# docs/dev/decisions/parent-child-work-items-and-conduct.md):
#   work.jsonl   — the item's own state; written ONLY by worker-side tooling
#                  (wlog.sh). Carries relay_sent / relay_resolved / escalated.
#   relays.jsonl — delivery state; written ONLY by the conductor (rlog.sh).
#                  Carries relay_received / relay_synced. Optional (absent on
#                  standalone items and legacy epic-bound items, whose delivery
#                  events live in work.jsonl).
# Relay sections fold BOTH logs; the Change Log folds work.jsonl only (the two
# logs have independent seq spaces).
#
# A parent item's manifest may carry a derived children board between
# <!-- BEGIN BOARD --> / <!-- END BOARD --> anchors, written by
# scripts/conduct-board.sh --write. This renderer PRESERVES that region
# verbatim across re-renders (it is the one part of the manifest this script
# does not own).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/lib/common.sh" ]]; then
  # shellcheck source=/dev/null
  source "$SCRIPT_DIR/lib/common.sh"
else
  log_error() { echo "[ERROR] $*" >&2; }
  log_info()  { echo "[INFO]  $*" >&2; }
fi

command -v jq >/dev/null 2>&1 || { log_error "wrender.sh requires jq"; exit 1; }
[[ $# -ge 1 ]] || { log_error "usage: wrender.sh <work-dir>"; exit 1; }

WORK_DIR="${1%/}"
LOG="$WORK_DIR/work.jsonl"
RLOG="$WORK_DIR/relays.jsonl"
OUT="$WORK_DIR/manifest.md"
[[ -f "$LOG" ]] || { log_error "no work.jsonl in $WORK_DIR"; exit 1; }
[[ -s "$LOG" ]] || { log_error "work.jsonl is empty in $WORK_DIR"; exit 1; }

# Relay sections fold work.jsonl + relays.jsonl (delivery log, conductor-owned).
RELAY_SRCS=("$LOG")
[[ -s "$RLOG" ]] && RELAY_SRCS+=("$RLOG")

WORK_ID="$(basename "$WORK_DIR")"

# ── scalar extractors (slurp the log once per query; tiny files) ───────────────
# latest value of a header field carried on a created|meta_changed event
field() { jq -rs --arg k "$1" \
  'map(select((.type=="created" or .type=="meta_changed") and has($k)) | .[$k]) | (last // "—")' "$LOG"; }

title="$(jq -rs 'map(select(.type=="created"))[0].title // "Untitled"' "$LOG")"
request="$(jq -rs 'map(select(.type=="created"))[0].request // ""' "$LOG")"
created_ts="$(jq -rs '.[0].ts // ""' "$LOG")"
updated_ts="$(jq -rs '.[-1].ts // ""' "$LOG")"
# Status: last status_changed — unless a later `escalated` event exists (an
# escalated item is out of play until a human decides; any subsequent
# status_changed resumes/settles it).
status_key="$(jq -rs '
  map(select(.type=="status_changed" or .type=="escalated")
      | (if .type=="escalated" then "escalated" else .to end))
  | (last // "proposed")' "$LOG")"
phase_done="$(jq -rs 'map(select(.type=="phase_done") | .phase) | (last // "—")' "$LOG")"
owner="$(field owner)"; epic="$(field epic)"; wishlist="$(field wishlist)"
priority="$(field priority)"; effort="$(field effort)"
parent="$(field parent)"; parent_project="$(field parent_project)"

created_date="${created_ts%%T*}"; [[ -n "$created_date" ]] || created_date="—"
updated_date="${updated_ts%%T*}"; [[ -n "$updated_date" ]] || updated_date="—"

# Original Request — the user's verbatim prompt (load-bearing context for git-resume).
# Rendered as a blockquote when present; omitted entirely when absent (legacy items).
request_section=""
if [[ -n "$request" ]]; then
  request_section="## Original Request"$'\n\n'"$(printf '%s\n' "$request" | sed 's/^/> /')"$'\n\n'
fi

# status key → badge (the only place the emoji vocabulary lives)
case "$status_key" in
  proposed)       badge="🎯 Proposed" ;;
  researching)    badge="📚 Researching" ;;
  requirements)   badge="📝 Requirements" ;;
  planning)       badge="🎨 Planning" ;;
  implementation) badge="🔄 In Implementation" ;;
  completed)      badge="✅ Completed" ;;
  blocked)        badge="🔴 Blocked" ;;
  escalated)      badge="🚨 Escalated" ;;
  on_hold)        badge="⏸️ On Hold" ;;
  cancelled)      badge="❌ Cancelled" ;;
  *)              badge="🎯 $status_key" ;;
esac

# Parent header line (parent-bound child items only; omitted when standalone).
parent_line=""
if [[ "$parent" != "—" ]]; then
  parent_line="**Parent**: $parent @ ${parent_project}"$'\n'
fi

# Preserve a conduct-board-written children board region across re-renders.
# The board lives between the anchors and is owned by conduct-board.sh --write;
# this renderer carries it forward verbatim (re-anchored after the header).
board_section=""
if [[ -f "$OUT" ]] && grep -qF '<!-- BEGIN BOARD -->' "$OUT"; then
  board_body="$(awk '/<!-- BEGIN BOARD -->/{f=1;next} /<!-- END BOARD -->/{f=0} f' "$OUT")"
  board_section="<!-- BEGIN BOARD -->"$'\n'"$board_body"$'\n'"<!-- END BOARD -->"$'\n\n'
fi

# ── section bodies ────────────────────────────────────────────────────────────
artifacts="$(jq -rs '
  map(select(.type=="artifact_added"))
  | if length==0 then "_None yet._"
    else map("- [\(.title // .path)](\(.path)) — \(.kind // "artifact")") | join("\n") end' "$LOG")"

# Both relay sections fold work.jsonl + relays.jsonl (delivery events live in
# relays.jsonl for parent-bound items; in work.jsonl for legacy epic-bound ones).
open_relays="$(jq -rs '
  (map(select(.type=="relay_resolved") | (.direction + "/" + .slug))) as $resolved
  | (map(select(.type=="relay_synced") | .slug)) as $synced
  | map(select(.type=="relay_sent" or .type=="relay_received")
        | { dir: (if .type=="relay_sent" then "outbound" else "inbound" end),
            peer: (.to // .from // "—"), slug: .slug,
            kind: (.relay_kind // "—"), phase: (.phase // "—"), ask: (.ask // "") })
  | map(select((.dir + "/" + .slug) as $k | ($resolved | index($k)) | not))
  | if length==0 then "_None._"
    else ( ["| Direction | Peer | Slug | Kind | Phase | Ask |",
            "|-----------|------|------|------|-------|-----|"]
           + map("| \(.dir)\(if (.dir=="outbound") and ([.slug] | inside($synced)) then " ✓" else "" end) | \(.peer) | \(.slug) | \(.kind) | \(.phase) | \(.ask) |") )
         | join("\n") end' "${RELAY_SRCS[@]}")"

upstream="$(jq -rs '
  map(select(.type=="relay_received"))
  | if length==0 then "_None._"
    else map("- [\(.ts[0:10])] from \(.from // "—"): **\(.slug)** — \(.ask // "")\(if .path then " (`"+.path+"`)" else "" end)") | join("\n") end' "${RELAY_SRCS[@]}")"

changelog="$(jq -rs '
  def summ:
    if   .type=="created"        then "created — \(.title // "")"
    elif .type=="status_changed" then "status → \(.to)"
    elif .type=="phase_done"     then "phase done: \(.phase)"
    elif .type=="artifact_added" then "artifact: \(.title // .path)"
    elif .type=="relay_sent"     then "relay → \(.to // "—"): \(.slug)"
    elif .type=="relay_received" then "relay ← \(.from // "—"): \(.slug)"
    elif .type=="relay_synced"   then "relay synced: \(.slug)"
    elif .type=="relay_resolved" then "relay resolved (\(.direction // "—")): \(.slug)"
    elif .type=="escalated"      then "🚨 escalated"
    elif .type=="meta_changed"   then "meta updated"
    elif .type=="note"           then "note"
    else .type end;
  map("- \(.ts[0:10]) · seq \(.seq) · \(summ)\(if .note then " — " + .note else "" end)")
  | join("\n")' "$LOG")"

# ── assemble (atomic write via temp + mv) ─────────────────────────────────────
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
cat > "$TMP" <<EOF
# Work Item: $title

<!-- GENERATED by scripts/wrender.sh from work.jsonl — DO NOT EDIT BY HAND. -->
<!-- To change state: scripts/wlog.sh $WORK_DIR <event-type> ... ; then scripts/wrender.sh $WORK_DIR -->

**ID**: $WORK_ID
**Status**: $badge
**Created**: $created_date
**Last Updated**: $updated_date
**Owner**: $owner
**Epic**: $epic
${parent_line}**Wishlist**: $wishlist
**Epic Phase Done**: $phase_done
**Priority**: $priority
**Estimated Effort**: $effort

${request_section}${board_section}## Artifacts

$artifacts

## Open Relays

$open_relays

## Upstream Messages

$upstream

## Change Log

$changelog
EOF

mv "$TMP" "$OUT"
trap - EXIT
log_info "rendered $OUT (status=$status_key, phase_done=$phase_done)"
