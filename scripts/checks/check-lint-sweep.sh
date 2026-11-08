#!/usr/bin/env bash
# check-lint-sweep.sh — per-module HERMETIC golangci-lint parity sweep.
#
# The parity gap this closes (2026-11-07 01-34 §e1, the gotcha-50 class):
# `nix run .#lint` lints in WORKSPACE mode, while CI's lint matrix runs
# GOWORK=off per module — so lint drift that only shows hermetically (the
# unnamed-self-import that sailed through a --no-verify push) is caught by
# CI AFTER the push and by nothing local. This gate is the local mirror of
# CI's matrix: every go.work member that carries a .golangci.yml, linted
# hermetically, same --timeout as CI.
#
# Module set is DERIVED (go.work + .golangci.yml presence), never
# hand-listed — a new gated module joins the sweep by adding its config.
#
# Usage: bash scripts/checks/check-lint-sweep.sh [-- <module-dir>]
# Env:   GOEXPERIMENT=jsonv2 forced; LINT_SWEEP_ROOT overrides the scanned
#        root (self-test hook, the DRIFT_ROOT pattern); golangci-lint from
#        the flake app's runtimeInputs.
# Exit:  0 = every gated module clean, 1 = at least one module has findings.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${LINT_SWEEP_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/go-cache-env.sh"

only=""
if [ "${1:-}" = "--" ] && [ -n "${2:-}" ]; then
  only="$2"
fi

# Hermetic per-module lint, exactly CI's shape (GOWORK=off, same timeout).
lint_module() { # <module-dir>
  local dir="$1"
  (
    cd "$REPO_ROOT/$dir" || exit 1
    env GOWORK=off GOEXPERIMENT=jsonv2 golangci-lint run --timeout=5m
  )
}

failed=0
checked=0

# Discover modules: every go.work use-entry that carries a .golangci.yml.
while IFS= read -r raw; do
  dir="${raw#./}"
  [ "$dir" = "." ] && dir="."

  if [ -n "$only" ] && [ "$dir" != "$only" ]; then
    continue
  fi

  if [ ! -f "$REPO_ROOT/$dir/.golangci.yml" ]; then
    continue
  fi

  checked=$((checked + 1))
  echo "==> golangci-lint (hermetic) $dir"
  if ! lint_module "$dir"; then
    echo "  ✗ lint-sweep: $dir has findings (CI's '${dir}' matrix leg would be red)"
    failed=1
  fi
done < <(cd "$REPO_ROOT" && env GOWORK= go work edit -json 2>/dev/null | jq -r '.Use[].DiskPath' | sed 's|^\./||')

echo ""
if [ "$checked" -eq 0 ]; then
  echo "lint-sweep: no gated modules found (every go.work member without a .golangci.yml is skipped by design)"
  exit 0
fi

if [ "$failed" -eq 0 ]; then
  echo "✓ lint-sweep: $checked module(s) clean under hermetic GOWORK=off golangci-lint"
else
  echo "✗ lint-sweep: hermetic findings above — the local workspace-mode lint cannot see these"
  exit 1
fi
