#!/usr/bin/env bash
# test-push-and-watch.sh — fixture self-test for push-and-watch.sh (offline)
#
#   T1  push + CI green                        -> exit 0, "CI GREEN"
#   T2  CI red                                 -> exit 1, failed job + triage hint
#   T3  run never appears                      -> exit 1, "no ci.yml run appeared"
#   T4  push fails                             -> exit 1, "push FAILED"
#   T5  tag-cache refresh fails                -> exit 1 BEFORE any push (sentinel absent)
#   T6  watch fails but conclusion=success     -> exit 0, "CI GREEN" (watch lied)
#   T7  conclusion=failure + jobs fetch fails  -> exit 1, RED + URL, step warning
#   T8  conclusion unreadable                  -> exit 1, honest UNKNOWN (no false red)
#
# Runtime: seconds (every command is a stub; a fixture git repo supplies the
# branch/SHA; the fake gh parses the tool's --jq expression for the SHA).
#
# Usage: ./scripts/selftests/test-push-and-watch.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
TOOL="$SCRIPT_DIR/../tools/push-and-watch.sh"

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

work=$(mktemp -d)
tmp=$(mktemp -d)
trap 'rm -rf "$work" "$tmp"' EXIT

# --- fake gh: behavior driven by $FAKE_STATE files ---
cat >"$tmp/gh" <<'EOF'
#!/usr/bin/env bash
STATE="${FAKE_STATE:?}"
case "$1" in
  run)
    sub="${2:-}"
    if [ "$sub" = "list" ]; then
      jqexpr=""
      prev=""
      for a in "$@"; do
        if [ "$prev" = "--jq" ]; then jqexpr="$a"; fi
        prev="$a"
      done
      want="$(cat "$STATE/sha" 2>/dev/null)"
      case "$jqexpr" in
      *"select(.headSha==\"$want\")"*)
        [ -f "$STATE/runid" ] && cat "$STATE/runid"
        ;;
      esac
      exit 0
    fi
    if [ "$sub" = "watch" ]; then
      if [ -f "$STATE/watch_fail" ]; then
        echo "fake gh: run RED"
        exit 1
      fi
      echo "fake gh: run GREEN"
      exit 0
    fi
    if [ "$sub" = "view" ]; then
      if [ -f "$STATE/view_fail" ]; then
        exit 1
      fi
      [ -f "$STATE/view_conclusion" ] && cat "$STATE/view_conclusion"
      exit 0
    fi
    exit 0
    ;;
  api)
    if [ -f "$STATE/api_fail" ]; then
      echo "fake gh: HTTP 502" >&2
      exit 1
    fi
    [ -f "$STATE/jobs" ] && cat "$STATE/jobs"
    exit 0
    ;;
  repo)
    echo "owner/repo"
    exit 0
    ;;
esac
exit 0
EOF
chmod +x "$tmp/gh"

# --- fake push: records a sentinel line; rc from FAKE_PUSH_RC ---
cat >"$tmp/push" <<'EOF'
#!/usr/bin/env bash
echo "pushed (fake) $*" >>"${FAKE_STATE:?}/pushes"
exit "${FAKE_PUSH_RC:-0}"
EOF
chmod +x "$tmp/push"

# --- train stubs (script FILES: an inline `exit 0` would terminate the
# tool's own shell — builtins run in-process, the train-preflight lesson) ---
printf '#!/usr/bin/env bash\nexit 0\n' >"$tmp/train_ok.sh"
printf '#!/usr/bin/env bash\necho "simulated refresh failure"; exit 3\n' >"$tmp/train_fail.sh"
chmod +x "$tmp/train_ok.sh" "$tmp/train_fail.sh"

reset_state() {
  rm -rf "$work"
  mkdir -p "$work"
}

run_tool() { # <repo> <outfile> <train-cmd> <push-rc>
  local repo="$1" out="$2" train="$3" pushrc="$4"
  (
    cd "$repo" || exit 1
    env \
      PAW_TRAIN_CMD="$train" \
      PAW_GIT_PUSH_CMD="bash $tmp/push" \
      PAW_GH_CMD="bash $tmp/gh" \
      PAW_POLL_ATTEMPTS="2" \
      PAW_POLL_SLEEP="0" \
      FAKE_PUSH_RC="$pushrc" \
      FAKE_STATE="$work" \
      bash "$TOOL"
  ) >"$out" 2>&1
}

echo "== test-push-and-watch.sh"

# T1 happy path: run appears, watch green
r="$tmp/t1"
new_repo "$r"
reset_state
printf '%s\n' "$(git -C "$r" rev-parse HEAD)" >"$work/sha"
printf '4242\n' >"$work/runid"
run_tool "$r" "$tmp/t1.out" "bash $tmp/train_ok.sh" 0
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "CI GREEN" "$tmp/t1.out" || ok=1
grep -q "run 4242" "$tmp/t1.out" || ok=1
[ -f "$work/pushes" ] || ok=1
report $ok "T1 push + green watch -> exit 0 (got $rc)"

