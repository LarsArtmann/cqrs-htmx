#!/usr/bin/env bash
# check-coverage-threshold.sh — the single-source coverage threshold check.
#
# Thresholds live in coverage-thresholds.tsv (consumed by this script AND by
# `nix run .#coverage-gate`); before this gate the numbers were duplicated in
# flake.nix and every CI coverage step, and a drift between the two was
# invisible.
#
# Fail-loud by construction (canary d): a missing profile, an unparsable
# total, or an under-threshold total all exit 1 — the old CI awk and the old
# flake bc path both silently PASSED when a failed `go test` produced no
# coverage at all.
#
# Usage: check-coverage-threshold.sh <module-row> [profile]
#   <module-row>  a row from coverage-thresholds.tsv ("." for the root)
#   [profile]     coverage profile path; default <module>/coverage.out
# Exit: 0 pass, 1 fail/missing/unparsable, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

module="${1:-}"
profile="${2:-}"
if [ -z "$module" ]; then
  echo "usage: check-coverage-threshold.sh <module-row> [profile]" >&2
  exit 2
fi

tsv="$SCRIPT_DIR/coverage-thresholds.tsv"
threshold=$(awk -F'\t' -v m="$module" '$1 == m { print $2 }' "$tsv")
if [ -z "$threshold" ]; then
  echo "✗ coverage-threshold: no threshold row for '$module' in $tsv" >&2
  exit 1
fi

if [ -z "$profile" ]; then
  profile="$REPO_ROOT/$module/coverage.out"
fi
if [ "$module" = "." ]; then
  profile="$REPO_ROOT/coverage.out"
fi

if [ ! -f "$profile" ]; then
  echo "✗ coverage-threshold: profile missing: $profile (run the tests with -coverprofile first)" >&2
  exit 1
fi

cov=$(go tool cover -func="$profile" | tail -1 | grep -oP '\d+\.\d+(?=%)' || true)
case "$cov" in
'' | *[!0-9.]*)
  echo "✗ coverage-threshold: could not parse total coverage from $profile" >&2
  exit 1
  ;;
esac

echo "$module coverage: ${cov}% (threshold: ${threshold}%)"
if ! awk "BEGIN { exit ($cov < $threshold) ? 1 : 0 }"; then
  echo "FAIL: $module coverage ${cov}% < ${threshold}%" >&2
  exit 1
fi
