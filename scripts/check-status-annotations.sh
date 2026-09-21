#!/usr/bin/env bash
# check-status-annotations.sh — presence gate for archived status-report annotations.
#
# Policy (documented in docs/status/README.md):
#   * Reports dated on or after ANNOTATION_EPOCH (2026-09-09) MUST carry inline
#     strikethrough (`~~`) for every resolved item AND a dated `> ANNOTATED`
#     blockquote. This is the current convention, established when the
#     docs-health ANNOTATE sweep started.
#   * Older reports are LEGACY-EXEMPT. They predate the inline convention and
#     use one of two earlier dialects: a prose `> ANNOTATED ...` blockquote
#     (2026-06 -> 2026-09-07) or no annotation at all (2026-05 -> 2026-08).
#     They are historically complete; re-annotating them would be busywork.
#
# Deliberately-mixed tables (struck rows = done, unstruck rows = open) are
# first-class and never flagged here; the row-integrity check lives in the
# docs-health skill's check-rows.py and is run manually per audit.
#
# Usage: bash scripts/check-status-annotations.sh [dir]
# Exit: 0 = every gated report annotated; 1 = at least one missing marker.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

ANNOTATION_EPOCH="2026-09-09"
DIR="${1:-docs/status/archived}"

if [ ! -d "$DIR" ]; then
  echo "✗ annotation gate: directory not found: $DIR"
  exit 1
fi

total=0
gated=0
legacy=0
failed=0

for f in "$DIR"/*.md; do
  [ -e "$f" ] || continue
  total=$((total + 1))

  base="$(basename "$f")"
  date="$(printf '%s' "$base" | grep -oE '^[0-9]{4}-[0-9]{2}-[0-9]{2}')" || date=""

  if [ -z "$date" ]; then
    echo "  ✗ NO-DATE: $base (cannot classify; expected YYYY-MM-DD prefix)"
    failed=$((failed + 1))
    continue
  fi

  numeric="$(printf '%s' "$date" | tr -d '-')"
  if [ "$numeric" -lt "${ANNOTATION_EPOCH//-/}" ]; then
    legacy=$((legacy + 1))
    continue
  fi

  gated=$((gated + 1))

  if ! grep -q '~~' "$f"; then
    echo "  ✗ MISSING STRIKETHROUGH: $base (dated $date >= $ANNOTATION_EPOCH)"
    failed=$((failed + 1))
    continue
  fi

  if ! grep -qi 'ANNOTATED' "$f"; then
    echo "  ✗ MISSING ANNOTATED BLOCKQUOTE: $base (dated $date >= $ANNOTATION_EPOCH)"
    failed=$((failed + 1))
  fi
done

echo ""
if [ "$failed" -gt 0 ]; then
  echo "✗ annotation gate: $failed of $gated gated report(s) incomplete (scan: $total files, $legacy legacy-exempt)"
  exit 1
fi

echo "✓ annotation gate: $gated gated report(s) annotated ($legacy legacy-exempt, $total scanned)"
exit 0
