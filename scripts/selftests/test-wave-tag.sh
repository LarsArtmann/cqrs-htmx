#!/usr/bin/env bash
# test-wave-tag.sh — offline fixture self-test for wave-tag.sh (every leg is
# a recording stub; no git operations, no network).
#
#   WT1 dry-run happy path        -> exit 0; guards ran; legs 1-4 printed as
#                                    commands, stubs for push/tag/sweep NOT called
#   WT2 dry-run, guards refuse    -> exit 1; no other leg attempted
#   WT3 execute happy path        -> exit 0; strict ORDER guards→push→tag→
#                                    train(--refresh-cache --strict-lag 0)→
#                                    sweep(--commit --lint, module-path$ anchor)
#   WT4 execute, push refuses     -> exit 1; tag leg NOT attempted
#   WT5 execute, train red        -> exit 1; sweep NOT attempted (gotcha 27
#                                    class: a red train stops the rhythm)
#   WT6 execute, sweep refuses    -> exit 1; names the leg
#   WT7 carriers reported         -> changed published go.mod prints the exact
#                                    next wave-tag invocation with next patch
#   WT8 usage                     -> missing args exit 2; unknown module exit 2
#
# Usage: ./scripts/selftests/test-wave-tag.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../tools/wave-tag.sh"

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
LOG="$work/calls.log"
: >"$LOG"

# Stub factory: <name> [exit-code] — records "name arg arg ..." per call.
make_stub() {
  local name="$1" rc="${2:-0}"
  cat >"$work/$name" <<EOF
#!/usr/bin/env bash
echo "$name \$*" >>"$LOG"
exit $rc
EOF
  chmod +x "$work/$name"
}

make_stub verify-tag
make_stub train
make_stub bump-dep
# git stub: subcommand-aware (push ok; diff --name-only fakes a carrier for WT7)
cat >"$work/git" <<EOF
#!/usr/bin/env bash
echo "git \$*" >>"$LOG"
case "\$1" in
  push) exit 0 ;;
  rev-parse) echo "HEAD~1" ;;
  diff) printf 'setup/go.mod\nsetup/go.sum\nusermgmt/README.md\n' ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$work/git"

run_tool() { # [args...] — sets OUT + RC, appends a marker between runs
  echo "---RUN---" >>"$LOG"
  OUT="$(WAVE_TAG_VERIFY_TAG_CMD="bash $work/verify-tag" \
    WAVE_TAG_TRAIN_CMD="bash $work/train" \
    WAVE_TAG_BUMP_DEP_CMD="bash $work/bump-dep" \
    WAVE_TAG_GIT_CMD="bash $work/git" \
    bash "$TOOL" "$@" 2>&1)"
  RC=$?
}

# A real module dir for path/module resolution (the fixture repo root is the
# REAL repo — wave-tag reads <module-dir>/go.mod from the repo root; only the
# TOOLS are stubbed). usermgmt/totp exists and requires nothing from network.
MOD=usermgmt/totp

echo ""
echo "=== Test Suite: wave-tag.sh ==="
echo ""

# --- WT1: dry-run happy path ---------------------------------------------------
run_tool "$MOD" v4.99.0
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "WT1 dry-run exits 0 (got $RC)"
report "$(echo "$OUT" | grep -q 'guards (verify-tag --dry-run)' && echo 0 || echo 1)" "WT1 guards leg ran for real"
report "$(echo "$OUT" | grep -q '\[dry-run\] push:' && echo 0 || echo 1)" "WT1 push leg printed as a command"
report "$([ "$(grep -c '^git push' "$LOG")" = "0" ] && echo 0 || echo 1)" "WT1 dry-run called NO real push"

# --- WT2: dry-run, guards refuse ------------------------------------------------
make_stub verify-tag 1
run_tool "$MOD" v4.99.0
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "WT2 guard refusal exits 1 (got $RC)"
report "$(echo "$OUT" | grep -q 'REFUSED' && echo 0 || echo 1)" "WT2 names the refused leg"
report "$(echo "$OUT" | grep -q 'Nothing was pushed or tagged' && echo 0 || echo 1)" "WT2 states nothing was pushed"
make_stub verify-tag

