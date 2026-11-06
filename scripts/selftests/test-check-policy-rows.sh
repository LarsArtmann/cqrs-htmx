#!/usr/bin/env bash
# test-check-policy-rows.sh — fixture self-test for check-policy-rows.sh.
# Offline; builds throwaway declaration and policy files.
#
# Cases:
#   1. every row matches a declared object (exit 0)
#   2. an orphan row fails and names the declared neighbours
#   3. a "*" row is never an orphan
#   4. a prefix row that covers a declared object passes
#   5. a prefix row that covers nothing fails
#   6. grouping rows (g/g2) and comments are ignored
#   7. a `policy-rows-allow` suppression is honoured and counted
#   8. no declarations is a usage error, not a green

set -uo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
checker="$script_dir/../checks/check-policy-rows.sh"

if [ ! -f "$checker" ]; then
  echo "FAIL: checker missing: $checker" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

failures=0
assert_rc() {
  local want="$1" got="$2" label="$3"
  if [ "$got" -ne "$want" ]; then
    echo "FAIL: $label: exit $got, want $want"
    failures=$((failures + 1))
  fi
}

declarations="$tmp/declarations.txt"
cat >"$declarations" <<'DECL'
# command dispatch objects declared at registration (gate.Manifest())
command/CreateItem
command/DeleteItem
query/ListItems
DECL

# --- case 1: clean --------------------------------------------------------
clean="$tmp/clean.csv"
cat >"$clean" <<'CSV'
# subject, domain, object, action, effect
p, role_user, *, command/CreateItem, execute, allow
p, role_admin, *, command/*, execute, allow
CSV

"$checker" --declarations "$declarations" --policies "$clean" >/dev/null
assert_rc 0 $? "clean rows"

# --- case 2: orphan row ---------------------------------------------------
orphan="$tmp/orphan.csv"
cat >"$orphan" <<'CSV'
p, role_user, *, command/CreateItem, execute, allow
p, role_user, *, command/DeleteEverything, execute, allow
CSV

out="$("$checker" --declarations "$declarations" --policies "$orphan" 2>&1)"
assert_rc 1 $? "orphan row"

if ! printf '%s' "$out" | grep -q 'command/DeleteEverything'; then
  echo "FAIL: orphan row not named: $out"
  failures=$((failures + 1))
fi

if ! printf '%s' "$out" | grep -q 'declared under command/: command/CreateItem'; then
  echo "FAIL: orphan row did not suggest its declared neighbours: $out"
  failures=$((failures + 1))
fi

# --- case 3: a bare star is never an orphan ------------------------------
star="$tmp/star.csv"
cat >"$star" <<'CSV'
p, role_admin, *, *, *, allow
CSV

"$checker" --declarations "$declarations" --policies "$star" >/dev/null
assert_rc 0 $? "star row"

# --- case 4: a covering prefix passes ------------------------------------
prefix_ok="$tmp/prefix_ok.csv"
cat >"$prefix_ok" <<'CSV'
p, role_user, *, query/*, read, allow
CSV

"$checker" --declarations "$declarations" --policies "$prefix_ok" >/dev/null
assert_rc 0 $? "covering prefix"

# --- case 5: a prefix that covers nothing fails --------------------------
prefix_bad="$tmp/prefix_bad.csv"
cat >"$prefix_bad" <<'CSV'
p, role_user, *, invoice/*, read, allow
CSV

"$checker" --declarations "$declarations" --policies "$prefix_bad" >/dev/null
assert_rc 1 $? "covering-nothing prefix"

# --- case 6: grouping rows and comments are ignored ----------------------
mixed="$tmp/mixed.csv"
cat >"$mixed" <<'CSV'
# a comment that looks like a row: p, x, *, nowhere/*, read, allow
g, alice, role_user, *
g2, role_admin, role_user
p, role_user, *, command/CreateItem, execute, allow
CSV

"$checker" --declarations "$declarations" --policies "$mixed" >/dev/null
assert_rc 0 $? "grouping rows ignored"

# --- case 7: suppression -------------------------------------------------
suppressed="$tmp/suppressed.csv"
cat >"$suppressed" <<'CSV'
p, role_user, *, command/CreateItem, execute, allow
# policy-rows-allow: pre-declared for the next release.
p, role_user, *, command/FutureThing, execute, allow
CSV

out="$("$checker" --declarations "$declarations" --policies "$suppressed" 2>&1)"
assert_rc 0 $? "suppressed orphan"

if ! printf '%s' "$out" | grep -q 'suppressed: 1'; then
  echo "FAIL: suppression not counted: $out"
  failures=$((failures + 1))
fi

# --- case 8: empty declarations is a usage error ------------------------
empty="$tmp/empty.txt"
: >"$empty"

"$checker" --declarations "$empty" --policies "$clean" >/dev/null 2>&1
assert_rc 2 $? "empty declarations"

if [ "$failures" -ne 0 ]; then
  echo "check-policy-rows self-test: $failures failure(s)"
  exit 1
fi

echo "check-policy-rows self-test: 8/8 cases pass"
