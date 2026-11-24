#!/usr/bin/env bash
# check-debug-files.sh — refuse a COMMITTED zz_* debug/scratch file.
#
# WHY: the zz_ prefix is the ad-hoc "sort me last" scratch naming an agent or
# human reaches for while debugging, and the auto-commit daemon sweeps the
# whole tree continuously — a scratch file that lives longer than one edit
# cycle gets committed without any human noticing. One was observed mid-session
# in the adminui next-wave (2026-11-24). A committed zz_* file is never a
# valid state: if the content matters it should be named for what it is, and
# if it does not it belongs in /tmp, not in history.
#
# Scope: every git-tracked file in the working tree — or, with --staged, every
# file in the INDEX (the exact content a commit would carry). Matched on the
# BASENAME so a nested zz_ scratch dies the same way; the pattern is anchored
# to the whole basename (zz_*) so legitimate names merely CONTAINING zz pass.
#
# Usage: check-debug-files.sh [--staged]
#   --staged  scan the index (pre-commit hook: catches a file staged and then
#             edited around, and does not block on untracked scratch)
#
# Env: DEBUG_FILES_ROOT  override the scan root (fixture self-test)
# Exit: 0 none found · 1 found · 2 the scan itself failed (never green)
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

ROOT="${DEBUG_FILES_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

if [ ! -d "$ROOT/.git" ]; then
  echo "✗ debug-files: $ROOT is not a git working tree (rc=2)" >&2
  exit 2
fi

if [ "$MODE" = index ]; then
  hits="$(git -C "$ROOT" ls-files --cached | grep -E '(^|/)zz[^/]*$' || true)"
else
  hits="$(git -C "$ROOT" ls-files | grep -E '(^|/)zz[^/]*$' || true)"
fi

if [ -n "$hits" ]; then
  count="$(printf '%s\n' "$hits" | wc -l | tr -d ' ')"
  echo "✗ debug-files: $count zz_* scratch file(s) tracked:" >&2
  printf '%s\n' "$hits" | sed 's/^/    /' >&2
  echo "  A committed zz_* scratch file is never valid — rename it for what it" >&2
  echo "  is (and keep it in the same commit), or delete it; /tmp is where debug" >&2
  echo "  output lives." >&2
  exit 1
fi

if [ "$MODE" = index ]; then
  scanned="$(git -C "$ROOT" ls-files --cached | wc -l | tr -d ' ')"
  echo "✓ debug-files: no zz_* scratch files in $scanned staged file(s)"
else
  scanned="$(git -C "$ROOT" ls-files | wc -l | tr -d ' ')"
  echo "✓ debug-files: no zz_* scratch files in $scanned tracked file(s)"
fi
