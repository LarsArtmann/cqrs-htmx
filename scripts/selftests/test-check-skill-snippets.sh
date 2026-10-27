#!/usr/bin/env bash
# test-check-skill-snippets.sh — fixture self-test for check-skill-snippets.sh.
#
# Cases:
#   1. a compilable fence → pass
#   2. a mangled fence (unbalanced brace) → fail
#   3. an api-tour fence (//doc:api-tour marker) → skipped, not failed
#   4. a signature-reference fence (no code lines) → skipped
#   5. a statement fragment fence → passes via the func wrap
#
# Usage: bash scripts/selftests/test-check-skill-snippets.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-skill-snippets.sh"

WORK=$(mktemp -d /tmp/test-skill-snippets-XXXXXX)
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

make_skill() {
  local r="$1" body="$2"
  mkdir -p "$r/.agents/skills/x"
  {
    echo "doc"
    echo '```go'
    printf '%s\n' "$body"
    echo '```'
  } >"$r/.agents/skills/x/SKILL.md"
}

# 1. compilable fence
R="$WORK/case1"
make_skill "$R" 'package x

func Hello() string { return "hi" }'
out=$(SKILL_SNIPPETS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "1 code fence"; then ok "1-compiles"; else bad "1-compiles" "rc=$rc $out"; fi

# 2. mangled fence
R="$WORK/case2"
make_skill "$R" 'func Broken() {
	fmt.Println("no closing brace"'
out=$(SKILL_SNIPPETS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "does not parse"; then ok "2-mangled-fails"; else bad "2-mangled-fails" "rc=$rc $out"; fi

# 3. api-tour marker skips
R="$WORK/case3"
make_skill "$R" '//doc:api-tour
b := cqrshtmx.NewBroadcaster()
b.Subscribe()   // -> chan SSEEvent'
out=$(SKILL_SNIPPETS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "1 signature-reference fence"; then ok "3-api-tour-skipped"; else bad "3-api-tour-skipped" "rc=$rc $out"; fi

# 4. signature-reference fence (no code lines) skips
R="$WORK/case4"
make_skill "$R" 'cqrshtmx.Foo(a) / cqrshtmx.Bar(b)   // annotated tour'
out=$(SKILL_SNIPPETS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ]; then ok "4-signature-skips"; else bad "4-signature-skips" "rc=$rc $out"; fi

# 5. statement fragment passes via the func wrap
R="$WORK/case5"
make_skill "$R" 'mux := http.NewServeMux()
_ = mux'
out=$(SKILL_SNIPPETS_ROOT="$R" bash "$TOOL" 2>&1)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "1 code fence"; then ok "5-func-wrap"; else bad "5-func-wrap" "rc=$rc $out"; fi

echo "skill-snippets self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
