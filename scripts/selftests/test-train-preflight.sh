#!/usr/bin/env bash
# test-train-preflight.sh — fixture self-test for train-preflight.sh (offline)
#
#   F1  all stub stages pass with canaries      -> exit 0, ALL STAGES GREEN
#   F2  one stage fails (tests)                 -> exit 1, stage named
#   F3  guard miss: lint rc=0 but zero markers  -> exit 1, GUARD MISS (gotcha 2)
#   F4  train guard miss: no "internal requires" -> exit 1, GUARD MISS
#   F5  keep-going runs every stage             -> exit 1, all five stages ran
#   F6  usage: outside a git repository         -> exit 2
#
# Runtime: seconds (all stages are shell stubs; no nix, no go).
#
# Usage: ./scripts/selftests/test-train-preflight.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/train-preflight.sh"

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

new_repo() { # <dir>
  git init -q -b main "$1" || return 1
  git -C "$1" config user.email test@example.com
  git -C "$1" config user.name test
  git -C "$1" config commit.gpgsign false
  git -C "$1" commit -q --allow-empty -m initial
}

# Stub stage commands. PASS_STAGES emit the canary lines the guards require;
# individual fixtures override the map entries they want to break.
stub_dir=$(mktemp -d)
trap 'rm -rf "$tmp" "$stub_dir"' EXIT
tmp=$(mktemp -d)

write_stubs() { # <fail-stage-name-or-empty>
  local broken="$1"
  for stage in quiescence surprise lint tests train; do
    local body="echo '==> module-$stage'; echo 'checked 1 internal requires'; exit 0"
    if [ "$stage" = "quiescence" ] || [ "$stage" = "surprise" ]; then
      body="echo 'ok'; exit 0"
    fi
    if [ "$stage" = "train" ]; then
      body="echo 'Checked 842 internal requires: 0 unpublished.'; exit 0"
    fi
    if [ "$stage" = "$broken" ]; then
      body="echo 'simulated $stage failure'; exit 1"
    fi
    printf '#!/usr/bin/env bash\n%s\n' "$body" >"$stub_dir/$stage.sh"
    chmod +x "$stub_dir/$stage.sh"
  done
}

echo "== test-train-preflight.sh"

# F1 all green
r="$tmp/f1"
new_repo "$r"
write_stubs ""
(
  cd "$r" || exit 1
  export TRAIN_PREFLIGHT_QUIET_CMD="bash $stub_dir/quiescence.sh"
  export TRAIN_PREFLIGHT_PREFLIGHT_CMD="bash $stub_dir/surprise.sh"
  export TRAIN_PREFLIGHT_LINT_CMD="bash $stub_dir/lint.sh"
  export TRAIN_PREFLIGHT_TEST_CMD="bash $stub_dir/tests.sh"
  export TRAIN_PREFLIGHT_TRAIN_CMD="bash $stub_dir/train.sh"
  bash "$CHECKER"
) >"$tmp/f1.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "ALL STAGES GREEN" "$tmp/f1.out" || ok=1
report $ok "F1 all stub stages green -> exit 0 (got $rc)"

# F2 tests stage fails
r="$tmp/f2"
new_repo "$r"
write_stubs "tests"
(
  cd "$r" || exit 1
  export TRAIN_PREFLIGHT_QUIET_CMD="bash $stub_dir/quiescence.sh"
  export TRAIN_PREFLIGHT_PREFLIGHT_CMD="bash $stub_dir/surprise.sh"
  export TRAIN_PREFLIGHT_LINT_CMD="bash $stub_dir/lint.sh"
  export TRAIN_PREFLIGHT_TEST_CMD="bash $stub_dir/tests.sh"
  export TRAIN_PREFLIGHT_TRAIN_CMD="bash $stub_dir/train.sh"
  bash "$CHECKER"
) >"$tmp/f2.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "tests FAILED" "$tmp/f2.out" || ok=1
report $ok "F2 failing tests stage -> exit 1, named (got $rc)"

# F3 guard miss: lint passes but prints NO candidate markers
r="$tmp/f3"
new_repo "$r"
write_stubs ""
printf '#!/usr/bin/env bash\nexit 0\n' >"$stub_dir/lint.sh"
(
  cd "$r" || exit 1
  export TRAIN_PREFLIGHT_QUIET_CMD="bash $stub_dir/quiescence.sh"
  export TRAIN_PREFLIGHT_PREFLIGHT_CMD="bash $stub_dir/surprise.sh"
  export TRAIN_PREFLIGHT_LINT_CMD="bash $stub_dir/lint.sh"
  export TRAIN_PREFLIGHT_TEST_CMD="bash $stub_dir/tests.sh"
  export TRAIN_PREFLIGHT_TRAIN_CMD="bash $stub_dir/train.sh"
  bash "$CHECKER"
) >"$tmp/f3.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "GUARD MISS" "$tmp/f3.out" || ok=1
grep -q "FALSE green" "$tmp/f3.out" || ok=1
report $ok "F3 lint zero-marker guard miss -> exit 1 (got $rc)"

# F4 guard miss: train passes without the requires tally
r="$tmp/f4"
new_repo "$r"
write_stubs ""
printf '#!/usr/bin/env bash\necho "no tally here"; exit 0\n' >"$stub_dir/train.sh"
(
  cd "$r" || exit 1
  export TRAIN_PREFLIGHT_QUIET_CMD="bash $stub_dir/quiescence.sh"
  export TRAIN_PREFLIGHT_PREFLIGHT_CMD="bash $stub_dir/surprise.sh"
  export TRAIN_PREFLIGHT_LINT_CMD="bash $stub_dir/lint.sh"
  export TRAIN_PREFLIGHT_TEST_CMD="bash $stub_dir/tests.sh"
  export TRAIN_PREFLIGHT_TRAIN_CMD="bash $stub_dir/train.sh"
  bash "$CHECKER"
) >"$tmp/f4.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "GUARD MISS" "$tmp/f4.out" || ok=1
report $ok "F4 train tally guard miss -> exit 1 (got $rc)"

# F5 keep-going runs every stage and reports aggregate
r="$tmp/f5"
new_repo "$r"
write_stubs "tests"
(
  cd "$r" || exit 1
  export TRAIN_PREFLIGHT_KEEP_GOING=1
  export TRAIN_PREFLIGHT_QUIET_CMD="bash $stub_dir/quiescence.sh"
  export TRAIN_PREFLIGHT_PREFLIGHT_CMD="bash $stub_dir/surprise.sh"
  export TRAIN_PREFLIGHT_LINT_CMD="bash $stub_dir/lint.sh"
  export TRAIN_PREFLIGHT_TEST_CMD="bash $stub_dir/tests.sh"
  export TRAIN_PREFLIGHT_TRAIN_CMD="bash $stub_dir/train.sh"
  bash "$CHECKER"
) >"$tmp/f5.out" 2>&1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
for stage in quiescence surprise lint tests train; do
  grep -q "=== train-preflight: $stage ===" "$tmp/f5.out" || ok=1
done
report $ok "F5 keep-going reports every stage, aggregate rc 1 (got $rc)"

# F6 outside a git repository -> exit 2
d="$tmp/f6"
mkdir -p "$d"
(cd "$d" && bash "$CHECKER") >"$tmp/f6.out" 2>&1
rc=$?
report "$([ "$rc" -eq 2 ] && echo 0 || echo 1)" "F6 outside git repo -> exit 2 (got $rc)"

echo ""
echo "test-train-preflight: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
exit 0
