#!/usr/bin/env bash
# test-check-libs-report-count.sh — fixture self-test for check-libs-report-count.sh.
#
# Cases:
#   1. matching series + rows + baseline passes --strict
#   2. report file without an index row fails and names it (the go-etag drop class)
#   3. index row without a report file fails and names it
#   4. FORMAT.md is invisible to the gate (conventions, not a report)
#   5. simultaneous file+row removal fails against the pinned size (shrink class)
#   6. duplicate index row fails
#   7. advisory mode passes on drift
#   8. --update-baseline refuses while the sides disagree
#   9. --update-baseline on a green tree writes the count, then passes
#  10. missing baseline fails even in advisory mode
#
# Usage: bash scripts/selftests/test-check-libs-report-count.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-libs-report-count.sh"

WORK=$(mktemp -d /tmp/test-libs-report-count-XXXXXX)
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

# fixture <dir> <report-name...> — a series dir with an index linking each name.
fixture() {
  local d="$1"
  shift
  mkdir -p "$d/docs/review/libs"
  printf '# conventions\n' >"$d/docs/review/libs/FORMAT.md"
  : >"$d/docs/review/libs/index.html"
  local name
  for name in "$@"; do
    printf '# report\n' >"$d/docs/review/libs/$name.md"
    printf '            <tr>\n              <td>x</td>\n              <td>%s</td>\n              <td><a href="%s.md">md</a></td>\n            </tr>\n' "$name" "$name" >>"$d/docs/review/libs/index.html"
  done
}

# run <root> <baseline> <args...>
run() {
  local r="$1" b="$2"
  shift 2
  LIBS_COUNT_GATE_ROOT="$r" LIBS_COUNT_BASELINE="$b" bash "$TOOL" "$@" 2>&1
}

# 1. green tree passes
R="$WORK/case1"
fixture "$R" go-alpha go-beta
B="$WORK/case1.baseline"
printf '2\n' >"$B"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "series intact at 2"; then ok "1-green-passes"; else bad "1-green-passes" "rc=$rc $out"; fi

# 2. file without a row (go-etag drop class)
printf '# report\n' >"$R/docs/review/libs/go-orph.md"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "go-orph"; then ok "2-file-without-row"; else bad "2-file-without-row" "rc=$rc $out"; fi
rm "$R/docs/review/libs/go-orph.md"

# 3. row without a file
fixture "$R" go-alpha go-beta go-ghost
printf '2\n' >"$B"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "go-ghost"; then ok "3-row-without-file"; else bad "3-row-without-file" "rc=$rc $out"; fi
fixture "$R" go-alpha go-beta

# 4. FORMAT.md invisible (extra conventions file never drifts the count)
printf '2\n' >"$B"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 0 ]; then ok "4-format-invisible"; else bad "4-format-invisible" "rc=$rc $out"; fi

# 5. simultaneous shrink against the pinned size
R="$WORK/case5"
fixture "$R" go-alpha go-beta go-gamma
B="$WORK/case5.baseline"
printf '3\n' >"$B"
rm "$R/docs/review/libs/go-gamma.md"
sed -i '/go-gamma\.md/d' "$R/docs/review/libs/index.html"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "pinned baseline 3"; then ok "5-shrink-caught"; else bad "5-shrink-caught" "rc=$rc $out"; fi

# 6. duplicate row fails
R="$WORK/case6"
fixture "$R" go-alpha
printf '            <tr><td><a href="go-alpha.md">md</a></td></tr>\n' >>"$R/docs/review/libs/index.html"
B="$WORK/case6.baseline"
printf '1\n' >"$B"
out=$(run "$R" "$B" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "duplicate"; then ok "6-duplicate-row"; else bad "6-duplicate-row" "rc=$rc $out"; fi

# 7. advisory mode passes on drift
out=$(run "$R" "$B")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "advisory"; then ok "7-advisory-passes"; else bad "7-advisory-passes" "rc=$rc $out"; fi

# 8. --update-baseline refuses while sides disagree
out=$(run "$R" "$B" --update-baseline)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "refuse"; then ok "8-update-refused"; else bad "8-update-refused" "rc=$rc $out"; fi

# 9. --update-baseline on a green tree writes, then passes
R="$WORK/case9"
fixture "$R" go-alpha go-beta
B="$WORK/case9.baseline"
out=$(run "$R" "$B" --update-baseline)
rc=$?
if [ "$rc" -eq 0 ] && grep -q '^2$' "$B"; then
  out=$(run "$R" "$B" --strict)
  rc=$?
  if [ "$rc" -eq 0 ]; then ok "9-update-then-green"; else bad "9-update-then-green" "still failing: $out"; fi
else
  bad "9-update-then-green" "rc=$rc baseline=$out"
fi

# 10. missing baseline fails even advisory
R="$WORK/case10"
fixture "$R" go-alpha
out=$(run "$R" "$WORK/absent.baseline")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "missing baseline"; then ok "10-missing-baseline"; else bad "10-missing-baseline" "rc=$rc $out"; fi

echo "libs-report-count self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
