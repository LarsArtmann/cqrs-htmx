#!/usr/bin/env bash
# test-check-docs-links.sh — Test cases for check-docs-links.sh
#
# Creates a temporary directory with carefully crafted .md files that exercise
# the link checker's awk-based extraction logic. Verifies:
#   - Known-good links are accepted (not flagged as broken)
#   - Known-broken links are detected
#   - Go generics like [T](mapper) are NOT treated as links
#   - Links inside fenced code blocks are ignored
#   - Anchor-only links (#section) are skipped
#   - Links with query strings are handled
#
# Usage: ./scripts/test-check-docs-links.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/check-docs-links.sh"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass=0
fail=0

assert_contains() {
    local label="$1" output="$2" pattern="$3"
    if echo "$output" | grep -qF "$pattern"; then
        echo "  PASS: $label"
        pass=$((pass + 1))
    else
        echo "  FAIL: $label"
        echo "        Expected output to contain: $pattern"
        fail=$((fail + 1))
    fi
}

assert_not_contains() {
    local label="$1" output="$2" pattern="$3"
    if echo "$output" | grep -qF "$pattern"; then
        echo "  FAIL: $label"
        echo "        Output should NOT contain: $pattern"
        fail=$((fail + 1))
    else
        echo "  PASS: $label"
        pass=$((pass + 1))
    fi
}

# Create test fixture files that the links will reference
mkdir -p "$TMPDIR/docs/guides"
touch "$TMPDIR/FEATURES.md"
touch "$TMPDIR/CHANGELOG.md"
touch "$TMPDIR/docs/guide.md"
touch "$TMPDIR/docs/guides/setup.md"
touch "$TMPDIR/scripts/run.sh"

echo ""
echo "=== Test Suite: check-docs-links.sh ==="
echo ""

# --- Test 1: Known-good links should NOT be flagged ---
cat > "$TMPDIR/test-good.md" << 'EOF'
# Good Links

See [FEATURES](FEATURES.md) for details.
See [Changelog](CHANGELOG.md).
See [Guide](docs/guide.md).
See [Setup](docs/guides/setup.md).
See [Script](scripts/run.sh).
Relative: [../FEATURES.md](../FEATURES.md) from docs/.
EOF

# We need to run the checker in the TMPDIR context.
# check-docs-links.sh uses REPO_ROOT from its own location, so we copy it
# into TMPDIR and run from there.
mkdir -p "$TMPDIR/scripts"
cp "$CHECKER" "$TMPDIR/scripts/check-docs-links.sh"

OUTPUT=$(cd "$TMPDIR" && bash scripts/check-docs-links.sh 2>&1 || true)

assert_not_contains "Good links not flagged (FEATURES.md)" "$OUTPUT" "BROKEN:.*FEATURES.md"
assert_contains "Good links summary shown" "$OUTPUT" "OK: All markdown links resolve correctly"

# --- Test 2: Known-broken links SHOULD be flagged ---
cat > "$TMPDIR/test-broken.md" << 'EOF'
# Broken Links

See [Nonexistent](does-not-exist.md) file.
See [Missing Guide](docs/missing.md).
EOF

OUTPUT=$(cd "$TMPDIR" && bash scripts/check-docs-links.sh 2>&1 || true)

assert_contains "Broken .md link detected" "$OUTPUT" "does-not-exist.md"
assert_contains "Broken docs/ link detected" "$OUTPUT" "docs/missing.md"
assert_contains "Failure count shown" "$OUTPUT" "broken link"

# --- Test 3: Go generics should NOT be treated as links ---
# [T](mapper) looks like [text](url) to a naive regex but is Go code
cat > "$TMPDIR/test-generics.md" << 'EOF'
# Go Generics

The function signature is:
func Map[T, U any](slice []T, mapper func(T) U) []U

Usage of CommandTyped[Q](app, ...) in code.
EOF

OUTPUT=$(cd "$TMPDIR" && bash scripts/check-docs-links.sh 2>&1 || true)

assert_not_contains "Go generic [T](mapper) not treated as link" "$OUTPUT" "mapper"

# --- Test 4: Links inside fenced code blocks are ignored ---
cat > "$TMPDIR/test-codeblock.md" << 'EOF'
# Code Block Links

Some text with [real link](FEATURES.md).

```go
// This [fake link](nonexistent.md) is inside a code block
// and should be ignored.
var x = [T](value)
```

[Another real link](CHANGELOG.md) after code block.
EOF

OUTPUT=$(cd "$TMPDIR" && bash scripts/check-docs-links.sh 2>&1 || true)

assert_not_contains "Code-block link ignored (nonexistent.md)" "$OUTPUT" "nonexistent.md"
assert_contains "Real link outside code block checked" "$OUTPUT" "OK"

# --- Test 5: Anchor-only and query-string links are skipped ---
cat > "$TMPDIR/test-anchor.md" << 'EOF'
# Anchor Links

See [section](#section) for details.
See [FAQ](CHANGELOG.md?version=1) with query.
See [labeled](FEATURES.md#features) with anchor.
EOF

OUTPUT=$(cd "$TMPDIR" && bash scripts/check-docs-links.sh 2>&1 || true)

assert_not_contains "Anchor-only link skipped" "$OUTPUT" "BROKEN"
assert_not_contains "Query-string link resolved (stripped)" "$OUTPUT" "BROKEN"

# --- Test 6: URL links are skipped ---
cat > "$TMPDIR/test-urls.md" << 'EOF'
# URL Links

See [GitHub](https://github.com/larsartmann/cqrs-htmx).
See [Email](mailto:test@example.com).
EOF

OUTPUT=$(cd "$TMPDIR" && bash scripts/check-docs-links.sh 2>&1 || true)

assert_not_contains "URL link skipped" "$OUTPUT" "github.com"
assert_not_contains "Mailto link skipped" "$OUTPUT" "mailto:"

# Clean up test-only md files so they don't interfere
rm -f "$TMPDIR"/test-*.md

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
    echo "FAIL: $fail test(s) failed."
    exit 1
fi

echo "All tests passed."
