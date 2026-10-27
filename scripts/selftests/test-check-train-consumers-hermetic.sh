#!/usr/bin/env bash
# test-check-train-consumers-hermetic.sh — fixture self-test for
# check-train-consumers-hermetic.sh (offline: stub pipeline, no toolchain).
#
#   F1  no changed go.mod/go.sum vs base      -> exit 0, explicit zero count
#   F2  changed module + green stub pipeline  -> exit 0, module named
#   F3  changed module + failing stub         -> exit 1, module + FAILED named
#   F4  unresolvable base ref                 -> exit 0 with SKIP message
#   F5  scripts/testdata changes are excluded -> zero-count pass despite dirt
#
# Usage: ./scripts/selftests/test-check-train-consumers-hermetic.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/check-train-consumers-hermetic.sh"

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

new_repo() { # <dir>
  git init -q -b main "$1" || return 1
  git -C "$1" config user.email test@example.com
  git -C "$1" config user.name test
  git -C "$1" config commit.gpgsign false
  mkdir -p "$1/somemodule" "$1/scripts/testdata/poison-fixture"
  printf 'module example.com/root\n\ngo 1.21\n' >"$1/go.mod"
  printf 'module example.com/somemodule\n\ngo 1.21\n' >"$1/somemodule/go.mod"
  printf 'module example.com/poison\n\ngo 1.21\n' >"$1/scripts/testdata/poison-fixture/go.mod"
  git -C "$1" add -A
  git -C "$1" commit -q -m initial
}

GREEN_STUB="echo 'stub pipeline ran for candidate'; exit 0"
RED_STUB="echo 'stub pipeline simulated hermetic failure'; exit 1"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "== test-check-train-consumers-hermetic.sh"

# F1 zero changed modules
r="$tmp/f1"
new_repo "$r"
(
  cd "$r" || exit 1
  TRAIN_HERMETIC_BASE=HEAD \
    TRAIN_HERMETIC_PIPELINE="$GREEN_STUB" \
    bash "$CHECKER"
) >"$tmp/f1.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "0 modules" "$tmp/f1.out" || ok=1
report $ok "F1 clean tree -> explicit zero-count pass (got $rc)"

# F2 changed module, green stub
r="$tmp/f2"
new_repo "$r"
printf '\nrequire example.com/dep v1.0.0\n' >>"$r/somemodule/go.mod"
(
  cd "$r" || exit 1
  TRAIN_HERMETIC_BASE=HEAD \
    TRAIN_HERMETIC_PIPELINE="$GREEN_STUB" \
    bash "$CHECKER"
) >"$tmp/f2.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "=== somemodule ===" "$tmp/f2.out" || ok=1
grep -q "1/1 module(s) green" "$tmp/f2.out" || ok=1
report $ok "F2 changed module passes hermetically (got $rc)"

# F3 changed module, red stub
r="$tmp/f3"
new_repo "$r"
printf '\nrequire example.com/dep v1.0.0\n' >>"$r/somemodule/go.mod"
(
  cd "$r" || exit 1
  TRAIN_HERMETIC_BASE=HEAD \
    TRAIN_HERMETIC_PIPELINE="$RED_STUB" \
    bash "$CHECKER"
) >"$tmp/f3.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "hermetic pipeline FAILED" "$tmp/f3.out" || ok=1
grep -q "publish AFTER this is green" "$tmp/f3.out" || ok=1
report $ok "F3 red module blocks publish with rc 1 (got $rc)"

# F4 unresolvable base -> explicit SKIP, rc 0
r="$tmp/f4"
new_repo "$r"
printf '\nrequire example.com/dep v1.0.0\n' >>"$r/somemodule/go.mod"
(
  cd "$r" || exit 1
  TRAIN_HERMETIC_BASE=no-such-ref-exists \
    TRAIN_HERMETIC_PIPELINE="$GREEN_STUB" \
    bash "$CHECKER"
) >"$tmp/f4.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "SKIP" "$tmp/f4.out" || ok=1
report $ok "F4 unresolvable base -> explicit SKIP rc 0 (got $rc)"

# F5 testdata changes excluded
r="$tmp/f5"
new_repo "$r"
printf '\nrequire example.com/poison v9.9.9\n' >>"$r/scripts/testdata/poison-fixture/go.mod"
(
  cd "$r" || exit 1
  TRAIN_HERMETIC_BASE=HEAD \
    TRAIN_HERMETIC_PIPELINE="$GREEN_STUB" \
    bash "$CHECKER"
) >"$tmp/f5.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "0 modules" "$tmp/f5.out" || ok=1
report $ok "F5 scripts/testdata changes excluded -> zero-count pass (got $rc)"

echo ""
echo "test-check-train-consumers-hermetic: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
exit 0
