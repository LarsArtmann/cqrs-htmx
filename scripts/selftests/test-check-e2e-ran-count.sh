#!/usr/bin/env bash
# test-check-e2e-ran-count.sh — fixture self-test for check-e2e-ran-count.sh.
#
# Cases:
#   1. every listed spec has a result                  -> exit 0
#   2. one spec result-less (serial cascade)           -> exit 1, names the spec
#   3. --expect-unrun 1 absorbs exactly one            -> exit 0
#   4. --expect-unrun 2 with one unrun                 -> exit 0 (allowance is a ceiling)
#   5. an intended test.skip() WRITES a skipped result -> counts as RUN
#   6. missing report file                             -> exit 2
#   7. no arguments                                    -> exit 2
#
# Usage: bash scripts/selftests/test-check-e2e-ran-count.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-e2e-ran-count.sh"

WORK=$(mktemp -d /tmp/test-e2e-ran-count-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

expect() {
  local want_rc="$1" want_grep="$2" desc="$3" rc out
  shift 3
  out="$("$@" 2>&1)"
  rc=$?
  if [ "$rc" -eq "$want_rc" ] && echo "$out" | grep -q "$want_grep"; then
    echo "  ok: $desc"
    PASS=$((PASS + 1))
  else
    echo "  FAIL: $desc (rc=$rc want=$want_rc; output did not match '$want_grep')" >&2
    echo "$out" | head -5 >&2
    FAIL=$((FAIL + 1))
  fi
}

# A minimal playwright JSON report: two suites, three specs. The shape
# mirrors `playwright test --reporter=json`: suites may nest; specs carry
# tests[] with results[] (an intended skip still writes a skipped result).
cat >"$WORK/all-ran.json" <<'EOF'
{
  "suites": [
    {
      "title": "root-htmx.spec.ts",
      "file": "tests/root-htmx.spec.ts",
      "specs": [
        {
          "title": "partial vs full",
          "file": "tests/root-htmx.spec.ts",
          "tests": [{ "projectName": "chromium", "results": [{ "status": "passed" }] }]
        },
        {
          "title": "a deliberately skipped probe",
          "file": "tests/root-htmx.spec.ts",
          "tests": [{ "projectName": "chromium", "results": [{ "status": "skipped" }] }]
        }
      ]
    },
    {
      "title": "bridges.spec.ts",
      "file": "tests/bridges.spec.ts",
      "specs": [
        {
          "title": "console guard",
          "file": "tests/bridges.spec.ts",
          "tests": [{ "projectName": "chromium", "results": [{ "status": "passed" }] }]
        }
      ]
    }
  ]
}
EOF

# Same report but the console-guard spec produced NO result (cascade).
cat >"$WORK/one-unrun.json" <<'EOF'
{
  "suites": [
    {
      "title": "bridges.spec.ts",
      "file": "tests/bridges.spec.ts",
      "specs": [
        {
          "title": "health dashboard renders",
          "file": "tests/bridges.spec.ts",
          "tests": [{ "projectName": "chromium", "results": [{ "status": "failed" }] }]
        },
        {
          "title": "console guard",
          "file": "tests/bridges.spec.ts",
          "tests": [{ "projectName": "chromium", "results": [] }]
        }
      ]
    }
  ]
}
EOF

echo "== check-e2e-ran-count self-test =="

expect 0 "3 listed specs, 0 produced no result" \
  "case 1: all specs ran (incl. intended skip)" \
  bash "$GATE" "$WORK/all-ran.json"

expect 1 "1 produced no result" \
  "case 2: one result-less spec fails" \
  bash "$GATE" "$WORK/one-unrun.json"

expect 1 "console guard" \
  "case 2b: the failing run NAMES the unrun spec" \
  bash "$GATE" "$WORK/one-unrun.json"

expect 0 "expected: 1" \
  "case 3: --expect-unrun 1 absorbs exactly one" \
  bash "$GATE" "$WORK/one-unrun.json" --expect-unrun 1

expect 0 "expected: 2" \
  "case 4: allowance is a ceiling, not an exact match" \
  bash "$GATE" "$WORK/one-unrun.json" --expect-unrun 2

expect 0 "3 listed specs, 0 produced no result" \
  "case 5: an intended test.skip() writes a skipped RESULT and counts as run" \
  bash "$GATE" "$WORK/all-ran.json"

expect 2 "report file missing" \
  "case 6: missing report file" \
  bash "$GATE" "$WORK/does-not-exist.json"

expect 2 "report file missing" \
  "case 7: no arguments" \
  bash "$GATE"

echo
if [ "$FAIL" -eq 0 ]; then
  echo "test-check-e2e-ran-count: $PASS/$PASS assertions PASSED"
  exit 0
fi

echo "test-check-e2e-ran-count: $FAIL FAILED (of $((PASS + FAIL)))" >&2
exit 1
