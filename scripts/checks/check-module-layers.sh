#!/usr/bin/env bash
# check-module-layers.sh — enforce the package layering inside the root module.
#
# The root package (cqrshtmx) was decomposed on 2026-11-02 into htmx,
# eventcatalog, httperr and the shared internal/hx plumbing. That split is only
# worth anything if the edges stay one-way: the moment a subpackage imports the
# root package it re-creates the very cycle internal/hx exists to break, the
# root facades grow shims to compensate, and the next extraction is blocked.
#
# Declared layering — each package may import ONLY what its row lists:
#
#   internal/hx    (nothing from this module; it is the floor)
#   htmx           internal/hx
#   eventcatalog   internal/hx
#   httperr        htmx, internal/hx
#   .              (the root package; top of the stack, may import all of them)
#
# Test imports are checked too (a *_test.go back-edge is the same defect).
#
# Usage: ./scripts/checks/check-module-layers.sh
# Env:   LAYERS_ROOT   module root to inspect   (default: this repo)
#        LAYERS_MODULE module import path       (default: the root module)
# Exit:  0 = layering holds, 1 = forbidden edge, missing package, or go list error.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ROOT="${LAYERS_ROOT:-$REPO_ROOT}"
MODULE="${LAYERS_MODULE:-github.com/larsartmann/cqrs-htmx/v4}"

cd "$ROOT" || exit 1

# Order matters only for readable output: floor first, then its dependents.
LAYERS=(
  "$MODULE/internal/hx"
  "$MODULE/htmx"
  "$MODULE/eventcatalog"
  "$MODULE/httperr"
)

allowed_for() {
  case "$1" in
    "$MODULE/internal/hx") echo "" ;;
    "$MODULE/htmx") echo "$MODULE/internal/hx" ;;
    "$MODULE/eventcatalog") echo "$MODULE/internal/hx" ;;
    "$MODULE/httperr") echo "$MODULE/htmx $MODULE/internal/hx" ;;
    *) echo "" ;;
  esac
}

echo "=== Package Layer Check ($MODULE) ==="

if [ ! -f "$ROOT/go.mod" ]; then
  echo "FAIL: no go.mod at $ROOT"
  exit 1
fi

for pkg in "${LAYERS[@]}"; do
  dir="${pkg#"$MODULE"}"
  dir="${dir#/}"
  if [ -n "$dir" ] && [ ! -d "$ROOT/$dir" ]; then
    echo "FAIL: declared layer package is missing on disk: $dir/"
    exit 1
  fi
done

# One go list call: package path + its (non-test and test) imports.
list_out=$(GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}" go list \
  -f '{{.ImportPath}}|{{join .Imports " "}} {{join .TestImports " "}}' \
  "${LAYERS[@]}" 2>&1)
list_rc=$?
if [ "$list_rc" -ne 0 ]; then
  echo "FAIL: go list could not load the layer packages (rc=$list_rc):"
  printf '%s\n' "$list_out"
  exit 1
fi

fail=0
while IFS='|' read -r pkg imports; do
  [ -n "$pkg" ] || continue
  allowed=" $(allowed_for "$pkg") "
  count=0

  for imp in $imports; do
    case "$imp" in
      "$MODULE" | "$MODULE"/*) ;;
      *) continue ;;
    esac
    # A package always "imports" itself in the test binary; not an edge.
    [ "$imp" = "$pkg" ] && continue

    count=$((count + 1))
    if [[ "$allowed" != *" $imp "* ]]; then
      echo "FAIL: $pkg imports $imp"
      echo "      allowed from $pkg:${allowed% }"
      fail=1
    fi
  done

  echo "  $(basename "$pkg"): $count in-module import(s)"
done <<<"$list_out"

if [ "$fail" -ne 0 ]; then
  echo "FAIL: layering violated — a subpackage must never import the root package,"
  echo "      and internal/hx must not import anything from this module."
  exit 1
fi

echo "OK: layering holds for ${#LAYERS[@]} packages"
