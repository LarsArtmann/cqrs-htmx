#!/usr/bin/env bash
# test-check-buildflow-skips.sh — fixture self-test for the skip_steps canary.
#
# Cases:
#   1. the documented skips all present → pass
#   2. one documented skip deleted → fail naming it (gotcha 35 class)
#   3. the whole skip_steps block deleted → fail
#   4. an undocumented skip added → pass with a warning
#   5. missing config file → fail
#
# Usage: bash scripts/selftests/test-check-buildflow-skips.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-buildflow-skips.sh"

WORK=$(mktemp -d /tmp/test-buildflow-skips-XXXXXX)
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

make_cfg() {
  cat >"$1" <<'YAML'
skip_steps:
  - templ-generate
  - tailwind-build
  - jest-test
  - vitest-test
  - pnpm-audit
  - deadnix
  - dprint-format

other_key: value
YAML
}

# 1. all documented skips present
make_cfg "$WORK/ok.yml"
out=$(BUILDFLOW_SKIPS_FILE="$WORK/ok.yml" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "all 7 documented skips"; then ok "1-all-present"; else bad "1-all-present" "rc=$rc $out"; fi

# 2. one documented skip deleted
sed '/- dprint-format/d' "$WORK/ok.yml" >"$WORK/missing.yml"
out=$(BUILDFLOW_SKIPS_FILE="$WORK/missing.yml" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "dprint-format.*MISSING"; then ok "2-deletion-fails"; else bad "2-deletion-fails" "rc=$rc $out"; fi

# 3. the whole block deleted
sed '/^skip_steps:/,/^$/d' "$WORK/ok.yml" >"$WORK/gone.yml"
out=$(BUILDFLOW_SKIPS_FILE="$WORK/gone.yml" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ]; then ok "3-block-gone-fails"; else bad "3-block-gone-fails" "rc=$rc $out"; fi

# 4. an undocumented skip added → warn but pass
sed 's/  - deadnix/  - deadnix\n  - mystery-tool/' "$WORK/ok.yml" >"$WORK/extra.yml"
out=$(BUILDFLOW_SKIPS_FILE="$WORK/extra.yml" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "undocumented skip 'mystery-tool'"; then ok "4-extra-warns"; else bad "4-extra-warns" "rc=$rc $out"; fi

# 5. missing config
out=$(BUILDFLOW_SKIPS_FILE="$WORK/nope.yml" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ]; then ok "5-missing-config"; else bad "5-missing-config" "rc=$rc $out"; fi

echo "buildflow-skips self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
