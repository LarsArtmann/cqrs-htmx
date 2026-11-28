#!/usr/bin/env bash
# test-check-module-isolation.sh — fixture self-test for check-module-isolation.sh.
#
# Builds a throwaway workspace out of stdlib-only modules and drives the gate
# through every verdict it can print:
#   1. clean tree, test leg off                     -> exit 0, plain OK
#   2. clean tree, two modules on the test list     -> exit 0, "OK (build, vet, test)"
#   3. a planted failing test on the list           -> exit 1, FAIL (test), the others still OK
#   4. the same failing test with the list unset    -> exit 0 (the list gates the leg)
#   5. the same failing test with the list `none`   -> exit 0, plain OK
#   6. a planted build failure                      -> exit 1, FAIL (build)
#   7. a planted vet failure                        -> exit 1, FAIL (vet)
#   8. an unknown flag                              -> exit 2
#   9. --examples with the test leg                 -> exit 0
#  10. the real gate lists integration_test         -> the leg is live on this repo
#
# Usage: bash scripts/selftests/test-check-module-isolation.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-module-isolation.sh"

WORK=$(mktemp -d /tmp/test-check-module-isolation-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

ok() {
  echo "  ok $1: $2"
  pass=$((pass + 1))
}
bad() {
  echo "  FAIL $1: $2" >&2
  fail=$((fail + 1))
}

ROOT="$WORK/root"
mkdir -p "$ROOT"

write_work() {
  {
    echo "go 1.21"
    echo ""
    echo "use ("
    local m
    for m in "$@"; do
      echo "	./$m"
    done
    echo ")"
  } >"$ROOT/go.work"
}

make_module() {
  mkdir -p "$ROOT/$1"
  printf 'module example.com/%s\n\ngo 1.21\n' "$1" >"$ROOT/$1/go.mod"
}

run_gate() {
  local list="${1-__unset__}"
  shift || true
  if [[ $list == "__unset__" ]]; then
    MODULE_ISOLATION_ROOT="$ROOT" bash "$GATE" "$@" 2>&1
  else
    MODULE_ISOLATION_ROOT="$ROOT" MODULE_ISOLATION_TEST_MODULES="$list" bash "$GATE" "$@" 2>&1
  fi
}

# alpha: builds, vets, and its test passes.
make_module alpha
printf 'package alpha\n\nfunc Add(a, b int) int { return a + b }\n' >"$ROOT/alpha/alpha.go"
printf 'package alpha\n\nimport "testing"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(1, 2) != 3 {\n\t\tt.Fatal("planted arithmetic failure")\n\t}\n}\n' >"$ROOT/alpha/alpha_test.go"

# beta: builds and vets; its test is rewritten per case.
make_module beta
printf 'package beta\n\nfunc Val() int { return 1 }\n' >"$ROOT/beta/beta.go"
beta_test_passing='package beta

import "testing"

func TestVal(t *testing.T) {
	if Val() != 1 {
		t.Fatal("planted value failure")
	}
}
'
beta_test_failing='package beta

import "testing"

func TestVal(t *testing.T) {
	t.Fatal("planted test failure")
}
'
printf '%s' "$beta_test_passing" >"$ROOT/beta/beta_test.go"

# gamma: a build failure (undefined symbol in production code).
make_module gamma
printf 'package gamma\n\nvar _ = undefinedSymbol\n' >"$ROOT/gamma/gamma.go"

# delta: builds, but vet refuses it (wrong printf argument type).
make_module delta
printf 'package delta\n\nimport "fmt"\n\nfunc Speak() { fmt.Printf("%%d", "text") }\n' >"$ROOT/delta/delta.go"

# Case 1: clean tree, test leg off (neither module is on the default list).
write_work alpha beta
if out=$(run_gate __unset__) && printf '%s' "$out" | grep -q 'example.com/alpha \.\.\. OK$' &&
  ! printf '%s' "$out" | grep -q 'vet, test'; then
  ok 1 "clean tree passes without the test leg"
else
  bad 1 "clean tree should pass with plain OK lines; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 2: clean tree, both modules on the test list.
if out=$(run_gate "alpha beta") && printf '%s' "$out" | grep -q 'example.com/alpha \.\.\. OK (build, vet, test)' &&
  printf '%s' "$out" | grep -q 'example.com/beta \.\.\. OK (build, vet, test)' &&
  printf '%s' "$out" | grep -q 'tests ran hermetically for: alpha, beta'; then
  ok 2 "listed modules report the test verdict and the summary names them"
else
  bad 2 "both listed modules should report OK (build, vet, test); output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

printf '%s' "$beta_test_failing" >"$ROOT/beta/beta_test.go"

# Case 3: the planted failing test is on the list.
if out=$(run_gate "alpha beta"); then
  bad 3 "a failing test on the list must fail the gate; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
elif printf '%s' "$out" | grep -q 'example.com/beta \.\.\. FAIL (test)' &&
  printf '%s' "$out" | grep -q 'cd beta && GOWORK=off go test' &&
  printf '%s' "$out" | grep -q 'gotcha 30' &&
  printf '%s' "$out" | grep -q 'example.com/alpha \.\.\. OK (build, vet, test)'; then
  ok 3 "a failing listed test fails the gate with the reproduce hint; the sibling stays OK"
else
  bad 3 "wrong failure output; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 4: the same failing test with the list unset (the default list names integration_test only).
if out=$(run_gate __unset__) && printf '%s' "$out" | grep -q 'example.com/beta \.\.\. OK$'; then
  ok 4 "the list gates the leg: an unlisted failing test does not run"
else
  bad 4 "unlisted failing test should not fail the gate; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 5: the explicit `none`.
if out=$(run_gate none) && printf '%s' "$out" | grep -q 'example.com/beta \.\.\. OK$'; then
  ok 5 "the literal none disables the leg"
else
  bad 5 "none should disable the leg; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 6: a build failure.
write_work alpha gamma
if out=$(run_gate none); then
  bad 6 "a build failure must fail the gate; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
elif printf '%s' "$out" | grep -q 'example.com/gamma \.\.\. FAIL (build)'; then
  ok 6 "a build failure reports FAIL (build)"
else
  bad 6 "wrong failure output; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 7: a vet failure.
write_work alpha delta
if out=$(run_gate none); then
  bad 7 "a vet failure must fail the gate; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
elif printf '%s' "$out" | grep -q 'example.com/delta \.\.\. FAIL (vet)'; then
  ok 7 "a vet failure reports FAIL (vet)"
else
  bad 7 "wrong failure output; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 8: usage error.
write_work alpha beta
printf '%s' "$beta_test_passing" >"$ROOT/beta/beta_test.go"
if out=$(run_gate none --bogus); then
  bad 8 "an unknown flag must exit 2; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
elif [ "$?" -eq 2 ] && printf '%s' "$out" | grep -q 'usage:'; then
  ok 8 "an unknown flag exits 2 with a usage line"
else
  bad 8 "wrong usage output (rc $?); output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 9: --examples with the test leg.
if out=$(run_gate alpha --examples) && printf '%s' "$out" | grep -q 'incl. e2e/ + examples/' &&
  printf '%s' "$out" | grep -q 'example.com/alpha \.\.\. OK (build, vet, test)'; then
  ok 9 "--examples keeps the test leg"
else
  bad 9 "--examples with a listed module should pass; output:
$(printf '%s\n' "$out" | sed 's/^/      /')"
fi

# Case 10: the real gate keeps the consumer-test module on its list.
if grep -q '^HERMETIC_TEST_MODULES=(integration_test)$' "$GATE"; then
  ok 10 "the real gate runs integration_test's tests hermetically"
else
  bad 10 "HERMETIC_TEST_MODULES no longer lists integration_test in $GATE"
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-check-module-isolation: $fail case(s) FAILED"
  exit 1
fi
echo "test-check-module-isolation: all cases green"
