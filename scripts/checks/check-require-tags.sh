#!/usr/bin/env bash
# check-require-tags.sh — keep every internal require PUBLISHABLE.
#
# Two failure classes, one gate:
#   1. Zero pseudo-versions (v0.0.0-00010101...) in any go.mod.
#   2. Tag existence for internal requires — the blind spot that let
#      examples/admin-demo require usermgmt/totp/v4 v4.8.0 (never
#      published) on 2026-08-17 and poison the whole workspace module
#      graph for 12 days: workspace mode masks unpublished requires, so
#      only hermetic builds or this gate see them. The strict drift check
#      is what enforces it; scripts/check-release-train.sh adds the
#      per-dependency UNPUBLISHED/TRAIN-LAG classification on top.
#
# Local (default) mode chains scripts/check-version-drift.sh --strict.
# Under CI=true the drift leg downgrades to advisory: tag existence needs
# `git ls-remote` auth for the private larsartmann repos, unverified on
# Actions runners so far.
#
# Usage: ./scripts/check-require-tags.sh
# Exit:  0 = clean, 1 = phantom version or strict drift failure.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=== Require Tag Check ==="
echo "Scanning go.mod files for zero pseudo-versions..."
found=0
result=$(rg 'v0\.0\.0-00010101000000-000000000000' --glob 'go.mod' . 2>/dev/null || true)
if [ -n "$result" ]; then
  echo "FAIL: zero pseudo-version detected:"
  echo "$result"
  found=1
fi
if [ "$found" -eq 0 ]; then
  echo "OK: No phantom versions detected."
else
  exit 1
fi

# Tag existence: strict locally (fail on unpublished requires), advisory
# under CI (ls-remote auth unverified there).
if [ "${CI:-false}" != "true" ]; then
  bash scripts/check-version-drift.sh --strict
else
  bash scripts/check-version-drift.sh || true
fi
