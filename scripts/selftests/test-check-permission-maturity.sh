#!/usr/bin/env bash
# test-check-permission-maturity.sh — fixture self-test for
# check-permission-maturity.sh. Offline; builds throwaway repo trees.
#
# The checker resolves its paths from its own location, so each case copies the
# script into a temp tree that looks like this repository.

set -uo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
checker="$script_dir/../checks/check-permission-maturity.sh"

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

new_tree() {
  local dir="$1"
  mkdir -p "$dir/scripts/checks" "$dir/docs"
  cp "$checker" "$dir/scripts/checks/check-permission-maturity.sh"
  chmod +x "$dir/scripts/checks/check-permission-maturity.sh"
}

doc_with() {
  # doc_with DIR SERVICE LEVEL
  cat >"$1/docs/PERMISSIONS.md" <<MD
| Service | Level | Posture |
| --- | --- | --- |
| \`$2\` | $3 | fixture |
MD
}

# --- case 1: same level passes ---------------------------------------------
t1="$tmp/same"
new_tree "$t1"
doc_with "$t1" svc-a L2
printf 'svc-a\tL2\n' >"$t1/docs/PERMISSIONS_BASELINE"

"$t1/scripts/checks/check-permission-maturity.sh" --ratchet >/dev/null
assert_rc 0 $? "unchanged level"

# --- case 2: a raised level passes -----------------------------------------
t2="$tmp/up"
new_tree "$t2"
doc_with "$t2" svc-a L3
printf 'svc-a\tL2\n' >"$t2/docs/PERMISSIONS_BASELINE"

"$t2/scripts/checks/check-permission-maturity.sh" --ratchet >/dev/null
assert_rc 0 $? "raised level"

# --- case 3: a dropped level fails under --ratchet -------------------------
t3="$tmp/down"
new_tree "$t3"
doc_with "$t3" svc-a L1
printf 'svc-a\tL2\n' >"$t3/docs/PERMISSIONS_BASELINE"

out="$("$t3/scripts/checks/check-permission-maturity.sh" --ratchet 2>&1)"
assert_rc 1 $? "dropped level"

if ! printf '%s' "$out" | grep -q 'REGRESSION: svc-a dropped from L2 to L1'; then
  echo "FAIL: drop not reported: $out"
  failures=$((failures + 1))
fi

# --- case 4: the same drop is advisory (exit 0) without --ratchet ----------
"$t3/scripts/checks/check-permission-maturity.sh" >/dev/null
assert_rc 0 $? "advisory mode"

# --- case 5: a service that vanished from the doc is a regression ----------
t5="$tmp/vanish"
new_tree "$t5"
doc_with "$t5" svc-a L2
printf 'svc-a\tL2\nsvc-b\tL1\n' >"$t5/docs/PERMISSIONS_BASELINE"

out="$("$t5/scripts/checks/check-permission-maturity.sh" --ratchet 2>&1)"
assert_rc 1 $? "vanished service"

if ! printf '%s' "$out" | grep -q 'REGRESSION: svc-b vanished'; then
  echo "FAIL: vanished service not reported: $out"
  failures=$((failures + 1))
fi

# --- case 6: '?' rows are unverified and not ratcheted --------------------
t6="$tmp/unknown"
new_tree "$t6"
doc_with "$t6" svc-a '?'
printf 'svc-a\tL3\n' >"$t6/docs/PERMISSIONS_BASELINE"

"$t6/scripts/checks/check-permission-maturity.sh" --ratchet >/dev/null
assert_rc 0 $? "unknown level is not ratcheted"

# --- case 7: a new service is reported, not a failure --------------------
t7="$tmp/new"
new_tree "$t7"
doc_with "$t7" svc-new L1
printf 'svc-a\tL2\n' >"$t7/docs/PERMISSIONS_BASELINE"
# Keep svc-a present so the vanished-service rule does not fire.
printf '| Service | Level |\n| --- | --- |\n| `svc-a` | L2 |\n| `svc-new` | L1 |\n' >"$t7/docs/PERMISSIONS.md"

out="$("$t7/scripts/checks/check-permission-maturity.sh" --ratchet 2>&1)"
assert_rc 0 $? "new service"

if ! printf '%s' "$out" | grep -q 'NEW: svc-new'; then
  echo "FAIL: new service not reported: $out"
  failures=$((failures + 1))
fi

# --- case 8: --update-baseline rewrites the baseline ---------------------
t8="$tmp/update"
new_tree "$t8"
doc_with "$t8" svc-a L3
printf 'svc-a\tL2\n' >"$t8/docs/PERMISSIONS_BASELINE"

"$t8/scripts/checks/check-permission-maturity.sh" --update-baseline >/dev/null
assert_rc 0 $? "update baseline"

if ! grep -q 'svc-a'$'\t''L3' "$t8/docs/PERMISSIONS_BASELINE"; then
  echo "FAIL: baseline was not raised to L3"
  failures=$((failures + 1))
fi

if [ "$failures" -ne 0 ]; then
  echo "check-permission-maturity self-test: $failures failure(s)"
  exit 1
fi

echo "check-permission-maturity self-test: 8/8 cases pass"
