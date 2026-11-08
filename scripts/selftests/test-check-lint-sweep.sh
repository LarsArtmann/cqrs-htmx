#!/usr/bin/env bash
# test-check-lint-sweep.sh — self-test for check-lint-sweep.sh.
#
# Coverage (the gate had NO test — it is new 2026-11-08, the 01-34 §e1
# parity gap):
#   1. a clean gated module passes the hermetic sweep
#   2. a gofmt-dirty gated module FAILS it (the finding class workspace
#      mode can mask: hermetic typecheck surfaces undeclared imports)
#   3. an ungated module (no .golangci.yml) is skipped, not linted
#   4. -- <module> scopes the sweep to one module
#
# Fixtures are real mini-modules (zero requires, own go.work) linted with a
# minimal config (gofmt only) so each golangci run is seconds, cold-cache.
#
# Usage: ./scripts/selftests/test-check-lint-sweep.sh
# Exit:  0 = all tests pass, 1 = at least one fails
# Requires: golangci-lint on PATH (flake app runtimeInputs), Go 1.27+.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SWEEP="$SCRIPT_DIR/../checks/check-lint-sweep.sh"

# shellcheck source=/dev/null
source "$SCRIPT_DIR/../lib/go-cache-env.sh"

pass=0
fail=0

report() {
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

SCRATCH="$(mktemp -d)"
defer_cleanup() { rm -rf "$SCRATCH"; }
trap defer_cleanup EXIT

go_version_short() { go env GOVERSION | sed 's/^go//;s/[^0-9.].*$//'; }

make_module() { # <name> <dirty:0|1>
  local d="$SCRATCH/root/$1"
  mkdir -p "$d"
  cat >"$d/go.mod" <<EOF
module example.com/$1

go $(go_version_short)
EOF
  cat >"$d/.golangci.yml" <<'YAML'
version: "2"
linters:
  default: none
  enable:
    - gofmt
YAML
  if [ "$2" = 1 ]; then
    printf 'package main\nfunc main() {\nx:=1\n_ = x\n}\n' >"$d/main.go"
  else
    printf 'package main\n\nfunc main() {\n\tx := 1\n\t_ = x\n}\n' >"$d/main.go"
  fi
}

# Fixture workspace: clean + dirty gated modules, one ungated.
mkdir -p "$SCRATCH/root"
make_module clean 0
make_module dirty 1
mkdir -p "$SCRATCH/root/ungated"
printf 'package main\n\nfunc main() {\nx := 1\n\t_ = x\n}\n' >"$SCRATCH/root/ungated/main.go"
printf 'module example.com/ungated\n\ngo %s\n' "$(go_version_short)" >"$SCRATCH/root/ungated/go.mod"
(cd "$SCRATCH/root" && env GOWORK= go work init ./clean ./dirty ./ungated) >/dev/null 2>&1

run_sweep() { # <extra args...>
  LINT_SWEEP_ROOT="$SCRATCH/root" bash "$SWEEP" "$@" 2>&1
}

echo "== clean gated module passes"
OUT="$(run_sweep -- clean)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && grep -q "1 module(s) clean" <<<"$OUT"; then
  report 0 "scoped clean module sweeps green"
else
  report 1 "clean module should pass (rc=${RC:-0}): $OUT"
fi
unset RC

echo "== dirty gated module fails"
OUT="$(run_sweep -- dirty)" || RC=$?
if [ "${RC:-0}" -eq 1 ] && grep -q "has findings" <<<"$OUT"; then
  report 0 "gofmt-dirty module reds the sweep (rc=1)"
else
  report 1 "dirty module should fail (rc=${RC:-0}): $OUT"
fi
unset RC

echo "== ungated module skipped, mixed sweep still red on dirty"
OUT="$(run_sweep)" || RC=$?
if [ "${RC:-0}" -eq 1 ] && grep -q "dirty" <<<"$OUT" && ! grep -q "ungated" <<<"$OUT"; then
  report 0 "full sweep reds on dirty and never lints the ungated module"
else
  report 1 "full sweep misclassified (rc=${RC:-0}): $OUT"
fi
unset RC

echo "== all-clean workspace is green"
rm -rf "$SCRATCH/root2"
mkdir -p "$SCRATCH/root2"
cp -r "$SCRATCH/root/clean" "$SCRATCH/root2/clean"
(cd "$SCRATCH/root2" && env GOWORK= go work init ./clean) >/dev/null 2>&1
OUT="$(LINT_SWEEP_ROOT="$SCRATCH/root2" bash "$SWEEP" 2>&1)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && grep -q "1 module(s) clean" <<<"$OUT"; then
  report 0 "all-clean workspace sweeps green"
else
  report 1 "all-clean workspace should pass (rc=${RC:-0}): $OUT"
fi
unset RC

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
