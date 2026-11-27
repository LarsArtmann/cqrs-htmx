#!/usr/bin/env bash
# test-check-debug-files.sh — fixture self-test for check-debug-files.sh.
#
# Cases:
#   1. a clean repo passes
#   2. a tracked zz_scratch.go fails and names the file
#   3. a nested-dir zz_ scratch fails (basename-anchored)
#   4. a name merely CONTAINING zz mid-basename (fizzbuzz.go) passes
#   5. an UNTRACKED zz_ file passes in tree mode (only commits carry it)
#   6. --staged fails on a zz_ file in the INDEX even after the working tree
#      copy was deleted (the pre-commit shape)
#   7. --staged passes when the index is clean but an untracked zz_ exists
#   8. a non-repo directory exits 2 (never green)
#   9. an unknown argument exits 2 instead of scanning something else
#  10. a linked worktree (where .git is a file) passes
#
# Usage: bash scripts/selftests/test-check-debug-files.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-debug-files.sh"

WORK=$(mktemp -d /tmp/test-debug-files-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0
ok() {
  echo "  ok $1"
  pass=$((pass + 1))
}
bad() {
  echo "  FAIL $1: $2" >&2
  fail=$((fail + 1))
}

run() {
  DEBUG_FILES_ROOT="$1" bash "$TOOL" 2>&1
}

run_staged() {
  DEBUG_FILES_ROOT="$1" bash "$TOOL" --staged 2>&1
}

new_repo() {
  local r="$1"
  mkdir -p "$r"
  git -C "$r" init -q
}

# --- 1. clean repo passes ---------------------------------------------------
r="$WORK/clean"
new_repo "$r"
printf 'alpha\n' >"$r/main.go"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "clean repo passes"; else bad "clean repo" "rc=$rc out=$out"; fi

# --- 2. tracked zz_scratch.go fails and names the file ----------------------
r="$WORK/scratch"
new_repo "$r"
printf 'package main\n' >"$r/zz_scratch.go"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'zz_scratch.go'; then
  ok "tracked zz_scratch fails and names the file"
else
  bad "tracked zz_scratch" "rc=$rc out=$out"
fi

# --- 3. nested-dir zz_ scratch fails ----------------------------------------
r="$WORK/nested"
new_repo "$r"
mkdir -p "$r/cmd/tool"
printf 'package main\n' >"$r/cmd/tool/zz_dump.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'cmd/tool/zz_dump.txt'; then
  ok "nested zz_ scratch fails with its path"
else
  bad "nested zz_ scratch" "rc=$rc out=$out"
fi

# --- 4. name merely containing zz mid-basename passes ------------------------
r="$WORK/contains"
new_repo "$r"
printf 'package main\n' >"$r/fizzbuzz.go"
printf 'package main\n' >"$r/buzzer.go"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "mid-basename zz passes (basename-anchored)"; else bad "mid-basename zz" "rc=$rc out=$out"; fi

# --- 5. untracked zz_ file passes in tree mode -------------------------------
r="$WORK/untracked"
new_repo "$r"
printf 'package main\n' >"$r/main.go"
git -C "$r" add -A
printf 'scratch\n' >"$r/zz_untracked.txt"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "untracked zz_ passes in tree mode"; else bad "untracked zz_" "rc=$rc out=$out"; fi

# --- 6. --staged catches a zz_ in the index after the worktree copy is gone --
r="$WORK/staged-remnant"
new_repo "$r"
printf 'package main\n' >"$r/main.go"
printf 'scratch\n' >"$r/zz_staged.txt"
git -C "$r" add -A
rm "$r/zz_staged.txt"
out="$(run_staged "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'zz_staged.txt'; then
  ok "--staged catches the index copy after worktree deletion"
else
  bad "--staged remnant" "rc=$rc out=$out"
fi

# --- 7. --staged passes with a clean index but untracked scratch -------------
r="$WORK/staged-clean"
new_repo "$r"
printf 'package main\n' >"$r/main.go"
git -C "$r" add -A
printf 'scratch\n' >"$r/zz_loose.txt"
out="$(run_staged "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "--staged passes with clean index"; else bad "--staged clean" "rc=$rc out=$out"; fi

# --- 8. non-repo directory exits 2 -------------------------------------------
r="$WORK/notarepo"
mkdir -p "$r"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "non-repo exits 2 (never green)"; else bad "non-repo" "rc=$rc out=$out"; fi

# --- 9. unknown argument exits 2 ---------------------------------------------
r="$WORK/argcheck"
new_repo "$r"
printf 'package main\n' >"$r/main.go"
git -C "$r" add -A
out="$(DEBUG_FILES_ROOT="$r" bash "$TOOL" --bogus 2>&1)"
rc=$?
if [ "$rc" -eq 2 ]; then ok "unknown argument exits 2"; else bad "unknown argument" "rc=$rc out=$out"; fi

# --- 10. a linked worktree passes (.git is a file there) ---------------------
r="$WORK/linked-main"
new_repo "$r"
printf 'package main\n' >"$r/main.go"
git -C "$r" add -A
git -C "$r" -c user.name=selftest -c user.email=selftest@example.invalid commit -q -m init
git -C "$r" worktree add -q "$WORK/linked-wt" -b linked
out="$(run "$WORK/linked-wt")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "linked worktree passes"; else bad "linked worktree" "rc=$rc out=$out"; fi

echo ""
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ] || exit 1
echo "OK: all fixtures behaved"
