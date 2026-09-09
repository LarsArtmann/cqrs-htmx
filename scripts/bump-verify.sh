#!/usr/bin/env bash
# bump-verify.sh — one-command verification bundle for dependency bumps.
#
# Closes the ladder hole that bit the 2026-09-09 bump: build passed while
# tests/lint/coverage still referenced the old dependency behavior, and the
# deferred suites (race, cqrs-lint) surfaced days later. Run this after ANY
# internal-train bump and the "probably fine" gap disappears.
#
# Stages (sequential, stop on first failure):
#   1. build           — nix run .#build          (hermetic, all modules)
#   2. test            — nix run .#test           (includes go vet)
#   3. lint            — nix run .#lint           (15 modules, 0-issue policy)
#   4. check-modules   — nix run .#check-modules --report (isolation, budgets,
#                        toolchain, drift --strict, release-train, codegen)
#   5. check-cqrs-lint — nix run .#check-cqrs-lint
#   6. bench sanity    — nix run .#bench-spike (skipped: --skip-bench, or when
#                        the machine is under load — never bench under load)
#
# Usage:
#   source scripts/lib/go-cache-env.sh && bash scripts/bump-verify.sh [--skip-bench]
# Exit: 0 = all requested stages green, 1 = first failing stage (named).
# Per-stage full logs: /tmp/bump-verify-<stage>.log

set -uo pipefail

SKIP_BENCH=false
if [[ ${1:-} == "--skip-bench" ]]; then
  SKIP_BENCH=true
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT" || exit 1

stage() { # <name> <args to nix run .#...>
  local name="$1"
  shift
  echo "=== [$name] nix run $*"
  if nix run "$@" >"/tmp/bump-verify-$name.log" 2>&1; then
    echo "=== [$name] OK"
    return 0
  fi
  echo "=== [$name] FAIL — tail of /tmp/bump-verify-$name.log:"
  tail -20 "/tmp/bump-verify-$name.log"
  return 1
}

FAILED=""
stage build .#build || FAILED="build"
[ -z "$FAILED" ] && { stage test .#test || FAILED="test"; }
[ -z "$FAILED" ] && { stage lint .#lint || FAILED="lint"; }
[ -z "$FAILED" ] && { stage check-modules .#check-modules -- --report || FAILED="check-modules"; }
[ -z "$FAILED" ] && { stage check-cqrs-lint .#check-cqrs-lint || FAILED="check-cqrs-lint"; }

if [ -z "$FAILED" ] && [ "$SKIP_BENCH" = "false" ]; then
  LOAD=$(cut -d' ' -f1-3 /proc/loadavg | tr ' ' '\n' | sort -rn | head -1 | cut -d. -f1)
  CORES=$(nproc)
  if [ "${LOAD:-99}" -ge "$CORES" ]; then
    echo "=== [bench-sanity] SKIPPED (load $LOAD >= cores $CORES; never bench under load)"
  else
    stage bench-sanity .#bench-spike || FAILED="bench-sanity"
  fi
fi

if [ -n "$FAILED" ]; then
  echo ""
  echo "BUMP-VERIFY: FAILED at stage [$FAILED]"
  exit 1
fi
echo ""
echo "BUMP-VERIFY: ALL GREEN"
