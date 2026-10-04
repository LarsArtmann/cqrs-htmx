#!/usr/bin/env bash
# test-check-status-rows.sh — fixture self-test for check-status-rows.py
#
# Asserts the row gate's policy: PARTIAL rows (cells within one row disagree)
# fail; deliberately-mixed tables (struck = done, unstruck = open) are counted
# but pass; tildes inside inline code spans never count as strikethrough.
#
# Usage: ./scripts/test-check-status-rows.sh
# Exit: 0 = all cases pass, 1 = at least one case fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-status-rows.py"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    [ -n "${DETAIL:-}" ] && echo "        ${DETAIL}"
    fail=$((fail + 1))
  fi
}

run_gate() {
  RC=0
  OUTPUT="$(python3 "$CHECKER" "$@" 2>&1)" || RC=$?
}

assert_rc() { # <label> <expected-rc>
  DETAIL="rc=$RC (expected $2)"
  printf '        | %s\n' "${OUTPUT//$'\n'/$'\n'        | }"
  if [ "$RC" -eq "$2" ]; then report 0 "$1"; else report 1 "$1"; fi
}

assert_contains() { # <label> <needle>
  DETAIL="output did not contain: $2"
  if printf '%s' "$OUTPUT" | grep -qF "$2"; then report 0 "$1"; else report 1 "$1"; fi
}

echo ""
echo "=== Test Suite: check-status-rows.py ==="
echo ""

# --- Test 1: uniformly struck table passes ---
cat >"$TMPDIR/all-struck.md" <<'EOF'
| # | Task |
| --- | --- |
| ~~1~~ | ~~Do the thing~~ |
| ~~2~~ | ~~Do another thing~~ |
EOF
run_gate "$TMPDIR/all-struck.md"
assert_rc "all-struck table passes" 0

# --- Test 2: PARTIAL row fails, with exact count and exact file name ---
cat >"$TMPDIR/partial.md" <<'EOF'
| # | Task | Owner |
| --- | --- | --- |
| ~~1~~ | ~~Done~~ | lars |
EOF
run_gate "$TMPDIR/partial.md"
assert_rc "PARTIAL row fails" 1
assert_contains "offender file named exactly" "partial.md: table at line 1 has 1 PARTIAL row(s)"
assert_contains "offending row listed with its own line" "line 3: | ~~1~~ | ~~Done~~ | lars |"
assert_contains "exact total in summary" "1 PARTIAL row(s) across 1 file(s)"

# --- Test 2b: multiple PARTIAL rows count exactly (regression: the first
# check_file implementation counted the per-table header line too and
# printed 2 for a single PARTIAL row) ---
cat >"$TMPDIR/partial-x2.md" <<'EOF'
| # | Task | Owner |
| --- | --- | --- |
| ~~1~~ | ~~Done~~ | lars |
| ~~2~~ | done text | lars |
EOF
run_gate "$TMPDIR/partial-x2.md"
assert_rc "two PARTIAL rows fail" 1
assert_contains "table header reports exactly 2" "has 2 PARTIAL row(s)"
assert_contains "summary reports exactly 2" "2 PARTIAL row(s) across 1 file(s)"

# --- Test 3: deliberately-mixed table passes but is reported ---
cat >"$TMPDIR/mixed.md" <<'EOF'
| # | Task |
| --- | --- |
| ~~1~~ | ~~Done already~~ |
| 2 | Still open |
EOF
run_gate "$TMPDIR/mixed.md"
assert_rc "mixed table passes" 0
assert_contains "mixed table counted" "1 deliberately-mixed table(s)"

# --- Test 4: ~~ inside inline code spans never counts ---
cat >"$TMPDIR/code-span.md" <<'EOF'
| # | Task |
| --- | --- |
| 1 | Literal `~~` marker documented |
EOF
run_gate "$TMPDIR/code-span.md"
assert_rc "code-span tildes are not strikethrough" 0
assert_contains "clean table reported as such" "free of PARTIAL rows"

# --- Test 5: header/separator rows are exempt ---
cat >"$TMPDIR/header-only.md" <<'EOF'
| # | Task |
| --- | --- |
EOF
run_gate "$TMPDIR/header-only.md"
assert_rc "header-only table passes" 0

# --- Test 6: a directory argument expands to *.md ---
mkdir -p "$TMPDIR/corpus"
cat >"$TMPDIR/corpus/2026-09-19_10-00_ok.md" <<'EOF'
| # | Task |
| --- | --- |
| ~~1~~ | ~~Done~~ |
EOF
run_gate "$TMPDIR/corpus"
assert_rc "directory argument expands" 0
assert_contains "directory scan counted the file" "1 file(s) free of PARTIAL rows"

# --- Test 7: a missing path fails loudly ---
run_gate "$TMPDIR/does-not-exist.md"
assert_rc "missing path fails" 1
assert_contains "missing path named" "MISSING"

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
  echo "FAIL: $fail test(s) failed."
  exit 1
fi

echo "All tests passed."
