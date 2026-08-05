#!/usr/bin/env bash
# test-check-domain-counts.sh — Test cases for check-domain-counts.sh
#
# Verifies the domain count drift detector:
#   - Default run passes (docs are in sync with source)
#   - The reported event/command counts are positive numbers
#   - Inflated count phrasing in a doc would be detected
#
# Usage: ./scripts/test-check-domain-counts.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-domain-counts.sh"

pass=0
fail=0

echo ""
echo "=== Test Suite: check-domain-counts.sh ==="
echo ""

# --- Test 1: Default run passes (docs in sync) ---
OUTPUT=$(bash "$CHECKER" 2>&1) || true
EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 0 ]; then
	echo "  PASS: Default run passes (docs in sync)"
	pass=$((pass + 1))
else
	echo "  FAIL: Default run should pass (docs in sync)"
	echo "        Output: $OUTPUT"
	fail=$((fail + 1))
fi

# --- Test 2: Output contains event and command counts ---
EVENT_COUNT=$(echo "$OUTPUT" | grep -oP '\d+(?= event)' | head -1)
COMMAND_COUNT=$(echo "$OUTPUT" | grep -oP '\d+(?= command)' | head -1)
if [ -n "$EVENT_COUNT" ] && [ "$EVENT_COUNT" -gt 0 ] 2>/dev/null; then
	echo "  PASS: Event count is a positive number ($EVENT_COUNT)"
	pass=$((pass + 1))
else
	echo "  FAIL: Could not extract positive event count"
	fail=$((fail + 1))
fi
if [ -n "$COMMAND_COUNT" ] && [ "$COMMAND_COUNT" -gt 0 ] 2>/dev/null; then
	echo "  PASS: Command count is a positive number ($COMMAND_COUNT)"
	pass=$((pass + 1))
else
	echo "  FAIL: Could not extract positive command count"
	fail=$((fail + 1))
fi

# --- Test 3: Verify the counts match known totals (21 events, 20 commands) ---
if [ "${EVENT_COUNT:-0}" -eq 21 ] 2>/dev/null; then
	echo "  PASS: Event count is 21 (matches known total)"
	pass=$((pass + 1))
else
	echo "  FAIL: Event count should be 21, got ${EVENT_COUNT:-?}"
	fail=$((fail + 1))
fi
if [ "${COMMAND_COUNT:-0}" -eq 20 ] 2>/dev/null; then
	echo "  PASS: Command count is 20 (matches known total)"
	pass=$((pass + 1))
else
	echo "  FAIL: Command count should be 20, got ${COMMAND_COUNT:-?}"
	fail=$((fail + 1))
fi

# --- Test 4: Temporarily inject drift into a doc, verify detection ---
# We modify AGENTS.md temporarily with a wrong count, run the checker,
# then restore it. This tests the drift detection path.
AGENTS_FILE="$SCRIPT_DIR/../AGENTS.md"
if [ -f "$AGENTS_FILE" ]; then
	cp "$AGENTS_FILE" "$AGENTS_FILE.bak"
	trap 'cp "$AGENTS_FILE.bak" "$AGENTS_FILE" 2>/dev/null; rm -f "$AGENTS_FILE.bak"' EXIT

	# Inject a wrong count: "99 events / 99 commands"
	sed -i 's/21 events \/ 20 commands/99 events \/ 99 commands/g' "$AGENTS_FILE"

	OUTPUT=$(bash "$CHECKER" 2>&1) || true
	EXIT_CODE=$?

	# Restore immediately
	cp "$AGENTS_FILE.bak" "$AGENTS_FILE"

	if [ "$EXIT_CODE" -ne 0 ]; then
		echo "  PASS: Drift in AGENTS.md correctly detected"
		pass=$((pass + 1))
	else
		echo "  FAIL: Drift in AGENTS.md should be detected"
		fail=$((fail + 1))
	fi
else
	echo "  SKIP: AGENTS.md not found for drift injection test"
fi

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
	echo "FAIL: $fail test(s) failed."
	exit 1
fi

echo "All tests passed."
