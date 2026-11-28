#!/usr/bin/env bash
# test-check-service-methods.sh — Test cases for check-service-methods.sh
#
# Verifies the *Service method counter:
#   - The default threshold (85) passes with the current codebase
#   - A low threshold triggers failure
#   - The reported count is a valid number
#   - Threshold at exact count passes
#   - Threshold below count fails
#   - A threshold one above the count passes
#   - Every focused sub-service is reported with its own count
#   - A low sub-service ceiling fails and names the type
#
# Usage: ./scripts/selftests/test-check-service-methods.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/check-service-methods.sh"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

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

# run_checker_subservice: same, with an explicit ceiling for the focused
# sub-services (the *Service ceiling stays at its default).
run_checker_subservice() {
  set +e
  OUTPUT=$(SERVICE_METHOD_LIMIT="" SUBSERVICE_METHOD_LIMIT="${1:-}" bash "$CHECKER" 2>&1)
  EXIT_CODE=$?
  set -e
}

echo ""
echo "=== Test Suite: check-service-methods.sh ==="
echo ""

# --- Test 1: Default threshold (85) passes ---
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

# --- Test 7: Every focused sub-service is reported with a numeric count ---
# The expected set is derived from the source the same way the checker derives
# it (receiver discovery), so the case holds as sub-services are added.
run_checker
SUBSERVICE_ROWS=$(echo "$OUTPUT" | grep -cP '^    \w+Service: \d+ methods \(limit \d+\)$' || true)
EXPECTED_TYPES=$(grep -rhoP 'func \(\w+ \*\w+Service\) \w+' --include='*.go' --exclude='*_test.go' "$REPO_ROOT/usermgmt" |
  grep -oP '(?<=\*)\w+Service(?=\))' | sort -u | wc -l)
if [ "$SUBSERVICE_ROWS" -ge 2 ] && [ "$SUBSERVICE_ROWS" -eq "$EXPECTED_TYPES" ] &&
  echo "$OUTPUT" | grep -qP '^    BotService: \d+ methods ' &&
  echo "$OUTPUT" | grep -qP '^    TenantService: \d+ methods '; then
  echo "  PASS: all ${EXPECTED_TYPES} focused sub-service(s) reported with counts (incl. BotService, TenantService)"
  pass=$((pass + 1))
else
  echo "  FAIL: expected ${EXPECTED_TYPES} per-sub-service rows, found ${SUBSERVICE_ROWS}"
  echo "        Output: $OUTPUT"
  fail=$((fail + 1))
fi

# --- Test 8: The facade row stays first (the count parse relies on it) ---
FIRST_COUNT_LINE=$(echo "$OUTPUT" | grep -P 'methods' | head -1)
if echo "$FIRST_COUNT_LINE" | grep -qP '^OK: \d+ methods, \d+ until the decomposition trigger\.$'; then
  echo "  PASS: The *Service verdict is the first 'methods' line"
  pass=$((pass + 1))
else
  echo "  FAIL: first 'methods' line was: $FIRST_COUNT_LINE"
  fail=$((fail + 1))
fi

# --- Test 9: A low sub-service ceiling fails and names the type ---
run_checker_subservice 5
if [ "$EXIT_CODE" -ne 0 ] && echo "$OUTPUT" | grep -qP 'FAIL: \*usermgmt\.\w+Service has \d+ methods'; then
  echo "  PASS: Low sub-service ceiling (5) fails and names the type"
  pass=$((pass + 1))
else
  echo "  FAIL: sub-service ceiling of 5 should fail (exit $EXIT_CODE)"
  echo "        Output: $OUTPUT"
  fail=$((fail + 1))
fi

# --- Test 10: The sub-service ceiling is inclusive like the facade's (-ge) ---
run_checker
BOT_COUNT=$(grep -oP '(?<=^    BotService: )\d+' <<<"$OUTPUT" | head -1)
if [ -n "${BOT_COUNT:-}" ]; then
  run_checker_subservice "$BOT_COUNT"
  if [ "$EXIT_CODE" -ne 0 ]; then
    echo "  PASS: Sub-service ceiling at an exact count (${BOT_COUNT} = BotService) triggers failure"
    pass=$((pass + 1))
  else
    echo "  FAIL: sub-service ceiling of ${BOT_COUNT} should fail for BotService (checker uses -ge)"
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
