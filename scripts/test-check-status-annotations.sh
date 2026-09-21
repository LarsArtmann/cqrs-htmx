#!/usr/bin/env bash
# test-check-status-annotations.sh — self-test for check-status-annotations.sh
#
# Builds throwaway fixture trees (one archived-status directory per case) and
# asserts the gate's exit code and message for each policy branch:
#   - a gated report with `~~` + `ANNOTATED` passes
#   - a gated report missing `~~` fails with MISSING STRIKETHROUGH
#   - a gated report with `~~` but no `ANNOTATED` fails with MISSING ANNOTATED
#   - the epoch boundary (exactly ANNOTATION_EPOCH) is gated, not exempt
#   - a legacy report (older than the epoch) passes unannotated
#   - a file without a YYYY-MM-DD prefix fails with NO-DATE
#   - a missing directory fails with a clear message
#
# Usage: ./scripts/test-check-status-annotations.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-status-annotations.sh"

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
    echo "        ${DETAIL:-}"
    fail=$((fail + 1))
  fi
}

mkdir -p "$TMPDIR/scripts" "$TMPDIR/docs/status/archived"
cp "$CHECKER" "$TMPDIR/scripts/check-status-annotations.sh"

# Fixture dirs are rebuilt per case so cases never leak into each other.
reset_fixtures() {
  rm -rf "$TMPDIR/docs/status/archived"
  mkdir -p "$TMPDIR/docs/status/archived"
}

# run_gate <extra-arg...>; sets RC and OUTPUT
run_gate() {
  RC=0
  OUTPUT="$( (cd "$TMPDIR" && bash scripts/check-status-annotations.sh "$@") 2>&1)" || RC=$?
}

assert_rc() { # <label> <expected-rc>
  DETAIL="rc=$RC (expected $2)"
  if [ "$RC" -eq "$2" ]; then
    report 0 "$1"
  else
    report 1 "$1"
  fi
}

assert_contains() { # <label> <needle>
  DETAIL="output did not contain: $2"
  if printf '%s' "$OUTPUT" | grep -qF "$2"; then
    report 0 "$1"
  else
    report 1 "$1"
  fi
}

echo ""
echo "=== Test Suite: check-status-annotations.sh ==="
echo ""

# --- Test 1: fully annotated gated report passes ---
reset_fixtures
cat >"$TMPDIR/docs/status/archived/2026-09-19_12-00_annotated.md" <<'EOF'
# Report

> ANNOTATED 2026-09-21

1. ~~Did the thing (verified 2026-09-21).~~
EOF
run_gate
assert_rc "gated + annotated passes" 0
assert_contains "green summary printed" "✓ annotation gate"

# --- Test 2: gated report missing strikethrough fails ---
reset_fixtures
cat >"$TMPDIR/docs/status/archived/2026-09-19_12-01_no-strike.md" <<'EOF'
# Report

> ANNOTATED 2026-09-21

1. Did the thing but nobody struck it.
EOF
run_gate
assert_rc "gated without ~~ fails" 1
assert_contains "strikethrough offender named" "MISSING STRIKETHROUGH"

# --- Test 3: gated report with ~~ but no ANNOTATED blockquote fails ---
reset_fixtures
cat >"$TMPDIR/docs/status/archived/2026-09-19_12-02_no-quote.md" <<'EOF'
# Report

1. ~~Did the thing (verified 2026-09-21).~~
EOF
run_gate
assert_rc "gated without ANNOTATED fails" 1
assert_contains "blockquote offender named" "MISSING ANNOTATED BLOCKQUOTE"

# --- Test 4: epoch boundary (exactly 2026-09-09) is gated ---
reset_fixtures
cat >"$TMPDIR/docs/status/archived/2026-09-09_10-00_epoch.md" <<'EOF'
# Report

1. Unannotated item on the boundary day.
EOF
run_gate
assert_rc "boundary date is gated, not exempt" 1
assert_contains "boundary offender named" "MISSING STRIKETHROUGH"

# --- Test 5: legacy report (pre-epoch) passes unannotated ---
reset_fixtures
cat >"$TMPDIR/docs/status/archived/2026-06-17_10-00_legacy.md" <<'EOF'
# Report

1. An old item with no annotation at all.
EOF
run_gate
assert_rc "legacy report is exempt" 0
assert_contains "legacy count reported" "legacy-exempt"

# --- Test 6: file without a date prefix fails ---
reset_fixtures
cat >"$TMPDIR/docs/status/archived/no-date-prefix.md" <<'EOF'
# Report

> ANNOTATED 2026-09-21

1. ~~Struck.~~
EOF
run_gate
assert_rc "undated file fails" 1
assert_contains "undated offender named" "NO-DATE"

# --- Test 7: missing directory fails clearly ---
run_gate docs/status/does-not-exist
assert_rc "missing dir fails" 1
assert_contains "missing dir named" "directory not found"

# --- Test 8: explicit dir argument is honored ---
reset_fixtures
mkdir -p "$TMPDIR/other-archive"
cat >"$TMPDIR/other-archive/2026-09-20_09-00_ok.md" <<'EOF'
# Report

> ANNOTATED 2026-09-21

1. ~~Struck (verified).~~
EOF
run_gate other-archive
assert_rc "explicit dir argument honored" 0
assert_contains "explicit dir scanned" "1 gated report"

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
  echo "FAIL: $fail test(s) failed."
  exit 1
fi

echo "All tests passed."
