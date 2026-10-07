#!/usr/bin/env bash
# check-streamid-identity.sh — D12 / gotcha-25 guard: StreamID().String()
# (and bare aggID/streamID .String()) is DISPLAY-form — the in-flight
# go-cqrs-lite id train brands it "StreamMarker:…" — never identity.
# Identity positions (ID derivation, typed-store keys, casbin subjects,
# event stream keys) must use .Get() (the bare value).
#
# Rule: every non-test, non-generated occurrence must sit in the pinned
# display allowlist below. An occurrence in an UNPINNED file fails; a PINNED
# file whose present occurrences drift from its pin fails (a new site is
# either a .Get() fix or needs an explicit allowlist extension with
# justification). A pinned file with zero occurrences (deleted or fully
# migrated to .Get()) is an improvement, not drift.
#
# Fixture self-test: scripts/selftests/test-check-streamid-identity.sh
# Override the scanned tree with STREAMID_GUARD_ROOT=<dir> (self-tests only).

set -uo pipefail

ROOT="${STREAMID_GUARD_ROOT:-}"
if [ -z "$ROOT" ]; then
  ROOT="$(cd "$(dirname "$0")/../.." && pwd)" || exit 1
fi

# Golden-pinned display surfaces (AGENTS.md gotcha 25): dashboardui
# export/detail/overview/events-filter, transport SSE envelope, datastar
# DOM-id doc, e2e wire echo, basic-demo broadcast payload.
allowlist=(
  "datastar/event_bridge.go:1"
  "dashboardui/core/events.go:1"
  "dashboardui/core/overview.go:1"
  "dashboardui/detail_items.go:2"
  "dashboardui/export.go:4"
  "e2e/server/main.go:1"
  "examples/basic/main.go:1"
  "transport/event_sse.go:1"
)

pattern='StreamID\(\)\.String\(\)|\baggID\.String\(\)|\bstreamID\.String\(\)'

tmp="$(mktemp /tmp/cqrs-htmx-streamid-guard-XXXXXX)"
trap 'rm -f "$tmp"' EXIT

grep -rnE "$pattern" \
  --include='*.go' \
  --exclude='*_test.go' \
  --exclude='*_templ.go' \
  --exclude-dir=.git \
  "$ROOT" >"$tmp" 2>/dev/null

total=$(wc -l <"$tmp" | tr -d ' ')
if [ "$total" -eq 0 ]; then
  echo "FAIL: 0 candidates scanned — pattern or tree drift lost the guard's target; refusing to false-green."
  exit 1
fi

# Observed counts per repo-relative file.
observed="$(sed "s|^$ROOT/||" "$tmp" | awk -F: '{count[$1]++} END {for (f in count) print f":"count[f]}' | sort)"

fail=0
pinned_total=0

for entry in "${allowlist[@]}"; do
  file="${entry%:*}"
  want="${entry##*:}"
  pinned_total=$((pinned_total + want))
  got="$(printf '%s\n' "$observed" | grep -F "$file:" | cut -d: -f2)"
  got="${got:-0}"
  if [ "$got" -ne 0 ] && [ "$got" -ne "$want" ]; then
    echo "FAIL: display-form count drift in pinned file $file: expected $want, found $got"
    echo "      New sites must either switch to .Get() (identity) or extend the"
    echo "      allowlist pin in scripts/checks/check-streamid-identity.sh with a"
    echo "      display justification (AGENTS.md gotcha 25)."
    fail=1
  fi
done

while IFS= read -r entry; do
  [ -z "$entry" ] && continue
  file="${entry%:*}"
  allowlisted=0
  for a in "${allowlist[@]}"; do
    if [ "${a%:*}" = "$file" ]; then
      allowlisted=1
      break
    fi
  done
  if [ "$allowlisted" -eq 0 ]; then
    echo "FAIL: unpinned display-form StreamID site(s) in $file (identity positions must use .Get()):"
    grep -F "$ROOT/$file" "$tmp" | sed "s|^$ROOT/||" | sed 's/^/    /'
    fail=1
  fi
done <<EOF
$observed
EOF

if [ "$fail" -ne 0 ]; then
  echo "StreamID display-form candidates: $total (allowlist pins $pinned_total across ${#allowlist[@]} files)"
  exit 1
fi

echo "OK: all $total StreamID display-form sites match the pinned allowlist (${#allowlist[@]} files, $pinned_total pinned); identity positions use .Get()"
