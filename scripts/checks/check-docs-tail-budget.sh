#!/usr/bin/env bash
# check-docs-tail-budget.sh — advisory gate on the live status-report tail.
#
# Convention (docs/status/README.md): the non-archived `docs/status/` tail is
# kept small (the docs-health sweep archives resolved reports). A growing tail
# is drift — the 2026-09 round-13/14 passes found the tail at 22 and re-found
# stale counts twice in one day (the d4 class). This gate makes the tail size a
# visible number at sweep boundary.
#
# ADVISORY by default: prints a warning and exits 0 (the tail legitimately
# holds a report or two mid-session; blocking on it would red builds for
# ordinary docs work). --strict exits 1 instead, for a manual/train-time check.
#
# Counts only non-recursive `docs/status/*.md`, excluding README.md (the
# convention doc, not a report).
#
# Usage: bash scripts/checks/check-docs-tail-budget.sh [--strict] [--budget N]
# Env:   TAIL_BUDGET_ROOT  override the scan root (fixture self-test).
# Exit:  0 always (advisory) unless --strict and the budget is exceeded.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${TAIL_BUDGET_ROOT:-$REPO_ROOT}"

BUDGET=3
STRICT=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --strict) STRICT=1 ;;
  --budget)
    shift
    BUDGET="${1:-3}"
    ;;
  *)
    echo "usage: check-docs-tail-budget.sh [--strict] [--budget N]" >&2
    exit 2
    ;;
  esac
  shift
done

status_dir="$ROOT/docs/status"
if [ ! -d "$status_dir" ]; then
  echo "✗ tail-budget: $status_dir not found"
  exit 1
fi

mapfile -t reports < <(find "$status_dir" -maxdepth 1 -name '*.md' ! -name 'README.md' | sed 's|.*/||' | sort)
count="${#reports[@]}"

echo "tail-budget: $count live report(s) in docs/status/ (budget $BUDGET)"
for r in "${reports[@]}"; do
  echo "  - $r"
done

if [ "$count" -le "$BUDGET" ]; then
  echo "✓ tail-budget: within budget"
  exit 0
fi

echo "⚠ tail-budget: $count reports exceed the budget of $BUDGET — run the docs-health ARCHIVE pass to move resolved reports to docs/status/archived/"
if [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
