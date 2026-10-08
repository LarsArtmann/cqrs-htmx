#!/usr/bin/env bash
# test-check-branching-flow.sh — offline fixture self-test for
# check-branching-flow.sh. Stubs the branching-flow binary (BRANCHING_FLOW_BIN)
# so no real tool, go toolchain, or network is needed. Covers: missing-binary
# guard (local fail / CI=true skip), missing-baseline guard, uncommitted-
# baseline false-green guard, rc pass-through (0 green / 1 new-findings /
# 69 tool-failure), and the flag wiring pin (baseline + exit-code both passed).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-branching-flow.sh"
TMP=$(mktemp -d /tmp/check-branching-flow-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT
failures=0

check() { # check <name> <expected-rc> <actual-rc> <required-grep> <output>
  local name="$1" want_rc="$2" got_rc="$3" pattern="$4" out="$5"
  if [ "$want_rc" != "$got_rc" ]; then
    echo "FAIL $name: rc=$got_rc want=$want_rc"
    echo "$out" | head -5
    failures=$((failures + 1))
    return
  fi
  if [ -n "$pattern" ] && ! printf '%s\n' "$out" | grep -q "$pattern"; then
    echo "FAIL $name: output missing '$pattern'"
    echo "$out" | head -5
    failures=$((failures + 1))
    return
  fi
  echo "ok   $name"
}

# Stubs: exit 0 (clean), 1 (new findings), 69 (tool failure). Each records its
# argv so the flag-wiring pin can assert --baseline/--exit-code are passed.
make_stub() { # make_stub <name> <exit-code>
  cat >"$TMP/stub-$1" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" > "$TMP/stub-$1.argv"
echo "Baseline stub: +0 added, -0 removed, ~0 modified, =0 unchanged" >&2
exit $2
STUB
  chmod +x "$TMP/stub-$1"
}
make_stub green 0
make_stub red 1
make_stub crash 69

# Misfire stub: tool rc=0 but matched NOTHING of the baseline (removed=659,
# everything else 0) — the zero-candidate shape the gate must refuse to green.
cat >"$TMP/stub-misfire" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" > "$TMP/stub-misfire.argv"
echo "Baseline stub: +0 added, -659 removed, ~0 modified, =0 unchanged" >&2
exit 0
STUB
chmod +x "$TMP/stub-misfire"

# Fixture: a git repo with a committed baseline file.
REPO="$TMP/repo"
mkdir -p "$REPO/docs/analysis"
git -C "$REPO" init -q
git -C "$REPO" config user.email selftest@example.invalid
git -C "$REPO" config user.name "selftest"
printf '{"version":"2.1.0"}\n' >"$REPO/docs/analysis/branching-flow-baseline.sarif"
git -C "$REPO" add -A
git -C "$REPO" commit -qm fixture

# 1) Missing binary, local: hard fail. (env -u CI: GitHub runners export
# CI=true globally, which would take the gate's runner-skip branch.)
out=$(env -u CI BRANCHING_FLOW_BIN="$TMP/definitely-missing" bash "$GATE" "$REPO" 2>&1)
check "missing-binary guard fails locally" 1 $? 'not found' "$out"

# 2) Missing binary, CI=true: SKIP (runner posture).
out=$(CI=true BRANCHING_FLOW_BIN="$TMP/definitely-missing" bash "$GATE" "$REPO" 2>&1)
check "missing-binary skips under CI=true" 0 $? 'SKIP' "$out"

# 3) Missing baseline: fail with refresh hint.
out=$(BRANCHING_FLOW_BIN="$TMP/stub-green" bash "$GATE" "$TMP" 2>&1)
check "missing-baseline guard fails" 1 $? 'baseline .* missing' "$out"

# 4) Uncommitted (dirty) baseline: false-green guard.
printf '{"version":"2.1.0","dirty": true}\n' >"$REPO/docs/analysis/branching-flow-baseline.sarif"
out=$(BRANCHING_FLOW_BIN="$TMP/stub-green" bash "$GATE" "$REPO" 2>&1)
check "uncommitted-baseline guard fails" 1 $? 'uncommitted' "$out"
git -C "$REPO" restore docs/analysis/branching-flow-baseline.sarif

# 5) Untracked baseline (never committed, ?? in porcelain): same guard.
git -C "$REPO" rm -q --cached docs/analysis/branching-flow-baseline.sarif
out=$(BRANCHING_FLOW_BIN="$TMP/stub-green" bash "$GATE" "$REPO" 2>&1)
check "untracked-baseline guard fails" 1 $? 'uncommitted' "$out"
git -C "$REPO" add -A && git -C "$REPO" commit -qm re-add-baseline

# 6) Stub rc 0 → gate green.
out=$(BRANCHING_FLOW_BIN="$TMP/stub-green" bash "$GATE" "$REPO" 2>&1)
check "stub rc0 passes gate" 0 $? 'no new findings' "$out"

# 7) Stub rc 1 → gate red, NEW-findings message.
out=$(BRANCHING_FLOW_BIN="$TMP/stub-red" bash "$GATE" "$REPO" 2>&1)
check "stub rc1 fails gate as NEW findings" 1 $? 'NEW findings' "$out"

# 8) Stub rc 69 → gate red, tool-failure message (distinct from findings).
out=$(BRANCHING_FLOW_BIN="$TMP/stub-crash" bash "$GATE" "$REPO" 2>&1)
check "stub rc69 fails gate as tool failure" 1 $? 'tool failure rc=69' "$out"

# 9) Zero-candidate misfire: rc=0 with detected=0/removed=659 → gate red with
# the loud guard message, never a false green.
out=$(BRANCHING_FLOW_BIN="$TMP/stub-misfire" bash "$GATE" "$REPO" 2>&1)
check "zero-candidate misfire fails gate" 1 $? 'ZERO candidates' "$out"

# 10) Green run prints the counts: the OK line carries the Baseline summary.
out=$(BRANCHING_FLOW_BIN="$TMP/stub-green" bash "$GATE" "$REPO" 2>/dev/null)
check "green output carries counts" 0 $? 'added' "$out"

# 11) Flag-wiring pin: the gate must pass --baseline, --exit-code, and the
# committed baseline path to the tool (drift here silently changes semantics).
bash "$GATE" "$REPO" >/dev/null 2>&1 </dev/null || true
argv=$(cat "$TMP/stub-green.argv")
for needle in "--baseline" "--exit-code" "docs/analysis/branching-flow-baseline.sarif" "--exclude-generated"; do
  if ! printf '%s' "$argv" | grep -q -- "$needle"; then
    echo "FAIL flag-wiring pin: tool argv missing '$needle'"
    echo "argv: $argv"
    failures=$((failures + 1))
  else
    echo "ok   flag-wiring pin: $needle"
  fi
done

if [ "$failures" -gt 0 ]; then
  echo "check-branching-flow self-test: $failures FAILURE(S)"
  exit 1
fi
echo "check-branching-flow self-test: all cases green"
