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
    echo "FAIL $name: rc=$got_rc want=$want_rc"; printf '%s\n' "$out" | head -8; failures=$((failures+1)); return
  fi
  if [ -n "$pattern" ] && ! printf '%s\n' "$out" | grep -q "$pattern"; then
    echo "FAIL $name: output missing '$pattern'"; printf '%s\n' "$out" | head -8; failures=$((failures+1)); return
  fi
  echo "ok   $name"
}

command -v cqrs-lint >/dev/null 2>&1 || {
  echo "FAIL prerequisite: cqrs-lint not in PATH (local-only gate — run inside the devShell or with the system profile)"; exit 1
}

# Offline Go env for the scratch modules: GOTOOLCHAIN=local + GOFLAGS=-mod=mod
# keep `go list` from touching the network; the fixture go.mod carries no
# requires. GOWORK=off must NOT leak the repo workspace into the fixtures.
export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOTOOLCHAIN=local
export GOPROXY=off

mkmod() { # mkmod <dir> <extra-go-line>
  mkdir -p "$1"
  printf 'module example.com/%s\n\ngo 1.27\n' "$(basename "$1")" >"$1/go.mod"
  printf 'package main\n\nfunc main() {}\n' >"$1/main.go"
}

# --- case 1: clean module passes ---
mkmod "$TMP/clean"
out=$(bash "$GATE" "$TMP/clean" 2>&1); rc=$?
check "clean module passes" 0 "$rc" "All modules pass" "$out"

# --- case 2: stale suppression fails ---
mkmod "$TMP/stale"
printf '//cqrs-lint:ignore(V006) fixture: go.mod-level rule cannot fire on a .go line\nfunc unused() {}\n' >>"$TMP/stale/main.go"
out=$(bash "$GATE" "$TMP/stale" 2>&1); rc=$?
check "stale suppression fails" 1 "$rc" "FAIL: cqrs-lint findings" "$out"

# --- case 3: both together — one failure fails the sweep ---
out=$(bash "$GATE" "$TMP/clean" "$TMP/stale" 2>&1); rc=$?
check "mixed sweep fails on the stale member" 1 "$rc" "candidates=2" "$out"

# --- case 4: zero-candidate guard is meaningless here (the gate takes an
# explicit list; a EMPTY invocation runs the default gate list) — pin that
# the script always prints its candidate count instead ---
out=$(bash "$GATE" "$TMP/clean" 2>&1)
printf '%s\n' "$out" | grep -q '^candidates=' || {
  echo "FAIL candidate-count: gate did not print candidates="; printf '%s\n' "$out" | head -5; failures=$((failures+1));
}
echo "ok   candidate-count printed"

if [ "$failures" -gt 0 ]; then
  echo "check-cqrs-lint self-test: $failures FAILURE(S)"; exit 1
fi
echo "check-cqrs-lint self-test: all cases green"
