#!/usr/bin/env bash
# wlog.sh — append ONE event to a work item's append-only event log (work.jsonl).
#
# Usage:
#   scripts/wlog.sh <work-dir> <event-type> [key=value ...]
#
# Examples:
#   scripts/wlog.sh docs/work/work-2607010322-dark-mode created \
#       title="Add dark mode" slug=dark-mode kind=work repo=ps-ui owner=dev0@platformsmith.com
#   scripts/wlog.sh docs/work/work-2607010322-dark-mode status_changed to=implementation
#   scripts/wlog.sh docs/work/work-2607010322-dark-mode phase_done phase=requirements \
#       note="acceptance criteria signed off"
#
# Contract (see docs/dev/decisions/append-only-work-event-log.md):
#   * This is the ONLY sanctioned writer of work.jsonl.
#   * It APPENDS exactly one JSON object (one line). It never edits the manifest,
#     never moves/deletes files, never reads other files. One job: append.
#   * `seq` (monotonic) and `ts` (UTC) are assigned here, not by the caller.
#   * `actor` defaults to the git email; override with actor=<...>.
#   * All other fields come from key=value args, JSON-encoded safely via jq
#     (so a body/ask/note containing quotes or newlines can never corrupt the log).
#
# Design rule #1 — "append an event, or it didn't happen": every state-relevant
# fact (status, phase, artifact, relay lifecycle) MUST be an event here. The
# manifest is a generated view (scripts/wrender.sh); a future control-plane reader
# relies on this log being complete.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# common.sh gives house-style logging when present; fall back to a minimal logger
# so wlog.sh stays usable inside a sandbox pod that only carries the repo.
if [[ -f "$SCRIPT_DIR/lib/common.sh" ]]; then
  # shellcheck source=/dev/null
  source "$SCRIPT_DIR/lib/common.sh"
else
  log_error() { echo "[ERROR] $*" >&2; }
  log_info()  { echo "[INFO]  $*" >&2; }
fi

command -v jq >/dev/null 2>&1 || { log_error "wlog.sh requires jq"; exit 1; }

[[ $# -ge 2 ]] || {
  log_error "usage: wlog.sh <work-dir> <event-type> [key=value ...]"
  exit 1
}

WORK_DIR="${1%/}"; shift
TYPE="$1"; shift

[[ -d "$WORK_DIR" ]] || { log_error "work dir not found: $WORK_DIR"; exit 1; }
[[ "$TYPE" =~ ^[a-z_]+$ ]] || { log_error "event-type must be lower_snake_case: $TYPE"; exit 1; }

LOG="$WORK_DIR/work.jsonl"
[[ -f "$LOG" ]] || : > "$LOG"

# Monotonic seq = existing line count + 1. A work item has a single writer at a
# time (one agent in one branch), so this needs no lock.
SEQ=$(( $(wc -l < "$LOG" | tr -d ' ') + 1 ))
TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# actor defaults to the git email unless the caller supplies actor=...
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
  # Later keys win on collision via object-merge (so actor=... overrides default).
  filter="$filter + {\"$key\": \$$key}"
done

# jq -nc emits one compact line + trailing newline → append.
jq -nc "${jq_args[@]}" "$filter" >> "$LOG"
log_info "appended seq=$SEQ type=$TYPE → $LOG"
