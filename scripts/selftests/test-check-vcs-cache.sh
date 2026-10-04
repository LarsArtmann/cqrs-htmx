#!/usr/bin/env bash
# test-check-vcs-cache.sh — self-test for check-vcs-cache.sh
#
# Builds throwaway GOMODCACHE-shaped fixture trees (offline; `git init
# --bare` only) and exercises the three verdicts:
#   F1  healthy bare repo (origin set)           -> PASS, "1" checked
#   F2  bare repo with origin REMOVED (the exact
#       /mnt/buildcache corruption class)        -> FAIL, repair hint shown
#   F3  missing cache/vcs directory              -> PASS, nothing to do
#
# Usage: ./scripts/test-check-vcs-cache.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-vcs-cache.sh"

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

echo ""
echo "=== Test Suite: check-vcs-cache.sh ==="
echo ""

# --- F1: healthy bare repo passes -------------------------------------------
F1="$(mktemp -d)"
mkdir -p "$F1/cache/vcs"
git -C "$F1/cache/vcs" init -q --bare github.com-larsartmann-go-datastar
git -C "$F1/cache/vcs/github.com-larsartmann-go-datastar" remote add origin https://github.com/larsartmann/go-datastar.git
OUT="$(VCS_CACHE_ROOT="$F1/cache/vcs" bash "$CHECKER" 2>&1)"
rc=$?
report "$rc" "F1 healthy bare repo exits 0 (got $rc)"
report "$(echo "$OUT" | grep -q "All 1 VCS cache entries" && echo 0 || echo 1)" "F1 reports 1 entry checked"
rm -rf "$F1"

# --- F2: origin-less bare repo fails with the repair hint -------------------
F2="$(mktemp -d)"
mkdir -p "$F2/cache/vcs"
git -C "$F2/cache/vcs" init -q --bare github.com-larsartmann-go-datastar
OUT="$(VCS_CACHE_ROOT="$F2/cache/vcs" bash "$CHECKER" 2>&1)"
rc=$?
report "$([ "$rc" -eq 1 ] && echo 0 || echo 1)" "F2 origin-less bare repo exits 1 (got $rc)"
report "$(echo "$OUT" | grep -q "NO remote.origin.url" && echo 0 || echo 1)" "F2 names the broken entry"
report "$(echo "$OUT" | grep -q "remote add origin" && echo 0 || echo 1)" "F2 prints the repair command"
rm -rf "$F2"

# --- F3: missing cache/vcs is a no-op pass ----------------------------------
F3="$(mktemp -d)"
OUT="$(VCS_CACHE_ROOT="$F3/cache/vcs" bash "$CHECKER" 2>&1)"
rc=$?
report "$rc" "F3 missing cache/vcs exits 0 (got $rc)"
report "$(echo "$OUT" | grep -q "nothing to do" && echo 0 || echo 1)" "F3 explains the no-op"
rm -rf "$F3"

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
  echo "FAIL: $fail test(s) failed."
  exit 1
fi

echo "All tests passed."