# T2 CI red: failed job named + triage hint
r="$tmp/t2"
new_repo "$r"
reset_state
printf '%s\n' "$(git -C "$r" rev-parse HEAD)" >"$work/sha"
printf '4243\n' >"$work/runid"
touch "$work/watch_fail"
printf 'failure\n' >"$work/view_conclusion"
printf 'lint adminui\nTest adminui submodule\n' >"$work/jobs"
run_tool "$r" "$tmp/t2.out" "bash $tmp/train_ok.sh" 0
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "run 4243 RED" "$tmp/t2.out" || ok=1
grep -q "lint adminui" "$tmp/t2.out" || ok=1
grep -q "Test adminui submodule" "$tmp/t2.out" || ok=1
grep -q "Triage:" "$tmp/t2.out" || ok=1
report $ok "T2 red watch -> exit 1, failed jobs + triage hint (got $rc)"

# T3 run never appears -> exit 1, loud no-run message
r="$tmp/t3"
new_repo "$r"
reset_state
printf '%s\n' "$(git -C "$r" rev-parse HEAD)" >"$work/sha"
run_tool "$r" "$tmp/t3.out" "bash $tmp/train_ok.sh" 0
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "no ci.yml run appeared" "$tmp/t3.out" || ok=1
grep -q -- "--workflow ci.yml" "$tmp/t3.out" || ok=1
[ -f "$work/pushes" ] || ok=1
report $ok "T3 missing run -> exit 1 with workflow-filter hint (got $rc)"

# T4 push fails -> exit 1, "push FAILED"
r="$tmp/t4"
new_repo "$r"
reset_state
run_tool "$r" "$tmp/t4.out" "bash $tmp/train_ok.sh" 1
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "push FAILED" "$tmp/t4.out" || ok=1
[ -f "$work/pushes" ] || ok=1
report $ok "T4 failed push -> exit 1 before watch (got $rc)"

# T5 refresh fails -> exit 1 BEFORE any push (sentinel absent)
r="$tmp/t5"
new_repo "$r"
reset_state
run_tool "$r" "$tmp/t5.out" "bash $tmp/train_fail.sh" 0
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "refresh FAILED" "$tmp/t5.out" || ok=1
[ ! -f "$work/pushes" ] || ok=1
report $ok "T5 failed refresh -> exit 1, no push attempted (got $rc)"

# T6 the watch dies transiently but the run is actually GREEN -> rc 0.
# (Live 2026-10-14, run 38466348791: gh's jobs poll hit HTTP 502 and the
# tool declared RED on a run that was still running.)
r="$tmp/t6"
new_repo "$r"
reset_state
printf '%s\n' "$(git -C "$r" rev-parse HEAD)" >"$work/sha"
printf '4244\n' >"$work/runid"
touch "$work/watch_fail"
printf 'success\n' >"$work/view_conclusion"
run_tool "$r" "$tmp/t6.out" "bash $tmp/train_ok.sh" 0
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
grep -q "CI GREEN" "$tmp/t6.out" || ok=1
grep -q "run 4244 RED" "$tmp/t6.out" && ok=1
report $ok "T6 watch lied (502) + conclusion=success -> exit 0 GREEN (got $rc)"

# T7 genuinely RED: conclusion=failure, but the jobs listing itself fails
# -> still RED (rc 1) with the Actions URL + the step-enumeration warning,
# never an empty "failed job(s):" list.
r="$tmp/t7"
new_repo "$r"
reset_state
printf '%s\n' "$(git -C "$r" rev-parse HEAD)" >"$work/sha"
printf '4245\n' >"$work/runid"
touch "$work/watch_fail"
printf 'failure\n' >"$work/view_conclusion"
touch "$work/api_fail"
run_tool "$r" "$tmp/t7.out" "bash $tmp/train_ok.sh" 0
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "run 4245 RED" "$tmp/t7.out" || ok=1
grep -q "github.com/owner/repo/actions/runs/4245" "$tmp/t7.out" || ok=1
grep -q "failed STEPS" "$tmp/t7.out" || ok=1
report $ok "T7 red + failed jobs fetch -> exit 1, URL + step warning (got $rc)"

# T8 watch fails AND the conclusion is unreadable -> honest UNKNOWN, exit 1,
# no false RED declaration.
r="$tmp/t8"
new_repo "$r"
reset_state
printf '%s\n' "$(git -C "$r" rev-parse HEAD)" >"$work/sha"
printf '4246\n' >"$work/runid"
touch "$work/watch_fail"
touch "$work/view_fail"
run_tool "$r" "$tmp/t8.out" "bash $tmp/train_ok.sh" 0
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
grep -q "could NOT be re-read" "$tmp/t8.out" || ok=1
grep -q "run 4246 RED" "$tmp/t8.out" && ok=1
grep -q "CI GREEN" "$tmp/t8.out" && ok=1
report $ok "T8 unreadable conclusion -> exit 1 UNKNOWN, no false verdict (got $rc)"

echo ""
echo "test-push-and-watch: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
exit 0
