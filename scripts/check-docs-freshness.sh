#!/usr/bin/env bash
# check-docs-freshness.sh — Scan .md files for stale version strings
# Compares version numbers mentioned in docs against actual go.mod versions.
# Usage: nix run .#check-docs-freshness
# Exit: 0 = docs fresh, 1 = stale versions found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

FAILED=0

echo "=== Docs Freshness Check ==="
echo ""

# Extract versions from go.mod files
extract_version() {
    local modpath="$1" dep="$2"
    grep "$dep" "$modpath/go.mod" 2>/dev/null | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo ""
}

# Check AGENTS.md dependency table against go.mod
check_agents_md() {
    local dep="$1" agents_pattern="$2"
    local root_ver usermgmt_ver
    
    root_ver=$(extract_version "." "$dep")
    usermgmt_ver=$(extract_version "usermgmt" "$dep")
    
    # The version mentioned in AGENTS.md
    local agents_ver
    agents_ver=$(grep -oP "${agents_pattern}\s+v\d+\.\d+\.\d+" AGENTS.md | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo "")
    
    if [ -n "$agents_ver" ] && [ -n "$root_ver" ]; then
        if [ "$agents_ver" != "$root_ver" ]; then
            echo "  STALE: AGENTS.md has $dep $agents_ver, go.mod has $root_ver"
            FAILED=1
        fi
    fi
}

echo "Checking AGENTS.md version strings..."
check_agents_md "go-webauthn" "go-webauthn"
check_agents_md "pquerna/otp" "pquerna/otp"
check_agents_md "coreos/go-oidc" "coreos/go-oidc"
check_agents_md "a-h/templ" "a-h/templ"

# Check for stale Go version references
echo ""
echo "Checking Go version references..."
GO_MOD_VERSION=$(head -3 go.mod | grep '^go ' | awk '{print $2}')
AGENTS_GO=$(grep -oP 'Go 1\.\d+\.\d+' AGENTS.md | head -1 | grep -oP '1\.\d+\.\d+' || echo "")
if [ -n "$AGENTS_GO" ] && [ "$AGENTS_GO" != "$GO_MOD_VERSION" ]; then
    echo "  INFO: AGENTS.md says Go $AGENTS_GO, go.mod says $GO_MOD_VERSION (may be intentional)"
fi

# Check for stale HTMX version references
echo ""
echo "Checking HTMX version references..."
HTMX_CONST=$(grep -oP 'htmxVersion\s*=\s*"\K[^"]+' htmx_embed.go 2>/dev/null || echo "")
HTMX_DOCS=$(grep -oP 'HTMX v\d+\.\d+\.\d+' AGENTS.md docs/ 2>/dev/null | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo "")
if [ -n "$HTMX_CONST" ] && [ -n "$HTMX_DOCS" ]; then
    DOCS_VER=$(echo "$HTMX_DOCS" | sed 's/v//')
    if [ "$HTMX_CONST" != "$DOCS_VER" ]; then
        echo "  STALE: htmx_embed.go has $HTMX_CONST, docs reference $HTMX_DOCS"
        FAILED=1
    fi
fi

# Check for deprecated API references in docs
echo ""
echo "Checking for deprecated API references..."
if grep -rn 'errors\.New\|fmt\.Errorf' --include="*.md" . 2>/dev/null | grep -v 'banned\|enforced\|Use event\.' | head -5; then
    echo "  INFO: Found stdlib error constructor references in docs (may be documentation of the ban)"
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
    echo "✓ Docs freshness check PASSED"
else
    echo "✗ Docs freshness check found stale references"
    exit 1
fi
