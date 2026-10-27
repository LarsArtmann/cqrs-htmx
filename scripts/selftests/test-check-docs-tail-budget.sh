#!/usr/bin/env bash
# test-check-docs-tail-budget.sh — fixture self-test for check-docs-tail-budget.sh.
#
# Cases:
#   1. tail within budget                              -> exit 0, "within budget"
#   2. tail over budget (advisory)                     -> exit 0, warns
#   3. tail over budget with --strict                  -> exit 1
#   4. README.md is excluded from the count
#   5. --budget override honored
#   6-10. ratchet mode: at baseline / over baseline / below baseline / missing / malformed
#
# Usage: bash scripts/selftests/test-check-docs-tail-budget.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-docs-tail-budget.sh"

WORK=$(mktemp -d /tmp/test-docs-tail-budget-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

make_root() {
  mkdir -p "$1/docs/status"
  echo "# Status Reports" >"$1/docs/status/README.md"
  local i=0
  while [ "$i" -lt "$2" ]; do
    echo "# report $i" >"$1/docs/status/2026-01-0${i}_report.md"
    i=$((i + 1))
  done
}

run_gate() {
  TAIL_BUDGET_ROOT="$1" bash "$GATE" "${@:2}" 2>&1
}

# Case 1: within budget.
make_root "$WORK/within" 2
if out=$(run_gate "$WORK/within") && printf '%s' "$out" | grep -q "within budget"; then
  echo "  ok 1: within budget passes"
  pass=$((pass + 1))
else
  echo "  FAIL 1: within-budget tail should pass; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 2: over budget, advisory.
make_root "$WORK/over" 5
if out=$(run_gate "$WORK/over"); then
  if printf '%s' "$out" | grep -q "exceed the budget"; then
    echo "  ok 2: over budget warns but stays green (advisory)"
    pass=$((pass + 1))
  else
    echo "  FAIL 2: over budget did not warn; output:" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
  fi
else
  echo "  FAIL 2: advisory over-budget run should exit 0" >&2
  fail=$((fail + 1))
fi

# Case 3: over budget with --strict.
if out=$(run_gate "$WORK/over" --strict); then
  echo "  FAIL 3: --strict over budget should exit 1; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
else
  echo "  ok 3: --strict over budget fails"
  pass=$((pass + 1))
fi

# Case 4: README.md excluded — 1 report + README is within a budget of 1.
make_root "$WORK/readme" 1
if out=$(run_gate "$WORK/readme" --budget 1) && printf '%s' "$out" | grep -q "1 live report"; then
  echo "  ok 4: README.md excluded from the count"
  pass=$((pass + 1))
else
  echo "  FAIL 4: README.md should not count; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 5: --budget override — budget 0 with 1 report is over.
make_root "$WORK/budget" 1
if out=$(run_gate "$WORK/budget" --budget 0) && printf '%s' "$out" | grep -q "exceed the budget of 0"; then
  echo "  ok 5: --budget override honored"
  pass=$((pass + 1))
else
  echo "  FAIL 5: --budget 0 should report over budget; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# --- ratchet mode (blocking) -------------------------------------------------

# Case 6: ratchet at the baseline is green.
make_root "$WORK/ratchet-at" 3
echo 3 >"$WORK/ratchet-at/docs/status/TAIL_BASELINE"
if out=$(run_gate "$WORK/ratchet-at" --ratchet) && printf '%s' "$out" | grep -q "ratchet"; then
  echo "  ok 6: ratchet at baseline passes"
  pass=$((pass + 1))
else
  echo "  FAIL 6: ratchet at baseline should pass; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 7: growth past the baseline is a hard failure.
echo 2 >"$WORK/ratchet-at/docs/status/TAIL_BASELINE"
if out=$(run_gate "$WORK/ratchet-at" --ratchet); then
  echo "  FAIL 7: ratchet over baseline should fail; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
else
  if printf '%s' "$out" | grep -q "exceed the ratchet baseline"; then
    echo "  ok 7: ratchet over baseline fails"
    pass=$((pass + 1))
  else
    echo "  FAIL 7: wrong failure text; output:" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
  fi
fi

# Case 8: shrinking below the baseline warns but passes (lower the baseline).
make_root "$WORK/ratchet-below" 2
echo 5 >"$WORK/ratchet-below/docs/status/TAIL_BASELINE"
if out=$(run_gate "$WORK/ratchet-below" --ratchet) && printf '%s' "$out" | grep -q "BELOW the baseline"; then
  echo "  ok 8: ratchet below baseline warns and passes"
  pass=$((pass + 1))
else
  echo "  FAIL 8: ratchet below baseline should warn; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 9: missing baseline is a failure (no silent pass).
make_root "$WORK/ratchet-none" 2
if out=$(run_gate "$WORK/ratchet-none" --ratchet); then
  echo "  FAIL 9: missing baseline should fail; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
else
  if printf '%s' "$out" | grep -q "needs .*TAIL_BASELINE"; then
    echo "  ok 9: missing baseline fails"
    pass=$((pass + 1))
  else
    echo "  FAIL 9: wrong failure text; output:" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
  fi
fi

# Case 10: malformed baseline is a failure.
echo "many" >"$WORK/ratchet-at/docs/status/TAIL_BASELINE"
if out=$(run_gate "$WORK/ratchet-at" --ratchet); then
  echo "  FAIL 10: malformed baseline should fail; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
else
  if printf '%s' "$out" | grep -q "single integer"; then
    echo "  ok 10: malformed baseline fails"
    pass=$((pass + 1))
  else
    echo "  FAIL 10: wrong failure text; output:" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
  fi
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-check-docs-tail-budget: $fail case(s) FAILED"
  exit 1
fi
echo "test-check-docs-tail-budget: all cases green"
