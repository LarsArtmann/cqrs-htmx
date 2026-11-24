#!/usr/bin/env bash
# test-e2e-kill-strays.sh — offline fixture self-test for the session-
# etiquette guard in scripts/tools/e2e-kill-strays.sh (f#41).
#
# The kill paths themselves need real servers on real ports and are proven by
# the live battery; this self-test pins the REFUSAL CONTRACT:
#   1. a simulated live battery (E2E_KILL_STRAYS_FAKE_LIVE=1) refuses (rc=3)
#      and names the etiquette, WITHOUT killing anything
#   2. --force overrides the simulated live battery (proceeds to the port
#      sweep; on an idle box all ports are clear, rc=0)
#   3. E2E_KILL_STRAYS_FORCE=1 overrides the same way
#   4. a quiet box without --force proceeds (rc=0)
#
# Usage: bash scripts/selftests/test-e2e-kill-strays.sh
# NOTE: cases 2-4 run the real port sweep — they pass only when the battery
# ports are clear (the pre-flight state the script exists to establish).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../tools/e2e-kill-strays.sh"

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

# --- 1. simulated live battery refuses with rc=3 -----------------------------
out="$(E2E_KILL_STRAYS_FAKE_LIVE=1 bash "$TOOL" 2>&1)"
rc=$?
if [ "$rc" -eq 3 ] && printf '%s' "$out" | grep -q 'appears LIVE'; then
  ok "simulated live battery refuses (rc=3)"
else
  bad "simulated live battery" "rc=$rc out=$out"
fi

# --- 2. --force overrides ------------------------------------------------------
out="$(E2E_KILL_STRAYS_FAKE_LIVE=1 bash "$TOOL" --force 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'all battery ports clear'; then
  ok "--force overrides a simulated live battery"
else
  bad "--force override" "rc=$rc out=$out"
fi

# --- 3. env-force overrides ----------------------------------------------------
out="$(E2E_KILL_STRAYS_FAKE_LIVE=1 E2E_KILL_STRAYS_FORCE=1 bash "$TOOL" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ]; then
  ok "E2E_KILL_STRAYS_FORCE=1 overrides"
else
  bad "env force" "rc=$rc out=$out"
fi

# --- 4. quiet box proceeds -----------------------------------------------------
# Tolerates a GENUINELY live battery: if a real `playwright test` process is
# running on this box, the refusal is the guard working correctly, not a
# self-test failure — record it as a conditional skip.
out="$(bash "$TOOL" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ]; then
  ok "quiet box proceeds without --force"
elif [ "$rc" -eq 3 ] && printf '%s' "$out" | grep -q 'appears LIVE'; then
  ok "quiet box SKIPPED (a real battery is live; the refusal is correct)"
else
  bad "quiet box" "rc=$rc out=$out"
fi

echo ""
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ] || exit 1
echo "OK: all fixtures behaved"
