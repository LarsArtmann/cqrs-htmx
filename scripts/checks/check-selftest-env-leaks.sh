#!/usr/bin/env bash
# check-selftest-env-leaks.sh — mechanize the gotcha-27c self-test env-leak
# audit: GitHub runners export CI=true globally, so any self-test that
# exercises a checker which BRANCHES on ${CI} must strip the variable
# (`env -u CI` / `unset CI`) or it silently covers the CI branch instead of
# the local branch — fixture coverage that is false on exactly the runner
# where it runs.
#
# Rule: for every scripts/checks/*.sh that READS ${CI...}, the matching
# scripts/selftests/test-<name>.sh must contain an env-guard. A checker that
# reads CI with NO self-test is reported as missing coverage (INFO today —
# writing those self-tests is their own atomic change — but listed so the
# debt is visible on every run).
#
# Fixture self-test: scripts/selftests/test-check-selftest-env-leaks.sh
# Override the scanned tree with SELFTEST_ENV_LEAKS_ROOT=<dir> (self-tests).

set -uo pipefail

ROOT="${SELFTEST_ENV_LEAKS_ROOT:-}"
if [ -z "$ROOT" ]; then
  ROOT="$(cd "$(dirname "$0")/../.." && pwd)" || exit 1
fi

checkers="$(grep -lE '\$\{?CI[:}]' "$ROOT"/scripts/checks/*.sh 2>/dev/null | sort)"
if [ -z "$checkers" ]; then
  echo "FAIL: 0 CI-branching checkers found — pattern or layout drift lost the audit's target; refusing to false-green."
  exit 1
fi

total=0
guarded=0
fail=0
missing_selftest=0

for checker in $checkers; do
  name="$(basename "$checker" .sh)"
  total=$((total + 1))
  selftest="$ROOT/scripts/selftests/test-$name.sh"
  if [ ! -f "$selftest" ]; then
    echo "INFO: $name reads \${CI} but has NO self-test (scripts/selftests/test-$name.sh) — coverage debt, not a leak"
    missing_selftest=$((missing_selftest + 1))
    continue
  fi
  if grep -qE 'env -u CI|unset CI|CI= *$|CI-=' "$selftest"; then
    guarded=$((guarded + 1))
  else
    echo "FAIL: $name branches on \${CI} but scripts/selftests/test-$name.sh never strips CI (env -u CI) — on GitHub runners the self-test covers the WRONG branch"
    fail=1
  fi
done

echo "CI-branching checkers: $total  self-tested+guarded: $guarded  missing self-test: $missing_selftest"

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "OK: no self-test env-leaks (every self-tested CI-branching checker strips CI)"
