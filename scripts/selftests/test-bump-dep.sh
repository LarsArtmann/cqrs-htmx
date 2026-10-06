#!/usr/bin/env bash
# test-bump-dep.sh — fixture self-test for bump-dep.sh discovery logic.
#
# Exercises the module-discovery rules offline (--dry-run touches nothing and
# needs no network):
#   1. BOTH the block-require form and the single-line `require <mod> <ver>`
#      form are discovered (the 9b3c2e18 regex gap);
#   2. a trailing `$` anchors the module path (prefix match excluded);
#   3. testdata trees are skipped;
#   4. non-matching modules are skipped;
#   5. a pattern with no match exits 0 ("nothing to do").
#
# Usage: bash scripts/selftests/test-bump-dep.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUMP="$SCRIPT_DIR/../tools/bump-dep.sh"

WORK=$(mktemp -d /tmp/test-bump-dep-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

mkdir -p "$WORK/a" "$WORK/b" "$WORK/c/testdata/mod" "$WORK/d" "$WORK/e"

cat >"$WORK/a/go.mod" <<'EOF'
module example.net/a

go 1.22

require (
	github.com/larsartmann/exmod v0.1.0
)
EOF

cat >"$WORK/b/go.mod" <<'EOF'
module example.net/b

go 1.22

require github.com/larsartmann/exmod v0.2.0
EOF

cat >"$WORK/c/testdata/mod/go.mod" <<'EOF'
module example.net/fixture

go 1.22

require github.com/larsartmann/exmod v0.1.0
EOF

cat >"$WORK/d/go.mod" <<'EOF'
module example.net/d

go 1.22

require github.com/larsartmann/othermod v0.1.0
EOF

cat >"$WORK/e/go.mod" <<'EOF'
module example.net/e

go 1.22

require (
	github.com/larsartmann/exmod/sub v0.1.0
)
EOF

run_bump() {
  BUMP_DEP_ROOT="$WORK" bash "$BUMP" "$@" 2>&1
}

# Case 1: prefix pattern discovers a, b, e; NOT testdata (c) or othermod (d).
out=$(run_bump 'larsartmann/exmod' v9.9.9 --dry-run)
if printf '%s' "$out" | grep -q '(dry-run) ./a' &&
  printf '%s' "$out" | grep -q '(dry-run) ./b' &&
  printf '%s' "$out" | grep -q '(dry-run) ./e' &&
  ! printf '%s' "$out" | grep -q './c/testdata' &&
  ! printf '%s' "$out" | grep -q '(dry-run) ./d'; then
  echo "  ok 1: block + single-line forms discovered; testdata + non-match skipped"
  pass=$((pass + 1))
else
  echo "  FAIL 1: discovery wrong; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 2: exact anchor excludes the submodule (e).
out=$(run_bump 'larsartmann/exmod$' v9.9.9 --dry-run)
if printf '%s' "$out" | grep -q '(dry-run) ./a' &&
  printf '%s' "$out" | grep -q '(dry-run) ./b' &&
  ! printf '%s' "$out" | grep -q '(dry-run) ./e'; then
  echo '  ok 2: trailing $ anchors the module path (submodule excluded)'
  pass=$((pass + 1))
else
  echo "  FAIL 2: anchor not honored; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 3: no match exits 0 with the nothing-to-do note.
if out=$(run_bump 'larsartmann/nonexistent' v1.0.0 --dry-run) &&
  printf '%s' "$out" | grep -q "nothing to do"; then
  echo "  ok 3: no-match pattern exits 0 with nothing-to-do"
  pass=$((pass + 1))
else
  echo "  FAIL 3: no-match pattern should be a clean no-op; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 4 (R18): a poisoned upstream target aborts BEFORE any mutation.
# file:// proxy keeps the fixture offline-deterministic; the .mod carries the
# templ-components v1.20.0 placeholder signature.
PROXY_DIR="$WORK/proxy/github.com/larsartmann/exmod/@v"
mkdir -p "$PROXY_DIR"
cat >"$PROXY_DIR/v9.9.9.mod" <<'EOF'
module github.com/larsartmann/exmod

go 1.22

require (
	github.com/larsartmann/exmod/sub v9.9.9-00010101000000-000000000000
)
EOF
before_a="$(cat "$WORK/a/go.mod")"
out=$(GOPROXY_BASE="file://$WORK/proxy" run_bump 'larsartmann/exmod' v9.9.9)
rc=$?
if [ "$rc" -eq 1 ] &&
  printf '%s' "$out" | grep -q "pre-flight github.com/larsartmann/exmod@v9.9.9" &&
  printf '%s' "$out" | grep -q "unconsumable" &&
  printf '%s' "$out" | grep -q "Nothing was changed" &&
  [ "$(cat "$WORK/a/go.mod")" = "$before_a" ]; then
  echo "  ok 4: poisoned upstream aborts pre-sweep; go.mod untouched"
  pass=$((pass + 1))
else
  echo "  FAIL 4: poisoned-target refusal wrong (rc=$rc); output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 5 (R18): BUMP_DEP_NO_NETWORK=1 skips the pre-flight (the sweep runs —
# the per-module go steps fail offline, but the run must get PAST pre-flight).
out=$(BUMP_DEP_NO_NETWORK=1 run_bump 'larsartmann/exmod' v9.9.9)
if ! printf '%s' "$out" | grep -q "pre-flight github.com/larsartmann/exmod@v9.9.9" &&
  printf '%s' "$out" | grep -q "==> \./a"; then
  echo "  ok 5: BUMP_DEP_NO_NETWORK skips the pre-flight"
  pass=$((pass + 1))
else
  echo "  FAIL 5: NO_NETWORK skip wrong; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-bump-dep: $fail case(s) FAILED"
  exit 1
fi
echo "test-bump-dep: all cases green"
