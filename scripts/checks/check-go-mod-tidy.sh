#!/usr/bin/env bash
# check-go-mod-tidy.sh — hermetic `go mod tidy -diff` over every go.work member.
#
# The 2026-11-21 seven-job CI red (build, test, mod-tidy, security, lint(.),
# module-architecture, route-postures' root leg — all one cause: a mid-flight
# Frankenstein go.sum missing go.mod-file hash entries) was one command away
# from detection: `go mod tidy -diff` per module, run hermetically, prints the
# drift and exits nonzero WITHOUT touching files. Workspace mode masks the
# class locally (go.work resolves its own graph), which is exactly why the
# tree can read green while CI reads red — so this gate runs GOWORK=off.
#
# Usage:
#   bash scripts/checks/check-go-mod-tidy.sh                 # gate (default)
#   TIDY_GATE_ROOT=<tree> bash scripts/checks/check-go-mod-tidy.sh   # fixture self-test
#
# Env: TIDY_GATE_ROOT override the repo root (fixture self-test).
# Exit: 0 all members tidy-clean / 1 drift found (prints the offending module
#       and diff head) / 2 usage or broken go.work.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${TIDY_GATE_ROOT:-$REPO_ROOT}"

GO_WORK="$ROOT/go.work"
if [ ! -f "$GO_WORK" ]; then
  echo "tidy-gate: no go.work at $GO_WORK" >&2
  exit 2
fi

# The module list comes from go.work's use-block (the CI mod-tidy job's derivation).
MODULES="$(awk '/^use \(/,/^\)/' "$GO_WORK" | sed 's/^[[:space:]]*//; s/[[:space:]]*\/\/.*$//; s/^use ([[:space:]]*$//; s/^[)[:space:]]*$//; s/^\.\///; /^[[:space:]]*$/d')"
if [ -z "$MODULES" ]; then
  echo "tidy-gate: go.work use-block is empty" >&2
  exit 2
fi

# Hermetic environment: writable GOCACHE fallback + GOTOOLCHAIN floor alignment
# (the repo's shared go-cache-env lib when available; raw env otherwise).
# shellcheck source=../lib/go-cache-env.sh
if [ -f "$SCRIPT_DIR/../lib/go-cache-env.sh" ]; then
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/../lib/go-cache-env.sh"
fi
export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"
export GOWORK=off

failures=0
checked=0
while IFS= read -r mod; do
  [ -z "$mod" ] && continue
  moddir="$ROOT/$mod"
  [ -f "$moddir/go.mod" ] || {
    echo "tidy-gate: $mod: no go.mod (go.work names it)" >&2
    failures=$((failures + 1))
    continue
  }
  checked=$((checked + 1))
  out="$(cd "$moddir" && go mod tidy -diff 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "tidy-gate: $mod: go.mod/go.sum DRIFT (tidy -diff rc=$rc):"
    printf '%s\n' "$out" | head -20 | sed 's/^/    /'
    [ "$(printf '%s\n' "$out" | wc -l)" -gt 20 ] && echo "    ... (truncated)"
    failures=$((failures + 1))
  fi
done <<<"$MODULES"

if [ "$failures" -gt 0 ]; then
  echo "✗ tidy-gate: $failures of $checked module(s) would change under go mod tidy — run 'go mod tidy' in the named module(s) and commit together"
  exit 1
fi
echo "✓ tidy-gate: $checked module(s) tidy-clean (GOWORK=off)"
