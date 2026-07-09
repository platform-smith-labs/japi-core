#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# Platform Smith - Conduct Board (parent/child cross-repo workflow conductor)
# =============================================================================
# Read-only dashboard for a PARENT work item and its CHILD work items (the
# successor of epic-board.sh — the epic entity is removed; a standalone work
# item plays the epic role). Computes the global phase barrier + relay-settle
# state and prints a single "DO THIS NEXT".
#
# Model (see docs/dev/decisions/parent-child-work-items-and-conduct.md, which
# builds on epic-conductor-barrier-workflow.md — the barrier semantics are
# preserved verbatim):
#   - A parent's children advance ONE phase at a time across ALL repos:
#       requirements -> planning -> implementation -> validation
#   - No child starts phase P+1 until every child has settled phase P AND
#     there are zero open relays.
#   - Membership is DERIVED: each child declares parent=<work-id> (+
#     parent_project=<repo>) on its created event. Nesting is N-LEVEL
#     (2026-07-06 relaxation): any node may parent children; discovery is
#     direct-children-only, so each level's board is scoped to its own cohort.
#     Creation-time validation (see /work --parent-work): the parent= chain
#     must terminate at a standalone root with no cycles.
#   - SETTLING RULE for mid-level nodes (a child that is itself a parent): it
#     may not settle its final/validation phase_done toward ITS parent while
#     its own children board is incomplete — its children's completion is its
#     evidence. The board prints the settling hint on completion.
#   - TWO logs per child, one writer each:
#       work.jsonl   (worker-owned)    — phase_done, relay_sent, relay_resolved,
#                                        escalated, status_changed, ...
#       relays.jsonl (conductor-owned) — relay_received, relay_synced (delivery)
#     This board folds BOTH.
#   - A1 rule: a relay is ONE thing keyed by slug; ONE relay_resolved anywhere
#     among the children closes BOTH legs.
#   - ESCALATED children (last status/escalated event = escalated) are OUT OF
#     PLAY: they do not drag the barrier (excluded from the min), but the run
#     cannot COMPLETE while any child is escalated — a human must decide.
#   - PRIME/SELF ROW (2026-07-09): the parent is often itself a worker strand — its
#     own repo does execution work (the conductor + worker hats live in ONE item,
#     e.g. a repo that is both prime and conductor for a ticket). It is included as a
#     first-class GATING member (rendered first, marked ★) so its own phase_done
#     counts in the barrier min and it gets a run-in-each-repo command, exactly like
#     a child. Only when the parent already has children (a childless item stays a
#     plain standalone item, unchanged). work_dir(PARENT_REPO,PARENT_ID)==PARENT_DIR,
#     so every per-member fold resolves against the parent's own logs with no special
#     casing. See docs/dev/decisions/conduct-board-prime-strand-row.md.
#   - Read-only by default. With --write it injects the derived board (the
#     **Barrier Phase** line + children table) into the PARENT's manifest
#     between <!-- BEGIN BOARD --> / <!-- END BOARD --> anchors (wrender.sh
#     preserves that region across re-renders).
#
# Usage:
#   ./scripts/conduct-board.sh <parent-id> [--watch] [--write]
#       parent-id  full id, or a slug fragment (glob-resolved; never arithmetic)
#       --watch    refresh every 5s (Ctrl-C to exit)
#       --write    inject the derived board into the parent manifest
#
# PS_ROOT env var overrides the monorepo root (used by tests).
# =============================================================================

ROOT="${PS_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
WATCH=0
WRITE=0
PARENT_ARG=""

for a in "$@"; do
  case "$a" in
    --watch) WATCH=1 ;;
    --write) WRITE=1 ;;
    -h|--help) sed -n '2,48p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) PARENT_ARG="$a" ;;
  esac
done

[[ -n "$PARENT_ARG" ]] || { echo "usage: conduct-board.sh <parent-id> [--watch] [--write]" >&2; exit 1; }

