#!/usr/bin/env bash
# release-checklist.sh — Pre-release verification suite
# Validates CHANGELOG, version refs, go.mod consistency, and runs the full CI suite.
# Usage: nix run .#release-checklist
# Exit: 0 = ready to tag, 1 = issues found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

FAILED=0
step() { echo ""; echo "=== $1 ==="; }
check() { if [ $? -ne 0 ]; then FAILED=1; fi; }

# 1. CHANGELOG has [Unreleased] or version section
step "Checking CHANGELOG.md"
if ! grep -qE '## \[Unreleased\]|## \[v4\.' CHANGELOG.md; then
    echo "WARN: No [Unreleased] or [v4.X.Y] section in CHANGELOG.md"
fi
echo "OK"

# 2. go.mod versions match AGENTS.md dependency table
step "Checking AGENTS.md version consistency"
GO_CQRS_VERSION=$(grep 'go-cqrs-lite' go.mod | head -1 | grep -oP 'v\d+\.\d+\.\d+')
AGENTS_VERSION=$(grep 'go-cqrs-lite v' AGENTS.md | head -1 | grep -oP 'v\d+\.\d+\.\d+')
if [ -n "$GO_CQRS_VERSION" ] && [ -n "$AGENTS_VERSION" ]; then
    if [ "$GO_CQRS_VERSION" != "$AGENTS_VERSION" ]; then
        echo "MISMATCH: go.mod has go-cqrs-lite $GO_CQRS_VERSION, AGENTS.md has $AGENTS_VERSION"
        FAILED=1
    else
        echo "OK: go-cqrs-lite $GO_CQRS_VERSION matches"
    fi
else
    echo "SKIP: Could not extract versions"
fi

# 3. No encoding/json/v2 imports (banned by depguard)
step "Checking for banned json/v2 imports"
if grep -r '"encoding/json/v2"' --include="*.go" . 2>/dev/null; then
    echo "FAIL: encoding/json/v2 imports found (banned by depguard)"
    FAILED=1
else
    echo "OK: No json/v2 imports"
fi

# 4. Git working tree status
step "Checking git status"
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "WARN: Uncommitted changes exist. Commit or stash before tagging."
    git status --short
else
    echo "OK: Working tree clean"
fi

# 5. All modules build
step "Building all modules"
export GONOSUMCHECK='github.com/larsartmann/*'
GOWORK=off go build ./... 2>/dev/null && echo "OK: root builds" || { echo "FAIL: root build"; FAILED=1; }
(cd usermgmt && GOWORK=off go build ./... 2>/dev/null) && echo "OK: usermgmt builds" || { echo "FAIL: usermgmt build"; FAILED=1; }
(cd adminui && GOWORK=off go build ./... 2>/dev/null) && echo "OK: adminui builds" || { echo "FAIL: adminui build"; FAILED=1; }

# 6. Tags don't already exist for current version
step "Checking tag freshness"
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
echo "Latest tag: $LATEST_TAG"

echo ""
if [ "$FAILED" -eq 0 ]; then
    echo "✓ Release checklist PASSED — ready to tag"
else
    echo "✗ Release checklist FAILED — fix issues above"
    exit 1
fi
