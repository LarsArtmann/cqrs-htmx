#!/usr/bin/env bash
# test-wait-tree-quiet.sh — fixture self-test for wait-tree-quiet.sh
#
#   F1  already-quiet repo        -> exit 0 after ~one quiet window
#   F2  dirty tree whole time     -> exit 1 (timeout)
#   F3  commit mid-window         -> keeps waiting, still exits 0
#   F4  outside a git repository  -> exit 2
#
# Windows are 2s with 1s polls; total runtime stays well under 30s.
#
# Usage: ./scripts/test-wait-tree-quiet.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
CHECKER="$SCRIPT_DIR/wait-tree-quiet.sh"

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
  git -C "$1" commit -q --allow-empty -m initial
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "== test-wait-tree-quiet.sh"

# F1 already quiet
r="$tmp/f1"
new_repo "$r"
bash "$CHECKER" --quiet 2 --poll 1 --max-wait 15 >"$tmp/f1.out" 2>&1
rc=$?
report $([ $rc -eq 0 ] && echo 0 || echo 1) "F1 quiet repo exits 0 (got $rc)"

# F2 dirty whole time -> timeout
r="$tmp/f2"
new_repo "$r" && echo x >"$r/dirty.txt"
bash "$CHECKER" --quiet 2 --poll 1 --max-wait 3 >"$tmp/f2.out" 2>&1
rc=$?
ok=1
[ $rc -eq 1 ] || ok=0
grep -q "TIMEOUT" "$tmp/f2.out" || ok=0
report $ok "F2 dirty tree times out with exit 1 (got $rc)"

# F3 commit lands mid-window
r="$tmp/f3"
new_repo "$r"
(
  sleep 1
  git -C "$r" commit -q --allow-empty -m "chore: auto-commit 1 changed file(s) (heuristic)"
) &
bash "$CHECKER" --quiet 3 --poll 1 --max-wait 20 >"$tmp/f3.out" 2>&1
rc=$?
report $([ $rc -eq 0 ] && echo 0 || echo 1) "F3 mid-window commit still exits 0 (got $rc)"
wait

# F4 outside a repo
mkdir -p "$tmp/f4" && cd "$tmp/f4" || exit 1
bash "$CHECKER" >"$tmp/f4.out" 2>&1
rc=$?
report $([ $rc -eq 2 ] && echo 0 || echo 1) "F4 outside repo exits 2 (got $rc)"
cd "$SCRIPT_DIR/.." || exit 1

echo "== $pass passed, $fail failed"
[ "$fail" -eq 0 ]
