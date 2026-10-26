#!/usr/bin/env bash
# check-root-files.sh — allowlist gate on what may live at the repo root.
#
# The repo root is the root Go module's package directory: *.go files come
# and go with the package, but every OTHER tracked file at depth 1 is a
# config, a doc, or a stray — the biome.json class (a config nothing
# consumed, sitting at root for months until the 2026-10-26 prove-or-remove
# audit). This gate freezes the tracked non-Go root set: a new root file
# fails until the baseline is bumped in the same, visible commit; a removed
# one requires lowering the baseline. Untracked/gitignored artifacts
# (coverage.out, caches, compiled binaries) are invisible by design — the
# daemon tracks real content, and generated artifacts are not content.
#
# Usage:
#   bash scripts/checks/check-root-files.sh                # report (advisory)
#   bash scripts/checks/check-root-files.sh --strict       # blocking (CI)
#   bash scripts/checks/check-root-files.sh --update-baseline
#
# Env: ROOT_FILES_GATE_ROOT    override the repo root (fixture self-test).
#      ROOT_FILES_BASELINE     override the baseline file (fixture self-test).
# Exit: 0 ok / advisory, 1 drift under --strict or missing baseline, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${ROOT_FILES_GATE_ROOT:-$REPO_ROOT}"
BASELINE="${ROOT_FILES_BASELINE:-$SCRIPT_DIR/root-files.baseline}"

STRICT=0
UPDATE=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --strict) STRICT=1 ;;
  --update-baseline) UPDATE=1 ;;
  *)
    echo "usage: check-root-files.sh [--strict] [--update-baseline]" >&2
    exit 2
    ;;
  esac
  shift
done

# Tracked, depth-1, non-Go files, sorted. ls-files lists the index, so a
# staged-but-uncommitted stray is already visible.
mapfile -t root_files < <(git -C "$ROOT" ls-files | grep -v '/' | grep -v '\.go$' | LC_ALL=C sort)

if [ "$UPDATE" -eq 1 ]; then
  if [ "${#root_files[@]}" -gt 0 ]; then
    printf '%s\n' "${root_files[@]}" >"$BASELINE"
  else
    : >"$BASELINE"
  fi
  echo "✓ root-files: baseline written: ${#root_files[@]} tracked non-Go root file(s)"
  exit 0
fi

echo "root-files: ${#root_files[@]} tracked non-Go file(s) at repo root"

if [ ! -f "$BASELINE" ]; then
  echo "✗ root-files: missing baseline $BASELINE (run with --update-baseline to create it)" >&2
  exit 1
fi

if diff_out=$(diff "$BASELINE" <(printf '%s\n' "${root_files[@]}")); then
  echo "✓ root-files: matches the baseline allowlist"
  exit 0
fi

echo "✗ root-files: root file set drifted from the baseline allowlist:" >&2
printf '%s\n' "$diff_out" | sed 's/^/    /' >&2

if [ "$STRICT" -eq 0 ]; then
  echo "root-files: advisory — re-run with --strict to block"
  exit 0
fi

echo "root-files: put the file where it belongs, or justify it and bump the baseline in the SAME commit: bash scripts/checks/check-root-files.sh --update-baseline" >&2
exit 1
