#!/usr/bin/env bash
# check-branching-flow.sh — ratchet gate over the full branching-flow
# (go-design-smells) analyzer set. The committed SARIF baseline freezes the
# 2026-10-05 triage verdicts (docs/analysis/triage-decisions.md); the gate
# fails ONLY on NEW findings, so the 663 existing adjudicated findings never
# block a commit (a gate failing on all of them would brick every push —
# the gotcha-6 advisory/strict split-brain lesson).
#
# Tool contract (verified empirically 2026-10-05 vs branching-flow 0.2.0):
#   rc 0     = no NEW findings (modified/removed findings do NOT fail)
#   rc 1     = new findings since the baseline (+N added)
#   rc 69/...= tool failure, e.g. the experimental panic linter cannot load
#              packages when `go` on PATH is below the go.work floor
#
# The baseline MUST be minified SARIF (`jq -c .`): the pretty form exceeds
# the 1 MB check-large-files limit, and the compact `finding` format does not
# round-trip through the baseline differ (byte-identical tree diffs as
# +610 added — false red — verified 2026-10-05). Refresh workflow:
# docs/analysis/README.md.
#
# BRANCHING_FLOW_BIN overrides the binary (the fixture self-test stubs it).
# CI=true with a missing binary → SKIP: runners do not ship branching-flow
# (same posture as check-css-bundle-classes — the fixture self-test is the
# CI coverage, the real gate runs locally via nix run .#check-branching-flow).
set -uo pipefail

ROOT="${1:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$ROOT" || exit 1

BASELINE="docs/analysis/branching-flow-baseline.sarif"
BIN="${BRANCHING_FLOW_BIN:-branching-flow}"

if ! command -v "$BIN" >/dev/null 2>&1; then
  if [ "${CI:-}" = "true" ]; then
    echo "check-branching-flow: SKIP — no branching-flow binary on this runner (fixture self-test is the CI coverage)"
    exit 0
  fi
  echo "check-branching-flow: FAILED — branching-flow binary not found ('$BIN')." >&2
  echo "  It is a system tool (go-design-smells); point BRANCHING_FLOW_BIN at it if it lives elsewhere." >&2
  exit 1
fi

if [ ! -f "$BASELINE" ]; then
  echo "check-branching-flow: FAILED — baseline $BASELINE missing." >&2
  echo "  Regenerate it: docs/analysis/README.md (branching-flow all + jq -c), then commit." >&2
  exit 1
fi

# Uncommitted-baseline false-green guard: a locally regenerated (or untracked)
# baseline must be committed before the gate can vouch for it.
if [ -n "$(git status --porcelain -- "$BASELINE" 2>/dev/null)" ]; then
  echo "check-branching-flow: FAILED — baseline $BASELINE is uncommitted." >&2
  echo "  Commit the refreshed baseline (or git restore it) so the gate compares against a reviewed state." >&2
  exit 1
fi

LOG="/tmp/check-branching-flow-$$-$(date +%s).log"
"$BIN" all . --format sarif --exclude-generated --no-emoji --no-experimental-warn \
  --baseline "$BASELINE" --exit-code >/dev/null 2>"$LOG"
rc=$?

# Parse the tool's Baseline summary line: `Baseline <path>: +N added, -M removed, ~K modified, =U unchanged`.
# detected = added+modified+unchanged is what the tool CURRENTLY sees; a committed
# non-empty baseline with detected=0 and removed>0 means the analyzer matched
# NOTHING (misfire: wrong go/go.work floor, SARIF parse failure) — rc=0 would be a
# false green, so it fails loudly instead.
baseline_line=$(grep 'Baseline' "$LOG" | tail -1 || true)
added=$(printf '%s' "$baseline_line" | sed -nE 's/.*\+([0-9]+) added.*/\1/p')
removed=$(printf '%s' "$baseline_line" | sed -nE 's/.*-([0-9]+) removed.*/\1/p')
modified=$(printf '%s' "$baseline_line" | sed -nE 's/.*~([0-9]+) modified.*/\1/p')
unchanged=$(printf '%s' "$baseline_line" | sed -nE 's/.*=([0-9]+) unchanged.*/\1/p')
detected=$(( ${added:-0} + ${modified:-0} + ${unchanged:-0} ))

case "$rc" in
0)
  if [ "$detected" -eq 0 ] && [ "${removed:-0}" -gt 0 ]; then
    echo "check-branching-flow: FAILED — ZERO candidates detected ($baseline_line)" >&2
    echo "  The tool matched none of the committed baseline's findings — analyzer" >&2
    echo "  misfire (go on PATH below the go.work floor, SARIF parse failure) or a" >&2
    echo "  full ratchet-down whose baseline was never re-pinned. Check the tool" >&2
    echo "  output, or refresh the baseline per docs/analysis/README.md." >&2
    rm -f "$LOG"
    exit 1
  fi
  echo "check-branching-flow: OK — no new findings vs committed baseline ($baseline_line)"
  rm -f "$LOG"
  exit 0
  ;;
1)
  echo "check-branching-flow: FAILED — NEW findings vs baseline ($baseline_line):" >&2
  echo "  Fix them, or (if adjudicated as deliberate) refresh the baseline per docs/analysis/README.md." >&2
  rm -f "$LOG"
  exit 1
  ;;
*)
  echo "check-branching-flow: FAILED — tool failure rc=$rc (not a findings verdict)." >&2
  echo "  Check: go on PATH >= go.work floor, GOEXPERIMENT=jsonv2, baseline parses as SARIF." >&2
  tail -5 "$LOG" >&2
  rm -f "$LOG"
  exit 1
  ;;
esac
