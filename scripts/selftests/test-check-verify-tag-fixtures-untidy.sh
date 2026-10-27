#!/usr/bin/env bash
# test-check-verify-tag-fixtures-untidy.sh — fixture self-test for the
# verify-tag poison canary.
#
# Cases:
#   1. the real fixtures intact → pass
#   2. a poison marker stripped → fail naming the fixture
#   3. a negative control violated (replace added to clean-go-mod) → fail
#
# Usage: bash scripts/selftests/test-check-verify-tag-fixtures-untidy.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-verify-tag-fixtures-untidy.sh"
REAL="$SCRIPT_DIR/../../scripts/testdata/verify-tag"

WORK=$(mktemp -d /tmp/test-fixtures-untidy-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0
ok() {
  echo "  ok $1"
  pass=$((pass + 1))
}
bad() {
  echo "  FAIL $1: $2" >&2
  fail=$((fail + 1))
}

# 1. real fixtures intact
out=$(bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "all 8 verify-tag fixtures"; then ok "1-real-intact"; else bad "1-real-intact" "rc=$rc $out"; fi

# 2. strip a poison marker (the gotcha-34 tidy-sweep damage class)
cp -r "$REAL" "$WORK/fixtures"
sed -i '/=> \.\.\/usermgmt/d' "$WORK/fixtures/family-dev-replace/go.mod"
out=$(FIXTURES_ROOT="$WORK/fixtures" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "family-dev-replace lost its poison"; then ok "2-stripped-marker-fails"; else bad "2-stripped-marker-fails" "rc=$rc $out"; fi

# 3. negative control violated: a replace sneaks into clean-go-mod
cp -r "$REAL" "$WORK/fixtures2"
printf '\nreplace github.com/larsartmann/go-sse => github.com/larsartmann/go-sse v0.6.0\n' \
  >>"$WORK/fixtures2/clean-go-mod/go.mod"
out=$(FIXTURES_ROOT="$WORK/fixtures2" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "clean-go-mod negative control violated"; then ok "3-negative-control"; else bad "3-negative-control" "rc=$rc $out"; fi

echo "fixtures-untidy self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
