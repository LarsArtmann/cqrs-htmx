#!/usr/bin/env bash
# check-css-bundles.sh — sanity gate for the committed Tailwind bundles
#
# The committed bundles (adminui/assets/admin-tw.css,
# dashboardui/assets/dashboard-tw.css) must always be the canonical
# MINIFIED builder output (nix run .#build-adminui-css /
# .#build-dashboardui-css). Two observed corruption classes:
#   1. A buildflow pre-commit/tailwind step rewriting the bundle WITHOUT
#      --minify (pretty multi-line form; class sets identical).
#   2. The 2026-09-22 ~13:05 incident: the git index briefly held a
#      near-empty ~8KB bundle (suspected empty content scan) while HEAD
#      had the full form — a family bump can re-trigger the class.
#
# Checks per bundle: exists, exactly 1 line (minified), tailwind banner
# header, byte floor (catches the near-empty class), and canary utility
# selectors that must survive every legitimate rebuild.
#
# Usage: scripts/check-css-bundles.sh
# Env (TEST HOOK): CHECK_CSS_BUNDLES_ROOT=<dir>  scan a fixture tree
#                  instead of the repo root.
# Exit: 0 = all bundles canonical; 1 = corruption-shaped bundle found

set -uo pipefail

REPO_ROOT="${CHECK_CSS_BUNDLES_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# Byte floor per bundle: generous absolute minimum. Legitimate rebuilds
# move this by a few KB; the corruption class lands 10x lower. If an
# intentional rebuild shrinks a bundle below the floor, re-pin here in
# the same change (with the justification in the commit message).
MIN_BYTES=30000

# Canary utility families emitted by the templ-components scan sources —
# present in every healthy bundle regardless of consumer class set.
CANARIES=(
  "bg-green-100"
  "text-gray-500"
  "border-gray-200"
  "--tc-sidebar"
)

BUNDLES=(
  "adminui/assets/admin-tw.css"
  "dashboardui/assets/dashboard-tw.css"
)

failures=0

for rel in "${BUNDLES[@]}"; do
  path="$REPO_ROOT/$rel"

  if [ ! -f "$path" ]; then
    echo "✗ $rel: MISSING (expected the committed minified bundle)"
    failures=$((failures + 1))
    continue
  fi

  lines=$(wc -l <"$path")
  if [ "$lines" -ne 1 ]; then
    echo "✗ $rel: $lines lines — not the canonical minified form (1 line)."
    echo "       A hook probably rewrote it without --minify."
    failures=$((failures + 1))
    continue
  fi

  if ! head -c 24 "$path" | grep -q '^/\*! tailwindcss v'; then
    echo "✗ $rel: missing the tailwindcss banner header — wrong producer."
    failures=$((failures + 1))
    continue
  fi

  bytes=$(wc -c <"$path")
  if [ "$bytes" -lt "$MIN_BYTES" ]; then
    echo "✗ $rel: $bytes bytes < floor $MIN_BYTES — near-empty bundle (empty content scan class)."
    failures=$((failures + 1))
    continue
  fi

  missing=""
  for canary in "${CANARIES[@]}"; do
    grep -q -- "$canary" "$path" || missing="$missing $canary"
  done
  if [ -n "$missing" ]; then
    echo "✗ $rel: canary utilities absent:$missing"
    echo "       The scan sources were probably empty (module not resolved)."
    failures=$((failures + 1))
    continue
  fi

  echo "✓ $rel: minified, $bytes bytes, canaries present"
done

echo ""

if [ "$failures" -gt 0 ]; then
  echo "✗ CSS bundle sanity check FAILED ($failures bundle(s))"
  echo "  FIX: re-run the canonical builders and commit the minified output:"
  echo "    nix run .#build-adminui-css     # adminui/assets/admin-tw.css"
  echo "    nix run .#build-dashboardui-css # dashboardui/assets/dashboard-tw.css"
  echo "  If a rebuild legitimately changes size below the floor, re-pin"
  echo "  MIN_BYTES in scripts/check-css-bundles.sh in the same change."
  exit 1
fi

echo "✓ CSS bundle sanity check PASSED (2 bundles canonical)"
