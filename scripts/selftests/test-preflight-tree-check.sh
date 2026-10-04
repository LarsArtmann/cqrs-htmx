#!/usr/bin/env bash
# test-preflight-tree-check.sh — fixture self-test for preflight-tree-check.sh
#
# Builds throwaway git repos and exercises the verdicts:
#   F1  clean tree, only old commits                     -> exit 0
#   F2  dirty tree                                       -> exit 1, names "dirty"
#   F3  fresh non-daemon commit                          -> exit 1, names "non-daemon"
#   F4  fresh daemon commit only (churn is expected)     -> exit 0
#   F5  daemon commit burst within the window            -> exit 1, names "commits within"
#   F6  run outside a git repository                     -> exit 2
#
# Usage: ./scripts/test-preflight-tree-check.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
CHECKER="$SCRIPT_DIR/preflight-tree-check.sh"

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
}

old_commit() { # <dir> <msg> — commit dated far in the past
  git -C "$1" commit -q --allow-empty -m "$2" \
    --date="2020-01-01T00:00:00" >/dev/null 2>&1
  GIT_COMMITTER_DATE="2020-01-01T00:00:00" git -C "$1" commit -q --allow-empty --amend -m "$2" >/dev/null 2>&1
}

tmp=$(mktemp -d)
if [ "${KEEP_TMP:-0}" != "1" ]; then trap 'rm -rf "$tmp"' EXIT; else echo "keep: $tmp"; fi

echo "== test-preflight-tree-check.sh"

# F1 clean + old
r="$tmp/f1"
new_repo "$r" && old_commit "$r" "initial"
(cd "$r" && PREFLIGHT_RECENCY_SECONDS=300 bash "$CHECKER") >"$tmp/f1.out" 2>&1
rc=$?
report "$([ "$rc" -eq 0 ] && echo 0 || echo 1)" "F1 clean/old exits 0 (got $rc)"

# F2 dirty
r="$tmp/f2"
new_repo "$r" && old_commit "$r" "initial" && echo x >"$r/uncommitted.txt"
(cd "$r" && PREFLIGHT_RECENCY_SECONDS=300 bash "$CHECKER") >"$tmp/f2.out" 2>&1
rc=$?
ok=0
[ $rc -eq 1 ] || ok=1
grep -q "dirty" "$tmp/f2.out" || ok=1
report $ok "F2 dirty tree exits 1 naming dirty (got $rc)"

# F3 fresh non-daemon commit
r="$tmp/f3"
new_repo "$r" && old_commit "$r" "initial" && git -C "$r" commit -q --allow-empty -m "feat: foreign in-flight work"
(cd "$r" && PREFLIGHT_RECENCY_SECONDS=300 bash "$CHECKER") >"$tmp/f3.out" 2>&1
rc=$?
ok=0
[ $rc -eq 1 ] || ok=1
grep -q "non-daemon" "$tmp/f3.out" || ok=1
report $ok "F3 fresh foreign commit exits 1 (got $rc)"

# F4 fresh daemon commit only
r="$tmp/f4"
new_repo "$r" && old_commit "$r" "initial" && git -C "$r" commit -q --allow-empty -m "chore: auto-commit 2 changed file(s) (heuristic)"
(cd "$r" && PREFLIGHT_RECENCY_SECONDS=300 bash "$CHECKER") >"$tmp/f4.out" 2>&1
rc=$?
report "$([ "$rc" -eq 0 ] && echo 0 || echo 1)" "F4 fresh daemon commit alone exits 0 (got $rc)"

# F5 daemon burst
r="$tmp/f5"
new_repo "$r" && old_commit "$r" "initial"
for i in 1 2 3 4 5; do
  git -C "$r" commit -q --allow-empty -m "chore: auto-commit $i changed file(s) (heuristic)"
done
(cd "$r" && PREFLIGHT_RECENCY_SECONDS=300 PREFLIGHT_MAX_RECENT_COMMITS=3 bash "$CHECKER") >"$tmp/f5.out" 2>&1
rc=$?
ok=0
[ $rc -eq 1 ] || ok=1
grep -q "commits within" "$tmp/f5.out" || ok=1
report $ok "F5 daemon burst exceeds velocity exits 1 (got $rc)"

# F6 outside a repo
mkdir -p "$tmp/f6" && cd "$tmp/f6" || exit 1
bash "$CHECKER" >"$tmp/f6.out" 2>&1
rc=$?
report "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "F6 outside repo exits 2 (got $rc)"
cd "$SCRIPT_DIR/.." || exit 1

echo "== $pass passed, $fail failed"
[ "$fail" -eq 0 ]
