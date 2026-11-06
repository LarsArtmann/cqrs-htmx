#!/usr/bin/env bash
# test-check-root-size.sh — fixture self-test for check-root-size.sh.
#
# Cases:
#   1. a tree at its baseline passes and prints the canonical sentence
#   2. LOC growth fails and names both numbers
#   3. a new production file fails even when LOC is unchanged-ish (file ratchet)
#   4. a shrink passes and hints that the baseline can be lowered
#   5. --advisory downgrades growth to a warning (exit 0)
#   6. a missing baseline exits 2 (never green)
#   7. a non-integer baseline exits 2
#   8. a baseline with a trailing field exits 2
#   9. --update-baseline rewrites to the measured value
#  10. test files and _templ.go are NOT counted (production-only measurement)
#  11. an empty root (no production .go) exits 2
#
# Usage: bash scripts/selftests/test-check-root-size.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-root-size.sh"

WORK=$(mktemp -d /tmp/test-check-root-size-XXXXXX)
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

# run <root> [args...] — baseline defaults to <root>/baseline
run() {
  local r="$1"
  shift
  ROOT_SIZE_ROOT="$r" ROOT_SIZE_BASELINE="$r/baseline" bash "$TOOL" "$@" 2>&1
}

# mkroot <dir> <lines-per-file> <files> — production files of <lines> lines each
mkroot() {
  local r="$1" lines="$2" n="$3" i
  mkdir -p "$r"
  for i in $(seq 1 "$n"); do
    yes 'package root' | head -n "$lines" >"$r/file$i.go"
  done
}

# --- 1. at baseline passes --------------------------------------------------
r="$WORK/at-baseline"
mkroot "$r" 10 2
echo '2 20' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '2 production files / 20 LOC'; then
  ok "tree at baseline passes with the canonical sentence"
else
  bad "at baseline" "rc=$rc out=$out"
fi

# --- 2. LOC growth fails ----------------------------------------------------
r="$WORK/loc-growth"
mkroot "$r" 10 2
echo '2 20' >"$r/baseline"
printf 'package root\n' >>"$r/file1.go"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'grew'; then
  ok "LOC growth fails and names the growth"
else
  bad "LOC growth" "rc=$rc out=$out"
fi

# --- 3. file-count growth fails --------------------------------------------
r="$WORK/file-growth"
mkroot "$r" 10 2
echo '2 20' >"$r/baseline"
printf 'package root\n' >"$r/extra.go"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ]; then ok "an extra production file fails the file ratchet"; else bad "file growth" "rc=$rc out=$out"; fi

# --- 4. shrink passes and hints -------------------------------------------
r="$WORK/shrink"
mkroot "$r" 10 2
echo '3 40' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'lower the baseline'; then
  ok "shrink passes and hints at the lower baseline"
else
  bad "shrink" "rc=$rc out=$out"
fi

# --- 5. --advisory downgrades growth --------------------------------------
r="$WORK/advisory"
mkroot "$r" 10 2
echo '2 20' >"$r/baseline"
printf 'package root\n' >>"$r/file1.go"
out="$(run "$r" --advisory)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'WARNING'; then
  ok "--advisory warns instead of failing"
else
  bad "advisory" "rc=$rc out=$out"
fi

# --- 6. missing baseline exits 2 ------------------------------------------
r="$WORK/no-baseline"
mkroot "$r" 10 2
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "missing baseline exits 2"; else bad "missing baseline" "rc=$rc out=$out"; fi

# --- 7. non-integer baseline exits 2 --------------------------------------
r="$WORK/bad-baseline"
mkroot "$r" 10 2
echo 'two twenty' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "non-integer baseline exits 2"; else bad "bad baseline" "rc=$rc out=$out"; fi

# --- 8. trailing field exits 2 --------------------------------------------
r="$WORK/extra-field"
mkroot "$r" 10 2
echo '2 20 99' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "trailing baseline field exits 2"; else bad "extra field" "rc=$rc out=$out"; fi

# --- 9. --update-baseline rewrites ----------------------------------------
r="$WORK/update"
mkroot "$r" 10 2
echo '9 99' >"$r/baseline"
out="$(run "$r" --update-baseline)"
rc=$?
if [ "$rc" -eq 0 ] && [ "$(cat "$r/baseline")" = "2 20" ]; then
  ok "--update-baseline rewrites to the measured value"
else
  bad "update baseline" "rc=$rc baseline=$(cat "$r/baseline") out=$out"
fi

# --- 10. tests and _templ.go are not counted ------------------------------
r="$WORK/production-only"
mkroot "$r" 10 1
printf 'package root\n' >"$r/file_test.go"
printf 'package root\n' >"$r/page_templ.go"
echo '1 10' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '1 production files / 10 LOC'; then
  ok "test and _templ files are excluded from the measurement"
else
  bad "production only" "rc=$rc out=$out"
fi

# --- 11. an empty root exits 2 --------------------------------------------
r="$WORK/empty"
mkdir -p "$r"
echo '0 0' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "an empty root exits 2 instead of reporting zero"; else bad "empty root" "rc=$rc out=$out"; fi

echo
if [ "$fail" -eq 0 ]; then
  echo "✓ root-size self-test: $pass/$pass cases passed"
  exit 0
fi
echo "✗ root-size self-test: $fail case(s) failed ($pass passed)" >&2
exit 1
