#!/usr/bin/env bash
# test-check-reexport-policy.sh — fixture self-test for check-reexport-policy.sh.
#
# The declarations are pre-baked TSVs (the same shape the Go scanner emits), so
# these cases are hermetic and never compile anything.
#
# Cases:
#   1. clean tree passes --strict
#   2. a file that shrinks is a note, not a violation
#   3. an unexported alias in a reexport file needs no deprecation marker
#   4. an exempt-class alias needs no deprecation marker
#   5. a new alias file fails --strict (R1)
#   6. alias growth in a baselined file fails --strict (R1)
#   7. a plain *_reexport.go with no alias declaration fails --strict (R1)
#   8. a non-deprecated exported alias fails --strict (R2)
#   9. a deprecation that never names the owning package fails --strict (R2)
#  10. an alias-bearing file with no baseline row fails --strict (R3)
#  11. an unknown class in the baseline fails --strict (R3)
#  12. advisory mode reports but exits 0
#  13. a missing baseline exits 2
#
# Usage: bash scripts/selftests/test-check-reexport-policy.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-reexport-policy.sh"

WORK=$(mktemp -d /tmp/test-reexport-policy-XXXXXX)
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

ROOT="$WORK/repo"
mkdir -p "$ROOT/pkg" "$ROOT/other"
# A *_reexport.go must exist on disk for the filename sweep to see anything.
printf 'package pkg\n' >"$ROOT/pkg/htmx_reexport.go"

# decls <file> — one TSV row: module file name kind target deprecated names_target pure
decls() {
  printf '# module\tfile\tname\tkind\ttarget_pkg\tdeprecated\tnames_target\tpure_alias_file\n'
  printf '.\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' "$@"
}

# baseline <path> <class> <count> [note]
baseline_row() {
  printf '%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "${4:-note}"
}

# run <baseline> <decls> [args...]
run() {
  local base="$1" decls_file="$2"
  shift 2
  REEXPORT_POLICY_ROOT="$ROOT" \
    REEXPORT_POLICY_BASELINE="$base" \
    REEXPORT_POLICY_DECLS="$decls_file" \
    bash "$TOOL" "$@" 2>&1
}

CLEAN_DECLS="$WORK/clean.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true true false >"$CLEAN_DECLS"
CLEAN_BASE="$WORK/clean.baseline"
baseline_row pkg/htmx_reexport.go reexport 1 v5 >"$CLEAN_BASE"

# 1. clean tree passes
out=$(run "$CLEAN_BASE" "$CLEAN_DECLS" --strict)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "ADR-0062 holds"; then
  ok "1-clean-passes"
else
  bad "1-clean-passes" "rc=$rc $out"
fi

