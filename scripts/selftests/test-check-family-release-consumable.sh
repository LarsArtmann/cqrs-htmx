#!/usr/bin/env bash
# test-check-family-release-consumable.sh — offline fixture self-test for
# check-family-release-consumable.sh (R2 guard).
#
# Cases (all via --check-gomod, no network):
#   1. clean go.mod            -> exit 0, requires scanned printed
#   2. placeholder go.mod      -> exit 1, "unconsumable (placeholder sub-requires)"
#   3. missing file            -> exit 2 (tool failure, not a finding)
#   4. no args                 -> exit 2 usage
#
# Usage: bash scripts/selftests/test-check-family-release-consumable.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-family-release-consumable.sh"

WORK="$(mktemp -d /tmp/test-family-consumable-XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

check() { # check <case-no> <expect-rc> <needle> <actual-rc> <output>
  local no="$1" want_rc="$2" needle="$3" rc="$4" out="$5"
  if [ "$rc" -ne "$want_rc" ]; then
    echo "  FAIL $no: expected rc=$want_rc, got rc=$rc; output:" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
    return
  fi
  if [ -n "$needle" ] && ! printf '%s' "$out" | grep -q "$needle"; then
    echo "  FAIL $no: output missing '$needle'; output:" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
    return
  fi
  echo "  ok $no"
  pass=$((pass + 1))
}

# Case 1: clean go.mod passes and prints the scanned-require count.
cat >"$WORK/clean.mod" <<'EOF'
module github.com/example/family

go 1.27

require (
	github.com/example/family/sub v1.20.1
	github.com/example/other v0.5.0
)
EOF
out=$(bash "$GATE" --check-gomod "$WORK/clean.mod")
rc=$?
check 1 0 "OK: no placeholder requires" "$rc" "$out"
if ! printf '%s' "$out" | grep -qE '\(2 requires scanned\)'; then
  echo "  FAIL 1b: candidate count not printed (got: $out)" >&2
  fail=$((fail + 1))
else
  echo "  ok 1b: scanned-require count printed"
  pass=$((pass + 1))
fi

# Case 2: the templ-components v1.20.0 poison class fails with the named error.
cat >"$WORK/poison.mod" <<'EOF'
module github.com/example/family

go 1.27

require (
	github.com/example/family/charts v1.20.0-00010101000000-000000000000
	github.com/example/family/htmx v1.20.0
)
EOF
out=$(bash "$GATE" --check-gomod "$WORK/poison.mod")
rc=$?
check 2 1 "unconsumable (placeholder sub-requires)" "$rc" "$out"
if ! printf '%s' "$out" | grep -q -- '-00010101000000-000000000000'; then
  echo "  FAIL 2b: offending require line not quoted" >&2
  fail=$((fail + 1))
else
  echo "  ok 2b: offending line quoted"
  pass=$((pass + 1))
fi

# Case 3: missing file is a tool failure (rc 2), not a finding.
out=$(bash "$GATE" --check-gomod "$WORK/absent.mod" 2>&1)
rc=$?
check 3 2 "not found" "$rc" "$out"

# Case 4: no arguments is a usage error (rc 2).
out=$(bash "$GATE" 2>&1)
rc=$?
check 4 2 "usage" "$rc" "$out"

echo "pass=$pass fail=$fail"
if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "OK: check-family-release-consumable fixture self-test"
