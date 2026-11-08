#!/usr/bin/env bash
# check-module-isolation.sh — Verify each Go module builds and vets standalone (GOWORK=off)
# Adapted from go-cqrs-lite's CI enforcement model.
#
# Catches: missing replace directives, version mismatches, stale go.work entries,
# and the root-tag-gap class (code on HEAD consuming a sibling API whose PINNED
# go.mod version predates it — workspace mode masks it via go.work replaces,
# hermetic mode fails with `undefined:` exactly like a consumer would).
# Usage: ./scripts/checks/check-module-isolation.sh [--examples]
#   --examples  ALSO cover e2e/ and examples/ (the CI build-job parity leg;
#               default covers production modules only, like flake.nix apps)
# Exit: 0 = all modules pass, 1 = at least one module fails

set -euo pipefail

INCLUDE_EXAMPLES=0
if [[ ${1:-} == "--examples" ]]; then
  INCLUDE_EXAMPLES=1
elif [[ -n ${1:-} ]]; then
  echo "usage: $0 [--examples]" >&2
  exit 2
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

# Fall back to /tmp when the ambient build cache is unwritable (e.g. a dead
# secondary disk), so the gate reports real isolation failures, not cache-init
# noise. Shared guard (flake.nix's bench-spike app uses the same lib).
# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/../lib/go-cache-env.sh"

# Production modules, auto-discovered from go.work (e2e/ and examples/ are
# main packages excluded by default — same exclusion flake.nix apps use;
# --examples drops the filter for the CI build-job parity leg). New modules
# added to go.work are picked up automatically; no manual list to drift.
if [[ $INCLUDE_EXAMPLES -eq 1 ]]; then
  mapfile -t MODULES < <(
    env GOWORK= go work edit -json |
      jq -r '.Use[].DiskPath' |
      sed 's|^\./||' |
      sort
  )
else
  mapfile -t MODULES < <(
    env GOWORK= go work edit -json |
      jq -r '.Use[].DiskPath' |
      sed 's|^\./||' |
      grep -Ev '^(e2e/|examples/)' |
      sort
  )
fi

failed=0

echo "=== Module Isolation Check${INCLUDE_EXAMPLES:+ (incl. e2e/ + examples/)} ==="
echo "Testing each module with GOWORK=off (standalone build)..."
echo ""

for mod in "${MODULES[@]}"; do
  mod_path="$REPO_ROOT/$mod"
  if [[ ! -f "$mod_path/go.mod" ]]; then
    echo "SKIP: $mod (no go.mod)"
    continue
  fi

  module_name=$(grep -m1 '^module ' "$mod_path/go.mod" | awk '{print $2}')
  echo -n "  $module_name ... "

  # Build check
  if ! (cd "$mod_path" && GOWORK=off GOEXPERIMENT=jsonv2 GOPRIVATE='github.com/larsartmann/*' go build ./... 2>/dev/null); then
    echo "FAIL (build)"
    echo "    Run: cd $mod && GOWORK=off go build ./..."
    failed=1
    continue
  fi

  # Vet check
  if ! (cd "$mod_path" && GOWORK=off GOEXPERIMENT=jsonv2 GOPRIVATE='github.com/larsartmann/*' go vet ./... 2>/dev/null); then
    echo "FAIL (vet)"
    echo "    Run: cd $mod && GOWORK=off go vet ./..."
    failed=1
    continue
  fi

  echo "OK"
done

echo ""
if [[ $failed -eq 0 ]]; then
  echo "✓ All modules build and vet standalone (GOWORK=off)"
else
  echo "✗ Module isolation check FAILED"
  exit 1
fi
