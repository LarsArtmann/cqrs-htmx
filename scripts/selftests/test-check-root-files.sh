#!/usr/bin/env bash
# test-check-root-files.sh — fixture self-test for check-root-files.sh.
#
# Cases:
#   1. baseline matching the tracked set passes --strict
#   2. an unbaselined root file fails --strict and names it
#   3. advisory mode passes on drift
#   4. --update-baseline writes the current set (and then passes)
#   5. missing baseline fails even in advisory mode
#   6. *.go files at root are invisible to the gate
#   7. nested files are invisible (depth-1 only)
#   8. a removed file drifts the baseline (removal must lower it)
#
# Usage: bash scripts/selftests/test-check-root-files.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-root-files.sh"

WORK=$(mktemp -d /tmp/test-root-files-XXXXXX)
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

# fixture_repo <dir> — a git repo with a baseline-worthy root set, staged.
fixture_repo() {
  local r="$1"
  mkdir -p "$r/subdir"
  git -C "$r" init -q
  printf 'module example.com/x\n' >"$r/go.mod"
  printf '# doc\n' >"$r/README.md"
  printf 'package x\n' >"$r/x.go"
  printf 'nested\n' >"$r/subdir/nested.txt"
  git -C "$r" add go.mod README.md x.go subdir/nested.txt
}

# run <repo> <baseline> <args...>
run() {
  local r="$1" b="$2"
  shift 2
  ROOT_FILES_GATE_ROOT="$r" ROOT_FILES_BASELINE="$b" bash "$TOOL" "$@" 2>&1
}

# 1. matching baseline passes
R="$WORK/case1"
fixture_repo "$R"
B="$WORK/case1.baseline"
printf 'README.md\ngo.mod\n' | LC_ALL=C sort >"$B"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "matches the baseline"; then ok "1-match-passes"; else bad "1-match-passes" "rc=$rc $out"; fi

# 2. unbaselined root file fails and names it
printf 'x\n' >"$R/stray.json"
git -C "$R" add stray.json
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "stray.json"; then ok "2-stray-fails"; else bad "2-stray-fails" "rc=$rc $out"; fi

# 3. advisory mode passes on drift
out=$(run "$R" "$B")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "advisory"; then ok "3-advisory-passes"; else bad "3-advisory-passes" "rc=$rc $out"; fi

# 4. --update-baseline captures the stray, then passes
out=$(run "$R" "$B" --update-baseline)
rc=$?
if [ "$rc" -eq 0 ] && grep -q "stray.json" "$B"; then
  out=$(run "$R" "$B" --strict)
  rc=$?
  if [ "$rc" -eq 0 ]; then ok "4-update-baseline"; else bad "4-update-baseline" "still failing after update: $out"; fi
else
  bad "4-update-baseline" "rc=$rc baseline did not capture stray.json"
fi

# 5. missing baseline fails even in advisory mode
rm -f "$B"
out=$(run "$R" "$B")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "missing baseline"; then ok "5-missing-baseline"; else bad "5-missing-baseline" "rc=$rc $out"; fi

# restore the baseline for the remaining cases
run "$R" "$B" --update-baseline >/dev/null

# 6. new *.go file at root is invisible
printf 'package x\n' >"$R/y.go"
git -C "$R" add y.go
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 0 ]; then ok "6-go-files-invisible"; else bad "6-go-files-invisible" "rc=$rc $out"; fi

# 7. nested file is invisible
printf 'nested\n' >"$R/subdir/other.txt"
git -C "$R" add subdir/other.txt
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 0 ]; then ok "7-nested-invisible"; else bad "7-nested-invisible" "rc=$rc $out"; fi

# 8. removing a baselined file drifts (baseline has an extra entry)
git -C "$R" rm -q --cached stray.json
rm -f "$R/stray.json"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "stray.json"; then ok "8-removal-drifts"; else bad "8-removal-drifts" "rc=$rc $out"; fi

echo "root-files self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
