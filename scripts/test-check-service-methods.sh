#!/usr/bin/env bash
# test-check-service-methods.sh — Test cases for check-service-methods.sh
#
# Verifies the *Service method counter:
#   - The default threshold (80) passes with the current codebase
#   - A low threshold triggers failure
#   - The reported count is a valid number
#   - Threshold at exact count passes
#   - Threshold below count fails
#
# Usage: ./scripts/test-check-service-methods.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-service-methods.sh"

pass=0
fail=0

# run_checker: runs the checker with an optional SERVICE_METHOD_LIMIT env,
# captures stdout+stderr into OUTPUT, and stores the real exit code in EXIT_CODE.
# Uses set +e because set -e would kill the script when the checker exits non-zero.
run_checker() {
	set +e
	OUTPUT=$(SERVICE_METHOD_LIMIT="${1:-}" bash "$CHECKER" 2>&1)
	EXIT_CODE=$?
	set -e
}

echo ""
echo "=== Test Suite: check-service-methods.sh ==="
echo ""

# --- Test 1: Default threshold (80) passes ---
run_checker
if [ "$EXIT_CODE" -eq 0 ]; then
	echo "  PASS: Default threshold passes"
	pass=$((pass + 1))
else
	echo "  FAIL: Default threshold should pass (exit $EXIT_CODE)"
	echo "        Output: $OUTPUT"
	fail=$((fail + 1))
fi

# --- Test 2: Output contains a numeric count ---
COUNT=$(echo "$OUTPUT" | grep -oP '\d+(?= methods)' | head -1)
if [ -n "$COUNT" ] && [ "$COUNT" -gt 0 ] 2>/dev/null; then
	echo "  PASS: Reported count is a positive number ($COUNT)"
	pass=$((pass + 1))
else
	echo "  FAIL: Could not extract a positive method count from output"
	fail=$((fail + 1))
fi

# --- Test 3: Low threshold (1) triggers failure ---
run_checker 1
if [ "$EXIT_CODE" -ne 0 ]; then
	echo "  PASS: Low threshold (1) correctly triggers failure"
	pass=$((pass + 1))
else
	echo "  FAIL: Threshold of 1 should trigger failure"
	fail=$((fail + 1))
fi

# --- Test 4: Threshold equal to current count FAILS (checker uses -ge) ---
if [ -n "${COUNT:-}" ]; then
	run_checker "$COUNT"
	if [ "$EXIT_CODE" -ne 0 ]; then
		echo "  PASS: Threshold at exact count ($COUNT) triggers failure (ceiling reached)"
		pass=$((pass + 1))
	else
		echo "  FAIL: Threshold at exact count ($COUNT) should fail (checker uses -ge)"
		fail=$((fail + 1))
	fi
fi

# --- Test 5: Threshold below current count fails ---
if [ -n "${COUNT:-}" ]; then
	BELOW=$((COUNT - 1))
	run_checker "$BELOW"
	if [ "$EXIT_CODE" -ne 0 ]; then
		echo "  PASS: Threshold below count ($BELOW) triggers failure"
		pass=$((pass + 1))
	else
		echo "  FAIL: Threshold below count ($BELOW) should fail"
		fail=$((fail + 1))
	fi
fi

# --- Test 6: Threshold one above current count passes ---
if [ -n "${COUNT:-}" ]; then
	ABOVE=$((COUNT + 1))
	run_checker "$ABOVE"
	if [ "$EXIT_CODE" -eq 0 ]; then
		echo "  PASS: Threshold above count ($ABOVE) passes"
		pass=$((pass + 1))
	else
		echo "  FAIL: Threshold above count ($ABOVE) should pass"
		fail=$((fail + 1))
	fi
fi

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
	echo "FAIL: $fail test(s) failed."
	exit 1
fi

echo "All tests passed."
