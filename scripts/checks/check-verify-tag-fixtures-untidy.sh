#!/usr/bin/env bash
# check-verify-tag-fixtures-untidy.sh — canary (b): the verify-tag fixtures
# must stay deliberately UNTIDY.
#
# Gotcha 34: the fixtures under scripts/testdata/verify-tag/ encode poison
# on purpose (unpublished requires, pseudo-versions, dev/family replaces,
# a missing go directive); a tidy sweep that does not exclude testdata
# strips exactly the lines the fixtures exist to carry, and the daemon
# captured the damage mid-loop twice. This gate pins the poison MARKERS per
# fixture — if a sweep (or anything else) strips one, the gate goes red
# before the 15-case verify-tag self-test suite silently loses its
# subject matter.
#
# Usage: bash scripts/checks/check-verify-tag-fixtures-untidy.sh
# Env: FIXTURES_ROOT  override the fixtures dir (fixture self-test).
# Exit: 0 all markers intact, 1 a marker is missing / negative control dirty.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FIXTURES="${FIXTURES_ROOT:-$REPO_ROOT/scripts/testdata/verify-tag}"

fail=0

# require_marker <fixture> <must-contain-grep-pattern> <description>
require_marker() {
  local fixture="$1" pattern="$2" desc="$3"
  local gomod="$FIXTURES/$fixture/go.mod"
  if [ ! -f "$gomod" ]; then
    echo "✗ fixtures-untidy: $gomod missing" >&2
    fail=1
    return
  fi
  if ! grep -qE "$pattern" "$gomod"; then
    echo "✗ fixtures-untidy: $fixture lost its poison ($desc) — a tidy sweep may have struck; see gotcha 34" >&2
    fail=1
  fi
}

# forbid_marker <fixture> <must-NOT-contain-grep-pattern> <description>
forbid_marker() {
  local fixture="$1" pattern="$2" desc="$3"
  local gomod="$FIXTURES/$fixture/go.mod"
  if [ ! -f "$gomod" ]; then
    echo "✗ fixtures-untidy: $gomod missing" >&2
    fail=1
    return
  fi
  if grep -qE "$pattern" "$gomod"; then
    echo "✗ fixtures-untidy: $fixture negative control violated ($desc)" >&2
    fail=1
  fi
}

require_marker family-dev-replace \
  'replace github\.com/larsartmann/cqrs-htmx/usermgmt/v4 => \.\./usermgmt' \
  'family dev replace to ../usermgmt'
forbid_marker missing-go-directive \
  '^go [0-9]' \
  'must keep lacking a go directive'
require_marker pseudo-version \
  'v4\.8\.1-0\.20260830120000-abcdefabcdef' \
  'the pseudo-version require'
require_marker real-setup-v4.8.1-poisoned \
  'replace github\.com/larsartmann/cqrs-htmx/dashboardui/v4 => \.\./dashboardui' \
  'the DEV-ONLY dashboardui replace'
require_marker sibling-local-replace \
  '=> \.\./\.\./go-cqrs-lite/metaengine/projectionadapter' \
  'the sibling filesystem replace'
require_marker unpublished-require \
  'usermgmt/v4 v4\.9\.0' \
  'the unpublished require'
require_marker version-override-replace \
  'replace github\.com/larsartmann/go-sse => github\.com/larsartmann/go-sse v0\.6\.0' \
  'the version-override replace'
forbid_marker clean-go-mod \
  '^replace ' \
  'must stay replace-free (the clean negative control)'

if [ "$fail" -eq 0 ]; then
  echo "✓ fixtures-untidy: all 8 verify-tag fixtures carry their poison"
fi
exit "$fail"
