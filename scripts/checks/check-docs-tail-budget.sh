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
# ordinary docs work). --strict exits 1 against the absolute budget, for a
# manual/train-time check.
#
# --ratchet is the BLOCKING mode used by check-modules + CI: it fails when the
# tail GROWS past the recorded baseline in docs/status/TAIL_BASELINE, and warns
# when the tail shrinks (lower the baseline in the same commit that archives).
# A ratchet prevents regrowth immediately, without pretending the corpus is
# already at its target — the absolute budget stays the goal, the baseline is
# the floor of the descent.
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
RATCHET=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --strict) STRICT=1 ;;
  --ratchet) RATCHET=1 ;;
  --budget)
    shift
    BUDGET="${1:-3}"
    ;;
  *)
    echo "usage: check-docs-tail-budget.sh [--strict] [--ratchet] [--budget N]" >&2
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

BASELINE_FILE="$ROOT/docs/status/TAIL_BASELINE"
baseline=""
if [ "$RATCHET" -eq 1 ]; then
  if [ ! -f "$BASELINE_FILE" ]; then
    echo "✗ tail-budget: --ratchet needs $BASELINE_FILE (one integer: the highest allowed count)" >&2
    exit 1
  fi
  baseline=$(tr -d '[:space:]' <"$BASELINE_FILE")
  case "$baseline" in
  '' | *[!0-9]*)
    echo "✗ tail-budget: $BASELINE_FILE must contain a single integer (got '$baseline')" >&2
    exit 1
    ;;
  esac
  echo "tail-budget: $count live report(s) in docs/status/ (budget $BUDGET, ratchet baseline $baseline)"
else
  echo "tail-budget: $count live report(s) in docs/status/ (budget $BUDGET)"
fi
for r in "${reports[@]}"; do
  echo "  - $r"
done

if [ "$RATCHET" -eq 1 ]; then
  if [ "$count" -gt "$baseline" ]; then
    echo "✗ tail-budget: $count reports exceed the ratchet baseline of $baseline — archive resolved reports before adding new ones"
    exit 1
  fi
  if [ "$count" -lt "$baseline" ]; then
    echo "⚠ tail-budget: $count reports is BELOW the baseline of $baseline — lower docs/status/TAIL_BASELINE to $count in this commit"
  fi
  if [ "$count" -le "$BUDGET" ]; then
    echo "✓ tail-budget: within budget and ratchet"
    exit 0
  fi
  echo "✓ tail-budget: within the ratchet (target budget still $BUDGET)"
  exit 0
fi

if [ "$count" -le "$BUDGET" ]; then
  echo "✓ tail-budget: within budget"
  exit 0
fi

echo "⚠ tail-budget: $count reports exceed the budget of $BUDGET — run the docs-health ARCHIVE pass to move resolved reports to docs/status/archived/"
if [ "$STRICT" -eq 1 ]; then
  exit 1
fi
exit 0
