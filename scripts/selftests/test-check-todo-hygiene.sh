#!/usr/bin/env bash
# test-check-todo-hygiene.sh — fixture self-test for check-todo-hygiene.sh.
#
# Cases (each planted failure must BE a failure — a gate that only passes is
# not a gate):
#   1. clean backlog                                  -> exit 0
#   2. `[x]` row                                      -> exit 1, RULE1
#   3. struck-title row                               -> exit 1, RULE2
#   4. inline strike INSIDE a live row                -> exit 0 (not a dead row)
#   5. section 40% dead                               -> exit 1, RULE3
#   6. header paragraph >300 chars                    -> exit 1, RULE4-PARA
#   7. header total >1,500 chars                       -> exit 1, RULE4-TOTAL
#   8. missing TODO_LIST.md                           -> exit 2
#   9. empty backlog (no rows)                        -> exit 0 (no divide-by-zero)
#  10. --verbose prints per-section row/dead counts   -> exit 0, SECTION lines
#
# Usage: bash scripts/selftests/test-check-todo-hygiene.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-todo-hygiene.sh"

WORK=$(mktemp -d /tmp/test-todo-hygiene-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

run_gate() {
  TODO_HYGIENE_ROOT="$1" bash "$GATE" "${@:2}" 2>&1
}

check_case() {
  local name="$1" root="$2" want_rc="$3" want_text="$4"
  shift 4
  local out rc
  out=$(run_gate "$root" "$@")
  rc=$?
  if [ "$rc" -ne "$want_rc" ]; then
    echo "  FAIL $name: exit $rc, want $want_rc" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
    return
  fi
  if [ -n "$want_text" ] && ! printf '%s' "$out" | grep -q "$want_text"; then
    echo "  FAIL $name: output lacks '$want_text'" >&2
    printf '%s\n' "$out" | sed 's/^/      /' >&2
    fail=$((fail + 1))
    return
  fi
  echo "  ok $name"
  pass=$((pass + 1))
}

mk() {
  mkdir -p "$1"
  cat >"$1/TODO_LIST.md"
}

# 1. clean
mk "$WORK/clean" <<'EOF'
# TODO List — fixture

> Short pointer. Done work lives in CHANGELOG.md.

**Updated:** 2026-10-24.

## P1 — High impact

- [ ] **First task.** Evidence: `file.go:1`. **Next:** do it.
- [~] **Second task.** Started; **Next:** finish it.

## P2 — Medium

- [ ] **Third task.** Evidence: `file.go:2`.
EOF
check_case 1-clean "$WORK/clean" 0 "todo-hygiene: clean"

# 2. [x] row
mk "$WORK/xrow" <<'EOF'
# TODO List — fixture

## P1

- [ ] **Open task.**
- [x] **Done task that must not live here.**
EOF
check_case 2-xrow "$WORK/xrow" 1 "RULE1"

# 3. struck-title row
mk "$WORK/dead" <<'EOF'
# TODO List — fixture

## P1

- [ ] ~~**Old dead row**~~ done at abc1234.
- [ ] **Live row.**
EOF
check_case 3-struck-title "$WORK/dead" 1 "RULE2"

# 4. inline strike inside a live row is legitimate
mk "$WORK/inline" <<'EOF'
# TODO List — fixture

## P1

- [~] **Live row.** ~~First half~~ done at abc1234; **Next:** second half.
EOF
check_case 4-inline-strike "$WORK/inline" 0 "todo-hygiene: clean"

# 5. section 40% dead
mk "$WORK/section" <<'EOF'
# TODO List — fixture

## P1

- [ ] ~~**Dead one**~~ done at a1.
- [ ] ~~**Dead two**~~ done at a2.
- [ ] **Live one.**
- [ ] **Live two.**
- [ ] **Live three.**
EOF
check_case 5-section-death "$WORK/section" 1 "RULE3"

# 6. header paragraph too long
LONG=$(printf 'x%.0s' $(seq 1 320))
mk "$WORK/longpara" <<EOF
# TODO List — fixture

> $LONG

## P1

- [ ] **Live row.**
EOF
check_case 6-long-paragraph "$WORK/longpara" 1 "RULE4-PARA"

# 7. header total too long (many short paragraphs)
{
  mkdir -p "$WORK/bighead"
  echo "# TODO List — fixture" >"$WORK/bighead/TODO_LIST.md"
  echo "" >>"$WORK/bighead/TODO_LIST.md"
  i=0
  while [ "$i" -lt 22 ]; do
    printf '> paragraph number %s with enough words to add up quickly over twenty-two blocks.\n\n' "$i" >>"$WORK/bighead/TODO_LIST.md"
    i=$((i + 1))
  done
  printf '## P1\n\n- [ ] **Live row.**\n' >>"$WORK/bighead/TODO_LIST.md"
}
check_case 7-header-total "$WORK/bighead" 1 "RULE4-TOTAL"

# 8. missing file
mkdir -p "$WORK/missing"
check_case 8-missing "$WORK/missing" 2 "not found"

# 9. empty backlog
mk "$WORK/empty" <<'EOF'
# TODO List — fixture

## P1

Nothing actionable.
EOF
check_case 9-empty "$WORK/empty" 0 "todo-hygiene: clean"

# 10. verbose sections
out=$(run_gate "$WORK/clean" --verbose)
if printf '%s' "$out" | grep -q "SECTION P1" && printf '%s' "$out" | grep -q "rows=2 dead=0"; then
  echo "  ok 10-verbose-sections"
  pass=$((pass + 1))
else
  echo "  FAIL 10-verbose-sections: per-section counts missing" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

echo "todo-hygiene self-test: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
