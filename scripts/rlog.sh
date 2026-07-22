#!/usr/bin/env bash
# rlog.sh — append ONE conductor-owned event to a work item's relays.jsonl.
#
# THE CONDUCTOR'S LOG. `relays.jsonl` is the conductor-owned ledger of a work
# item and has exactly ONE writer: the conductor (`/conduct sync` +
# `conduct-board.sh --write`). Worker-side tooling never writes here — workers
# append to `work.jsonl` via wlog.sh. This writer partition (every jsonl has
# exactly one writer) is the file-mode mirror of the product substrate's
# ownership model: delivery state is platform-owned (conversation_message),
# node state is worker-owned (ps_task writes).
# See docs/dev/decisions/parent-child-work-items-and-conduct.md.
#
# Two event families live here, both conductor-owned:
#   * relay delivery — relay_received / relay_synced (push of a peer's message)
#   * barrier push   — barrier_advanced (the conductor PUSHES the derived barrier
#     into the child's OWN tree so an isolated `/work auto` worker reads the
#     barrier locally instead of reaching across the repo boundary for the parent
#     manifest; fields: phase + state). See
#     docs/dev/decisions/conductor-pushes-barrier-into-child-territory.md.
#
# Usage:
#   scripts/rlog.sh <work-dir> <relay_received|relay_synced|barrier_advanced> [key=value ...]
#
# Examples:
#   # conductor delivered an inbound relay INTO this work item:
#   scripts/rlog.sh repos/alpha/docs/work/work-...-x relay_received \
#       from=beta slug=beta-needs relay_kind=blocks phase=planning \
#       ask="expose the seam" path=relays/inbound/from-beta--beta-needs.md
#   # conductor delivered this work item's outbound relay to its target:
#   scripts/rlog.sh docs/work/work-...-y relay_synced slug=beta-needs
#   # conductor pushed the current barrier into this child (conduct-board.sh --write):
#   scripts/rlog.sh repos/alpha/docs/work/work-...-x barrier_advanced phase=planning state=open
#
# Contract:
#   * ONLY the conductor invokes this (delivery + barrier push are conductor-driven).
#   * Event types are restricted to the conductor vocabulary below — everything
#     else (relay_sent, relay_resolved, phase_done, …) belongs to work.jsonl.
#   * `seq`/`ts` assigned here; `actor` defaults to the git email.
#   * wrender.sh folds work.jsonl + relays.jsonl into the manifest's Open Relays /
#     Upstream Messages sections (selecting relay_* types only — barrier_advanced
#     is ignored by the renderer, no manifest noise); conduct-board.sh folds both
#     when deriving the barrier.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/lib/common.sh" ]]; then
  # shellcheck source=/dev/null
  source "$SCRIPT_DIR/lib/common.sh"
else
  log_error() { echo "[ERROR] $*" >&2; }
  log_info()  { echo "[INFO]  $*" >&2; }
fi

command -v jq >/dev/null 2>&1 || { log_error "rlog.sh requires jq"; exit 1; }

[[ $# -ge 2 ]] || {
  log_error "usage: rlog.sh <work-dir> <relay_received|relay_synced|barrier_advanced> [key=value ...]"
  exit 1
}

WORK_DIR="${1%/}"; shift
TYPE="$1"; shift

[[ -d "$WORK_DIR" ]] || { log_error "work dir not found: $WORK_DIR"; exit 1; }
case "$TYPE" in
  relay_received|relay_synced|barrier_advanced) ;;
  *) log_error "rlog.sh only appends conductor events (relay_received|relay_synced|barrier_advanced); '$TYPE' belongs in work.jsonl via wlog.sh"; exit 1 ;;
esac

LOG="$WORK_DIR/relays.jsonl"
[[ -f "$LOG" ]] || : > "$LOG"

# Monotonic seq within relays.jsonl. Single writer (the conductor) — no lock.
SEQ=$(( $(wc -l < "$LOG" | tr -d ' ') + 1 ))
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

has_actor=0
for kv in "$@"; do [[ "${kv%%=*}" == "actor" ]] && has_actor=1; done
ACTOR_DEFAULT="$(git -C "$WORK_DIR" config user.email 2>/dev/null || echo "${USER:-unknown}")"

jq_args=(--argjson seq "$SEQ" --arg ts "$TS" --arg type "$TYPE")
filter='{seq:$seq, ts:$ts, type:$type}'
if [[ $has_actor -eq 0 ]]; then
  jq_args+=(--arg actor "$ACTOR_DEFAULT")
  filter="$filter + {actor:\$actor}"
fi

for kv in "$@"; do
  [[ "$kv" == *=* ]] || { log_error "bad arg (need key=value): $kv"; exit 1; }
  key="${kv%%=*}"
  val="${kv#*=}"
  [[ "$key" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]] || { log_error "invalid key: $key"; exit 1; }
  jq_args+=(--arg "$key" "$val")
  filter="$filter + {\"$key\": \$$key}"
done

jq -nc "${jq_args[@]}" "$filter" >> "$LOG"
log_info "appended seq=$SEQ type=$TYPE → $LOG (conductor delivery log)"
