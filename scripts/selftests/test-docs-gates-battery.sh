#!/usr/bin/env bash
# test-docs-gates-battery.sh — fixture self-test for docs-gates-battery.sh.
#
# The wrapper's contract is resolved-verdict honesty, pinned here against a
# STUB binary (no network, no nix):
#   1. a green battery (stub rc 0) passes and surfaces the verdict matrix
#   2. findings (stub rc 1) stay advisory — wrapper exits 0 with a WARNING
#   3. the same findings under --strict become exit 1
#   4. a broken battery (stub rc 2) is never a findings verdict — exit 2
#   5. the broken battery stays exit 2 even under --strict (a crash is not
#      promotable to a verdict)
#   6. absent binary + DOCS_GATES_NO_NIX=1 is an honest SKIP (exit 0, reason)
#   7. a non-executable DOCS_GATES_BIN is treated as absent, not a crash
#   8. an unknown argument exits 2 instead of scanning something else
#   9. --json reaches the binary's argv
#
# Usage: bash scripts/selftests/test-docs-gates-battery.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/docs-gates-battery.sh"
BASH_BIN="$(command -v bash)"

WORK=$(mktemp -d /tmp/test-docs-gates-battery-XXXXXX)
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

ROOT="$WORK/repo-root"
mkdir -p "$ROOT"

# The stub mimics the real binary: emit findings + a verdict matrix, exit
# $STUB_RC, and record its argv so the passthrough case can assert on it.
STUB="$WORK/docs-gates-stub"
cat >"$STUB" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >"${STUB_ARGS:-/dev/null}"
echo "freshness: 4 unharvested item(s)"
echo "verdict matrix:"
echo "  freshness  FAIL  4 unharvested item(s) in the newest report"
echo "  audit-all  FAIL  1 above audit_max: 322"
echo "  annotate   PASS"
echo "  routing    PASS"
echo "  cadence    FAIL  20 live reports vs budget 15"
exit "${STUB_RC:-0}"
EOF
chmod +x "$STUB"

env_stub() {
  local rc="$1"
  shift
  DOCS_GATES_BIN="$STUB" DOCS_GATES_NO_NIX=1 DOCS_GATES_ROOT="$ROOT" STUB_RC="$rc" \
    bash "$TOOL" "$@" 2>&1
}

# --- 1. green battery passes and surfaces the matrix ------------------------
out="$(env_stub 0)"
rc=$?
if [ "$rc" -eq 0 ] &&
  printf '%s' "$out" | grep -q 'all five gates PASS' &&
  printf '%s' "$out" | grep -q 'cadence    FAIL'; then
  ok "green battery passes and prints the matrix"
else
  bad "green battery" "rc=$rc out=$out"
fi

# --- 2. findings stay advisory (exit 0 + WARNING) ---------------------------
out="$(env_stub 1)"
rc=$?
if [ "$rc" -eq 0 ] &&
  printf '%s' "$out" | grep -q 'WARNING' &&
  printf '%s' "$out" | grep -q 'audit-all  FAIL'; then
  ok "findings are advisory (exit 0) with the matrix surfaced"
else
  bad "advisory findings" "rc=$rc out=$out"
fi

# --- 3. the same findings under --strict exit 1 ------------------------------
out="$(env_stub 1 --strict)"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'FAIL under --strict'; then
  ok "--strict turns findings into exit 1"
else
  bad "strict findings" "rc=$rc out=$out"
fi

# --- 4. a broken battery exits 2, never as a findings verdict ----------------
out="$(env_stub 2)"
rc=$?
if [ "$rc" -eq 2 ] && printf '%s' "$out" | grep -q 'battery itself failed'; then
  ok "broken battery exits 2"
else
  bad "broken battery" "rc=$rc out=$out"
fi

# --- 5. --strict cannot promote a crash into a verdict -----------------------
out="$(env_stub 3 --strict)"
rc=$?
if [ "$rc" -eq 2 ]; then
  ok "--strict keeps a crash at exit 2"
else
  bad "strict crash" "rc=$rc out=$out"
fi

# --- 6. absent binary + NO_NIX=1 is an honest SKIP ---------------------------
EMPTY_BIN="$WORK/empty-bin"
mkdir -p "$EMPTY_BIN"
out="$(PATH="$EMPTY_BIN" DOCS_GATES_NO_NIX=1 DOCS_GATES_ROOT="$ROOT" \
  "$BASH_BIN" "$TOOL" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'SKIP'; then
  ok "absent binary is an honest SKIP (exit 0, reason printed)"
else
  bad "absent binary" "rc=$rc out=$out"
fi

# --- 7. a non-executable DOCS_GATES_BIN is treated as absent ------------------
NONEXEC="$WORK/not-executable"
printf '#!/usr/bin/env bash\nexit 0\n' >"$NONEXEC"
out="$(PATH="$EMPTY_BIN" DOCS_GATES_BIN="$NONEXEC" DOCS_GATES_NO_NIX=1 \
  DOCS_GATES_ROOT="$ROOT" "$BASH_BIN" "$TOOL" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] &&
  printf '%s' "$out" | grep -q 'not executable' &&
  printf '%s' "$out" | grep -q 'SKIP'; then
  ok "non-executable DOCS_GATES_BIN degrades to SKIP with a warning"
else
  bad "non-executable binary" "rc=$rc out=$out"
fi

# --- 8. an unknown argument exits 2 ------------------------------------------
out="$(DOCS_GATES_BIN="$STUB" DOCS_GATES_NO_NIX=1 DOCS_GATES_ROOT="$ROOT" \
  bash "$TOOL" --bogus 2>&1)"
rc=$?
if [ "$rc" -eq 2 ] && printf '%s' "$out" | grep -q 'usage:'; then
  ok "unknown argument exits 2 with usage"
else
  bad "unknown argument" "rc=$rc out=$out"
fi

# --- 9. --json reaches the binary's argv --------------------------------------
ARGS_FILE="$WORK/stub-args"
out="$(STUB_ARGS="$ARGS_FILE" DOCS_GATES_BIN="$STUB" DOCS_GATES_NO_NIX=1 \
  DOCS_GATES_ROOT="$ROOT" bash "$TOOL" --json 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] && grep -q -- '--json' "$ARGS_FILE"; then
  ok "--json is passed through to the binary"
else
  bad "--json passthrough" "rc=$rc out=$out args=$(cat "$ARGS_FILE" 2>/dev/null || echo MISSING)"
fi

echo
if [ "$fail" -eq 0 ]; then
  echo "✓ docs-gates-battery self-test: $pass/$pass cases passed"
  exit 0
fi
echo "✗ docs-gates-battery self-test: $fail case(s) failed ($pass passed)" >&2
exit 1
