#!/usr/bin/env bash
# check-family-release-consumable.sh — R2 guard (templ-components v1.20.0
# poison class): a published tag whose go.mod requires PLACEHOLDER
# pseudo-versions (`vX.Y.Z-00010101000000-000000000000` — Go's zero-time
# form for a commit that was never tagged) can never be consumed: every
# consumer sweep dies with a misleading downstream "unknown revision".
# This gate names the upstream fault instead.
#
# Modes:
#   check-family-release-consumable.sh <module-path> <version>
#       Fetch <module>@<version>.mod from the module proxy; check its
#       requires; then walk every same-family submodule require (path sharing
#       the candidate's family prefix) the same way. Exit 1 = unconsumable
#       (placeholder requires or unpublished version), 2 = tool failure
#       (network), 0 = consumable.
#   check-family-release-consumable.sh --check-gomod <file> [family-prefix]
#       Offline mode: check a local go.mod file (self-tests, bump-dep
#       pre-flight after its own fetch).
#
# Fixture self-test: scripts/selftests/test-check-family-release-consumable.sh

set -uo pipefail

PROXY_BASE="${GOPROXY_BASE:-https://proxy.golang.org}"

scan_gomod() { # <gomod-file> <label> <family-prefix>
  local file="$1" label="$2" family="$3" hits
  hits="$(grep -nE -- '-00010101000000' "$file" 2>/dev/null || true)"
  if [ -n "$hits" ]; then
    echo "FAIL: upstream release $label is unconsumable (placeholder sub-requires):"
    printf '%s\n' "$hits" | sed 's/^/    /'
    return 1
  fi
  return 0
}

walk_family_subs() { # <gomod-file> <candidate-module> <version>
  local file="$1" candidate="$2" version="$3"
  local family candidate_escaped path subfile rc
  family="$(printf '%s' "$candidate" | sed 's|/[^/]*$||')"
  candidate_escaped="$(printf '%s' "$candidate" | sed 's/[^a-zA-Z0-9/._-]/\\&/g')"
  rc=0
  while IFS= read -r path; do
    [ -n "$path" ] || continue
    subfile="$(mktemp /tmp/cqrs-htmx-consumable-sub-XXXXXX)"
    if ! curl -fsSL --max-time 30 "$PROXY_BASE/$path/@v/$version.mod" >"$subfile" 2>/dev/null; then
      echo "FAIL: submodule $path has no published $version on the module proxy (unconsumable family release)"
      rm -f "$subfile"
      rc=1
      continue
    fi
    if ! scan_gomod "$subfile" "$path@$version" "$family"; then
      rc=1
    fi
    rm -f "$subfile"
  done <<EOF
$(grep -E "^[[:space:]]*\"?$candidate_escaped/" "$file" | grep -v '//' | awk '{print $1}' | tr -d '"' | sort -u)
EOF
  return "$rc"
}

if [ "${1:-}" = "--check-gomod" ]; then
  [ -n "${2:-}" ] || {
    echo "usage: $0 --check-gomod <file>" >&2
    exit 2
  }
  [ -f "$2" ] || {
    echo "FAIL: go.mod file not found: $2" >&2
    exit 2
  }
  if scan_gomod "$2" "local file $(basename "$2")" ""; then
    requires=$(grep -cE '^[[:space:]]*[a-zA-Z0-9/._-]+ v' "$2" 2>/dev/null || true)
    echo "OK: no placeholder requires in $(basename "$2") (${requires:-0} requires scanned)"
    exit 0
  fi
  exit 1
fi

[ $# -eq 2 ] || {
  echo "usage: $0 <module-path> <version> | --check-gomod <file>" >&2
  exit 2
}

module="$1"
version="$2"
tmp="$(mktemp /tmp/cqrs-htmx-consumable-XXXXXX)"
trap 'rm -f "$tmp"' EXIT

if ! curl -fsSL --max-time 30 "$PROXY_BASE/$module/@v/$version.mod" >"$tmp"; then
  curl_rc=$?
  if [ "$curl_rc" -eq 22 ]; then
    echo "FAIL: $module@$version is UNPUBLISHED on the module proxy (404) — the release cannot be consumed at all"
    exit 1
  fi
  echo "TOOL FAILURE: module proxy unreachable (curl rc=$curl_rc) — verdict unknown, not a finding"
  exit 2
fi

scan_gomod "$tmp" "$module@$version" "" || exit 1
echo "candidates: $(grep -cE 'v[0-9]+\.[0-9]+' "$tmp") requires in $module@$version root go.mod; walking same-family submodules"
walk_family_subs "$tmp" "$module" "$version" || exit 1
echo "OK: $module@$version is consumable (no placeholder requires in root or same-family submodules)"
