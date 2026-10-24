#!/usr/bin/env bash
# test-evict-archive.sh — fixture self-test for evict-archive.sh.
#
# Cases:
#   1. dry run reports the right counts and removes nothing
#   2. --list prints every target path
#   3. --apply without --yes is refused (exit 2)
#   4. a stale keep-list entry is refused (exit 1)
#   5. an empty keep-list is refused (exit 1)
#   6. --apply --yes removes exactly the targets; keep entries + README survive
#   7. a second run is a no-op ("nothing to remove")
#
# Usage: bash scripts/selftests/test-evict-archive.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../tools/evict-archive.sh"

WORK=$(mktemp -d /tmp/test-evict-archive-XXXXXX)
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
  EVICT_ROOT="$1" bash "$TOOL" "${@:2}" 2>&1
}

make_repo() {
  local r="$1"
  mkdir -p "$r/docs/status/archived" "$r/docs/planning/archived" "$r/docs/feedback/processed" "$r/docs/status"
  echo "# manifest" >"$r/docs/status/archived/README.md"
  printf 'docs/status/archived/keep-me.md\n' >"$r/docs/status/ARCHIVE_KEEP.txt"
  echo "# keep" >"$r/docs/status/archived/keep-me.md"
  echo "# drop 1" >"$r/docs/status/archived/drop-one.md"
  echo "# drop 2" >"$r/docs/status/archived/drop-two.md"
  echo "# drop 3" >"$r/docs/planning/archived/drop-three.md"
  echo "# drop 4" >"$r/docs/feedback/processed/drop-four.md"
  (
    cd "$r" || exit 1
    git init -q .
    git add -A
    git -c user.email=t@t -c user.name=t commit -qm fixture
  )
}

R="$WORK/repo"
make_repo "$R"

# 1. dry run
out=$(run "$R")
if printf '%s' "$out" | grep -q "1 kept, 4 to remove"; then ok "1-dry-run-counts"; else bad "1-dry-run-counts" "$out"; fi
if [ -f "$R/docs/status/archived/drop-one.md" ]; then ok "1-dry-run-removes-nothing"; else bad "1-dry-run-removes-nothing" "file vanished"; fi

# 2. --list
out=$(run "$R" --list)
if printf '%s' "$out" | grep -q "docs/status/archived/drop-one.md" && printf '%s' "$out" | grep -q "docs/feedback/processed/drop-four.md"; then
  ok "2-list-paths"
else
  bad "2-list-paths" "$out"
fi

# 3. apply without --yes
out=$(run "$R" --apply)
rc=$?
if [ "$rc" -eq 2 ] && printf '%s' "$out" | grep -q "requires --yes"; then ok "3-refuse-apply-no-yes"; else bad "3-refuse-apply-no-yes" "rc=$rc $out"; fi

# 4. stale keep entry
echo "docs/status/archived/does-not-exist.md" >>"$R/docs/status/ARCHIVE_KEEP.txt"
out=$(run "$R")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "stale keep-list"; then ok "4-refuse-stale-keep"; else bad "4-refuse-stale-keep" "rc=$rc $out"; fi

# 5. empty keep list
printf '# only comments\n' >"$R/docs/status/ARCHIVE_KEEP.txt"
out=$(run "$R")
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "keep-list is empty"; then ok "5-refuse-empty-keep"; else bad "5-refuse-empty-keep" "rc=$rc $out"; fi

# restore the real keep list
printf 'docs/status/archived/keep-me.md\n' >"$R/docs/status/ARCHIVE_KEEP.txt"

# 6. apply
out=$(run "$R" --apply --yes)
rc=$?
if [ "$rc" -eq 0 ]; then ok "6-apply-exit"; else bad "6-apply-exit" "rc=$rc $out"; fi
alive=0
for f in docs/status/archived/keep-me.md docs/status/archived/README.md docs/status/ARCHIVE_KEEP.txt; do
  [ -e "$R/$f" ] || alive=1
done
[ "$alive" -eq 0 ] && ok "6-keeps-survive" || bad "6-keeps-survive" "a keep entry was removed"
gone=0
for f in docs/status/archived/drop-one.md docs/status/archived/drop-two.md docs/planning/archived/drop-three.md docs/feedback/processed/drop-four.md; do
  [ -e "$R/$f" ] && gone=1
done
[ "$gone" -eq 0 ] && ok "6-targets-removed" || bad "6-targets-removed" "a target survived"

# 7. idempotent
out=$(run "$R")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "nothing to remove"; then ok "7-idempotent"; else bad "7-idempotent" "rc=$rc $out"; fi

echo "evict-archive self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
