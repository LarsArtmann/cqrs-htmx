#!/usr/bin/env bash
# test-check-shellcheck.sh — fixture self-test for check-shellcheck.sh.
# Covers: a clean fixture tree passes, a planted SC2034 fails with the
# finding visible, the empty-file-list trap errors loudly, and the REAL
# scripts/ tree is clean (the parity claim this gate exists to enforce).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECK="$SCRIPT_DIR/../checks/check-shellcheck.sh"
TMP=$(mktemp -d /tmp/shellcheck-gate-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT

failures=0

note_fail() {
  echo "FAIL $1"
  failures=$((failures + 1))
}

# --- case 1: clean fixture tree passes ------------------------------------
ok_tree="$TMP/scripts"
mkdir -p "$ok_tree/checks"
printf '#!/usr/bin/env bash\nset -euo pipefail\necho "clean"\n' >"$ok_tree/checks/clean.sh"
out="$(SHELLCHECK_SCRIPTS_DIR="$ok_tree" bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '1 script(s)'; then
  echo "ok   clean fixture tree passes"
else
  note_fail "clean fixture tree: $out"
fi

# --- case 2: planted SC2034 fails and the finding is visible ---------------
bad_tree="$TMP/bad-scripts"
mkdir -p "$bad_tree/checks"
printf '#!/usr/bin/env bash\nset -uo pipefail\nGATE=1\necho "boom"\n' >"$bad_tree/checks/bad.sh"
out="$(SHELLCHECK_SCRIPTS_DIR="$bad_tree" bash "$CHECK" 2>&1)"
if [ $? -eq 1 ] && printf '%s' "$out" | grep -q 'SC2034'; then
  echo "ok   planted SC2034 fails with the finding visible"
else
  note_fail "planted SC2034: $out"
fi

# --- case 3: the empty-file-list trap errors loudly (exit 2) ---------------
empty_tree="$TMP/empty-scripts"
mkdir -p "$empty_tree"
out="$(SHELLCHECK_SCRIPTS_DIR="$empty_tree" bash "$CHECK" 2>&1)"
if [ $? -eq 2 ] && printf '%s' "$out" | grep -q 'no .sh files'; then
  echo "ok   empty file list is a loud error, not a silent pass"
else
  note_fail "empty file list: $out"
fi

# --- case 4: the REAL scripts/ tree is clean (the parity claim) ------------
out="$(bash "$CHECK" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ]; then
  echo "ok   real scripts/ tree clean ($out)"
else
  note_fail "real tree: $out"
fi

echo ""
if [ "$failures" -gt 0 ]; then
  echo "shellcheck-gate self-test: $failures case(s) FAILED"
  exit 1
fi
echo "shellcheck-gate self-test: all cases green"
exit 0
