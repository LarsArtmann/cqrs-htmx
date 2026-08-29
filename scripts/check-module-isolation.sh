#!/usr/bin/env bash
# check-module-isolation.sh — Verify each Go module builds and vets standalone (GOWORK=off)
# Adapted from go-cqrs-lite's CI enforcement model.
#
# Catches: missing replace directives, version mismatches, stale go.work entries.
# Usage: ./scripts/check-module-isolation.sh
# Exit: 0 = all modules pass, 1 = at least one module fails

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Fall back to /tmp when the ambient build cache is unwritable (e.g. a dead
# secondary disk), so the gate reports real isolation failures, not cache-init
# noise. Mirrors the guard in flake.nix's bench-spike app.
if ! mkdir -p "${GOCACHE:-$HOME/.cache/go-build}" 2>/dev/null; then
	GOCACHE="/tmp/go-build-cache"
	GOMODCACHE="/tmp/go-mod-cache"
	export GOCACHE GOMODCACHE
	mkdir -p "$GOCACHE" "$GOMODCACHE"
fi

# Production modules, auto-discovered from go.work (excludes e2e/ and
# examples/ — main packages, same exclusion flake.nix apps use). New modules
# added to go.work are picked up automatically; no manual list to drift.
mapfile -t MODULES < <(
	env GOWORK= go work edit -json |
		jq -r '.Use[].DiskPath' |
		sed 's|^\./||' |
		grep -Ev '^(e2e/|examples/)' |
		sort
)

failed=0

echo "=== Module Isolation Check ==="
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
if [[ "$failed" -eq 0 ]]; then
	echo "✓ All modules build and vet standalone (GOWORK=off)"
else
	echo "✗ Module isolation check FAILED"
	exit 1
fi
