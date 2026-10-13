#!/usr/bin/env bash
# test-verify-remote-tags.sh — fixture self-test for verify-remote-tags.sh
# (offline: the "remote" is a local bare clone; real git ls-remote/rev-parse).
#
#   T1  no patterns                          -> exit 2 (usage)
#   T2  pushed annotated + lightweight tags  -> exit 0, candidates=2, all peel
#   T3  pattern with zero matches            -> exit 1, FALSE-green guard
#   T4  local-only tag (never pushed)        -> exit 1, MISSING named
#   T5  locally moved tag (remote differs)   -> exit 1, MISMATCH named
#
# Runtime: seconds.
#
# Usage: ./scripts/selftests/test-verify-remote-tags.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
TOOL="$SCRIPT_DIR/../tools/verify-remote-tags.sh"

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

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

new_repo() { # <dir>
  git init -q -b main "$1" || return 1
  git -C "$1" config user.email test@example.com
  git -C "$1" config user.name test
  git -C "$1" config commit.gpgsign false
  git -C "$1" config tag.gpgsign false
}

echo "== test-verify-remote-tags.sh"

# Fixture: work repo + bare remote; annotated tag at c1, lightweight at c2,
# one local-only tag at c3 (never pushed).
r="$tmp/work"
new_repo "$r"
git -C "$r" commit -q --allow-empty -m c1
git -C "$r" tag -a setup/v4.14.3 -m release
git -C "$r" commit -q --allow-empty -m c2
git -C "$r" tag usermgmt/v4.13.0
git -C "$r" commit -q --allow-empty -m c3
git -C "$r" clone -q --bare "$r" "$tmp/remote.git"
git -C "$r" remote add origin "$tmp/remote.git"
git -C "$r" push -q origin --tags
# Local-only tag: created AFTER the push, never published.
git -C "$r" tag auditlog/v4.12.9

# T1 usage: no patterns
(cd "$r" && bash "$TOOL") >"$tmp/t1.out" 2>&1
rc=$?
report "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "T1 no patterns -> exit 2 (got $rc)"

# T2 both pushed tags verify (peeled comparison on BOTH sides)
(cd "$r" && bash "$TOOL" --remote "$tmp/remote.git" 'setup/v4.14.3' 'usermgmt/v4.13.0') >"$tmp/t2.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "candidates=2" "$tmp/t2.out" || ok=1
grep -q "missing=0 mismatch=0" "$tmp/t2.out" || ok=1
grep -q "all remote tags peel" "$tmp/t2.out" || ok=1
report $ok "T2 pushed annotated+lightweight tags -> exit 0 (got $rc)"

# T3 zero candidates -> exit 1, false-green guard
(cd "$r" && bash "$TOOL" --remote "$tmp/remote.git" 'zzz/*') >"$tmp/t3.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "zero candidates" "$tmp/t3.out" || ok=1
report $ok "T3 zero-candidate pattern -> exit 1 (got $rc)"

# T4 local-only tag -> exit 1, MISSING named
(cd "$r" && bash "$TOOL" --remote "$tmp/remote.git" 'auditlog/*') >"$tmp/t4.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "MISSING" "$tmp/t4.out" || ok=1
grep -q "auditlog/v4.12.9" "$tmp/t4.out" || ok=1
report $ok "T4 unpushed local tag -> exit 1, MISSING named (got $rc)"

# T5 locally moved annotated tag -> exit 1, MISMATCH named
git -C "$r" tag -fa setup/v4.14.3 -m moved >/dev/null 2>&1
(cd "$r" && bash "$TOOL" --remote "$tmp/remote.git" 'setup/*') >"$tmp/t5.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "MISMATCH" "$tmp/t5.out" || ok=1
grep -q "setup/v4.14.3" "$tmp/t5.out" || ok=1
report $ok "T5 locally moved tag -> exit 1, MISMATCH named (got $rc)"

echo ""
echo "test-verify-remote-tags: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
exit 0
