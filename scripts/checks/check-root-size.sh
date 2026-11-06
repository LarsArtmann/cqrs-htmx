#!/usr/bin/env bash
# check-root-size.sh — measure the ROOT PACKAGE (top-level *.go, production only)
# and ratchet it against a committed baseline.
#
# WHY: docs kept quoting the root package's size by hand and drifted — the
# 2026-11-06 re-export plan and `TODO_LIST.md` both claimed "53 production files
# / 7,594 LOC" while the tree measured 54 / 8,547 (the hand sum had dropped a
# file and counted a subset of lines). A number nobody recomputes is a
# split-brain waiting to happen, so the number gets ONE source: this gate.
#
# The ratchet direction is DOWNWARD: the root package is being burnt down
# (ADR-0062 aliases, extractable packages). Growth is allowed, but only as a
# deliberate act: re-run with --update-baseline in the same commit, which makes
# the growth visible in review instead of drifting in prose.
#
# Usage: check-root-size.sh [--print | --advisory | --update-baseline]
#   --print            print only the canonical sentence (docs quote this)
#   --advisory         never fail on growth (prints WARNING, exit 0)
#   --update-baseline  rewrite the baseline to the measured value, exit 0
#
# Env: ROOT_SIZE_ROOT / ROOT_SIZE_BASELINE  override paths (fixture self-test)
# Exit: 0 within/under the ratchet · 1 growth · 2 the measurement or baseline
#       itself failed (never a false green)
set -uo pipefail

MODE=check
case "${1:-}" in
'') ;;
--print) MODE=print ;;
--advisory) MODE=advisory ;;
--update-baseline) MODE=update ;;
*)
  echo "usage: $(basename "$0") [--print|--advisory|--update-baseline]" >&2
  exit 2
  ;;
esac

ROOT="${ROOT_SIZE_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
BASELINE="${ROOT_SIZE_BASELINE:-$ROOT/scripts/checks/root-size.baseline}"

if [ ! -d "$ROOT" ]; then
  echo "✗ root-size: $ROOT is not a directory (rc=2)" >&2
  exit 2
fi

files_tmp="$(mktemp /tmp/cqrs-htmx-root-size-XXXXXX)"
trap 'rm -f "$files_tmp"' EXIT

find "$ROOT" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' ! -name '*_templ.go' \
  | LC_ALL=C sort >"$files_tmp"

files="$(wc -l <"$files_tmp" | tr -d ' ')"
if [ "$files" -eq 0 ]; then
  echo "✗ root-size: 0 production .go files under $ROOT — the measurement lost its target (rc=2)" >&2
  exit 2
fi

# shellcheck disable=SC2046 # one path per line, no whitespace in these names
loc="$(xargs wc -l <"$files_tmp" | awk 'END {print $1}')"
# xargs prints a "total" line last; for a single file there is no total line.
if [ -z "$loc" ]; then
  loc="$(wc -l <"$files_tmp" | tr -d ' ')"
fi
loc="$(printf '%s' "$loc" | tr -d ' ')"
case "$loc" in
'' | *[!0-9]*)
  echo "✗ root-size: could not measure line count (got '$loc') (rc=2)" >&2
  exit 2
  ;;
esac

# Canonical sentence: this is the ONLY place the number is produced.
sentence="$files production files / $(printf '%s' "$loc" | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta') LOC"

if [ "$MODE" = print ]; then
  printf '%s\n' "$sentence"
  exit 0
fi

if [ "$MODE" = update ]; then
  printf '%s %s\n' "$files" "$loc" >"$BASELINE"
  echo "✓ root-size: baseline updated to $sentence ($BASELINE)"
  exit 0
fi

if [ ! -f "$BASELINE" ]; then
  echo "✗ root-size: no baseline at $BASELINE — nothing to ratchet against (rc=2)" >&2
  echo "  Create it once with: $0 --update-baseline" >&2
  exit 2
fi

read -r bfiles bloc extra <"$BASELINE" || true
case "${bfiles:-}" in
'' | *[!0-9]*)
  echo "✗ root-size: baseline $BASELINE must start with an integer file count (got '${bfiles:-}') (rc=2)" >&2
  exit 2
  ;;
esac
case "${bloc:-}" in
'' | *[!0-9]*)
  echo "✗ root-size: baseline $BASELINE must carry an integer LOC count on the same line (got '${bloc:-}') (rc=2)" >&2
  exit 2
  ;;
esac
if [ -n "${extra:-}" ]; then
  echo "✗ root-size: baseline $BASELINE must be 'files loc' and nothing else (found trailing '$extra') (rc=2)" >&2
  exit 2
fi

grown=0
[ "$files" -gt "$bfiles" ] && grown=1
[ "$loc" -gt "$bloc" ] && grown=1

if [ "$grown" -eq 0 ]; then
  echo "✓ root-size: $sentence (baseline $bfiles files / $bloc LOC)"
  if [ "$loc" -lt "$bloc" ] || [ "$files" -lt "$bfiles" ]; then
    echo "  ↓ root burndown: lower the baseline to '$files $loc' in $BASELINE in this commit."
  fi
  exit 0
fi

msg="✗ root-size: the ROOT PACKAGE grew — measured $sentence vs baseline $bfiles files / $bloc LOC"
if [ "$MODE" = advisory ]; then
  echo "WARNING: $msg" >&2
  echo "  (advisory mode: exit 0)" >&2
  exit 0
fi

echo "$msg" >&2
echo "  The root package is meant to burn DOWN (ADR-0062). If the growth is" >&2
echo "  deliberate, re-run with --update-baseline in the SAME commit and say why." >&2
echo "  If it is not, extract the code into a subpackage instead of growing root." >&2
exit 1