# ---- phase ladder (verbatim from epic-board.sh) ------------------------------
phase_ord() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    *complete*|*done*)        echo 5 ;;
    *validat*)                echo 4 ;;
    *implement*)              echo 3 ;;
    *plan*)                   echo 2 ;;
    *requirement*)            echo 1 ;;
    *) echo 0 ;;
  esac
}
phase_name() { case "$1" in 1) echo requirements;; 2) echo planning;; 3) echo implementation;; 4) echo validation;; 5) echo done;; *) echo "(pre-requirements)";; esac; }
# repo_cwd <repo> -> directory to open a Claude session in (solution alias → root).
repo_cwd() { if [[ "$1" == "solution" ]]; then echo "."; else echo "repos/$1"; fi; }
# phase_slash <ord> <wid> <repo> -> the exact command for that phase. Validation
# is TWO-LAYER (children run local suites + relay e2e-needs; the parent's repo —
# normally solution — drives the cross-repo e2e and is the final gate).
phase_slash() {
  local ord="$1" wid="$2" repo="${3:-}"
  case "$ord" in
    1) echo "/work $wid" ;;
    2) echo "/planv0 --work $wid" ;;
    3) echo "/implement_plan docs/work/$wid/plans/master.md" ;;
    4) if [[ "$repo" == "$PARENT_REPO" ]]; then
         echo "author + DRIVE the cross-repo e2e (parent Success Criteria) on a live stack — you are the e2e gate; resolve each repo's e2e-needs relay as it's covered+passing, then: scripts/wlog.sh docs/work/$wid phase_done phase=validation"
       else
         echo "run LOCAL suite; then relay e2e-needs → $PARENT_REPO: write relays/outbound/to-$PARENT_REPO--$repo-e2e-needs.md + scripts/wlog.sh docs/work/$wid relay_sent to=$PARENT_REPO slug=$repo-e2e-needs relay_kind=blocks phase=validation ask=\"e2e coverage for $repo\"; then settle: scripts/wlog.sh docs/work/$wid phase_done phase=validation"
       fi ;;
    *) echo "/work $wid" ;;
  esac
}

# ---- combined-log helper -----------------------------------------------------
# child_logs <wd> -> the child's log files (work.jsonl always; relays.jsonl when present)
child_logs() {
  local wd="$1"
  [[ -f "$wd/work.jsonl" ]] && printf '%s\n' "$wd/work.jsonl"
  [[ -s "$wd/relays.jsonl" ]] && printf '%s\n' "$wd/relays.jsonl"
}

