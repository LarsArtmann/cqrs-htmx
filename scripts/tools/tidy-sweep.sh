#!/usr/bin/env bash
# tidy-sweep.sh — per-module hermetic `go mod tidy` sweep (canary a: gotcha
# 34 as code).
#
# The union workspace hides module-local drift; only a per-module GOWORK=off
# tidy catches it (gotcha 36 companion: the auditlog stale go.sum hash that
# no train gate ever visited). scripts/testdata/ is excluded BY CONSTRUCTION
# — the verify-tag fixtures are deliberately untidy (gotcha 34), and this
# script must never be the sweep that destroys them.
#
# Usage:
#   bash scripts/tools/tidy-sweep.sh                  # report candidates (-diff)
#   bash scripts/tools/tidy-sweep.sh --fix            # apply tidy where needed
#   bash scripts/tools/tidy-sweep.sh -- <module-dir>  # single module
# Env: GOEXPERIMENT=jsonv2 forced; GOCACHE/GOMODCACHE via lib/go-cache-env.sh.
# Exit: 0 clean (or all fixed), 1 candidates remain in check mode, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
# shellcheck source=/dev/null # hermetic env lives outside this script's dir
source "$SCRIPT_DIR/../lib/go-cache-env.sh"

MODE=check
only=""
while [ "$#" -gt 0 ]; do
  case "$1" in
  --fix) MODE=fix ;;
  --)
    shift
    only="${1:-}"
    break
    ;;
  *)
    echo "usage: tidy-sweep.sh [--fix] [-- <module-dir>]" >&2
    exit 2
    ;;
  esac
  shift
done

export GOEXPERIMENT=jsonv2

if [ -n "$only" ]; then
  members=("$only")
else
  mapfile -t members < <(
    awk '/^use \(/{f=1;next} /^\)/{f=0} f{print $1}' "$REPO_ROOT/go.work" |
      sed 's#^\./##' |
      grep -v '^scripts/testdata'
  )
fi

candidates=0
for mod in "${members[@]}"; do
  dir="$REPO_ROOT/$mod"
  if [ ! -f "$dir/go.mod" ]; then
    echo "  ? $mod has no go.mod — skipped"
    continue
  fi

  if [ "$MODE" = "fix" ]; then
    if (cd "$dir" && GOWORK=off go mod tidy 2>/dev/null); then
      echo "  ✓ $mod tidied"
    else
      echo "  ✗ $mod tidy FAILED"
      candidates=$((candidates + 1))
    fi
    continue
  fi

  if ! diff_out=$(cd "$dir" && GOWORK=off go mod tidy -diff 2>&1); then
    if grep -q '^diff ' <<<"$diff_out"; then
      echo "  ✗ $mod is NOT tidy (go mod tidy -diff reports changes)"
      candidates=$((candidates + 1))
    else
      echo "  ✗ $mod tidy -diff errored:"
      printf '%s\n' "$diff_out" | head -5 | sed 's/^/      /'
      candidates=$((candidates + 1))
    fi
  fi
done

if [ "$candidates" -gt 0 ]; then
  echo "tidy-sweep: $candidates module(s) need tidy (run with --fix)"
  exit 1
fi

echo "tidy-sweep: all ${#members[@]} module(s) tidy (testdata excluded)"