# 2. shrink is a note
DECLS="$WORK/shrink.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true true false >"$DECLS"
BASE="$WORK/shrink.baseline"
baseline_row pkg/htmx_reexport.go reexport 3 v5 >"$BASE"
out=$(run "$BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "shrank to 1"; then
  ok "2-shrink-is-a-note"
else
  bad "2-shrink-is-a-note" "rc=$rc $out"
fi

# 3. unexported alias needs no marker
DECLS="$WORK/unexported.decls"
decls pkg/htmx_reexport.go errCommandsNil var httperr false false false >"$DECLS"
BASE="$WORK/unexported.baseline"
baseline_row pkg/htmx_reexport.go reexport 1 v5 >"$BASE"
out=$(run "$BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 0 ]; then ok "3-unexported-exempt"; else bad "3-unexported-exempt" "rc=$rc $out"; fi

# 4. exempt class needs no marker
DECLS="$WORK/exempt.decls"
decls context.go UserID type id false false false >"$DECLS"
BASE="$WORK/exempt.baseline"
baseline_row context.go exempt:domain-vocabulary 1 owned >"$BASE"
baseline_row pkg/htmx_reexport.go reexport 1 v5 >>"$BASE"
out=$(run "$BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 0 ]; then ok "4-exempt-class"; else bad "4-exempt-class" "rc=$rc $out"; fi

# 5. new alias file fails
DECLS="$WORK/new.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true true false >"$DECLS"
printf '.\tpkg/extra_reexport.go\tExtra\tvar\thtmx\ttrue\ttrue\tfalse\n' >>"$DECLS"
out=$(run "$CLEAN_BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R1 new-alias-file: pkg/extra_reexport.go"; then
  ok "5-new-alias-file"
else
  bad "5-new-alias-file" "rc=$rc $out"
fi

# 6. growth fails
DECLS="$WORK/growth.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true true false >"$DECLS"
printf '.\tpkg/htmx_reexport.go\tHTMXPrompt\tvar\thtmx\ttrue\ttrue\tfalse\n' >>"$DECLS"
out=$(run "$CLEAN_BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R1 alias-growth: pkg/htmx_reexport.go holds 2"; then
  ok "6-alias-growth"
else
  bad "6-alias-growth" "rc=$rc $out"
fi

# 7. empty *_reexport.go fails
DECLS="$WORK/empty.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true true false >"$DECLS"
printf 'package pkg\n' >"$ROOT/pkg/ghost_reexport.go"
out=$(run "$CLEAN_BASE" "$DECLS" --strict)
rc=$?
rm -f "$ROOT/pkg/ghost_reexport.go"
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R1 new-alias-file: pkg/ghost_reexport.go"; then
  ok "7-empty-reexport-file"
else
  bad "7-empty-reexport-file" "rc=$rc $out"
fi

# 8. non-deprecated exported alias fails
DECLS="$WORK/nodep.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx false false false >"$DECLS"
out=$(run "$CLEAN_BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R2 not-deprecated: pkg/htmx_reexport.go: HTMXTarget"; then
  ok "8-not-deprecated"
else
  bad "8-not-deprecated" "rc=$rc $out"
fi

# 9. deprecation that never names the target fails
DECLS="$WORK/unnamed.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true false false >"$DECLS"
out=$(run "$CLEAN_BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R2 unnamed-target"; then
  ok "9-unnamed-target"
else
  bad "9-unnamed-target" "rc=$rc $out"
fi

# 10. unclassified alias-bearing file fails
DECLS="$WORK/unclassified.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx true true false >"$DECLS"
printf '.\tother/util.go\terrForbidden\tvar\tidentitymodel\tfalse\tfalse\tfalse\n' >>"$DECLS"
out=$(run "$CLEAN_BASE" "$DECLS" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R3 unclassified: other/util.go"; then
  ok "10-unclassified-file"
else
  bad "10-unclassified-file" "rc=$rc $out"
fi

# 11. unknown class fails
BASE="$WORK/unknown.baseline"
baseline_row pkg/htmx_reexport.go vendor-facade 1 nope >"$BASE"
out=$(run "$BASE" "$CLEAN_DECLS" --strict)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "R3 unclassified: pkg/htmx_reexport.go declares class"; then
  ok "11-unknown-class"
else
  bad "11-unknown-class" "rc=$rc $out"
fi

# 12. advisory reports but exits 0
DECLS="$WORK/advisory.decls"
decls pkg/htmx_reexport.go HTMXTarget var htmx false false false >"$DECLS"
out=$(run "$CLEAN_BASE" "$DECLS")
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "advisory"; then
  ok "12-advisory-exits-zero"
else
  bad "12-advisory-exits-zero" "rc=$rc $out"
fi

# 13. missing baseline exits 2
out=$(run "$WORK/nope.baseline" "$CLEAN_DECLS" --strict)
rc=$?
if [ "$rc" -eq 2 ] && printf '%s' "$out" | grep -q "missing baseline"; then
  ok "13-missing-baseline"
else
  bad "13-missing-baseline" "rc=$rc $out"
fi

echo "reexport-policy self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