# --- WT3: execute happy path — order + flags -------------------------------------
run_tool "$MOD" v4.99.0 --execute
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "WT3 execute exits 0 (got $RC)"
order_ok=$(awk '
  /---RUN---/ {seen=0}
  /verify-tag usermgmt\/totp v4\.99\.0 --dry-run/ {if (!seen) {order = order " G"; seen=1}}
  /^git push$/ {order = order " P"}
  /verify-tag usermgmt\/totp v4\.99\.0 --push/ {order = order " T"}
  /train --refresh-cache --strict-lag 0/ {order = order " R"}
  /bump-dep github\.com\/larsartmann\/cqrs-htmx\/usermgmt\/totp\/v4\$ v4\.99\.0 --commit --lint/ {order = order " S"}
  END {print (order ~ /G.*P.*T.*R.*S/) ? "OK" : "BAD:" order}
' "$LOG")
report "$([ "$order_ok" = "OK" ] && echo 0 || echo 1)" "WT3 strict leg order guards→push→tag→train→sweep ($order_ok)"
report "$(grep -q 'bump-dep github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4\$' "$LOG" && echo 0 || echo 1)" "WT3 sweep anchors the exact module path (\$ suffix)"

# --- WT4: execute, push refuses ---------------------------------------------------
cat >"$work/git" <<EOF
#!/usr/bin/env bash
echo "git \$*" >>"$LOG"
case "\$1" in
  push) exit 1 ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$work/git"
run_tool "$MOD" v4.99.0 --execute
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "WT4 push refusal exits 1 (got $RC)"
report "$(awk '/---RUN---/{n++} n==4' "$LOG" | grep -q -- '--push' | head -1; [ "$(awk '/---RUN---/{n++} n==4' "$LOG" | grep -c -- 'verify-tag usermgmt/totp v4.99.0 --push')" = "0" ] && echo 0 || echo 1)" "WT4 tag leg NOT attempted after push refusal"

# --- WT5: execute, train red stops the sweep ---------------------------------------
cat >"$work/git" <<'EOF'
#!/usr/bin/env bash
echo "git $*" >>"$LOG"
case "$1" in
  rev-parse) echo "HEAD~1" ;;
  diff) exit 0 ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$work/git"
make_stub train 3
run_tool "$MOD" v4.99.0 --execute
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "WT5 train red exits 1 (got $RC)"
report "$([ "$(awk '/---RUN---/{n++} n==5' "$LOG" | grep -c '^bump-dep')" = "0" ] && echo 0 || echo 1)" "WT5 sweep NOT attempted after train red"
make_stub train

# --- WT6: execute, sweep refuses ----------------------------------------------------
make_stub bump-dep 1
run_tool "$MOD" v4.99.0 --execute
report "$([ "$RC" -eq 1 ] && echo 0 || echo 1)" "WT6 sweep refusal exits 1 (got $RC)"
report "$(echo "$OUT" | grep -q "leg 'sweep' REFUSED" && echo 0 || echo 1)" "WT6 names the refused leg (sweep)"
make_stub bump-dep

# --- WT7: carriers reported with the next invocation --------------------------------
mkdir -p "$work/cache"
printf 'usermgmt/totp/v4.13.0\nsetup/v4.14.4\nroot-irrelevant\n' >"$work/cache/cqrs-htmx.tags"
# git diff now reports setup/go.mod changed (published per the seeded cache)
cat >"$work/git" <<'EOF'
#!/usr/bin/env bash
echo "git $*" >>"$LOG"
case "$1" in
  push) exit 0 ;;
  rev-parse) echo "HEAD~1" ;;
  diff) printf 'setup/go.mod\n' ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$work/git"
OUT="$(WAVE_TAG_VERIFY_TAG_CMD="bash $work/verify-tag" \
  WAVE_TAG_TRAIN_CMD="bash $work/train" \
  WAVE_TAG_BUMP_DEP_CMD="bash $work/bump-dep" \
  WAVE_TAG_GIT_CMD="bash $work/git" \
  TRAIN_TAG_CACHE_DIR="$work/cache" \
  bash "$TOOL" "$MOD" v4.99.0 --execute 2>&1)"
RC=$?
report "$([ "$RC" -eq 0 ] && echo 0 || echo 1)" "WT7 carrier wave completes exit 0 (got $RC)"
report "$(echo "$OUT" | grep -q 'setup (latest v4.14.4 → next v4.14.5)' && echo 0 || echo 1)" "WT7 carrier prints latest + next patch"
report "$(echo "$OUT" | grep -q 'wave-tag.sh setup v4.14.5 --execute' && echo 0 || echo 1)" "WT7 prints the exact next invocation"

# --- WT8: usage errors -----------------------------------------------------------------
run_tool
report "$([ "$RC" -eq 2 ] && echo 0 || echo 1)" "WT8 no args exits 2 (got $RC)"
run_tool no/such/module v1.0.0
report "$([ "$RC" -eq 2 ] && echo 0 || echo 1)" "WT8 unknown module dir exits 2 (got $RC)"

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
echo "All tests passed."
