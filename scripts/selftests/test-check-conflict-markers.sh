#!/usr/bin/env bash
# test-check-conflict-markers.sh — fixture self-test for check-conflict-markers.sh.
#
# Cases:
#   1. a clean repo passes
#   2. a `<<<<<<<` line at line start fails and names the file
#   3. a bare `=======` line fails
#   4. a `>>>>>>>` line fails
#   5. a marker quoted mid-line (prose) passes
#   6. a marker quoted in a code span on the same line passes
#   7. an UNTRACKED conflicted file passes (only commits can carry the defect)
#   8. a non-repo directory exits 2 (never green)
#   9. every marker line in a multi-marker file is reported
#  10. --staged fails on a marker in the INDEX even after the working tree was
#      repaired (the pre-commit shape: the commit still carries the marker)
#  11. --staged passes when the index is clean but the working tree is dirty
#  12. an unknown argument exits 2 instead of scanning something else
#
# Usage: bash scripts/selftests/test-check-conflict-markers.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-conflict-markers.sh"

WORK=$(mktemp -d /tmp/test-conflict-markers-XXXXXX)
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
  CONFLICT_MARKERS_ROOT="$1" bash "$TOOL" 2>&1
}

run_staged() {
  CONFLICT_MARKERS_ROOT="$1" bash "$TOOL" --staged 2>&1
}

new_repo() {
  local r="$1"
  mkdir -p "$r"
  git -C "$r" init -q
}

# --- 1. clean repo passes ---------------------------------------------------
r="$WORK/clean"
new_repo "$r"
printf 'alpha\nbeta\n' >"$r/file.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "clean repo passes"; else bad "clean repo" "rc=$rc out=$out"; fi

# --- 2. <<<<<<< at line start fails ----------------------------------------
r="$WORK/open"
new_repo "$r"
printf 'one\n<<<<<<< Updated upstream\ntwo\n' >"$r/file.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'file.txt'; then
  ok "<<<<<<< line fails and names the file"
else
  bad "<<<<<<< line" "rc=$rc out=$out"
fi

# --- 3. bare ======= fails --------------------------------------------------
r="$WORK/sep"
new_repo "$r"
printf 'one\n=======\ntwo\n' >"$r/file.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ]; then ok "======= line fails"; else bad "======= line" "rc=$rc out=$out"; fi

# --- 4. >>>>>>> fails -------------------------------------------------------
r="$WORK/close"
new_repo "$r"
printf 'one\n>>>>>>> Stashed changes\ntwo\n' >"$r/file.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ]; then ok ">>>>>>> line fails"; else bad ">>>>>>> line" "rc=$rc out=$out"; fi

# --- 5. marker quoted mid-line passes --------------------------------------
r="$WORK/prose"
new_repo "$r"
printf '%s\n' 'The file held a <<<<<<< Updated upstream marker inside prose.' >"$r/file.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "mid-line quote passes"; else bad "mid-line quote" "rc=$rc out=$out"; fi

# --- 6. a quoted marker after indentation passes ---------------------------
r="$WORK/codespan"
new_repo "$r"
printf '%s\n' '1. **docs/X was committed broken** (markers: <<<<<<< A ======= B >>>>>>> S).' >"$r/file.md"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "code-span quote passes"; else bad "code-span quote" "rc=$rc out=$out"; fi

# --- 7. untracked conflicted file passes ----------------------------------
r="$WORK/untracked"
new_repo "$r"
printf 'alpha\n' >"$r/tracked.txt"
git -C "$r" add -A
printf '<<<<<<< HEAD\nx\n=======\ny\n>>>>>>> other\n' >"$r/untracked.txt"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then ok "untracked conflicted file passes"; else bad "untracked" "rc=$rc out=$out"; fi

# --- 8. non-repo exits 2 ---------------------------------------------------
r="$WORK/notarepo"
mkdir -p "$r"
printf '<<<<<<< HEAD\n' >"$r/file.txt"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "non-repo exits 2"; else bad "non-repo" "rc=$rc out=$out"; fi

# --- 9. two markers in one file are both reported -------------------------
r="$WORK/two"
new_repo "$r"
printf '<<<<<<< HEAD\na\n=======\nb\n>>>>>>> other\n' >"$r/file.txt"
git -C "$r" add -A
out="$(run "$r")"
rc=$?
hits="$(printf '%s\n' "$out" | grep -c 'file.txt:')"
if [ "$rc" -eq 1 ] && [ "$hits" -eq 3 ]; then
  ok "three marker lines in one file all reported"
else
  bad "multi-marker file" "rc=$rc hits=$hits out=$out"
fi

# --- 10. --staged fails on an INDEX marker after the tree was repaired ------
# The committed-conflict class exactly: the marker reached the index, the file
# was then "fixed" in place, and a plain working-tree scan would go green while
# the commit still carried the marker.
r="$WORK/staged-dirty-index"
new_repo "$r"
printf 'one\n<<<<<<< Updated upstream\ntwo\n' >"$r/file.txt"
git -C "$r" add -A
printf 'one\ntwo\n' >"$r/file.txt" # working tree repaired, index still dirty
out="$(run_staged "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'file.txt'; then
  ok "--staged fails on an index marker the working tree no longer shows"
else
  bad "--staged dirty index" "rc=$rc out=$out"
fi

# --- 11. --staged passes with a clean index and a dirty working tree --------
r="$WORK/staged-clean-index"
new_repo "$r"
printf 'alpha\n' >"$r/file.txt"
git -C "$r" add -A
printf 'alpha\n<<<<<<< HEAD\n' >"$r/file.txt" # unstaged marker only
out="$(run_staged "$r")"
rc=$?
if [ "$rc" -eq 0 ]; then
  ok "--staged passes when only the working tree carries a marker"
else
  bad "--staged clean index" "rc=$rc out=$out"
fi

# --- 12. unknown argument exits 2 ------------------------------------------
r="$WORK/bad-arg"
new_repo "$r"
printf 'alpha\n' >"$r/file.txt"
git -C "$r" add -A
out="$(CONFLICT_MARKERS_ROOT="$r" bash "$TOOL" --bogus 2>&1)"
rc=$?
if [ "$rc" -eq 2 ]; then ok "unknown argument exits 2"; else bad "unknown argument" "rc=$rc out=$out"; fi

echo
if [ "$fail" -eq 0 ]; then
  echo "✓ conflict-markers self-test: $pass/$pass cases passed"
  exit 0
fi
echo "✗ conflict-markers self-test: $fail case(s) failed ($pass passed)" >&2
exit 1
