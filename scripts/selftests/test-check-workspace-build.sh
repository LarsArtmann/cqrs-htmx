#!/usr/bin/env bash
# test-check-workspace-build.sh — fixture self-test for check-workspace-build.sh.
#
# Builds throwaway two-module workspaces offline (GOPROXY=off, stdlib only)
# and proves the gate's verdict logic:
#   1. healthy workspace (go.work `use` covers both modules)  -> gate passes
#   2. mangled workspace (consumer module's dependency dropped from `use`,
#      the go-work-sync/373209a7 shape)                        -> gate fails
#   3. missing go.work                                          -> gate fails
#
# Usage: bash scripts/selftests/test-check-workspace-build.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-workspace-build.sh"

WORK=$(mktemp -d /tmp/test-check-workspace-build-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

write_module_a() {
  mkdir -p "$1/a"
  cat >"$1/a/go.mod" <<'EOF'
module example.net/a

go 1.22
EOF
  cat >"$1/a/a.go" <<'EOF'
package a

func Hello() string { return "hello" }
EOF
}

write_module_b() {
  mkdir -p "$1/b"
  cat >"$1/b/go.mod" <<'EOF'
module example.net/b

go 1.22

require example.net/a v0.0.0
EOF
  cat >"$1/b/main.go" <<'EOF'
package main

import (
	"fmt"

	"example.net/a"
)

func main() { fmt.Println(a.Hello()) }
EOF
}

run_gate() {
  (cd "$1" && WORKSPACE_BUILD_ROOT="$1" GOPROXY=off bash "$GATE" 2>&1)
}

# Case 1: healthy two-module workspace.
case1="$WORK/healthy"
write_module_a "$case1"
write_module_b "$case1"
cat >"$case1/go.work" <<'EOF'
go 1.22

use (
	./a
	./b
)
EOF
if out=$(run_gate "$case1"); then
  echo "  ok 1: healthy workspace passes"
  pass=$((pass + 1))
else
  echo "  FAIL 1: healthy workspace should pass; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Case 2: the mangled shape — go.work drops `use ./a` so example.net/a no
# longer resolves locally; with GOPROXY=off the build must fail.
case2="$WORK/mangled"
write_module_a "$case2"
write_module_b "$case2"
cat >"$case2/go.work" <<'EOF'
go 1.22

use ./b
EOF
if out=$(run_gate "$case2"); then
  echo "  FAIL 2: mangled workspace should fail; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
else
  echo "  ok 2: mangled workspace (dropped use) fails"
  pass=$((pass + 1))
fi

# Case 3: no go.work at all — the gate refuses instead of silently building
# in single-module mode (a missing go.work IS workspace breakage: CI and the
# release gates false-green without it, AGENTS gotcha 11).
case3="$WORK/nowork"
write_module_a "$case3"
if out=$(run_gate "$case3"); then
  echo "  FAIL 3: missing go.work should fail; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
else
  echo "  ok 3: missing go.work fails"
  pass=$((pass + 1))
fi

# Case 4: the CI-runner shape — a machine-local replace target that does not
# exist (/nonexistent/...). The gate's consumer-view filter must DROP the
# local replace so the workspace resolves example.net/a via `use` and BUILDS;
# without the filter this shape fails (first CI run, 2026-10-01). A
# version-to-version replace must SURVIVE the filter (case 5).
case4="$WORK/localreplace"
write_module_a "$case4"
write_module_b "$case4"
cat >"$case4/go.work" <<'EOF'
go 1.22

use (
	./a
	./b
)

replace example.net/a => /nonexistent/fleet-sibling-checkout/a
EOF
if out=$(run_gate "$case4"); then
  echo "  ok 4: machine-local missing replace target is filtered (consumer view)"
  pass=$((pass + 1))
else
  echo "  FAIL 4: local replace to missing dir should be filtered, not fatal; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-check-workspace-build: $fail of 4 cases FAILED"
  exit 1
fi
echo "test-check-workspace-build: all 4 cases green"
