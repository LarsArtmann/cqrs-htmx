#!/usr/bin/env bash
# test-check-module-layers.sh — fixture self-test for the package layering gate.
#
# The gate's whole value is that it fails on a back-edge, so every case below
# plants one (or removes a layer) and asserts the exact verdict. Offline: the
# fixtures are throwaway modules with no dependencies.
#
# Cases:
#   1. clean layering                              → pass
#   2. htmx imports the root package               → fail, names the edge
#   3. internal/hx imports htmx (floor violation)   → fail
#   4. httperr imports eventcatalog (off-row edge)  → fail
#   5. back-edge hidden in a _test.go file          → fail (TestImports checked)
#   6. a declared layer missing from disk           → fail, names the directory
#   7. no go.mod at the module root                 → fail
#   8. clean layering, explicit allowed edges       → pass
#
# Usage: bash scripts/selftests/test-check-module-layers.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-module-layers.sh"

WORK=$(mktemp -d /tmp/test-module-layers-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0
ok() {
  echo "  ok $1"
  pass=$((pass + 1))
}
bad() {
  echo "  FAIL $1: $2" >&2
  fail=$((fail + 1))
}

MOD="example.com/layers"

mk_mod() { # dir
  mkdir -p "$1"
  printf 'module %s\n\ngo 1.21\n' "$MOD" >"$1/go.mod"
}

mk_pkg() { # dir pkgname [imports...]
  local dir="$1" name="$2"
  shift 2
  mkdir -p "$dir"
  {
    printf 'package %s\n\n' "$name"
    if [ "$#" -gt 0 ]; then
      printf 'import (\n'
      for imp in "$@"; do printf '\t_ "%s"\n' "$imp"; done
      printf ')\n\n'
    fi
    printf '// X is the fixture export.\nconst X = 1\n'
  } >"$dir/fixture.go"
}

mk_clean() { # dir
  local d="$1"
  mk_mod "$d"
  mk_pkg "$d" layers
  mk_pkg "$d/internal/hx" hx
  mk_pkg "$d/htmx" htmx "$MOD/internal/hx"
  mk_pkg "$d/eventcatalog" eventcatalog "$MOD/internal/hx"
  mk_pkg "$d/httperr" httperr "$MOD/htmx" "$MOD/internal/hx"
}

run_gate() { # dir
  LAYERS_ROOT="$1" LAYERS_MODULE="$MOD" bash "$TOOL" 2>&1
}

# 1. clean layering
mk_clean "$WORK/clean"
out=$(run_gate "$WORK/clean")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "OK: layering holds"; then
  ok "1-clean-passes"
else
  bad "1-clean-passes" "rc=$rc $out"
fi

# 2. htmx imports the root package
mk_clean "$WORK/back-edge"
mk_pkg "$WORK/back-edge/htmx" htmx "$MOD/internal/hx" "$MOD"
out=$(run_gate "$WORK/back-edge")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -qxF "FAIL: $MOD/htmx imports $MOD"; then
  ok "2-root-back-edge-fails"
else
  bad "2-root-back-edge-fails" "rc=$rc $out"
fi

# 3. internal/hx must import nothing from the module
mk_clean "$WORK/floor"
mk_pkg "$WORK/floor/internal/hx" hx "$MOD"
out=$(run_gate "$WORK/floor")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -qxF "FAIL: $MOD/internal/hx imports $MOD"; then
  ok "3-floor-violation-fails"
else
  bad "3-floor-violation-fails" "rc=$rc $out"
fi

# 4. httperr may not import eventcatalog
mk_clean "$WORK/offrow"
mk_pkg "$WORK/offrow/httperr" httperr "$MOD/htmx" "$MOD/internal/hx" "$MOD/eventcatalog"
out=$(run_gate "$WORK/offrow")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -qxF "FAIL: $MOD/httperr imports $MOD/eventcatalog"; then
  ok "4-off-row-edge-fails"
else
  bad "4-off-row-edge-fails" "rc=$rc $out"
fi

# 5. a back-edge hidden in a test file is still a back-edge
mk_clean "$WORK/testedge"
printf 'package eventcatalog\n\nimport (\n\t_ "%s"\n)\n' "$MOD" >"$WORK/testedge/eventcatalog/fixture_test.go"
out=$(run_gate "$WORK/testedge")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -qxF "FAIL: $MOD/eventcatalog imports $MOD"; then
  ok "5-test-import-back-edge-fails"
else
  bad "5-test-import-back-edge-fails" "rc=$rc $out"
fi

# 6. a declared layer missing from disk
mk_clean "$WORK/missing"
rm -rf "$WORK/missing/httperr"
out=$(run_gate "$WORK/missing")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "missing on disk: httperr/"; then
  ok "6-missing-layer-fails"
else
  bad "6-missing-layer-fails" "rc=$rc $out"
fi

# 7. no go.mod
mkdir -p "$WORK/bare"
out=$(run_gate "$WORK/bare")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "no go.mod"; then
  ok "7-no-gomod-fails"
else
  bad "7-no-gomod-fails" "rc=$rc $out"
fi

# 8. the allowed rows are exactly what the gate advertises
mk_clean "$WORK/allowed"
out=$(run_gate "$WORK/allowed")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "httperr: 2 in-module import(s)"; then
  ok "8-allowed-rows-counted"
else
  bad "8-allowed-rows-counted" "rc=$rc $out"
fi

# 9. a genuine import cycle fails loudly instead of being silently skipped
mk_clean "$WORK/cycle"
mk_pkg "$WORK/cycle/internal/hx" hx "$MOD/htmx"
out=$(run_gate "$WORK/cycle")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "go list could not load the layer packages"; then
  ok "9-cycle-fails-loudly"
else
  bad "9-cycle-fails-loudly" "rc=$rc $out"
fi

echo ""
echo "module-layers selftest: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
