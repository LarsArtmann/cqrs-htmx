#!/usr/bin/env bash
# verify-remote-tags.sh — paranoia audit: LOCAL tags must peel (^{}) on the
# remote to the SAME commit.
#
# WHY: annotated tags have TWO hashes — the tag object and the peeled
# commit. `git ls-remote` reports the TAG OBJECT for annotated tags; the
# 2026-10-13 wave audit compared it against `rev-parse <tag>^{commit}`
# without peeling the REMOTE side and "failed" 13/13 healthy tags. A
# verification step that cries wolf erodes the trust it exists to build.
# The correct comparison is peeled-vs-peeled (lightweight tags carry only
# the commit; the script falls back to the raw ref for those).
#
# Usage: verify-remote-tags.sh [--remote NAME_OR_URL] <pattern>...
#   e.g. verify-remote-tags.sh 'setup/v4.14.3' 'usermgmt/v4.13.*'
#   The remote defaults to origin; at least one pattern is required.
# Exit: 0 = every candidate matches; 1 = any mismatch/missing tag OR zero
#       candidates (gotcha-2: an audit that scans nothing is a false
#       green); 2 = usage

set -uo pipefail

REMOTE=origin
PATTERNS=()
while [ $# -gt 0 ]; do
  case "$1" in
  --remote)
    [ $# -ge 2 ] || {
      echo "verify-remote-tags: --remote needs a value" >&2
      exit 2
    }
    REMOTE="$2"
    shift 2
    ;;
  -h | --help)
    sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  -*)
    echo "verify-remote-tags: unknown option: $1" >&2
    exit 2
    ;;
  *)
    PATTERNS+=("$1")
    shift
    ;;
  esac
done
[ ${#PATTERNS[@]} -ge 1 ] || {
  echo "usage: $0 [--remote NAME_OR_URL] <pattern>..." >&2
  exit 2
}

candidates=()
declare -A seen=()
for p in "${PATTERNS[@]}"; do
  for t in $(git tag --list "$p"); do
    [ -n "${seen[$t]:-}" ] && continue
    seen[$t]=1
    candidates+=("$t")
  done
done

echo "verify-remote-tags: remote=$REMOTE patterns=${PATTERNS[*]} candidates=${#candidates[@]}"
if [ "${#candidates[@]}" -eq 0 ]; then
  echo "verify-remote-tags: FAIL — zero candidates matched the pattern(s)." >&2
  echo "  An audit that scans nothing is a FALSE green (gotcha 2)." >&2
  exit 1
fi

missing=0
mismatch=0
for t in "${candidates[@]}"; do
  local_peeled="$(git rev-parse "$t^{commit}" 2>/dev/null)" || {
    echo "  ? $t: not resolvable locally"
    missing=$((missing + 1))
    continue
  }
  remote_peeled="$(git ls-remote "$REMOTE" "refs/tags/$t^{}" 2>/dev/null | awk 'NR==1{print $1}')"
  if [ -z "$remote_peeled" ]; then
    remote_peeled="$(git ls-remote "$REMOTE" "refs/tags/$t" 2>/dev/null | awk 'NR==1{print $1}')"
  fi
  if [ -z "$remote_peeled" ]; then
    echo "  ✗ $t: MISSING on $REMOTE (local → ${local_peeled:0:12})"
    missing=$((missing + 1))
  elif [ "$remote_peeled" != "$local_peeled" ]; then
    echo "  ✗ $t: MISMATCH local=${local_peeled:0:12} remote=${remote_peeled:0:12}"
    mismatch=$((mismatch + 1))
  else
    echo "  ✓ $t → ${local_peeled:0:12}"
  fi
done

echo "verify-remote-tags: candidates=${#candidates[@]} ok=$((${#candidates[@]} - missing - mismatch)) missing=$missing mismatch=$mismatch"
if [ "$missing" -ne 0 ] || [ "$mismatch" -ne 0 ]; then
  echo "✗ verify-remote-tags: RED — investigate before trusting the wave (never re-push a moved version: the proxy caches it)." >&2
  exit 1
fi
echo "✓ all remote tags peel to the local commits"
exit 0
