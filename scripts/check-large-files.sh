#!/usr/bin/env bash
# check-large-files.sh — Reject large files and binary executables from the repo
#
# Prevents tracked binaries and oversized assets from entering version control
# (root cause of the 32 MB binary accident during the httputil migration).
# Two modes:
#   ./scripts/check-large-files.sh            → check staged NEW files (pre-commit)
#   ./scripts/check-large-files.sh --all      → check all git-tracked files (CI/manual)
# Exit: 0 = clean, 1 = violation found
# Override the size limit via LARGE_FILE_LIMIT (bytes); default 1 MB.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

MAX_SIZE="${LARGE_FILE_LIMIT:-1048576}" # 1 MB

# Determine which files to check
if [ "${1:-}" = "--all" ]; then
	files=$(git ls-files)
	mode="all tracked"
else
	files=$(git diff --cached --name-only --diff-filter=A 2>/dev/null || true)
	mode="staged new"
fi

if [ -z "$files" ]; then
	echo "=== Large-file / binary guard === ($mode: nothing to check)"
	exit 0
fi

violation=0

while IFS= read -r f; do
	[ -z "$f" ] && continue
	[ -f "$f" ] || continue

	# Size check
	size=$(wc -c <"$f" 2>/dev/null || echo 0)
	if [ "$size" -gt "$MAX_SIZE" ]; then
		echo "FAIL: $f is ${size} bytes (limit ${MAX_SIZE})." >&2
		echo "       Use Git LFS, .gitignore, or shrink the asset." >&2
		violation=1
		continue
	fi

	# Magic-byte check for common executable formats (first 4 bytes)
	hex=$(head -c 4 "$f" 2>/dev/null | od -An -tx1 | tr -d ' \n')
	case "$hex" in
	7f454c46) binfmt="ELF" ;;
	cffaedfe | cefaedfe | feedfacf | feedface) binfmt="Mach-O" ;;
	cafebabe) binfmt="Mach-O universal / Java class" ;;
	4d5a*) binfmt="PE/Windows executable" ;;
	*) binfmt="" ;;
	esac
	if [ -n "$binfmt" ]; then
		echo "FAIL: $f has ${binfmt} magic bytes (0x${hex})." >&2
		echo "       Binary executables must not be tracked in git." >&2
		violation=1
	fi
done <<<"$files"

if [ "$violation" -eq 0 ]; then
	echo "=== Large-file / binary guard === ($mode: OK)"
else
	exit 1
fi
