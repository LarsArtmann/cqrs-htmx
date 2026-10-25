#!/usr/bin/env bash
# test-check-upstream-treadmill.sh — offline fixture self-test for
# check-upstream-treadmill.sh (no network: seeded tag cache + a recording
# fetch stub).
#
#   T1  aligned pin (warm cache)               -> exit 0, "[aligned]"
#   T2  lag 2, advisory                         -> exit 0, "[LAG 2]"
#   T3  lag 2, --strict-lag 1                   -> exit 3, names the row
#   T4  unknown prefix (sqliteengine trap)      -> exit 1, "UNKNOWN"
#   T5  cold fetch failure                      -> exit 4, repo named (loud)
#   T6  pseudo pin: non-exempt -> PSEUDO error; replace-exempt -> skipped
#   T7  cqrs-htmx self-requires ignored         -> no row, exit 0
#   T8  workspace disagreement                  -> DISAGREE warning, exit 0
#   T9  warm cache never calls the fetch stub   -> 0 fetch calls
#   T10 nested prefix + single-line require     -> metaengine/sqliteengine/
#                                                 derived; require-form parsed
#
# Usage: ./scripts/selftests/test-check-upstream-treadmill.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-upstream-treadmill.sh"

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

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

# new_fixture <name> — echoes the fixture tree dir; seeds:
#   $tree/mod-a/go.mod, $tree/mod-b/go.mod
#   $cache/<repo>.tags per SEED_TAGS_* content
#   $fetchlog — the recording fetch stub writes one line per call
new_fixture() {
  local tree="$work/$1"
  mkdir -p "$tree/mod-a" "$tree/mod-b" "$tree/cache"
  : >"$work/fetchlog"
  printf '#!/usr/bin/env bash\necho "FETCH-CALLED $*" >>%q\n' "$work/fetchlog" >>"$tree/fetch"
  return 0
}

run_gate() { # <fixture-dir> [args...] — echoes output; sets RC
  # env -u CI: the LOCAL branch is the default under test; T11 covers the
  # CI=true downgrade explicitly (selftest-env-leaks requires the strip).
  local tree="$1"
  shift
  OUT="$(env -u CI TREADMILL_ROOT="$tree" \
    TRAIN_TAG_CACHE_DIR="$tree/cache" \
    TRAIN_TAG_CACHE_TTL=86400 \
    TREADMILL_FETCH_TAGS_CMD="bash $tree/fetch" \
    bash "$GATE" "$@" 2>&1)"
  RC=$?
}

seed_tags() { # <fixture> <repo> <tag lines...>
  local fixture="$1" repo="$2"
  shift 2
  mkdir -p "$work/$fixture/cache"
  printf '%s\n' "$@" >"$work/$fixture/cache/$repo.tags"
}

mod_a() { # <fixture> <go.mod content>
  printf '%s\n' "$2" >"$work/$1/mod-a/go.mod"
}

mod_b() { # <fixture> <go.mod content>
  printf '%s\n' "$2" >"$work/$1/mod-b/go.mod"
}

echo ""
echo "=== Test Suite: check-upstream-treadmill.sh ==="
echo ""

# --- T1: aligned pin, warm cache --------------------------------------------
new_fixture t1
mod_a t1 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-retry v0.8.0 // indirect
)'
seed_tags t1 go-retry 'v0.7.0' 'v0.8.0'
run_gate "$work/t1"
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "T1 aligned pin exits 0 (got $RC)"
report "$(echo "$OUT" | grep -q '\[aligned\]' && echo 0 || echo 1)" "T1 row reports [aligned]"

# --- T2: lag 2, advisory ------------------------------------------------------
new_fixture t2
mod_a t2 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-sse v0.6.2
)'
seed_tags t2 go-sse 'v0.6.2' 'v0.6.3' 'v0.6.4'
run_gate "$work/t2"
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "T2 lag 2 advisory exits 0 (got $RC)"
report "$(echo "$OUT" | grep -q '\[LAG 2\]' && echo 0 || echo 1)" "T2 row reports [LAG 2]"

# --- T3: lag 2, strict-lag 1 --------------------------------------------------
run_gate "$work/t2" --strict-lag 1
report "$([ "$RC" -eq 3 ] && echo 0 || echo 1)" "T3 lag 2 over budget 1 exits 3 (got $RC)"
report "$(echo "$OUT" | grep -q 'worst lag 2 exceeds budget 1' && echo 0 || echo 1)" "T3 verdict names worst lag and budget"

# --- T4: unknown prefix (the sqliteengine trap) -------------------------------
new_fixture t4
mod_a t4 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-cqrs-lite/sqliteengine/v4 v4.6.0
)'
seed_tags t4 go-cqrs-lite 'metaengine/sqliteengine/v4.6.0' 'metaengine/sqliteengine/v4.6.1'
run_gate "$work/t4"
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "T4 wrong-prefix derivation exits 1 (got $RC)"
report "$(echo "$OUT" | grep -q 'UNKNOWN' && echo 0 || echo 1)" "T4 reports UNKNOWN, not a silent zero"

