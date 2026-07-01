#!/usr/bin/env bash
# check-version-drift.sh — Detect sibling modules referencing different versions
# Adapted from go-cqrs-lite's CI version drift detection.
#
# Catches: module A requires sibling at v3.3.0 while module B requires v3.4.0.
# In go.work mode this is masked; under GOWORK=off it causes build failures.
# Usage: ./scripts/check-version-drift.sh
# Exit: 0 = no drift, 1 = drift detected

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Collect all internal module paths (from go.mod module declarations)
declare -A MODULE_VERSIONS

echo "=== Version Drift Check ==="
echo ""

# Find all go.mod files
for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | sort); do
    moddir=$(dirname "$modfile")
    # Extract all require entries that reference larsartmann modules
    while IFS= read -r line; do
        # Parse "github.com/larsartmann/.../vX vY.Z.W" 
        mod_path=$(echo "$line" | awk '{print $1}')
        version=$(echo "$line" | awk '{print $2}')
        
        if [[ "$mod_path" =~ ^github\.com/larsartmann/ ]]; then
            key="${mod_path} ${version}"
            MODULE_VERSIONS["$key"]+="${moddir},"
        fi
    done < <(cd "$moddir" && awk '
        /^require \(/ { in_req=1; next }
        /^\)/ { in_req=0 }
        in_req && /^\t/ && /github\.com\/larsartmann\// { print $0 }
    ' go.mod 2>/dev/null)
done

# Check for drift: same module path at different versions
declare -A MODULE_ALL_VERSIONS
for key in "${!MODULE_VERSIONS[@]}"; do
    mod_path=$(echo "$key" | awk '{print $1}')
    version=$(echo "$key" | awk '{print $2}')
    MODULE_ALL_VERSIONS["$mod_path"]+="${version},"
done

failed=0
drift_found=false

for mod_path in "${!MODULE_ALL_VERSIONS[@]}"; do
    versions=$(echo "${MODULE_ALL_VERSIONS[$mod_path]}" | tr ',' '\n' | grep -v '^$' | sort -u)
    version_count=$(echo "$versions" | wc -l)

    if [[ "$version_count" -gt 1 ]]; then
        echo "  DRIFT: $mod_path referenced at multiple versions:"
        echo "$versions" | while read -r v; do
            echo "    - $v"
        done
        drift_found=true
        failed=1
    fi
done

if [[ "$failed" -eq 0 ]]; then
    echo "✓ No version drift detected"
else
    echo ""
    echo "✗ Version drift detected — siblings reference different versions"
    echo "  Fix: ensure all modules reference the same version of internal deps"
    exit 1
fi
