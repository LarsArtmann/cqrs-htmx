#!/usr/bin/env bash
# test-check-service-methods.sh — Test cases for check-service-methods.sh
#
# Verifies the *Service method counter:
#   - The default threshold (80) passes with the current codebase
#   - A low threshold triggers failure
#   - The reported count is a valid number
#
# Usage: ./scripts/test-check-service-methods.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-service-methods.sh"

pass=0
fail=0

echo ""
echo "=== Test Suite: check-service-methods.sh ==="
echo ""

# --- Test 1: Default threshold (80) passes ---
OUTPUT=$(bash "$CHECKER" 2>&1) || true
EXIT_CODE=$?
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
OUTPUT=$(SERVICE_METHOD_LIMIT=1 bash "$CHECKER" 2>&1) || true
EXIT_CODE=$?
if [ "$EXIT_CODE" -ne 0 ]; then
	echo "  PASS: Low threshold (1) correctly triggers failure"
	pass=$((pass + 1))
else
	echo "  FAIL: Threshold of 1 should trigger failure"
	fail=$((fail + 1))
fi

# --- Test 4: Threshold equal to current count passes ---
if [ -n "$COUNT" ]; then
	OUTPUT=$(SERVICE_METHOD_LIMIT="$COUNT" bash "$CHECKER" 2>&1) || true
	EXIT_CODE=$?
	if [ "$EXIT_CODE" -eq 0 ]; then
		echo "  PASS: Threshold at exact count ($COUNT) passes"
		pass=$((pass + 1))
	else
		echo "  FAIL: Threshold at exact count ($COUNT) should pass"
		fail=$((fail + 1))
	fi
fi

# --- Test 5: Threshold below current count fails ---
if [ -n "$COUNT" ]; then
	BELOW=$((COUNT - 1))
	OUTPUT=$(SERVICE_METHOD_LIMIT="$BELOW" bash "$CHECKER" 2>&1) || true
	EXIT_CODE=$?
	if [ "$EXIT_CODE" -ne 0 ]; then
		echo "  PASS: Threshold below count ($BELOW) triggers failure"
		pass=$((pass + 1))
	else
		echo "  FAIL: Threshold below count ($BELOW) should fail"
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