# --- T5: cold fetch failure is LOUD -------------------------------------------
new_fixture t5
mod_a t5 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-health v0.7.0
)'
printf '#!/usr/bin/env bash\nexit 1\n' >"$work/t5/fetch"
run_gate "$work/t5"
report "$([ "$RC" -eq 4 ] && echo 0 || echo 1)" "T5 fetch failure exits 4 (got $RC)"
report "$(echo "$OUT" | grep -q 'go-health' && echo 0 || echo 1)" "T5 names the unverifiable repo"

# --- T6: pseudo pins — non-exempt errors, replace-exempt skips ----------------
new_fixture t6
mod_a t6 'module example.com/a
go 1.27

require github.com/larsartmann/go-output v0.0.0-20261010120000-abcdefabcdef

replace github.com/larsartmann/go-output => ../go-output
'
mod_b t6 'module example.com/b
go 1.27

require (
	github.com/larsartmann/go-ndjson v0.0.0-20261001120000-123456789012
)'
seed_tags t6 go-output 'v0.38.7'
run_gate "$work/t6"
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "T6 non-exempt pseudo pin exits 1 (got $RC)"
report "$(echo "$OUT" | grep -q 'PSEUDO' && echo 0 || echo 1)" "T6 reports PSEUDO"
report "$(echo "$OUT" | grep 'PSEUDO' | grep -q 'go-ndjson' && echo "$OUT" | grep -c 'PSEUDO' | grep -q '^1$' && echo 0 || echo 1)" "T6 replace-exempt pseudo (go-output) is skipped, not PSEUDO"

# --- T7: cqrs-htmx self-requires are the release-train's lane -----------------
new_fixture t7
mod_a t7 'module example.com/a
go 1.27

require (
	github.com/larsartmann/cqrs-htmx/usermgmt/v4 v4.16.0
)'
run_gate "$work/t7"
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "T7 self-repo require ignored, exit 0 (got $RC)"
report "$(echo "$OUT" | grep -q 'usermgmt' && echo 1 || echo 0)" "T7 no row for the self-require"

# --- T8: workspace disagreement warns but stays advisory-green -----------------
new_fixture t8
mod_a t8 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-retry v0.8.0
)'
mod_b t8 'module example.com/b
go 1.27

require (
	github.com/larsartmann/go-retry v0.7.0
)'
seed_tags t8 go-retry 'v0.8.0'
run_gate "$work/t8"
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "T8 disagreement stays advisory exit 0 (got $RC)"
report "$(echo "$OUT" | grep -q 'DISAGREE' && echo 0 || echo 1)" "T8 reports DISAGREE warning"

# --- T9: warm cache never calls the fetch stub ---------------------------------
: >"$work/fetchlog-check"
new_fixture t9
mod_a t9 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-retry v0.8.0
)'
seed_tags t9 go-retry 'v0.8.0'
run_gate "$work/t9"
fetch_calls=$(wc -l <"$work/fetchlog")
report "$([ "$fetch_calls" -eq 0 ] && echo 0 || echo 1)" "T9 warm cache: 0 fetch calls (got $fetch_calls)"

# --- T10: nested prefix derivation + single-line require form -------------------
new_fixture t10
mod_a t10 'module example.com/a
go 1.27

require github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4 v4.6.0'
seed_tags t10 go-cqrs-lite 'metaengine/sqliteengine/v4.6.0' 'metaengine/sqliteengine/v4.6.1' 'metaengine/v4.20.0'
run_gate "$work/t10"
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "T10 nested prefix + single-line require exit 0 (got $RC)"
report "$(echo "$OUT" | grep -q 'metaengine/sqliteengine/' && echo 0 || echo 1)" "T10 derives metaengine/sqliteengine/ (the trap, positive case)"
report "$(echo "$OUT" | grep -q '\[LAG 1\]' && echo 0 || echo 1)" "T10 counts lag 1 against v4.6.1"

# --- T11: CI=true downgrades fetch failure to a warning (house rule) -------------
new_fixture t11
mod_a t11 'module example.com/a
go 1.27

require (
	github.com/larsartmann/go-health v0.7.0
)'
printf '#!/usr/bin/env bash\nexit 1\n' >"$work/t11/fetch"
OUT="$(env CI=true TREADMILL_ROOT="$work/t11" \
  TRAIN_TAG_CACHE_DIR="$work/t11/cache" \
  TRAIN_TAG_CACHE_TTL=86400 \
  TREADMILL_FETCH_TAGS_CMD="bash $work/t11/fetch" \
  bash "$GATE" 2>&1)"
RC=$?
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "T11 CI=true fetch failure downgrades to exit 0 (got $RC)"
report "$(echo "$OUT" | grep -q 'WARN: cannot list tags of go-health' && echo 0 || echo 1)" "T11 emits the CI advisory warning"

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
echo "All tests passed."
