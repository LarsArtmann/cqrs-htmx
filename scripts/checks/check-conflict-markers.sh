#!/usr/bin/env bash
# check-conflict-markers.sh — refuse a COMMITTED merge-conflict marker.
#
# WHY: the auto-commit daemon has twice committed a stash-pop conflict into a
# one-number gate file — `docs/CORPUS_BASELINE` in `158e86f6` (2026-10-30) and
# again in `24b653d6` (2026-11-06). Both times the corpus ratchet then refused
# the file outright ("must contain a single integer"), i.e. a blocking gate was
# red for every committer, and both times it was found by accident, by hand.
# A conflicted WORKING TREE is normal mid-merge; a committed marker is never a
# valid state, so the invariant belongs in a gate.
#
# Scope: every git-tracked file in the working tree — or, with --staged, every
# file in the INDEX (the exact content a commit would carry). Markers are
# matched ANCHORED at line start, because a real marker always starts its line —
# a doc that quotes one inside prose or a code span stays green.
#
# Usage: check-conflict-markers.sh [--staged]
#   --staged  scan the index (pre-commit hook: catches a marker that was staged
#             and then edited out of the working tree, and does not block on a
#             marker the commit would not carry)
#
# Env: CONFLICT_MARKERS_ROOT  override the scan root (fixture self-test)
# Exit: 0 no markers · 1 markers found · 2 the scan itself failed (never green)
set -uo pipefail

MODE=tree
case "${1:-}" in
'') ;;
--staged) MODE=index ;;
*)
  echo "usage: $(basename "$0") [--staged]" >&2
  exit 2
  ;;
esac

ROOT="${CONFLICT_MARKERS_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

if [ ! -d "$ROOT/.git" ]; then
  echo "✗ conflict-markers: $ROOT is not a git working tree (rc=2)" >&2
  exit 2
fi

grep_args=(-n -I -E '^(<{7}( |$)|={7}$|>{7}( |$))' -- .)
if [ "$MODE" = index ]; then
  grep_args=(--cached "${grep_args[@]}")
fi

output="$(git -C "$ROOT" grep "${grep_args[@]}" 2>&1)"
rc=$?

case "$rc" in
0)
  count="$(printf '%s\n' "$output" | wc -l | tr -d ' ')"
  echo "✗ conflict-markers: $count committed conflict-marker line(s):" >&2
  printf '%s\n' "$output" | sed 's/^/    /' >&2
  echo "  A committed conflict marker is never valid — resolve the file and re-commit." >&2
  echo "  If a doc must quote a marker, keep it inside prose or a code span (this check is line-start anchored)." >&2
  exit 1
  ;;
1)
  if [ "$MODE" = index ]; then
    scanned="$(git -C "$ROOT" ls-files --cached | wc -l | tr -d ' ')"
    echo "✓ conflict-markers: no markers in $scanned staged file(s)"
  else
    scanned="$(git -C "$ROOT" ls-files | wc -l | tr -d ' ')"
    echo "✓ conflict-markers: no committed markers in $scanned tracked file(s)"
  fi
  exit 0
  ;;
*)
  echo "✗ conflict-markers: git grep refused (rc=$rc) — the scan verified nothing:" >&2
  printf '%s\n' "$output" | head -5 >&2
  exit 2
  ;;
esac