# open_inbound_slugs <wd> <resolved-slugs> -> space-separated slugs of this child's
# OPEN inbound relays (received here — relays.jsonl — and not settled anywhere).
open_inbound_slugs() {
  local wd="$1" resolved="${2:-}"
  command -v jq >/dev/null 2>&1 || return 0
  local -a srcs; mapfile -t srcs < <(child_logs "$wd")
  (( ${#srcs[@]} )) || return 0
  jq -rs --arg res "$resolved" '
    ($res|split(" ")|map(select(length>0))) as $r
    | [ .[] | select(.type=="relay_received") | select(.slug as $s | ($r|index($s))|not) | .slug ] | join(" ")' "${srcs[@]}"
}

# ---- resolve parent dir ------------------------------------------------------
# Parents may live in any repo's docs/work (default seat: the monorepo root's
# docs/work, i.e. the "solution" repo). Exact match first, then slug glob.
resolve_parent() {
  local arg="$1" d m
  for d in "$ROOT/docs/work/$arg" "$ROOT"/repos/*/docs/work/"$arg"; do
    [[ -f "$d/work.jsonl" ]] && { echo "$d"; return; }
  done
  local matches=()
  for m in "$ROOT"/docs/work/*"$arg"*/work.jsonl "$ROOT"/repos/*/docs/work/*"$arg"*/work.jsonl; do
    [[ -f "$m" ]] && matches+=("$(dirname "$m")")
  done
  if (( ${#matches[@]} == 1 )); then echo "${matches[0]}"; return; fi
  if (( ${#matches[@]} == 0 )); then echo "ERR: parent '$arg' not found" >&2; return 1; fi
  echo "ERR: parent '$arg' is ambiguous: ${matches[*]}" >&2; return 1
}

# ---- work dir for a repo + workitem -----------------------------------------
work_dir() {
  local repo="$1" wid="$2"
  if [[ "$repo" == "solution" ]]; then echo "$ROOT/docs/work/$wid"; else echo "$ROOT/repos/$repo/docs/work/$wid"; fi
}

# relay_counts <wd> <resolved-slugs> -> "<open-inbound> <open-outbound>"
# A1 RULE preserved verbatim: a slug in the run-global resolved set closes BOTH
# legs. relay_sent lives in work.jsonl (worker-owned); relay_received lives in
# relays.jsonl (conductor-owned) — fold both.
relay_counts() {
  local wd="$1" resolved="${2:-}" oin=0 oout=0
  command -v jq >/dev/null 2>&1 || { echo "0 0"; return; }
  local -a srcs; mapfile -t srcs < <(child_logs "$wd")
  (( ${#srcs[@]} )) || { echo "0 0"; return; }
  oin=$(jq -rs --arg res "$resolved" '($res|split(" ")|map(select(length>0))) as $r
        | [ .[] | select(.type=="relay_received") | select(.slug as $s | ($r|index($s))|not) ] | length' "${srcs[@]}")
  oout=$(jq -rs --arg res "$resolved" '($res|split(" ")|map(select(length>0))) as $r
        | [ .[] | select(.type=="relay_sent") | select(.slug as $s | ($r|index($s))|not) ] | length' "${srcs[@]}")
  echo "${oin:-0} ${oout:-0}"
}

# relay_list <wd> <repo> <resolved-slugs> -> one line per OPEN relay leg.
relay_list() {
  local wd="$1" repo="$2" resolved="${3:-}"
  command -v jq >/dev/null 2>&1 || return 0
  local -a srcs; mapfile -t srcs < <(child_logs "$wd")
  (( ${#srcs[@]} )) || return 0
  jq -rs --arg repo "$repo" --arg res "$resolved" '
    ($res|split(" ")|map(select(length>0))) as $r
    | [ .[] | select(.type=="relay_sent" or .type=="relay_received")
        | {dir:(if .type=="relay_sent" then "outbound" else "inbound" end), slug:.slug, peer:(.to // .from // "—")} ]
    | map(select(.slug as $s | ($r|index($s))|not))
    | .[] | "  \($repo): \(.dir) \(.peer) — \(.slug)"' "${srcs[@]}"
}

# inject_block — verbatim from epic-board.sh.
inject_block() {
  local file="$1" begin="$2" end="$3" cf="$4"
  if grep -qF "$begin" "$file"; then
    awk -v b="$begin" -v e="$end" -v cf="$cf" '
      index($0,b){ print; while((getline line < cf)>0) print line; close(cf); skip=1; next }
      index($0,e){ skip=0; print; next }
      !skip { print }
    ' "$file" > "$file.tmp" && mv "$file.tmp" "$file"
  else
    { echo ""; echo "$begin"; cat "$cf"; echo "$end"; } >> "$file"
  fi
}

# discover_children <parent-id> <parent-repo> -> "repo|wid" per child whose log
# declares parent=<parent-id>. parent_project, when recorded, must match the
# parent's repo (guards against a cross-repo id collision). Sorted + de-duped.
discover_children() {
  local parent_id="$1" parent_repo="$2" f wd wid repo belongs
  command -v jq >/dev/null 2>&1 || return 0
  for f in "$ROOT"/docs/work/*/work.jsonl "$ROOT"/repos/*/docs/work/*/work.jsonl; do
    [[ -f "$f" ]] || continue
    belongs="$(jq -rs --arg p "$parent_id" --arg pp "$parent_repo" '
      any(.[]; (.type=="created" or .type=="meta_changed") and (.parent==$p)
               and ((.parent_project // $pp)==$pp))' "$f" 2>/dev/null || echo false)"
    [[ "$belongs" == "true" ]] || continue
    wd="$(dirname "$f")"; wid="$(basename "$wd")"
    repo="$(jq -rs 'map(select(.type=="created"))[0].repo // empty' "$f" 2>/dev/null)"
    if [[ -z "$repo" ]]; then
      if [[ "$wd" == "$ROOT/repos/"* ]]; then repo="${wd#"$ROOT"/repos/}"; repo="${repo%%/*}"; else repo="solution"; fi
    fi
    echo "$repo|$wid"
  done | sort -u
}

# run_resolved_slugs <parent-id> <parent-repo> -> the run-global settled-relay
# slug set (any relay_resolved in any child's work.jsonl) — feeds the A1 rule.
run_resolved_slugs() {
  # A1 verbatim: "a relay is ONE thing keyed by slug; ONE relay_resolved ANYWHERE
  # closes BOTH legs." So the settled set is the RUN-GLOBAL union of every
  # relay_resolved slug across ALL work logs — not just this parent's direct
  # children. This matters at every nesting level:
  #   • a child→PARENT relay (e.g. validation e2e-needs) is resolved on the PARENT's
  #     own log (conductor, direction=inbound); and
  #   • a mid-level node's outbound relay to ITS OWN child is resolved on a
  #     GRANDCHILD's log.
  # A resolved-set scoped to direct children would miss both and render a settled
  # relay as "open" (the parent-side + subtree bugs). Slugs are descriptive and
  # effectively unique per run, so a global union is the faithful, level-agnostic
  # reading of A1. ($1/$2 kept for signature compatibility; the set is global.)
  command -v jq >/dev/null 2>&1 || return 0
  { for f in "$ROOT"/docs/work/*/work.jsonl "$ROOT"/repos/*/docs/work/*/work.jsonl; do
      [[ -f "$f" ]] || continue
      jq -rs '.[] | select(.type=="relay_resolved") | .slug' "$f" 2>/dev/null
    done; } | sort -u | tr '\n' ' '
}

render() {
  local PARENT_DIR PARENT_MD PARENT_ID
  PARENT_DIR="$(resolve_parent "$PARENT_ARG")" || exit 1
  PARENT_MD="$PARENT_DIR/manifest.md"
  PARENT_ID="$(basename "$PARENT_DIR")"
  # The parent's repo is the authoritative one it recorded on its own created event
  # (children declare parent_project=<that repo>). Deriving it from the path breaks
  # when a repo runs its OWN conductor locally (pod = one repo): the parent then sits
  # under $ROOT/docs/work, not $ROOT/repos/*, and a path fallback would wrongly say
  # "solution" — which then fails discover_children's parent_project guard. Read the
  # created-event repo first (same pattern discover_children uses), path only as fallback.
  PARENT_REPO="$(jq -rs 'map(select(.type=="created"))[0].repo // empty' "$PARENT_DIR/work.jsonl" 2>/dev/null)"
  if [[ -z "$PARENT_REPO" ]]; then
    if [[ "$PARENT_DIR" == "$ROOT/repos/"* ]]; then
      PARENT_REPO="${PARENT_DIR#"$ROOT"/repos/}"; PARENT_REPO="${PARENT_REPO%%/*}"
    else
      PARENT_REPO="solution"
    fi
  fi

  # N-level (2026-07-06): a child may itself conduct children. Surface its own
  # parent in the header for context; no guard.
  local OWN_PARENT
  OWN_PARENT="$(jq -rs 'map(select((.type=="created" or .type=="meta_changed") and has("parent")) | .parent) | (last // "")' "$PARENT_DIR/work.jsonl" 2>/dev/null)"

  local PARENT_TITLE=""
  [[ -f "$PARENT_MD" ]] && PARENT_TITLE="$(awk -F': ' '/^# Work Item:/{sub(/^# Work Item: /,""); print; exit}' "$PARENT_MD")"
  local WISH=""
  [[ -f "$PARENT_MD" ]] && WISH="$(awk -F': ' '/^\*\*Wishlist\*\*:/{print $2; exit}' "$PARENT_MD" | sed 's/ *$//')"
  local UPDATED=""
  [[ -f "$PARENT_MD" ]] && UPDATED="$(awk -F': ' '/^\*\*Last Updated\*\*:/{print $2; exit}' "$PARENT_MD" | cut -c1-10)"

  # A1: run-global settled-relay set.
  local RESOLVED; RESOLVED="$(run_resolved_slugs "$PARENT_ID" "$PARENT_REPO")"

  local ROWS; ROWS="$(discover_children "$PARENT_ID" "$PARENT_REPO")"
  if [[ -z "$ROWS" ]]; then
    printf '\033[1m%s\033[0m\n' "PARENT $PARENT_ID"
    [[ -n "$PARENT_TITLE" ]] && printf '  %s\n' "$PARENT_TITLE"
    printf '  no children yet — this item has no child work items declaring parent=%s\n' "$PARENT_ID"
    printf '  scaffold one (conductor-driven, or in the target repo):\n'
    printf '    /work --parent-work %s --parent-project %s "strand prompt"\n' "$PARENT_ID" "$PARENT_REPO"
    return
  fi

  # PRIME/SELF ROW: the parent is also a worker strand (its own repo work). Prepend
  # it as a first-class gating member so it renders first, counts in the barrier min,
  # and gets its own run command — like a child. Guarded to only fire when children
  # exist (above), so a childless standalone item is unaffected.
  ROWS="$PARENT_REPO|$PARENT_ID"$'\n'"$ROWS"

  # gather per-child state (escalated children are OUT OF PLAY: excluded from
  # the barrier min, block completion, and get no run-in-each-repo command)
  local base=99 total_open=0 total_esc=0
  local -a R_repo R_wid R_phase R_done R_in R_out R_state R_esc R_prime
  local i=0
  while IFS='|' read -r repo wid; do
    [[ -n "$repo" ]] || continue
    local wd; wd="$(work_dir "$repo" "$wid")"
    local donejson=""
    [[ -f "$wd/work.jsonl" ]] && \
      donejson="$(jq -rs 'map(select(.type=="phase_done")|.phase)|(last // "")' "$wd/work.jsonl" 2>/dev/null)"
    local doneword="${donejson:-—}"
    local dord; dord="$(phase_ord "$doneword")"
    local esc="false"
    [[ -f "$wd/work.jsonl" ]] && esc="$(jq -rs '
      map(select(.type=="status_changed" or .type=="escalated")
          | (if .type=="escalated" then "escalated" else .to end))
      | ((last // "") == "escalated")' "$wd/work.jsonl" 2>/dev/null)"
    local oin oout; read -r oin oout < <(relay_counts "$wd" "$RESOLVED")
    R_repo[$i]="$repo"; R_wid[$i]="$wid"; R_phase[$i]="$doneword"; R_done[$i]="$dord"
    R_in[$i]="$oin"; R_out[$i]="$oout"; R_esc[$i]="$esc"
    if [[ "$repo" == "$PARENT_REPO" && "$wid" == "$PARENT_ID" ]]; then R_prime[$i]=1; else R_prime[$i]=0; fi
    if [[ "$esc" == "true" ]]; then
      total_esc=$(( total_esc + 1 ))
    else
      (( dord < base )) && base=$dord
    fi
    total_open=$(( total_open + oin + oout ))
    i=$(( i+1 ))
  done <<< "$ROWS"
  local n=$i
  (( base==99 )) && base=0
  local target=$(( base + 1 )); (( target > 5 )) && target=5

  # ---- unambiguous barrier label (semantics verbatim from epic-board.sh) ----
  local BARRIER_SHORT BARRIER_LONG
  if (( base >= 4 && total_open == 0 && total_esc == 0 )); then
    BARRIER_SHORT="complete"; BARRIER_LONG="complete"
  elif (( total_open > 0 )); then
    local _hp; _hp="$(phase_name $(( target > 4 ? 4 : target )))"
    BARRIER_SHORT="$_hp (HELD)"
    BARRIER_LONG="$_hp — HELD ($total_open open relay(s); resolve them before any repo starts $_hp)"
  elif (( base >= 4 && total_esc > 0 )); then
    BARRIER_SHORT="validation (HELD)"
    BARRIER_LONG="validation — HELD ($total_esc escalated child(ren) need a human decision before completion)"
  else
    local _tp; _tp="$(phase_name $target)"
    BARRIER_SHORT="$_tp (OPEN)"
    if (( base == 0 )); then
      BARRIER_LONG="$_tp — OPEN (kickoff — every repo may run $_tp now)"
    else
      BARRIER_LONG="$_tp — OPEN (all repos settled $(phase_name $base), zero open relays — every repo may run $_tp now)"
    fi
  fi

  # per-child state label
  for ((j=0;j<n;j++)); do
    if [[ "${R_esc[$j]}" == "true" ]]; then R_state[$j]="🚨 ESCALATED — needs a human decision (out of play)"
    elif (( ${R_in[$j]} > 0 )); then R_state[$j]="🟢 ACT — ${R_in[$j]} inbound ask(s); answer + reply"
    elif (( ${R_out[$j]} > 0 )); then R_state[$j]="⏳ BLOCKED — ${R_out[$j]} reply pending (run /conduct sync)"
    elif (( ${R_done[$j]} >= 4 )); then R_state[$j]="✅ complete (validated)"
    elif (( ${R_done[$j]} == base )); then R_state[$j]="🔵 WORKING — owes $(phase_name $target)"
    else R_state[$j]="✅ settled @ $(phase_name ${R_done[$j]}) (at barrier)"; fi
    [[ "${R_prime[$j]}" == "1" ]] && R_state[$j]="★ ${R_state[$j]}"
  done

  # ---- header ----
  printf '\033[1m%s\033[0m\n' "PARENT $PARENT_ID  ($PARENT_REPO)"
  [[ -n "$PARENT_TITLE" ]] && printf '  %s\n' "$PARENT_TITLE"
  [[ -n "$OWN_PARENT" ]] && printf '  ↑ itself a child of: %s\n' "$OWN_PARENT"
  printf '  barrier: \033[1m%s\033[0m   ·   wishlist: %s   ·   last updated: %s\n' "$BARRIER_SHORT" "${WISH:-—}" "${UPDATED:---}"
  printf '%s\n' "────────────────────────────────────────────────────────────────────────────"
  printf '%-14s %-34s %-14s %s\n' "REPO" "WORK ITEM" "DONE" "STATE"
  for ((j=0;j<n;j++)); do
    printf '%-14s %-34s %-14s %s\n' "${R_repo[$j]}" "${R_wid[$j]:0:34}" "${R_phase[$j]}" "${R_state[$j]}"
  done
  printf '%s\n' "────────────────────────────────────────────────────────────────────────────"
  printf '\033[2m%s\033[0m\n' "★ = prime/conductor's own strand (this parent's repo work; gates the barrier like a child)"

  # ---- NEXT ----
  local NEXT="" STATEMSG=""
  if (( total_esc > 0 )); then
    STATEMSG="🚨 $total_esc escalated child(ren) — human decision required (they are out of play; the run cannot complete)"
  fi
  if (( total_open > 0 )); then
    [[ -n "$STATEMSG" ]] && printf '%s\n' "$STATEMSG"
    STATEMSG="⏳ Settling $(phase_name $target) — relays in flight (barrier held)"
    local nact=0
    for ((j=0;j<n;j++)); do [[ "${R_esc[$j]}" != "true" ]] && (( ${R_in[$j]} > 0 )) && nact=$(( nact + 1 )); done
    if (( nact > 0 )); then
      NEXT="👉 act in the $nact 🟢 ACT repo(s) — see 'run in each repo' below; then /conduct sync at the $PARENT_REPO root."
    else
      NEXT="👉 run  /conduct $PARENT_ID sync  to deliver $total_open pending relay(s)."
    fi
  elif (( base >= 4 && total_esc == 0 )); then
    STATEMSG="✅ All children settled validation — run complete."
    if [[ -n "$OWN_PARENT" ]]; then
      NEXT="🎉 Children complete — this node may now settle its own parent-facing validation: scripts/wlog.sh docs/work/$PARENT_ID phase_done phase=validation (settling rule satisfied), then status_changed to=completed."
    else
      NEXT="🎉 Run complete — set the parent's status: scripts/wlog.sh docs/work/$PARENT_ID status_changed to=completed (and advance the roadmap/wishlist if linked)."
    fi
  elif (( base >= 4 )); then
    NEXT="👉 decide the escalated child(ren): resume (status_changed) or cancel — then the run can complete."
  else
    [[ -n "$STATEMSG" ]] && printf '%s\n' "$STATEMSG"
    STATEMSG="▶ $(phase_name $base) settled — barrier OPEN, advance to $(phase_name $target)"
    local laggards=""
    for ((j=0;j<n;j++)); do [[ "${R_esc[$j]}" != "true" ]] && (( ${R_done[$j]} == base )) && laggards+="${R_repo[$j]} "; done
    NEXT="👉 advance ${laggards% } to $(phase_name $target) — see 'run in each repo' below."
  fi
  printf '%s\n' "$STATEMSG"
  printf '\033[1m%s\033[0m\n' "$NEXT"

  # ---- open relay detail ----
  if (( total_open > 0 )); then
    printf '%s\n' "open relays:"
    for ((j=0;j<n;j++)); do
      local wd; wd="$(work_dir "${R_repo[$j]}" "${R_wid[$j]}")"
      relay_list "$wd" "${R_repo[$j]}" "$RESOLVED"
    done
  fi

  # ---- per-repo action commands (escalated children intentionally omitted) --
  local any_act=0 printed_hdr=0
  for ((j=0;j<n;j++)); do
    [[ "${R_esc[$j]}" == "true" ]] && continue
    local owed=0
    if   (( ${R_in[$j]} > 0 )); then owed=$target
    elif (( ${R_out[$j]} == 0 && ${R_done[$j]} < target && ${R_done[$j]} < 4 )); then owed=$target
    else continue; fi
    if (( printed_hdr == 0 )); then
      printf '%s\n' "run in each repo (open a Claude session in the dir, then run the command):"
      printed_hdr=1
    fi
    local cwd cmd; cwd="$(repo_cwd "${R_repo[$j]}")"; cmd="$(phase_slash "$owed" "${R_wid[$j]}" "${R_repo[$j]}")"
    printf '  %-13s cd %-22s → in Claude:  %s\n' "${R_repo[$j]}" "$cwd" "$cmd"
    if (( ${R_in[$j]} > 0 )); then
      any_act=1
      local wd slugs s; wd="$(work_dir "${R_repo[$j]}" "${R_wid[$j]}")"; slugs="$(open_inbound_slugs "$wd" "$RESOLVED")"
      for s in $slugs; do
        printf '  %-13s   then close inbound: scripts/wlog.sh docs/work/%s relay_resolved direction=inbound slug=%s && scripts/wrender.sh docs/work/%s\n' \
          "" "${R_wid[$j]}" "$s" "${R_wid[$j]}"
      done
    fi
  done
  if (( any_act == 1 )); then
    printf '%s\n' "then, back at the $PARENT_REPO root:  /conduct $PARENT_ID sync   (delivers replies, re-derives the barrier)"
  fi

  # ---- --write: inject the derived board into the PARENT manifest ----
  if (( WRITE )); then
    local bf; bf="$(mktemp)"
    {
      printf '**Barrier Phase**: %s\n\n' "$BARRIER_LONG"
      printf '| Repo | Work Item | Phase | Status |\n'
      printf '|------|-----------|-------|--------|\n'
      for ((j=0;j<n;j++)); do
        printf '| %s | %s | %s | %s |\n' "${R_repo[$j]}" "${R_wid[$j]}" "${R_phase[$j]}" "${R_state[$j]}"
      done
    } > "$bf"
    inject_block "$PARENT_MD" "<!-- BEGIN BOARD -->" "<!-- END BOARD -->" "$bf"
    rm -f "$bf"
    printf '\033[2m%s\033[0m\n' "ⓘ wrote derived board into $PARENT_MD (between BOARD anchors; wrender.sh preserves it)."
  else
    printf '\033[2m%s\033[0m\n' "ⓘ read-only snapshot. Run with --write to refresh the parent manifest board, or /conduct $PARENT_ID board (Claude) to sync-then-show."
  fi
}

PARENT_REPO="solution"   # set properly inside render(); default for phase_slash

if (( WATCH )); then
  while true; do clear; render; printf '\n(refreshing every 5s · Ctrl-C to exit)\n'; sleep 5; done
else
  render
fi
