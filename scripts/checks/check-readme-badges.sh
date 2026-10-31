#!/usr/bin/env bash
# check-readme-badges.sh — resolve every img.shields.io badge in README.md to
# its JSON truth (round-45 M15). A badge endpoint that starts answering
# "no tags" / "inaccessible" / non-2xx (the sunset-host class) silently
# renders a LYING image in the README; this gate fetches each badge's .json
# form and fails on those verdicts.
#
# Network policy (check-upstream-treadmill precedent): fetch failure is a
# WARNING under CI=true (exit 0 with a warning line) and an ERROR locally
# (exit 4) — a hermetic CI runner must not redden on egress, but a local
# "green" that never reached the network is not a verdict.
#
# BADGE_FETCHER overrides the fetch command for the offline fixture self-test
# (args: <url>; stdout: body; exit code: HTTP status semantics, 0 = 200).
#
# Usage: bash scripts/checks/check-readme-badges.sh [--gate] [readme-path]
set -uo pipefail

GATE=0
README=""
for arg in "$@"; do
  case "$arg" in
  --gate) GATE=1 ;;
  *) README="$arg" ;;
  esac
done
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
README="${README:-$REPO_ROOT/README.md}"

[ -f "$README" ] || {
  echo "check-readme-badges: FAILED — no README at $README" >&2
  exit 1
}

fetch() {
  if [ -n "${BADGE_FETCHER:-}" ]; then
    "$BADGE_FETCHER" "$1"
    return $?
  fi
  local body status url="$1"
  body="$(curl -fsSL --max-time 15 "$url" 2>/dev/null)"
  status=$?
  [ "$status" -eq 0 ] || return 1
  printf '%s' "$body"
}

# Extract shields URLs and normalize to their .json form (insert .json before
# the query string; append when there is none).
urls="$(grep -oE 'https://img\.shields\.io/[^) "]+[^) ".,]' "$README" | sort -u || true)"
if [ -z "$urls" ]; then
  echo "check-readme-badges: no img.shields.io badges found — nothing to resolve"
  exit 0
fi

count=0
red=0
warned=0
while IFS= read -r url; do
  [ -z "$url" ] && continue
  count=$((count + 1))
  case "$url" in
  *.json*) json_url="$url" ;;
  *\?*) json_url="${url%%\?*}.json?${url#*\?}" ;;
  *) json_url="${url}.json" ;;
  esac

  if ! body="$(fetch "$json_url")"; then
    if [ "${CI:-}" = "true" ]; then
      echo "  WARN (ci): $json_url unreachable (network-egress runner) — advisory"
      warned=$((warned + 1))
      continue
    fi
    echo "check-readme-badges: FAILED — fetch error for $json_url (exit 4; network down or host gone)" >&2
    exit 4
  fi

  # The JSON form is {"name":...,"value":...}; the lying classes render
  # value "no tags" / "inaccessible" with HTTP 200.
  if printf '%s' "$body" | grep -Eq '"value"\s*:\s*"(no tags|inaccessible|not found|invalid)"'; then
    echo "check-readme-badges: FAILED — badge resolves to a lying verdict: $json_url" >&2
    red=$((red + 1))
    continue
  fi
  echo "  ✅ $json_url"
done <<<"$urls"

[ "$red" -gt 0 ] && {
  echo "check-readme-badges: $red lying badge(s) of $count" >&2
  exit 1
}
echo "check-readme-badges: $count badge(s) resolved${warned:+, $warned CI-warned}"
exit 0
