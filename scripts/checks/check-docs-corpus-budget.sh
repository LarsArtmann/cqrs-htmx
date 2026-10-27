#!/usr/bin/env bash
# check-docs-corpus-budget.sh — ratchet on the size of the whole docs corpus.
#
# The tail ratchet guards ONE symptom (a growing live status tail). This gate
# guards the thing the 2026-10-24 audit actually measured: the docs tree itself.
# 806 markdown files / ~149k lines, 43% of it closed history, sit inside every
# grep and every agent's context window. Nothing stopped it from growing except
# attention.
#
# A ratchet, not a ceiling: `docs/CORPUS_BASELINE` holds the measured line
# count, and the gate fails when the corpus GROWS past it. Deliberate growth is
# fine but must be visible: bump the baseline in the same commit that adds the
# content, exactly like the status-tail ratchet.
#
# Usage:
#   bash scripts/checks/check-docs-corpus-budget.sh              # report (advisory)
#   bash scripts/checks/check-docs-corpus-budget.sh --ratchet     # blocking (CI)
#   bash scripts/checks/check-docs-corpus-budget.sh --update-baseline
#   bash scripts/checks/check-docs-corpus-budget.sh --files       # per-directory table
#
# Env: CORPUS_BUDGET_ROOT  override the repo root (fixture self-test).
# Exit: 0 ok / advisory, 1 ratchet growth or missing/malformed baseline, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${CORPUS_BUDGET_ROOT:-$REPO_ROOT}"

RATCHET=0
UPDATE=0
FILES=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --ratchet) RATCHET=1 ;;
  --update-baseline) UPDATE=1 ;;
  --files) FILES=1 ;;
  *)
    echo "usage: check-docs-corpus-budget.sh [--ratchet] [--update-baseline] [--files]" >&2
    exit 2
    ;;
  esac
  shift
done

docs_dir="$ROOT/docs"
if [ ! -d "$docs_dir" ]; then
  echo "✗ corpus-budget: $docs_dir not found"
  exit 1
fi

mapfile -t md_files < <(find "$docs_dir" -name '*.md' -type f | sort)
files="${#md_files[@]}"
lines=0
if [ "$files" -gt 0 ]; then
  lines=$(printf '%s\0' "${md_files[@]}" | xargs -0 cat | wc -l | tr -d '[:space:]')
fi

baseline_file="$ROOT/docs/CORPUS_BASELINE"

if [ "$UPDATE" -eq 1 ]; then
  printf '%s\n' "$lines" >"$baseline_file"
  echo "✓ corpus-budget: baseline written: $lines lines across $files file(s)"
  exit 0
fi

echo "corpus-budget: $lines line(s) across $files markdown file(s) under docs/"

if [ "$FILES" -eq 1 ]; then
  find "$docs_dir" -name '*.md' -type f -printf '%h\n' | sort | uniq -c | sort -rn | sed 's/^/  /'
fi

if [ "$RATCHET" -eq 0 ]; then
  exit 0
fi

if [ ! -f "$baseline_file" ]; then
  echo "✗ corpus-budget: --ratchet needs $baseline_file (one integer: the highest allowed line count)" >&2
  exit 1
fi
baseline=$(tr -d '[:space:]' <"$baseline_file")
case "$baseline" in
'' | *[!0-9]*)
  echo "✗ corpus-budget: $baseline_file must contain a single integer (got '$baseline')" >&2
  exit 1
  ;;
esac

if [ "$lines" -gt "$baseline" ]; then
  echo "✗ corpus-budget: $lines lines exceed the baseline of $baseline by $((lines - baseline)) — shrink the corpus or justify the growth by bumping docs/CORPUS_BASELINE in this commit"
  exit 1
fi
if [ "$lines" -lt "$baseline" ]; then
  echo "⚠ corpus-budget: $lines lines is BELOW the baseline of $baseline — lower docs/CORPUS_BASELINE to $lines in this commit"
fi
echo "✓ corpus-budget: within the ratchet (baseline $baseline)"
exit 0
