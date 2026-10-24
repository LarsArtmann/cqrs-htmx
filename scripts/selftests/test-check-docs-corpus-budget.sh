#!/usr/bin/env bash
# test-check-docs-corpus-budget.sh — fixture self-test for check-docs-corpus-budget.sh.
#
# Cases:
#   1. at the baseline passes
#   2. above the baseline fails with the delta
#   3. below the baseline passes and warns (ratchet down)
#   4. missing baseline fails
#   5. malformed baseline fails
#   6. --update-baseline writes the measured count
#   7. report mode (no --ratchet) never fails on growth
#   8. --files prints per-directory counts
#   9. an empty docs tree counts zero (and still ratchets)
#
# Usage: bash scripts/selftests/test-check-docs-corpus-budget.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-docs-corpus-budget.sh"

WORK=$(mktemp -d /tmp/test-corpus-budget-XXXXXX)
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
  CORPUS_BUDGET_ROOT="$1" bash "$TOOL" "${@:2}" 2>&1
}

make_docs() {
  local r="$1"
  mkdir -p "$r/docs/status" "$r/docs/guides"
  printf 'one\ntwo\nthree\n' >"$r/docs/status/a.md" # 3 lines
  printf 'one\ntwo\n' >"$r/docs/guides/b.md"        # 2 lines
  printf 'not markdown\n' >"$r/docs/notes.txt"      # ignored
}

R="$WORK/case1"
make_docs "$R"

# 1. at the baseline
printf '5\n' >"$R/docs/CORPUS_BASELINE"
out=$(run "$R" --ratchet)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "within the ratchet"; then ok "1-at-baseline"; else bad "1-at-baseline" "rc=$rc $out"; fi

# 2. above the baseline
printf '4\n' >"$R/docs/CORPUS_BASELINE"
out=$(run "$R" --ratchet)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "exceed the baseline of 4 by 1"; then ok "2-growth-fails"; else bad "2-growth-fails" "rc=$rc $out"; fi

# 3. below the baseline warns but passes
printf '9\n' >"$R/docs/CORPUS_BASELINE"
out=$(run "$R" --ratchet)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "BELOW the baseline of 9"; then ok "3-below-warns"; else bad "3-below-warns" "rc=$rc $out"; fi

# 4. missing baseline
rm -f "$R/docs/CORPUS_BASELINE"
out=$(run "$R" --ratchet)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "needs .*CORPUS_BASELINE"; then ok "4-missing-baseline"; else bad "4-missing-baseline" "rc=$rc $out"; fi

# 5. malformed baseline
printf 'many\n' >"$R/docs/CORPUS_BASELINE"
out=$(run "$R" --ratchet)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "must contain a single integer"; then ok "5-malformed-baseline"; else bad "5-malformed-baseline" "rc=$rc $out"; fi

# 6. --update-baseline
out=$(run "$R" --update-baseline)
rc=$?
if [ "$rc" -eq 0 ] && [ "$(tr -d '[:space:]' <"$R/docs/CORPUS_BASELINE")" = "5" ]; then ok "6-update-baseline"; else bad "6-update-baseline" "rc=$rc $(cat "$R/docs/CORPUS_BASELINE") $out"; fi

# 7. report mode never fails on growth
printf '1\n' >"$R/docs/CORPUS_BASELINE"
out=$(run "$R")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "5 line(s) across 2 markdown file(s)"; then ok "7-report-mode"; else bad "7-report-mode" "rc=$rc $out"; fi

# 8. --files table
out=$(run "$R" --files)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "docs/status" && printf '%s' "$out" | grep -q "docs/guides"; then ok "8-files-table"; else bad "8-files-table" "rc=$rc $out"; fi

# 9. empty docs tree
E="$WORK/empty"
mkdir -p "$E/docs"
printf '0\n' >"$E/docs/CORPUS_BASELINE"
out=$(run "$E" --ratchet)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "0 line(s) across 0 markdown file(s)"; then ok "9-empty-tree"; else bad "9-empty-tree" "rc=$rc $out"; fi

echo "corpus-budget self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
