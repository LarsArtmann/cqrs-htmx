#!/usr/bin/env bash
# test-verify-tag.sh — fixture self-test for verify-tag.sh
#
# Runs every guard of verify-tag.sh against a corpus of fixture go.mods
# (scripts/testdata/verify-tag/) inside a throwaway git clone with a local
# bare remote as `origin`. Everything is off-line; --dry-run means no tag
# is ever created and no signing key is needed.
#
# The corpus includes the REAL poisoned go.mod of setup/v4.8.1 — the
# regression anchor for the incident that produced this protocol.
#
# Usage: ./scripts/test-verify-tag.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERIFY="$SCRIPT_DIR/verify-tag.sh"
FIXTURES="$SCRIPT_DIR/testdata/verify-tag"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

echo ""
echo "=== Test Suite: verify-tag.sh (fixture corpus: $FIXTURES) ==="
echo ""

git init --bare -q "$WORK/origin.git"
git clone -q "$WORK/origin.git" "$WORK/repo"
cd "$WORK/repo"
git config user.email fixture@example.com
git config user.name fixture
git config commit.gpgsign false
git config tag.gpgsign false

cp -R "$FIXTURES" ./fixtures
git add fixtures
git commit -q -m fixtures

# Seed the fixture origin with a tag for every cqrs-htmx require in the
# corpus EXCEPT the deliberately-unpublished version, so each poisoned
# fixture fails ONLY the guard it exists to test.
while read -r dep ver; do
  [ -n "$dep" ] || continue
  sub="${dep#github.com/larsartmann/cqrs-htmx}"
  sub="${sub#/}"
  case "$sub" in
  v*) ref="$ver" ;;
  *) ref="${sub%/v*}/$ver" ;;
  esac
  git tag "$ref" HEAD
  git push -q origin "refs/tags/$ref"
  git tag -d "$ref" >/dev/null
done <<EOF
$(cat fixtures/*/go.mod |
  awk '/^\s*github\.com\/larsartmann\/cqrs-htmx\// && $2 != "v4.9.0" {print $1 " " $2}' |
  sort -u)
EOF

expect_accept() { # <name> <moddir> [extra args...]
  local name="$1" moddir="$2"
  shift 2
  local out rc
  set +e
  out="$(bash "$VERIFY" "$moddir" v4.8.0 --dry-run "$@" 2>&1)"
  rc=$?
  set -e
  if [ "$rc" -eq 0 ]; then
    echo "  PASS: $name accepted (exit 0)"
    pass=$((pass + 1))
  else
    echo "  FAIL: $name should be ACCEPTED, got exit $rc:"
    # shellcheck disable=SC2001
    echo "$out" | sed 's/^/        /'
    fail=$((fail + 1))
  fi
}

expect_reject() { # <name> <moddir> <phrase-in-output> [extra args...]
  local name="$1" moddir="$2" phrase="$3"
  shift 3
  local out rc
  set +e
  out="$(bash "$VERIFY" "$moddir" v4.8.0 --dry-run "$@" 2>&1)"
  rc=$?
  set -e
  if [ "$rc" -eq 0 ]; then
    echo "  FAIL: $name should be REJECTED but verify-tag exited 0"
    fail=$((fail + 1))
  elif [[ $out != *"$phrase"* ]]; then
    echo "  FAIL: $name rejected (exit $rc) but for the WRONG reason — output lacks '$phrase':"
    # shellcheck disable=SC2001
    echo "$out" | sed 's/^/        /'
    fail=$((fail + 1))
  else
    echo "  PASS: $name rejected for the right reason ('$phrase')"
    pass=$((pass + 1))
  fi
}

# --- 1. Clean go.mod is accepted -------------------------------------------
expect_accept "clean go.mod" fixtures/clean-go-mod

# --- 2. Family dev-replace (setup/v4.8.1 class) is refused -----------------
expect_reject "family dev-replace" fixtures/family-dev-replace "LOCAL PATHS"

# --- 3. Sibling local replace (systemadapter class, the 2026-08-30 hole) ---
expect_reject "sibling local replace" fixtures/sibling-local-replace "LOCAL PATHS"

# --- 4. --allow-replace-exempt lets the sibling case through ---------------
expect_accept "sibling local replace + --allow-replace-exempt" \
  fixtures/sibling-local-replace --allow-replace-exempt

# --- 5. REGRESSION ANCHOR: the real setup/v4.8.1 poisoned go.mod -----------
expect_reject "REAL setup/v4.8.1 poisoned go.mod" \
  fixtures/real-setup-v4.8.1-poisoned "LOCAL PATHS"

# --- 6. larsartmann pseudo-version require is refused ----------------------
expect_reject "pseudo-version require" fixtures/pseudo-version "pseudo-version"

# --- 7. Unpublished cqrs-htmx require is refused ---------------------------
expect_reject "unpublished require" fixtures/unpublished-require "unpublished"

# --- 8. Version-override replace (target = module + version) passes --------
expect_accept "version-override replace" fixtures/version-override-replace

# --- 9. Missing go directive is refused ------------------------------------
expect_reject "missing go directive" fixtures/missing-go-directive "directive"

# --- 10. Uncommitted module tree is refused --------------------------------
echo "# dirty" >>fixtures/version-override-replace/go.mod
expect_reject "uncommitted module tree" fixtures/version-override-replace "commit"

# --- 11. Pre-existing local tag is refused ---------------------------------
git tag "fixtures/clean-go-mod/v4.8.0" HEAD
expect_reject "pre-existing local tag" fixtures/clean-go-mod "already exists locally"
git tag -d "fixtures/clean-go-mod/v4.8.0" >/dev/null

# --- 12. Usage error exits 2 ------------------------------------------------
set +e
bash "$VERIFY" >/dev/null 2>&1
rc=$?
set -e
if [ "$rc" -eq 2 ]; then
  echo "  PASS: usage error exits 2"
  pass=$((pass + 1))
else
  echo "  FAIL: usage error should exit 2, got $rc"
  fail=$((fail + 1))
fi

echo ""
echo "Results: $pass passed, $fail failed"
echo ""

if [ "$fail" -gt 0 ]; then
  echo "FAIL: $fail test(s) failed."
  exit 1
fi

echo "All tests passed."
