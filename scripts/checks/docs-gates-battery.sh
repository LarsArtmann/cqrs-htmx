#!/usr/bin/env bash
# docs-gates-battery.sh — run the docs-health five-gate battery as ONE matrix,
# with an honest self-skip instead of a red build for an absent binary.
#
# WHY: the battery is a local-advisory surface (`.docs-gates.json` pins
# `audit_max`). It is not a flake input and it is not in nixpkgs, so CI cannot
# assume it exists. The repo's precedent for a tool that may be absent is
# check-branching-flow.sh: self-skip with a printed reason, never a false green
# and never a red for a missing tool. This wrapper adds the one thing CI needs:
# the five verdicts printed as a matrix in the log, so drift is visible even
# when the step is advisory.
#
# Resolution order: docs-gates on PATH · `nix run github:Artmann-Technologies/
# docs-gates` (when nix + network are usable) · SKIP (rc=0, reason printed).
#
# Usage: docs-gates-battery.sh [--strict] [--json]
#   --strict  a FAIL verdict becomes this script's exit 1 (manual/CI-blocking use)
#   --json    pass through to the binary's machine-readable mode
#
# Env: DOCS_GATES_BIN      force a binary path (fixture self-test)
#      DOCS_GATES_NO_NIX  1 = never fall back to `nix run`
#      DOCS_GATES_ROOT     repo root to scan (default: script's ../..)
# Exit: 0 clean or self-skipped/advisory · 1 findings under --strict ·
#       2 usage error
set -uo pipefail

STRICT=0
PASSTHRU=()
for arg in "$@"; do
  case "$arg" in
  --strict) STRICT=1 ;;
  --json) PASSTHRU+=(--json) ;;
  *)
    echo "usage: $(basename "$0") [--strict] [--json]" >&2
    exit 2
    ;;
  esac
done

ROOT="${DOCS_GATES_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

if [ ! -d "$ROOT" ]; then
  echo "✗ docs-gates-battery: $ROOT is not a directory (rc=2)" >&2
  exit 2
fi

bin="${DOCS_GATES_BIN:-$(command -v docs-gates || true)}"
if [ -n "$bin" ] && [ ! -x "$bin" ]; then
  echo "WARNING: docs-gates-battery: $bin is not executable — treating docs-gates as absent" >&2
  bin=""
fi

run_args=(run "$ROOT")
if [ "${#PASSTHRU[@]}" -gt 0 ]; then
  run_args+=("${PASSTHRU[@]}")
fi

if [ -n "$bin" ]; then
  echo "docs-gates-battery: using $bin"
  out="$("$bin" "${run_args[@]}" 2>&1)"
  rc=$?
elif [ "${DOCS_GATES_NO_NIX:-0}" != "1" ] && command -v nix >/dev/null 2>&1; then
  echo "docs-gates-battery: docs-gates not on PATH — trying nix run github:Artmann-Technologies/docs-gates"
  out="$(nix run github:Artmann-Technologies/docs-gates -- "${run_args[@]}" 2>&1)"
  rc=$?
  # A nix failure that never reached the battery is SKIP, not a verdict.
  if ! printf '%s' "$out" | grep -q 'verdict matrix'; then
    echo "SKIP: docs-gates-battery — could not run the battery (nix/network):"
    printf '%s\n' "$out" | tail -5 | sed 's/^/    /'
    echo "SKIP: advisory surface, exit 0 — run it locally: nix run github:Artmann-Technologies/docs-gates -- run ."
    exit 0
  fi
else
  echo "SKIP: docs-gates-battery — docs-gates is not installed and nix fallback is disabled/unavailable"
  echo "SKIP: advisory surface, exit 0 — install it or run: nix run github:Artmann-Technologies/docs-gates -- run ."
  exit 0
fi

# Always surface the matrix, even when the binary floods the log above it.
printf '%s\n' "$out" | sed -n '/verdict matrix/,$p'

if [ "$rc" -eq 0 ]; then
  echo "✓ docs-gates-battery: all five gates PASS"
  exit 0
fi

if [ "$rc" -ne 1 ]; then
  echo "✗ docs-gates-battery: the battery itself failed (rc=$rc) — that is not a findings verdict" >&2
  exit 2
fi

if [ "$STRICT" -eq 1 ]; then
  echo "✗ docs-gates-battery: FAIL under --strict (see the matrix above)" >&2
  exit 1
fi

echo "WARNING: docs-gates-battery: gate(s) FAILED — advisory mode (exit 0)."
echo "  The pin/ratchet decisions live in TODO_LIST.md; raise with --strict to block."
exit 0
