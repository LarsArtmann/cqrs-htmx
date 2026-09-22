#!/usr/bin/env bash
# test-check-dep-budgets.sh — self-test for check-dep-budgets.sh
#
# Pins the go.mod dep-counting awk against fixtures via the DEP_BUDGETS_ROOT
# test hook, with focus on the 2026-09-22 regression class: standalone
# comment lines inside require blocks (e.g. //cqrs-lint:ignore(V006)
# suppressions) MUST NOT count as dependencies. Nothing prevents a rewrite
# of the awk from reintroducing that — this fixture corpus does.
#
# Fixtures (root module only; every other budget key skips when its
# fixture go.mod is absent — the "." budget of 19 is the probe):
#   F1  18 real deps + 2 indirect + 3 standalone comment lines -> PASS at
#       exactly "18 deps" (comment/indirect miscounting pushes 21/20 over
#       the 19 budget and flips the verdict)
#   F2  20 real deps -> OVER BUDGET, exit 1
#   F3  empty require block + comment lines only -> "0 deps", PASS
#
# Usage: ./scripts/test-check-dep-budgets.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-dep-budgets.sh"

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

# write_fixture <dir> <real-deps> <indirect> <comments> <single-line-requires>
write_fixture() {
  local dir="$1" real="$2" indirect="$3" comments="$4" single="$5"
  mkdir -p "$dir"
  {
    echo "module example.com/fixture"
    echo ""
    echo "go 1.27.1"
    echo ""
    echo "require ("
    local i
    for ((i = 1; i <= real; i++)); do
      printf '\texample.com/real%02d v1.0.%d\n' "$i" "$i"
    done
    for ((i = 1; i <= indirect; i++)); do
      printf '\texample.com/ind%02d v2.0.%d // indirect\n' "$i" "$i"
    done
    for ((i = 1; i <= comments; i++)); do
      printf '\t//cqrs-lint:ignore(V006) per-module trains, not lockstep — comment %d\n' "$i"
    done
    echo ")"
    for ((i = 1; i <= single; i++)); do
      printf 'require example.com/single%02d v3.0.%d\n' "$i" "$i"
    done
  } >"$dir/go.mod"
}

echo ""
echo "=== Test Suite: check-dep-budgets.sh ==="
echo ""

# --- F1: comment lines + indirect deps must not count -----------------------
F1="$(mktemp -d)"
write_fixture "$F1" 18 2 3 1
OUT="$(DEP_BUDGETS_ROOT="$F1" bash "$CHECKER" 2>&1)"
rc=$?
report "$rc" "F1 18 block + 1 single-line real deps (2 indirect + 3 comments excluded) exit 0 (got $rc)"
report "$(echo "$OUT" | grep -q "example.com/fixture: 19 deps (budget: 19)" && echo 0 || echo 1)" "F1 counts exactly 19 deps (18 block + 1 single-line; comment/indirect exclusion intact)"
rm -rf "$F1"

# --- F2: over budget fails ---------------------------------------------------
F2="$(mktemp -d)"
write_fixture "$F2" 20 0 0 0
OUT="$(DEP_BUDGETS_ROOT="$F2" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F2 20 real deps over budget 19 -> exit 1 (got $rc)"
report "$(echo "$OUT" | grep -q "OVER BUDGET" && echo 0 || echo 1)" "F2 reports OVER BUDGET"
rm -rf "$F2"

# --- F3: empty require block + comments only --------------------------------
F3="$(mktemp -d)"
write_fixture "$F3" 0 0 4 0
OUT="$(DEP_BUDGETS_ROOT="$F3" bash "$CHECKER" 2>&1)"
rc=$?
report "$rc" "F3 zero real deps exit 0 (got $rc)"
report "$(echo "$OUT" | grep -q "example.com/fixture: 0 deps" && echo 0 || echo 1)" "F3 counts exactly 0 deps"
rm -rf "$F3"

# --- F4: the real repo still passes -----------------------------------------
OUT="$(bash "$CHECKER" 2>&1)"
rc=$?
report "$rc" "F4 real repo within all budgets (got $rc)"

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
  echo "FAIL: $fail test(s) failed."
  exit 1
fi

echo "All tests passed."
