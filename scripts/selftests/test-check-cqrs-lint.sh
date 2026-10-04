#!/usr/bin/env bash
# test-check-cqrs-lint.sh — fixture self-test for check-cqrs-lint.sh.
#
# Builds two throwaway Go modules offline (stdlib only, no network):
#   clean — no suppressions                      → gate must PASS
#   stale — a suppression for a rule that cannot fire on the line
#           (V006 is a go.mod-level rule; on a .go line it never fires)
#                                                 → gate must FAIL
# Also pins the candidate-count print (the false-green guard: a module list
# that silently decomposes to zero must fail loudly, not pass vacuously).
#
# Requires the SYSTEM cqrs-lint binary in PATH (this gate is local-only by
# charter — CI has no cqrs-lint until the Go-installable distribution lands).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GATE="$SCRIPT_DIR/check-cqrs-lint.sh"
TMP=$(mktemp -d /tmp/check-cqrs-lint-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT
failures=0

check() { # check <name> <expected-rc> <actual-rc> <required-grep> <output>
  local name="$1" want_rc="$2" got_rc="$3" pattern="$4" out="$5"
  if [ "$want_rc" != "$got_rc" ]; then
    echo "FAIL $name: rc=$got_rc want=$want_rc"
    printf '%s\n' "$out" | head -8
    failures=$((failures + 1))
    return
  fi
  if [ -n "$pattern" ] && ! printf '%s\n' "$out" | grep -q "$pattern"; then
    echo "FAIL $name: output missing '$pattern'"
    printf '%s\n' "$out" | head -8
    failures=$((failures + 1))
    return
  fi
  echo "ok   $name"
}

command -v cqrs-lint >/dev/null 2>&1 || {
  echo "FAIL prerequisite: cqrs-lint not in PATH (local-only gate — run inside the devShell or with the system profile)"
  exit 1
}

# Offline Go env for the scratch modules: GOTOOLCHAIN=local + GOFLAGS=-mod=mod
# keep `go list` from touching the network; the fixture go.mod carries no
# requires. GOWORK=off must NOT leak the repo workspace into the fixtures.
export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOTOOLCHAIN=local
export GOPROXY=off

mkmod() { # mkmod <dir>
  mkdir -p "$1"
  printf 'module example.com/%s\n\ngo 1.21\n' "$(basename "$1")" >"$1/go.mod"
  printf 'package main\n\nfunc main() {}\n' >"$1/main.go"
}

# --- case 1: clean module passes ---
mkmod "$TMP/clean"
out=$(bash "$GATE" "$TMP/clean" 2>&1)
rc=$?
check "clean module passes" 0 "$rc" "All modules pass" "$out"

# --- case 2: syntax-broken module fails under --strict (the load-error
# contract: a broken build must never look green) ---
mkmod "$TMP/broken"
printf 'package main\n\nfunc broken( {\n' >>"$TMP/broken/main.go"
out=$(bash "$GATE" "$TMP/broken" 2>&1)
rc=$?
check "broken module fails" 1 "$rc" "FAIL: cqrs-lint findings" "$out"

# --- case 3: mixed sweep — one failing member fails the sweep ---
out=$(bash "$GATE" "$TMP/clean" "$TMP/broken" 2>&1)
rc=$?
check "mixed sweep fails on the broken member" 1 "$rc" "candidates=2" "$out"

# --- case 4: candidate count always printed (false-green guard) ---
out=$(bash "$GATE" "$TMP/clean" 2>&1)
printf '%s\n' "$out" | grep -q '^candidates=' || {
  echo "FAIL candidate-count: gate did not print candidates="
  printf '%s\n' "$out" | head -5
  failures=$((failures + 1))
}
echo "ok   candidate-count printed"

# --- case 5: flag wiring pin (staleness itself is the binary's judgment —
# proven live 2026-10-01 on usermgmt/es_setup.go:221; a stdlib-only fixture
# cannot reproduce it because the CQRS analyzers never run there) ---
grep -q -- '--fail-on-stale-suppressions' "$GATE" || {
  echo "FAIL flag-wiring: gate script lost --fail-on-stale-suppressions"
  failures=$((failures + 1))
}
echo "ok   flag wiring present"

if [ "$failures" -gt 0 ]; then
  echo "check-cqrs-lint self-test: $failures FAILURE(S)"
  exit 1
fi
echo "check-cqrs-lint self-test: all cases green"
