#!/usr/bin/env bash
# test-check-css-bundles.sh — self-test for check-css-bundles.sh
#
# Builds throwaway fixture trees (offline; copies of the real bundles,
# surgically corrupted per case) and exercises the verdicts:
#   F1  pristine copies of the real bundles        -> PASS, 2 reported
#   F2  missing bundle                             -> FAIL, names the file
#   F3  pretty multi-line rewrite (hook class)     -> FAIL, names the file
#   F4  near-empty ~8KB truncation (incident class)-> FAIL, names the file
#   F5  canary-stripped (empty scan, size ok)      -> FAIL, names canaries
#   F6  empty file                                 -> FAIL, names the file
#
# Usage: ./scripts/test-check-css-bundles.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-css-bundles.sh"
REPO="$(cd "$SCRIPT_DIR/.." && pwd)"

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

make_fixture() { # <dir> — pristine bundle copies
  mkdir -p "$1/adminui/assets" "$1/dashboardui/assets"
  cp "$REPO/adminui/assets/admin-tw.css" "$1/adminui/assets/"
  cp "$REPO/dashboardui/assets/dashboard-tw.css" "$1/dashboardui/assets/"
}

echo ""
echo "=== Test Suite: check-css-bundles.sh ==="
echo ""

# --- F1: pristine real bundles pass ------------------------------------------
F1="$(mktemp -d)"
make_fixture "$F1"
OUT="$(CHECK_CSS_BUNDLES_ROOT="$F1" bash "$CHECKER" 2>&1)"
rc=$?
report "$rc" "F1 pristine bundles exit 0 (got $rc)"
report "$(echo "$OUT" | grep -q "2 bundles canonical" && echo 0 || echo 1)" "F1 reports both bundles canonical"
rm -rf "$F1"

# --- F2: missing bundle fails, naming the file --------------------------------
F2="$(mktemp -d)"
make_fixture "$F2"
rm "$F2/dashboardui/assets/dashboard-tw.css"
OUT="$(CHECK_CSS_BUNDLES_ROOT="$F2" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F2 missing bundle exits 1 (got $rc)"
report "$(echo "$OUT" | grep -q "dashboard-tw.css: MISSING" && echo 0 || echo 1)" "F2 names the missing bundle"
rm -rf "$F2"

# --- F3: pretty multi-line rewrite fails (the hook class) ---------------------
F3="$(mktemp -d)"
make_fixture "$F3"
python3 -c 'import sys; d = open(sys.argv[1]).read(); open(sys.argv[2], "w").write(d.replace(";", ";\n"))' \
  "$REPO/dashboardui/assets/dashboard-tw.css" "$F3/dashboardui/assets/dashboard-tw.css"
OUT="$(CHECK_CSS_BUNDLES_ROOT="$F3" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F3 pretty multi-line rewrite exits 1 (got $rc)"
report "$(echo "$OUT" | grep -q "not the canonical minified form" && echo 0 || echo 1)" "F3 names the minified-form failure"
rm -rf "$F3"

# --- F4: near-empty truncation fails (the 2026-09-22 incident class) ----------
F4="$(mktemp -d)"
make_fixture "$F4"
head -c 8192 "$REPO/adminui/assets/admin-tw.css" >"$F4/adminui/assets/admin-tw.css"
OUT="$(CHECK_CSS_BUNDLES_ROOT="$F4" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F4 8KB truncation exits 1 (got $rc)"
report "$(echo "$OUT" | grep -q "near-empty bundle" && echo 0 || echo 1)" "F4 names the near-empty class"
rm -rf "$F4"

# --- F5: canary-stripped bundle fails (empty scan, size otherwise ok) ---------
F5="$(mktemp -d)"
make_fixture "$F5"
sed 's/--tc-sidebar[^;]*;//g; s/\.bg-green-100[^}]*}//g' \
  "$REPO/dashboardui/assets/dashboard-tw.css" >"$F5/dashboardui/assets/dashboard-tw.css"
OUT="$(CHECK_CSS_BUNDLES_ROOT="$F5" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F5 canary-stripped exits 1 (got $rc)"
report "$(echo "$OUT" | grep -q "canary utilities absent" && echo 0 || echo 1)" "F5 names the canary failure"
rm -rf "$F5"

# --- F6: empty file fails ------------------------------------------------------
F6="$(mktemp -d)"
make_fixture "$F6"
: >"$F6/adminui/assets/admin-tw.css"
OUT="$(CHECK_CSS_BUNDLES_ROOT="$F6" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F6 empty bundle exits 1 (got $rc)"
rm -rf "$F6"

echo ""
if [ "$fail" -gt 0 ]; then
  echo "✗ $fail test(s) failed ($pass passed)"
  exit 1
fi
echo "✓ All $pass tests passed"
