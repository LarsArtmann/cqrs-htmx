#!/usr/bin/env bash
# check-replace-directives.sh — Verify no absolute paths in replace directives
# Adapted from go-cqrs-lite's CI portability check.
#
# Absolute paths break portability: they work on one machine but fail on CI,
# Docker, and other developers' machines. Use relative paths (../sibling).
# Usage: ./scripts/check-replace-directives.sh
# Exit: 0 = all relative, 1 = absolute path found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=== Replace Directive Check ==="
echo "Checking for absolute paths in replace directives..."
echo ""

failed=0

for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | sort); do
    # Check for absolute paths (starting with / on Unix)
    if grep -E 'replace.*=> */' "$modfile" 2>/dev/null; then
        echo "  ABSOLUTE PATH in $modfile"
        failed=1
    fi
done

if [[ "$failed" -eq 0 ]]; then
    echo "✓ All replace directives use relative paths"
else
    echo ""
    echo "✗ Absolute paths found in replace directives"
    echo "  Fix: use relative paths like '../sibling' instead of '/home/user/sibling'"
    exit 1
fi
