#!/usr/bin/env bash
# test-check-docs-freshness.sh — fixture self-test for the /v4-suffix rule
# (scripts/lib/docs-import-paths.sh, wired into check-docs-freshness.sh).
#
# Asserts: /v4-carrying subpackage paths pass; /v4-less subpackage paths are
# named with file and line; bare repo mentions (URLs) are NOT flagged;
# missing files are skipped silently.
#
# Usage: ./scripts/test-check-docs-freshness.sh
# Exit: 0 = all cases pass, 1 = at least one case fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/docs-import-paths.sh"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

echo ""
echo "=== Test Suite: docs-import-paths (/v4 rule) ==="
echo ""

# --- Test 1: correct paths pass ---
cat >"$TMPDIR/ok.md" <<'EOF'
Install with `go get github.com/larsartmann/cqrs-htmx/v4` and import
`github.com/larsartmann/cqrs-htmx/v4/htmx`. The repo lives at
https://github.com/larsartmann/cqrs-htmx (bare mention is fine).
Submodules: `github.com/larsartmann/cqrs-htmx/usermgmt/v4` and
`github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4` carry the
major suffix after the submodule path. Web URLs
(`github.com/larsartmann/cqrs-htmx/commits/master`) and workspace-only
modules (`github.com/larsartmann/cqrs-htmx/integration_test`) are fine.
EOF
CHECK_FAILED=0
OUTPUT="$(check_import_paths "$TMPDIR/ok.md")" || CHECK_FAILED=1
report "$([ "$CHECK_FAILED" -eq 0 ] && echo 0 || echo 1)" "v4-carrying paths + bare mentions pass (CHECK_FAILED=$CHECK_FAILED)"

# --- Test 2: /v4-less subpackage path fails and is named ---
cat >"$TMPDIR/bad.md" <<'EOF'
Import the helpers:

```go
import "github.com/larsartmann/cqrs-htmx/htmx"
```
EOF
CHECK_FAILED=0
OUTPUT="$(check_import_paths "$TMPDIR/bad.md")" || CHECK_FAILED=1
report "$([ "$CHECK_FAILED" -eq 1 ] && echo 0 || echo 1)" "/v4-less subpackage path fails"
report "$(printf '%s' "$OUTPUT" | grep -qF "$TMPDIR/bad.md:4" && echo 0 || echo 1)" "offender named with file:line (got: $OUTPUT)"

# --- Test 3: root-module /v4-less import (quote-terminated) is caught ---
cat >"$TMPDIR/badroot.md" <<'EOF'
`import "github.com/larsartmann/cqrs-htmx/transport"` also lacks v4.
EOF
CHECK_FAILED=0
OUTPUT="$(check_import_paths "$TMPDIR/badroot.md")" || CHECK_FAILED=1
report "$([ "$CHECK_FAILED" -eq 1 ] && echo 0 || echo 1)" "transport-style /v4-less path fails"

# --- Test 4: missing file is skipped silently ---
CHECK_FAILED=0
OUTPUT="$(check_import_paths "$TMPDIR/nope-does-not-exist.md")" || CHECK_FAILED=1
report "$([ "$CHECK_FAILED" -eq 0 ] && [ -z "$OUTPUT" ] && echo 0 || echo 1)" "missing file skipped silently"

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
  echo "FAIL: $fail test(s) failed."
  exit 1
fi

echo "All tests passed."
