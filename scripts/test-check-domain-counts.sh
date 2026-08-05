#!/usr/bin/env bash
# test-check-domain-counts.sh — Test cases for check-domain-counts.sh
#
# Verifies the domain count drift detector:
#   - Default run passes (docs are in sync with source)
#   - The reported event/command counts are positive numbers
#   - The counts match known totals (21 events, 20 commands)
#   - Drift is detected using an isolated fake repo (no real files modified)
#
# Usage: ./scripts/test-check-domain-counts.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-domain-counts.sh"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass=0
fail=0

echo ""
echo "=== Test Suite: check-domain-counts.sh ==="
echo ""

# Helper: run the real checker and capture exit code safely under set -e
run_checker() {
	set +e
	OUTPUT=$(bash "$CHECKER" 2>&1)
	EXIT_CODE=$?
	set -e
}

# --- Test 1: Default run passes (docs in sync) ---
run_checker
if [ "$EXIT_CODE" -eq 0 ]; then
	echo "  PASS: Default run passes (docs in sync)"
	pass=$((pass + 1))
else
	echo "  FAIL: Default run should pass (docs in sync)"
	echo "        Output: $OUTPUT"
	fail=$((fail + 1))
fi

# --- Test 2: Output contains event and command counts ---
# The checker outputs "Events:   NN (...)" and "Commands: NN (...)".
# Use grep -P with the actual format; wrap in set +e because grep
# returns 1 on no-match and pipefail would kill the script.
set +eo pipefail
EVENT_COUNT=$(echo "$OUTPUT" | grep -oP '^Events:\s+\K\d+')
COMMAND_COUNT=$(echo "$OUTPUT" | grep -oP '^Commands:\s+\K\d+')
set -eo pipefail

if [ -n "${EVENT_COUNT:-}" ] && [ "${EVENT_COUNT:-0}" -gt 0 ] 2>/dev/null; then
	echo "  PASS: Event count is a positive number ($EVENT_COUNT)"
	pass=$((pass + 1))
else
	echo "  FAIL: Could not extract positive event count"
	fail=$((fail + 1))
fi

if [ -n "${COMMAND_COUNT:-}" ] && [ "${COMMAND_COUNT:-0}" -gt 0 ] 2>/dev/null; then
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

# --- Test 4: Drift detection using an isolated fake repo ---
# Create a fake repo with known event/command counts in source files,
# then inject wrong numbers in a doc and verify the checker detects it.
FAKE_REPO="$TMPDIR/fake-repo"
mkdir -p "$FAKE_REPO/identity-model"

# Source files: 2 event payloads, 2 commands
cat >"$FAKE_REPO/identity-model/events.go" <<'GOEOF'
package identitymodel
type UserCreatedPayload struct{}
type UserDeletedPayload struct{}
GOEOF

cat >"$FAKE_REPO/identity-model/commands.go" <<'GOEOF'
package identitymodel
type CreateUserCmd struct{}
type DeleteUserCmd struct{}
GOEOF

# Create a copy of the checker that uses the fake repo as REPO_ROOT
sed 's|^REPO_ROOT=.*|REPO_ROOT="'"$FAKE_REPO"'"|' "$CHECKER" >"$TMPDIR/fake-checker.sh"
chmod +x "$TMPDIR/fake-checker.sh"

# 4a: In-sync doc should pass
echo "We have 2 event payload structs and 2 command structs." >"$FAKE_REPO/AGENTS.md"
set +e
OUTPUT=$(bash "$TMPDIR/fake-checker.sh" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -eq 0 ]; then
	echo "  PASS: Fake repo with in-sync counts passes"
	pass=$((pass + 1))
else
	echo "  FAIL: Fake repo with correct counts should pass (exit $EXIT_CODE)"
	echo "        Output: $OUTPUT"
	fail=$((fail + 1))
fi

# 4b: Drifted event count should fail
echo "We have 99 event payload structs and 2 command structs." >"$FAKE_REPO/AGENTS.md"
set +e
OUTPUT=$(bash "$TMPDIR/fake-checker.sh" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
	echo "  PASS: Drifted event count (99 vs 2) correctly detected"
	pass=$((pass + 1))
else
	echo "  FAIL: Drifted event count should be detected"
	fail=$((fail + 1))
fi

# 4c: Drifted command count should fail
echo "We have 2 event payload structs and 99 command structs." >"$FAKE_REPO/AGENTS.md"
set +e
OUTPUT=$(bash "$TMPDIR/fake-checker.sh" 2>&1)
EXIT_CODE=$?
set -e
if [ "$EXIT_CODE" -ne 0 ]; then
	echo "  PASS: Drifted command count (99 vs 2) correctly detected"
	pass=$((pass + 1))
else
	echo "  FAIL: Drifted command count should be detected"
	fail=$((fail + 1))
fi

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
	echo "FAIL: $fail test(s) failed."
	exit 1
fi

echo "All tests passed."
