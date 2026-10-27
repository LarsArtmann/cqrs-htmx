#!/usr/bin/env bash
# evict-archive.sh — remove closed history that no living document cites.
#
# The repo carries ~43% closed history inside every grep and file listing. It
# is recoverable from git, but only if the removal is one reversible command
# with a recorded keep-list. This script IS that command: it derives the
# removal set from the tree, excludes `docs/status/ARCHIVE_KEEP.txt` (plus any
# archive README), and prints exactly what it would remove.
#
# Dry run is the DEFAULT. `--apply` performs the removal with `git rm`, so the
# deletion is staged and reversible with the restore command in
# `docs/status/ARCHIVE_EVICTED.md`.
#
# Usage:
#   bash scripts/tools/evict-archive.sh                 # dry run (prints counts)
#   bash scripts/tools/evict-archive.sh --list          # dry run + every path
#   bash scripts/tools/evict-archive.sh --apply --yes   # perform the removal
#
# Env: EVICT_ROOT  override the scan root (fixture self-test).
# Exit: 0 ok, 1 refusal/failure, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${EVICT_ROOT:-$REPO_ROOT}"
KEEP_FILE="$ROOT/docs/status/ARCHIVE_KEEP.txt"

APPLY=0
LIST=0
YES=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --apply) APPLY=1 ;;
  --list) LIST=1 ;;
  --yes) YES=1 ;;
  *)
    echo "usage: evict-archive.sh [--list] [--apply --yes]" >&2
    exit 2
    ;;
  esac
  shift
done

if [ "$APPLY" -eq 1 ] && [ "$YES" -ne 1 ]; then
  echo "✗ evict-archive: --apply requires --yes (this removes tracked files)" >&2
  exit 2
fi
if [ ! -f "$KEEP_FILE" ]; then
  echo "✗ evict-archive: keep-list missing: $KEEP_FILE" >&2
  exit 1
fi

mapfile -t keep < <(grep -v -E '^[[:space:]]*(#|$)' "$KEEP_FILE" | sed 's/[[:space:]]*$//')
if [ "${#keep[@]}" -eq 0 ]; then
  echo "✗ evict-archive: keep-list is empty — refusing to evict everything" >&2
  exit 1
fi

stale=0
for k in "${keep[@]}"; do
  if [ ! -e "$ROOT/$k" ]; then
    echo "✗ evict-archive: keep-list entry does not exist: $k" >&2
    stale=$((stale + 1))
  fi
done
if [ "$stale" -gt 0 ]; then
  echo "✗ evict-archive: $stale stale keep-list entr(ies); re-derive the list first" >&2
  exit 1
fi

keep_set=$(printf '%s\n' "${keep[@]}" | sort)

targets=$(
  find "$ROOT/docs" -type d \( -name archived -o -name processed \) -print0 |
    xargs -0 -I{} find {} -type f -print |
    sed "s|^$ROOT/||" |
    grep -v -E '(^|/)README\.md$' |
    sort |
    comm -23 - <(printf '%s\n' "$keep_set")
)

total=$(printf '%s\n' "$targets" | grep -c . || true)
echo "evict-archive: ${#keep[@]} kept, $total to remove"
while IFS= read -r d; do
  rel="${d#"$ROOT"/}"
  tot=$(find "$d" -type f | wc -l)
  rem=$(printf '%s\n' "$targets" | grep -c "^$rel/" || true)
  printf "  %-45s %3d total, %3d to remove\n" "$rel" "$tot" "$rem"
done < <(find "$ROOT/docs" -type d \( -name archived -o -name processed \) | sort)

if [ "$LIST" -eq 1 ]; then
  printf '%s\n' "$targets" | sed 's/^/  /'
fi

if [ "$total" -eq 0 ]; then
  echo "✓ evict-archive: nothing to remove"
  exit 0
fi

if [ "$APPLY" -eq 0 ]; then
  echo "dry run: nothing removed. Re-run with --apply --yes to remove $total file(s)."
  exit 0
fi

printf '%s\n' "$targets" | (cd "$ROOT" && xargs -d '\n' git rm -q --) || {
  echo "✗ evict-archive: git rm failed" >&2
  exit 1
}
echo "✓ evict-archive: removed $total file(s); restore with the command in docs/status/ARCHIVE_EVICTED.md"
