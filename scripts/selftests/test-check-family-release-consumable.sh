#!/usr/bin/env bash
# test-check-family-release-consumable.sh — offline fixture self-test for
# check-family-release-consumable.sh (R2 guard).
#
# Cases (all offline, no network):
#   1. clean go.mod            -> exit 0, requires scanned printed
#   2. placeholder go.mod      -> exit 1, "unconsumable (placeholder sub-requires)"
#   3. missing file            -> exit 2 (tool failure, not a finding)
#   4. no-args sweep, poison present -> exit 1, names the poison go.mod path
#   5. no-args sweep, clean tree     -> exit 0, per-file + require tally
#   6. no-args sweep, non-git root   -> exit 2 (loud, not a false green)
#   7. bogus flag              -> exit 2 usage
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

# Case 4: no-args repo sweep fails and NAMES the poison go.mod (label = tracked path).
SWEEP="$(mktemp -d "$WORK/sweep-XXXXXX")"
git -C "$SWEEP" init -q
mkdir -p "$SWEEP/poison" "$SWEEP/sub"
cp "$WORK/clean.mod" "$SWEEP/go.mod"
cp "$WORK/clean.mod" "$SWEEP/sub/go.mod"
cp "$WORK/poison.mod" "$SWEEP/poison/go.mod"
git -C "$SWEEP" add -A
out=$(CONSUMABLE_ROOT="$SWEEP" bash "$GATE" 2>&1)
rc=$?
check 4 1 "poison/go.mod" "$rc" "$out"

# Case 5: no-args sweep over a clean tree passes with the file/require tally.
git -C "$SWEEP" rm -q --cached poison/go.mod
rm "$SWEEP/poison/go.mod"
out=$(CONSUMABLE_ROOT="$SWEEP" bash "$GATE" 2>&1)
rc=$?
check 5 0 "2 tracked go.mod files" "$rc" "$out"

# Case 6: no-args sweep on a non-git root exits 2 loudly (never a false green).
NOTGIT="$(mktemp -d "$WORK/notgit-XXXXXX")"
out=$(CONSUMABLE_ROOT="$NOTGIT" bash "$GATE" 2>&1)
rc=$?
check 6 2 "TOOL FAILURE" "$rc" "$out"

# Case 7: a bogus flag is a usage error (rc 2).
out=$(bash "$GATE" --bogus 2>&1)
rc=$?
check 7 2 "usage" "$rc" "$out"

# Cases 8-9: the family walk demands each submodule's OWN required version,
# not the candidate's tag (multi-train families — httputil root v1.4.x +
# server_timing v1.0.x — must not be swept onto the root's version). A
# same-version family (templ-components) is still caught: the parent's
# require line carries the (placeholder or missing) version verbatim, and
# fetching exactly that version is what fails. Exercised offline through a
# file:// stub proxy.
PROXY="$WORK/proxy"
mkdir -p "$PROXY/github.com/example/family/@v" "$PROXY/github.com/example/family/sub/@v"
cat >"$PROXY/github.com/example/family/@v/v1.4.2.mod" <<'EOF'
module github.com/example/family

go 1.27

require github.com/example/family/sub v1.0.1
EOF
cat >"$PROXY/github.com/example/family/sub/@v/v1.0.1.mod" <<'EOF'
module github.com/example/family/sub

go 1.27
EOF

# Case 8: multi-train family — the submodule is healthy at its own version.
out=$(GOPROXY_BASE="file://$PROXY" bash "$GATE" github.com/example/family v1.4.2 2>&1)
rc=$?
check 8 0 "sub@v1.0.1" "$rc" "$out"

# Case 9: the parent requires a submodule version the proxy does not have.
cat >"$PROXY/github.com/example/family/@v/v1.4.3.mod" <<'EOF'
module github.com/example/family

go 1.27

require github.com/example/family/sub v1.0.2
EOF
out=$(GOPROXY_BASE="file://$PROXY" bash "$GATE" github.com/example/family v1.4.3 2>&1)
rc=$?
check 9 1 "no published v1.0.2" "$rc" "$out"

echo "pass=$pass fail=$fail"
if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "OK: check-family-release-consumable fixture self-test"
