#!/usr/bin/env bash
# check-module-isolation.sh — Verify each Go module builds and vets standalone (GOWORK=off)
# Adapted from go-cqrs-lite's CI enforcement model.
#
# Catches: missing replace directives, version mismatches, stale go.work entries,
# and the root-tag-gap class (code on HEAD consuming a sibling API whose PINNED
# go.mod version predates it — workspace mode masks it via go.work replaces,
# hermetic mode fails with `undefined:` exactly like a consumer would).
#
# For the modules on HERMETIC_TEST_MODULES the tests run hermetically too
# (since 2026-11-28): build and vet cannot see an assertion. Those modules exist
# to assert a SIBLING's behavior, and under GOWORK=off they resolve the PUBLISHED
# siblings exactly like CI's test job does, while a workspace run sees the local
# tree. A test that hard-codes something only the unreleased tree serves is green
# locally and red in CI until the sibling is tagged (gotcha 30, consumer-test
# variant: six hours on master over a dashboardui route rename).
# Self-test: scripts/selftests/test-check-module-isolation.sh.
#
# Usage: ./scripts/checks/check-module-isolation.sh [--examples]
#   --examples  ALSO cover e2e/ and examples/ (the CI build-job parity leg;
#               default covers production modules only, like flake.nix apps)
# Env: MODULE_ISOLATION_ROOT          override the repo root (fixture self-test)
#      MODULE_ISOLATION_TEST_MODULES  space-separated go.work paths whose tests
#                                      run hermetically; the literal `none`
#                                      disables the test leg
# Exit: 0 = all modules pass, 1 = at least one module fails, 2 = usage

set -euo pipefail

INCLUDE_EXAMPLES=0
examples_label=""
if [[ ${1:-} == "--examples" ]]; then
  INCLUDE_EXAMPLES=1
  examples_label=" (incl. e2e/ + examples/)"
elif [[ -n ${1:-} ]]; then
  echo "usage: $0 [--examples]" >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${MODULE_ISOLATION_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
cd "$REPO_ROOT"

# Fall back to /tmp when the ambient build cache is unwritable (e.g. a dead
# secondary disk), so the gate reports real isolation failures, not cache-init
# noise. Shared guard (flake.nix's bench-spike app uses the same lib).
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/go-cache-env.sh"

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

# The modules whose tests assert a sibling's behavior across the module
# boundary. Their tests run here too, for the reason in the header.
HERMETIC_TEST_MODULES=(integration_test)
if [[ -n ${MODULE_ISOLATION_TEST_MODULES:-} ]]; then
  read -r -a HERMETIC_TEST_MODULES <<<"$MODULE_ISOLATION_TEST_MODULES"
  if [[ ${HERMETIC_TEST_MODULES[0]} == "none" ]]; then
    HERMETIC_TEST_MODULES=()
  fi
fi

is_hermetic_test_module() {
  local candidate
  for candidate in "${HERMETIC_TEST_MODULES[@]}"; do
    if [[ $candidate == "$1" ]]; then
      return 0
    fi
  done
  return 1
}

failed=0
tested=""

echo "=== Module Isolation Check${examples_label} ==="
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

  # Test check (hermetic-test modules only)
  if is_hermetic_test_module "$mod"; then
    if ! (cd "$mod_path" && GOWORK=off GOEXPERIMENT=jsonv2 GOPRIVATE='github.com/larsartmann/*' go test ./... -count=1 >/dev/null 2>&1); then
      echo "FAIL (test)"
      echo "    Run: cd $mod && GOWORK=off go test ./... -count=1"
      echo "    A test in this module sees the PUBLISHED siblings, not the local tree (gotcha 30)."
      failed=1
      continue
    fi
    tested="${tested:+$tested, }$mod"
    echo "OK (build, vet, test)"
    continue
  fi

  echo "OK"
done

echo ""
if [[ $failed -eq 0 ]]; then
  echo "✓ All modules build and vet standalone (GOWORK=off)${tested:+; tests ran hermetically for: $tested}"
else
  echo "✗ Module isolation check FAILED"
  exit 1
fi
