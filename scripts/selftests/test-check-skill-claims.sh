#!/usr/bin/env bash
# test-check-skill-claims.sh — fixture self-test for check-skill-claims.sh.
#
# Cases:
#   1. a doc naming a symbol that exists → pass
#   2. a doc naming a removed symbol → fail naming it (the drift class)
#   3. root-alias leakage: cqrshtmx.X must resolve in ROOT files only, not
#      in a sub-module's files (the bug this gate once had)
#   4. per-doc allowlist suppresses a documented false positive
#   5. explicit doc args (the split-brain wrapper's shape)
#
# Usage: bash scripts/selftests/test-check-skill-claims.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-skill-claims.sh"

WORK=$(mktemp -d /tmp/test-skill-claims-XXXXXX)
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

make_tree() {
  local r="$1"
  mkdir -p "$r/.agents/skills/x" "$r/usermgmt"
  printf 'package cqrshtmx\n\nfunc RealThing() {}\n' >"$r/root.go"
  printf 'package usermgmt\n\nvar SubOnly = 1\n' >"$r/usermgmt/sub.go"
}

# 1. existing symbol
R="$WORK/case1"
make_tree "$R"
printf 'Use `cqrshtmx.RealThing` for stuff.\n' >"$R/.agents/skills/x/SKILL.md"
out=$(SKILL_CLAIMS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "1 documented symbol"; then ok "1-exists"; else bad "1-exists" "rc=$rc $out"; fi

# 2. removed symbol
printf 'Use `cqrshtmx.GoneThing` for stuff.\n' >"$R/.agents/skills/x/SKILL.md"
out=$(SKILL_CLAIMS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "cqrshtmx.GoneThing"; then ok "2-drift-fails"; else bad "2-drift-fails" "rc=$rc $out"; fi

# 3. root-alias leakage: SubOnly exists ONLY in usermgmt
printf 'Use `cqrshtmx.SubOnly` for stuff.\n' >"$R/.agents/skills/x/SKILL.md"
out=$(SKILL_CLAIMS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "root package"; then ok "3-no-root-leakage"; else bad "3-no-root-leakage" "rc=$rc $out"; fi

# 4. per-doc allowlist: the built-in auditlog/README.md entries suppress the
# local-variable (`setup.Opts`) / upstream-package (`auditlog.Config`) names
mkdir -p "$R/auditlog"
printf '`setup.Opts` and `auditlog.Config` shadow upstream names here.\n' >"$R/auditlog/README.md"
out=$(SKILL_CLAIMS_ROOT="$R" bash "$TOOL" "$R/auditlog/README.md" 2>&1)
rc=$?
if [ "$rc" -eq 0 ]; then ok "4-doc-allowlist"; else bad "4-doc-allowlist" "rc=$rc $out"; fi

# 5. explicit doc args
printf 'Root README mentions `cqrshtmx.RealThing`.\n' >"$R/README.md"
out=$(SKILL_CLAIMS_ROOT="$R" bash "$TOOL" "$R/README.md" 2>&1)
rc=$?
if [ "$rc" -eq 0 ]; then ok "5-explicit-args"; else bad "5-explicit-args" "rc=$rc $out"; fi

echo "skill-claims self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
