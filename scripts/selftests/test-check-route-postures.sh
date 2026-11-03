#!/usr/bin/env bash
# test-check-route-postures.sh — fixture self-test for check-route-postures.sh
# (offline: stub go binary + fixture module trees; no network, no real compile).
# Covers: a passing pin, a failing pin, the empty-match trap, and the REAL
# tree (the parity claim this gate exists to enforce).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK="$SCRIPT_DIR/../checks/check-route-postures.sh"
TMP=$(mktemp -d /tmp/route-postures-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT

failures=0

note_fail() {
  echo "FAIL $1"
  failures=$((failures + 1))
}

# Stub go: honors GO_TEST_MODE to decide the outcome. The gate always calls
# `go test -run <pattern> -count=1 -v ./...` inside the module dir.
cat >"$TMP/go" <<'STUB'
#!/usr/bin/env bash
# shellcheck disable=SC2124  # capture the full invocation for the mode check
args="$@"
case "${GO_TEST_MODE:-pass}" in
pass)
  echo "--- PASS: TestRoutes_Fixture (0.00s)"
  echo "PASS"
  exit 0
  ;;
fail)
  echo "--- FAIL: TestRoutes_Fixture (0.00s)"
  echo "    fixture_test.go:9: manifest drifted"
  echo "FAIL"
  exit 1
  ;;
empty)
  echo "testing: warning: no tests to run"
  echo "PASS"
  exit 0
  ;;
esac
exit 9
STUB
chmod +x "$TMP/go"

mkdir -p "$TMP/fixture-mod"
printf 'module fixture.mod/x\n\ngo 1.27\n' >"$TMP/fixture-mod/go.mod"

# --- case 1: passing pins --------------------------------------------------
out="$(ROUTE_POSTURE_ROOT="$TMP" ROUTE_POSTURE_MODULES="fixture-mod" ROUTE_POSTURE_GO="$TMP/go" GO_TEST_MODE=pass bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '1 manifest test(s) green'; then
  echo "ok   passing pin is green with a nonzero test count"
else
  note_fail "passing pin: rc=$rc out=$out"
fi

# --- case 2: failing pin ---------------------------------------------------
out="$(ROUTE_POSTURE_ROOT="$TMP" ROUTE_POSTURE_MODULES="fixture-mod" ROUTE_POSTURE_GO="$TMP/go" GO_TEST_MODE=fail bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'manifest pins FAILED'; then
  echo "ok   failing pin is red and names the module"
else
  note_fail "failing pin: rc=$rc out=$out"
fi

# --- case 3: the empty-match trap ------------------------------------------
out="$(ROUTE_POSTURE_ROOT="$TMP" ROUTE_POSTURE_MODULES="fixture-mod" ROUTE_POSTURE_GO="$TMP/go" GO_TEST_MODE=empty bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'ZERO manifest tests'; then
  echo "ok   zero matched tests is a loud red, not a silent green"
else
  note_fail "empty-match trap: rc=$rc out=$out"
fi

# --- case 4: missing module dir --------------------------------------------
out="$(ROUTE_POSTURE_ROOT="$TMP" ROUTE_POSTURE_MODULES="nope-mod" ROUTE_POSTURE_GO="$TMP/go" bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'nope-mod'; then
  echo "ok   missing module dir is red"
else
  note_fail "missing module: rc=$rc out=$out"
fi

# --- case 5: the REAL tree (the parity claim) ------------------------------
out="$(bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ]; then
  echo "ok   real tree green ($out)"
else
  note_fail "real tree: rc=$rc out=$out"
fi

echo ""
if [ "$failures" -gt 0 ]; then
  echo "route-postures gate self-test: $failures case(s) FAILED"
  exit 1
fi
echo "route-postures gate self-test: all cases green"
exit 0
