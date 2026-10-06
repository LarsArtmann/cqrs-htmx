#!/usr/bin/env bash
# test-check-selftest-env-leaks.sh — fixture self-test for
# check-selftest-env-leaks.sh (gotcha-27c audit).
#
# Cases:
#   1. CI-branching checker + guarded self-test     -> exit 0
#   2. CI-branching checker + unguarded self-test   -> exit 1, names the leak
#   3. CI-branching checker with NO self-test       -> exit 0, INFO debt line
#   4. zero CI-branching checkers                   -> exit 1 (fail-loud)
#
# Usage: bash scripts/selftests/test-check-selftest-env-leaks.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-selftest-env-leaks.sh"

WORK="$(mktemp -d /tmp/test-selftest-env-leaks-XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

check() { # check <case-no> <expect-rc> <needle> <rc> <output>
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

mk_checker() { # $1 = dir
  mkdir -p "$1/scripts/checks" "$1/scripts/selftests"
  printf '#!/usr/bin/env bash\nif [ "${CI:-false}" = "true" ]; then exit 0; fi\nexit 0\n' >"$1/scripts/checks/check-fakegate.sh"
}

# Case 1: guarded self-test passes.
mk_checker "$WORK/good"
printf '#!/usr/bin/env bash\nenv -u CI bash scripts/checks/check-fakegate.sh\n' >"$WORK/good/scripts/selftests/test-check-fakegate.sh"
out=$(SELFTEST_ENV_LEAKS_ROOT="$WORK/good" bash "$GATE")
rc=$?
check 1 0 "no self-test env-leaks" "$rc" "$out"

# Case 2: unguarded self-test leaks — named FAIL.
mk_checker "$WORK/leak"
printf '#!/usr/bin/env bash\nbash scripts/checks/check-fakegate.sh\n' >"$WORK/leak/scripts/selftests/test-check-fakegate.sh"
out=$(SELFTEST_ENV_LEAKS_ROOT="$WORK/leak" bash "$GATE")
rc=$?
check 2 1 "never strips CI" "$rc" "$out"

# Case 3: no self-test -> INFO debt, still green.
mk_checker "$WORK/debt"
out=$(SELFTEST_ENV_LEAKS_ROOT="$WORK/debt" bash "$GATE")
rc=$?
check 3 0 "coverage debt, not a leak" "$rc" "$out"

# Case 4: no CI-branching checkers at all -> fail loud.
mkdir -p "$WORK/empty/scripts/checks"
printf '#!/usr/bin/env bash\nexit 0\n' >"$WORK/empty/scripts/checks/check-plain.sh"
out=$(SELFTEST_ENV_LEAKS_ROOT="$WORK/empty" bash "$GATE")
rc=$?
check 4 1 "0 CI-branching checkers" "$rc" "$out"

echo "pass=$pass fail=$fail"
if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "OK: check-selftest-env-leaks fixture self-test"
