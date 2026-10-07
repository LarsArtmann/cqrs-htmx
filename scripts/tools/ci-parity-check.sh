#!/usr/bin/env bash
# ci-parity-check.sh — the gotcha-27c lesson as a command: CI only surfaces
# the FIRST failure per job, and several gates (module-architecture
# self-tests, docs freshness, CSS class-set equality, fixture self-tests
# under CI=true) are invisible to build+test. The ONLY local replication
# that found every remaining red in ONE pass was this battery — so it is now
# one command to run before a push (or after any multi-lane batch).
#
# Stages (stop on first failing stage; later stages assume earlier ones):
#   1. CI=true fixture self-tests   — every scripts/selftests/test-*.sh under
#                                     the env GitHub runners export, so
#                                     local-mode assertions leak instantly
#   2. go mod tidy -diff loop       — union-graph go.sum drift per go.mod
#   3. nix run .#lint               — golangci-lint over all gated modules
#   4. nix run .#test               — race tests over workspace members
#   5. nix run .#check-modules      — the full gate bundle
#
# Usage: bash scripts/tools/ci-parity-check.sh [stage...]   (default: all)
# Env:   PARITY_SKIP=<comma,stage-names> to skip stages deliberately.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT" || exit 1

LOG_TAG="$$-$(date +%s)" # fixed per-run tag: no per-invocation word splitting

stage_wanted() { # <name> [restricting-names...] — true unless PARITY_SKIP
  local name="$1"
  shift
  if [ "${PARITY_SKIP:-}" != "" ]; then
    case ",${PARITY_SKIP:-}," in
    *",$name,"*) return 1 ;;
    esac
  fi
  if [ $# -eq 0 ]; then
    return 0
  fi
  local w
  for w in "$@"; do
    [ "$w" = "$name" ] && return 0
  done
  return 1
}

overall=0

if stage_wanted selftests "$@"; then
  echo "== stage 1: CI=true fixture self-tests"
  selftest_fail=0
  shopt -s nullglob
  for t in scripts/selftests/test-*.sh; do
    if ! CI=true bash "$t" >"/tmp/ci-parity-selftest-$LOG_TAG.log" 2>&1; then
      echo "  FAIL $t (log: /tmp/ci-parity-selftest-$LOG_TAG.log)"
      selftest_fail=1
    fi
  done
  shopt -u nullglob
  if [ "$selftest_fail" -ne 0 ]; then
    overall=1
  else
    echo "  all fixture self-tests green under CI=true"
  fi
fi

if stage_wanted tidy "$@"; then
  echo "== stage 2: go mod tidy -diff per go.mod"
  # shellcheck disable=SC1091
  source scripts/lib/go-cache-env.sh || true
  tidy_fail=0
  while IFS= read -r gomod; do
    dir="$(dirname "$gomod")"
    # Fixture go.mods (scripts/testdata/*) are deliberate non-buildable
    # fixtures — bump-dep excludes them; so does this loop.
    case "$dir" in
    */testdata/*) continue ;;
    esac
    if ! (cd "$dir" && GOWORK=off GOEXPERIMENT=jsonv2 go mod tidy -diff >/dev/null 2>&1); then
      echo "  TIDY DRIFT: $gomod"
      tidy_fail=1
    fi
  done < <(find . -name go.mod -not -path './.git/*' | sort)
  if [ "$tidy_fail" -ne 0 ]; then
    overall=1
  else
    echo "  all go.mods tidy-clean"
  fi
fi

if stage_wanted lint "$@"; then
  echo "== stage 3: nix run .#lint"
  if ! nix run .#lint >"/tmp/ci-parity-lint-$LOG_TAG.log" 2>&1; then
    echo "  FAIL (log: /tmp/ci-parity-lint-$LOG_TAG.log)"
    overall=1
  else
    echo "  lint green"
  fi
fi

if stage_wanted test "$@"; then
  echo "== stage 4: nix run .#test"
  if ! nix run .#test >"/tmp/ci-parity-test-$LOG_TAG.log" 2>&1; then
    echo "  FAIL (log: /tmp/ci-parity-test-$LOG_TAG.log)"
    overall=1
  else
    echo "  tests green"
  fi
fi

if stage_wanted checkmodules "$@"; then
  echo "== stage 5: nix run .#check-modules"
  if ! nix run .#check-modules >"/tmp/ci-parity-checkmodules-$LOG_TAG.log" 2>&1; then
    echo "  FAIL (log: /tmp/ci-parity-checkmodules-$LOG_TAG.log)"
    overall=1
  else
    echo "  check-modules green"
  fi
fi

if [ "$overall" -ne 0 ]; then
  echo "ci-parity-check: FAILED — fix every stage above before pushing (one pass, all reds)"
  exit 1
fi
echo "ci-parity-check: ALL STAGES GREEN — local state matches CI's expectations"
