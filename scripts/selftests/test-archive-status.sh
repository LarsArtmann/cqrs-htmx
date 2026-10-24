#!/usr/bin/env bash
# test-archive-status.sh — fixture self-test for archive-status.sh + rebase_links.py.
#
# The ritual this tool replaces failed twice on ORDER, so the order is what the
# cases pin: validate and row-gate BEFORE the move, re-base links AFTER it, and
# lower the ratchet baseline in the same run.
#
# Cases:
#   1. dry run validates and moves nothing
#   2. --apply moves, annotates (SUPERSEDED + ~~ marker), re-bases links, appends the manifest row
#   3. link anchors survive the re-base
#   4. --update-baseline lowers TAIL_BASELINE to the measured tail
#   5. missing successor is refused before anything moves
#   6. missing report is refused
#   7. an already-archived report is refused
#   8. a reason containing '|' is refused (it would break the manifest table)
#   9. a spec line without three tab-separated fields is refused
#  10. a malformed --date is refused (exit 2)
#
# Usage: bash scripts/selftests/test-archive-status.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../tools/archive-status.sh"

WORK=$(mktemp -d /tmp/test-archive-status-XXXXXX)
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

make_repo() {
  local r="$1"
  mkdir -p "$r/docs/status/archived" "$r/docs/planning"
  echo "# status" >"$r/docs/status/README.md"
  printf '| Pass | File | Classification | Deciding reason |\n| --- | --- | --- | --- |\n' \
    >"$r/docs/status/archived/README.md"
  printf '2\n' >"$r/docs/status/TAIL_BASELINE"
  cat >"$r/docs/status/2026-10-01_00-00_old.md" <<'EOF'
# Old report

**Date:** 2026-10-01.

See the [successor](2026-10-02_00-00_new.md) and the [plan](../planning/x.md).

| Item | State |
| --- | --- |
| one | done |
EOF
  cat >"$r/docs/status/2026-10-01_00-01_anchored.md" <<'EOF'
# Anchored report

Read [the successor's b) section](2026-10-02_00-00_new.md#b-done).
EOF
  printf '# Successor\n\n## b) Done\n' >"$r/docs/status/2026-10-02_00-00_new.md"
  printf '# plan\n' >"$r/docs/planning/x.md"
  (
    cd "$r" || exit 1
    git init -q .
    git add -A
    git -c user.email=t@t -c user.name=t commit -qm fixture
  )
}

run() {
  ARCHIVE_STATUS_ROOT="$1" ARCHIVE_STATUS_SKIP_GATES=1 bash "$TOOL" "${@:2}" 2>&1
}

# 1. dry run
R="$WORK/case1"
make_repo "$R"
printf '2026-10-01_00-00_old.md\t2026-10-02_00-00_new.md\tre-covers the arc with the outcome\n' >"$WORK/spec1.tsv"
out=$(run "$R" "$WORK/spec1.tsv")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "dry run: nothing moved"; then ok "1-dry-run"; else bad "1-dry-run" "rc=$rc $out"; fi
if [ -f "$R/docs/status/2026-10-01_00-00_old.md" ]; then ok "1-dry-run-moves-nothing"; else bad "1-dry-run-moves-nothing" "file moved"; fi

# 2. apply
out=$(run "$R" --apply --date 2026-10-24 --pass "round-23 (2026-10-24)" "$WORK/spec1.tsv")
rc=$?
if [ "$rc" -eq 0 ]; then ok "2-apply-exit"; else bad "2-apply-exit" "rc=$rc $out"; fi
NEW="$R/docs/status/archived/2026-10-01_00-00_old.md"
if [ -f "$NEW" ] && [ ! -f "$R/docs/status/2026-10-01_00-00_old.md" ]; then ok "2-moved"; else bad "2-moved" "$out"; fi
if grep -q '~~this report~~ \*\*SUPERSEDED\*\*' "$NEW" && grep -q 'ANNOTATED 2026-10-24' "$NEW"; then
  ok "2-annotation"
else
  bad "2-annotation" "$(head -6 "$NEW")"
fi
if grep -q '(\.\./2026-10-02_00-00_new\.md)' "$NEW"; then ok "2-rebased-live-link"; else bad "2-rebased-live-link" "$(grep successor "$NEW")"; fi
if grep -q '(\.\./\.\./planning/x\.md)' "$NEW"; then ok "2-rebased-parent-link"; else bad "2-rebased-parent-link" "$(grep plan "$NEW")"; fi
if grep -q 're-covers the arc with the outcome' "$R/docs/status/archived/README.md"; then ok "2-manifest-row"; else bad "2-manifest-row" "no row"; fi
if grep -q 'ARCHIVE — superseded' "$R/docs/status/archived/README.md"; then ok "2-manifest-classification"; else bad "2-manifest-classification" "no classification"; fi

# 3. anchor survives
R="$WORK/case3"
make_repo "$R"
printf '2026-10-01_00-01_anchored.md\t2026-10-02_00-00_new.md\tre-covers the anchored series\n' >"$WORK/spec3.tsv"
out=$(run "$R" --apply --date 2026-10-24 "$WORK/spec3.tsv")
rc=$?
if [ "$rc" -eq 0 ] && grep -q '(\.\./2026-10-02_00-00_new\.md#b-done)' "$R/docs/status/archived/2026-10-01_00-01_anchored.md"; then
  ok "3-anchor-preserved"
else
  bad "3-anchor-preserved" "rc=$rc $out"
fi

# 4. baseline lowered
R="$WORK/case4"
make_repo "$R"
printf '3\n' >"$R/docs/status/TAIL_BASELINE"
out=$(run "$R" --apply --update-baseline --date 2026-10-24 "$WORK/spec1.tsv")
rc=$?
if [ "$rc" -eq 0 ] && [ "$(tr -d '[:space:]' <"$R/docs/status/TAIL_BASELINE")" = "2" ]; then
  ok "4-baseline-lowered"
else
  bad "4-baseline-lowered" "rc=$rc baseline=$(cat "$R/docs/status/TAIL_BASELINE") $out"
fi

# 5. missing successor
R="$WORK/case5"
make_repo "$R"
printf '2026-10-01_00-00_old.md\tnope.md\tx\n' >"$WORK/spec5.tsv"
out=$(run "$R" --apply "$WORK/spec5.tsv")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "successor not found" && [ -f "$R/docs/status/2026-10-01_00-00_old.md" ]; then
  ok "5-refuse-missing-successor"
else
  bad "5-refuse-missing-successor" "rc=$rc $out"
fi

# 6. missing report
printf '2026-10-09_00-00_gone.md\t2026-10-02_00-00_new.md\tx\n' >"$WORK/spec6.tsv"
out=$(run "$R" --apply "$WORK/spec6.tsv")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "report not found"; then ok "6-refuse-missing-report"; else bad "6-refuse-missing-report" "rc=$rc $out"; fi

# 7. already archived
mkdir -p "$R/docs/status/archived"
printf '# dup\n' >"$R/docs/status/archived/2026-10-03_00-00_dup.md"
printf '2026-10-03_00-00_dup.md\t2026-10-02_00-00_new.md\tx\n' >"$WORK/spec7.tsv"
out=$(run "$R" --apply "$WORK/spec7.tsv")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "already archived"; then ok "7-refuse-archived"; else bad "7-refuse-archived" "rc=$rc $out"; fi

# 8. pipe in reason
printf '2026-10-01_00-00_old.md\t2026-10-02_00-00_new.md\ta | b\n' >"$WORK/spec8.tsv"
out=$(run "$R" --apply "$WORK/spec8.tsv")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "breaks the manifest table"; then ok "8-refuse-pipe-reason"; else bad "8-refuse-pipe-reason" "rc=$rc $out"; fi

# 9. short spec line
printf '2026-10-01_00-00_old.md\tonly-two\n' >"$WORK/spec9.tsv"
out=$(run "$R" --apply "$WORK/spec9.tsv")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "3 tab-separated fields"; then ok "9-refuse-short-spec"; else bad "9-refuse-short-spec" "rc=$rc $out"; fi

# 10. bad date
out=$(run "$R" --date 24-10-2026 "$WORK/spec1.tsv")
rc=$?
if [ "$rc" -eq 2 ] && printf '%s' "$out" | grep -q "must be YYYY-MM-DD"; then ok "10-refuse-bad-date"; else bad "10-refuse-bad-date" "rc=$rc $out"; fi

# 11. resolved verdict (every item done; successor carries the next arc)
R="$WORK/case11"
make_repo "$R"
printf '2026-10-01_00-00_old.md\t2026-10-02_00-00_new.md\tcarries the patch train this report opened\tresolved\n' >"$WORK/spec11.tsv"
out=$(run "$R" --apply --date 2026-10-24 "$WORK/spec11.tsv")
rc=$?
NEW="$R/docs/status/archived/2026-10-01_00-00_old.md"
if [ "$rc" -eq 0 ] && grep -q 'every open item in this report is resolved' "$NEW" && ! grep -q 'SUPERSEDED' "$NEW"; then
  ok "11-resolved-verdict"
else
  bad "11-resolved-verdict" "rc=$rc $(head -5 "$NEW" 2>/dev/null) $out"
fi
if grep -q 'ARCHIVE — resolved' "$R/docs/status/archived/README.md"; then ok "11-resolved-manifest"; else bad "11-resolved-manifest" "no resolved classification"; fi

# 12. unknown verdict is refused
printf '2026-10-02_00-00_new.md\t2026-10-01_00-00_old.md\tx\tobsolete\n' >"$WORK/spec12.tsv"
out=$(run "$R" --apply "$WORK/spec12.tsv")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "verdict must be"; then ok "12-refuse-bad-verdict"; else bad "12-refuse-bad-verdict" "rc=$rc $out"; fi

echo "archive-status self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
