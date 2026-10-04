#!/usr/bin/env bash
# test-check-css-bundle-classes.sh — self-test for check-css-bundle-classes.sh
#
# Exercises the verdicts in fixture mode (CHECK_CSS_BUNDLE_CLASSES_NO_BUILD=1,
# no nix, no real-tree mutation):
#   F1  identical committed/fresh class sets      -> PASS, exact reported
#   F2  fresh build missing one class             -> FAIL, names the class
#   F3  fresh build with one extra class          -> FAIL, names the class
#   F4  fixture missing the .fresh file           -> FAIL, names the file
#   F5  formatting-only difference (same classes) -> PASS (the gate's purpose)
#
# Usage: ./scripts/selftests/test-check-css-bundle-classes.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/check-css-bundle-classes.sh"

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

BASE_CSS='.bg-gray-100{color:red}.text-sm\:bold{font-weight:700}@media(min-width:768px){.md\:flex{display:flex}}'

make_fixture() { # <dir> <fresh-css>  — bundles for BOTH modules
  mkdir -p "$1/adminui/assets" "$1/dashboardui/assets"
  printf '%s' "$BASE_CSS" >"$1/adminui/assets/admin-tw.css"
  printf '%s' "$BASE_CSS" >"$1/dashboardui/assets/dashboard-tw.css"
  printf '%s' "$2" >"$1/adminui/assets/admin-tw.css.fresh"
  printf '%s' "$2" >"$1/dashboardui/assets/dashboard-tw.css.fresh"
}

echo ""
echo "=== Test Suite: check-css-bundle-classes.sh ==="
echo ""

# The checker resolves bundles from its own REPO_ROOT — fixture mode needs
# the fixture tree to BE that root. Run via a subshell cd so the checker's
# dirname-based resolution lands inside the fixture: copy the checker in.
run_in_fixture() { # <fixture-dir> — sets OUT + RC via capture files
  local fx="$1"
  mkdir -p "$fx/scripts/checks"
  cp "$CHECKER" "$fx/scripts/checks/"
  (
    cd "$fx" || exit 1
    CHECK_CSS_BUNDLE_CLASSES_NO_BUILD=1 bash scripts/checks/check-css-bundle-classes.sh >"$fx/.out" 2>&1
    echo "$?" >"$fx/.rc"
  )
  RC="$(cat "$fx/.rc")"
  OUT="$(cat "$fx/.out")"
}

# --- F1: identical class sets pass -------------------------------------------
F1="$(mktemp -d)"
make_fixture "$F1" "$BASE_CSS"
run_in_fixture "$F1"
report "$RC" "F1 identical sets exit 0 (got $RC)"
report "$(echo "$OUT" | grep -q '2 bundles class-set exact' && echo 0 || echo 1)" "F1 reports both bundles exact"
rm -rf "$F1"

# --- F2: fresh build missing a class fails, naming it -------------------------
F2="$(mktemp -d)"
MISSING_CSS='.bg-gray-100{color:red}.text-sm\:bold{font-weight:700}'
make_fixture "$F2" "$MISSING_CSS"
run_in_fixture "$F2"
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "F2 missing-class exits 1 (got $RC)"
report "$(echo "$OUT" | grep -qF 'md\:flex' && echo 0 || echo 1)" "F2 output names the missing md:flex token"
rm -rf "$F2"

# --- F3: fresh build with an extra class fails, naming it ---------------------
F3="$(mktemp -d)"
EXTRA_CSS="$BASE_CSS.bg-red-500{background:red}"
make_fixture "$F3" "$EXTRA_CSS"
run_in_fixture "$F3"
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "F3 extra-class exits 1 (got $RC)"
report "$(echo "$OUT" | grep -q 'bg-red-500' && echo 0 || echo 1)" "F3 output names the extra bg-red-500 token"
rm -rf "$F3"

# --- F4: missing .fresh file fails, naming the file ---------------------------
F4="$(mktemp -d)"
make_fixture "$F4" "$BASE_CSS"
rm "$F4/dashboardui/assets/dashboard-tw.css.fresh"
run_in_fixture "$F4"
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "F4 missing .fresh exits 1 (got $RC)"
report "$(echo "$OUT" | grep -q 'dashboard-tw.css.fresh' && echo 0 || echo 1)" "F4 names the missing fixture file"
rm -rf "$F4"

# --- F5: formatting-only difference passes (the gate's purpose) ---------------
F5="$(mktemp -d)"
PRETTY_CSS='/* banner */
.bg-gray-100 {
  color: red;
}
.text-sm\:bold {
  font-weight: 700;
}
@media (min-width: 768px) {
  .md\:flex {
    display: flex;
  }
}
'
make_fixture "$F5" "$PRETTY_CSS"
run_in_fixture "$F5"
report "$RC" "F5 formatting-only difference exits 0 (got $RC)"
rm -rf "$F5"

echo ""
if [ "$fail" -eq 0 ]; then
  echo "✓ All check-css-bundle-classes self-tests passed ($pass)"
  exit 0
fi
echo "✗ $fail self-test(s) failed"
exit 1
