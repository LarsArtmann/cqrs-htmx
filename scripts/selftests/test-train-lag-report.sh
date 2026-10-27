#!/usr/bin/env bash
# test-train-lag-report.sh — fixture self-test for train-lag-report.sh (offline)
#
#   F1  aligned gate (rc 0 + tally)        -> exit 0, "aligned" verdict
#   F2  lag gate (rc 3 + tally + recipe)   -> exit 0 advisory, recipe shown
#   F3  lag gate + TRAIN_LAG_REPORT_STRICT -> exit 3
#   F4  broken gate (rc 1)                 -> exit 1 propagated
#   F5  tally-less output                  -> exit 1 GUARD MISS (gotcha 2)
#
# Usage: ./scripts/selftests/test-train-lag-report.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/train-lag-report.sh"

pass=0
fail=0
report() {
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

stub() { # <name> <body>
  printf '#!/usr/bin/env bash\n%s\n' "$2" >"$tmp/$1.sh"
  chmod +x "$tmp/$1.sh"
}

ALIGNED='echo "Checked 843 internal requires: 0 unpublished, 0 replace-exempted, 0 train lag."; exit 0'
LAGGED='cat <<X
TRAIN LAG github.com/larsartmann/go-foo/v4: current v1.0.0, published v1.1.0
Checked 843 internal requires: 0 unpublished, 0 replace-exempted, 1 train lag.
FIX RECIPE — one exact-anchor sweep per lagging module
  scripts/tools/bump-dep.sh '"'"'larsartmann/go-foo/v4$'"'"' v1.1.0
X
exit 3'
BROKEN='cat <<X
UNPUBLISHED: github.com/larsartmann/go-foo/v4@v1.1.0 required by root but never tagged
Checked 843 internal requires: 1 unpublished, 0 replace-exempted, 0 train lag.
X
exit 2'
NO_TALLY='echo "something went wrong quietly"; exit 0'

echo "== test-train-lag-report.sh"

r="$tmp/f1"
git init -q -b main "$r"
stub aligned "$ALIGNED"
(cd "$r" && TRAIN_LAG_REPORT_GATE_CMD="bash $tmp/aligned.sh" bash "$CHECKER") >"$tmp/f1.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "aligned — 0 lag" "$tmp/f1.out" || ok=1
report $ok "F1 aligned -> exit 0 (got $rc)"

r="$tmp/f2"
git init -q -b main "$r"
stub lagged "$LAGGED"
(cd "$r" && TRAIN_LAG_REPORT_GATE_CMD="bash $tmp/lagged.sh" bash "$CHECKER") >"$tmp/f2.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "EARLY WARNING" "$tmp/f2.out" || ok=1
grep -q "FIX RECIPE" "$tmp/f2.out" || ok=1
grep -q "bump-dep.sh" "$tmp/f2.out" || ok=1
report $ok "F2 lag advisory -> exit 0 with recipe (got $rc)"

r="$tmp/f3"
git init -q -b main "$r"
(cd "$r" && TRAIN_LAG_REPORT_STRICT=1 TRAIN_LAG_REPORT_GATE_CMD="bash $tmp/lagged.sh" bash "$CHECKER") >"$tmp/f3.out" 2>&1
rc=$?
report "$([ "$rc" -eq 3 ] && echo 0 || echo 1)" "F3 strict propagates lag rc 3 (got $rc)"

r="$tmp/f4"
git init -q -b main "$r"
stub broken "$BROKEN"
(cd "$r" && TRAIN_LAG_REPORT_GATE_CMD="bash $tmp/broken.sh" bash "$CHECKER") >"$tmp/f4.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 2 ] || ok=1
grep -q "never mask a broken gate" "$tmp/f4.out" || ok=1
report $ok "F4 broken gate rc 2 propagated (got $rc)"

r="$tmp/f5"
git init -q -b main "$r"
stub notally "$NO_TALLY"
(cd "$r" && TRAIN_LAG_REPORT_GATE_CMD="bash $tmp/notally.sh" bash "$CHECKER") >"$tmp/f5.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "GUARD MISS" "$tmp/f5.out" || ok=1
report $ok "F5 tally-less output -> GUARD MISS rc 1 (got $rc)"

echo ""
echo "test-train-lag-report: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
exit 0
