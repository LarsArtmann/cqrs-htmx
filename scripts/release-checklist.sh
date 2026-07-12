#!/usr/bin/env bash
# release-checklist.sh — Pre-release verification suite
# Validates the full CONTRIBUTING.md pre-release checklist automatically.
# Usage: nix run .#release-checklist
# Exit: 0 = ready to tag, 1 = issues found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

FAILED=0
step() { echo ""; echo "=== $1 ==="; }
pass() { echo "  ✓ $1"; }
fail() { echo "  ✗ $1"; FAILED=1; }

# 1. CHANGELOG has a version section (not just [Unreleased])
step "CHANGELOG.md"
if grep -qE '## \[v4\.[0-9]+\.[0-9]+\]' CHANGELOG.md; then
    pass "Version section found in CHANGELOG.md"
else
    fail "No [v4.X.Y] section in CHANGELOG.md — move [Unreleased] items"
fi

# 2. go.mod versions match AGENTS.md
step "Version consistency"
GO_CQRS_VERSION=$(grep 'go-cqrs-lite' go.mod | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo "")
AGENTS_VERSION=$(grep 'go-cqrs-lite v' AGENTS.md | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo "")
if [ -n "$GO_CQRS_VERSION" ] && [ -n "$AGENTS_VERSION" ]; then
    if [ "$GO_CQRS_VERSION" = "$AGENTS_VERSION" ]; then
        pass "go-cqrs-lite $GO_CQRS_VERSION matches between go.mod and AGENTS.md"
    else
        fail "go.mod has $GO_CQRS_VERSION, AGENTS.md has $AGENTS_VERSION"
    fi
fi

# 3. Git working tree clean
step "Git status"
if git diff --quiet && git diff --cached --quiet; then
    pass "Working tree clean"
else
    fail "Uncommitted changes — commit or stash before tagging"
    git status --short
fi

# 4. Run full verification suite (matching CONTRIBUTING.md pre-release checklist)
step "Tests (nix run .#test)"
if nix run .#test 2>&1 | tail -5; then
    pass "All module tests pass"
else
    fail "Tests failed"
fi

step "Build (nix run .#build)"
if nix run .#build 2>&1 | tail -5; then
    pass "All modules build"
else
    fail "Build failed"
fi

step "Lint (nix run .#lint)"
if nix run .#lint 2>&1 | tail -5; then
    pass "Lint clean (0 issues all modules)"
else
    fail "Lint issues found"
fi

step "ErrorFamily (nix run .#errorfamily)"
if nix run .#errorfamily 2>&1 | tail -5; then
    pass "Zero stdlib error constructors"
else
    fail "ErrorFamily violations"
fi

step "Module checks (nix run .#check-modules)"
if nix run .#check-modules 2>&1 | tail -5; then
    pass "Module isolation + dep budgets OK"
else
    fail "Module architecture issues"
fi

step "Coverage gate (nix run .#coverage-gate)"
if nix run .#coverage-gate 2>&1 | tail -5; then
    pass "Coverage above thresholds"
else
    fail "Coverage below thresholds"
fi

step "Formatting (nix fmt)"
nix fmt 2>&1 || true
if git diff --quiet; then
    pass "Formatting stable"
else
    fail "nix fmt changed files — re-commit"
    git status --short
fi

step "Flake check (nix flake check)"
if nix flake check 2>&1 | tail -5; then
    pass "Flake checks pass"
else
    fail "Flake check failed"
fi

# 5. Tag freshness
step "Tag status"
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "none")
echo "  Latest tag: $LATEST_TAG"
HEAD_SHORT=$(git rev-parse --short HEAD)
echo "  HEAD: $HEAD_SHORT"

echo ""
if [ "$FAILED" -eq 0 ]; then
    echo "✓ Release checklist PASSED — ready to tag"
    echo "  Next: tag all modules and push (see CONTRIBUTING.md Tagging section)"
else
    echo "✗ Release checklist FAILED — fix issues above"
    exit 1
fi
