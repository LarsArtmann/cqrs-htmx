#!/usr/bin/env bash
# train-lag-report.sh — early-warning report for release-train lag
# (plan T08; flattens the reactive-afternoon alignment cycle).
#
# The 2026-10-08 push-unblock found 33 lag findings ~24h after the previous
# alignment; this report surfaces lag NIGHTLY instead of at the next push.
#
# Behavior:
#   - Runs the release-train gate in strict mode (the exact CI/pre-push
#     flags) and reports its output verbatim.
#   - rc 0 (aligned) or rc 3 (lag — the early-warning SIGNAL): exit 0 by
#     default so the nightly stays green and informational. Set
#     TRAIN_LAG_REPORT_STRICT=1 to propagate rc 3 (manual/train-time check).
#   - ANY other rc is a real gate failure and propagates unchanged — the
#     wrapper never masks a broken gate.
#   - Guard: the output must contain the "internal requires" tally line;
#     a tally-less run is a silent no-op and fails loudly (gotcha 2).
#
# Env overrides (fixture self-test):
#   TRAIN_LAG_REPORT_GATE_CMD   gate command (default: the real gate)
#   TRAIN_LAG_REPORT_STRICT     1 = propagate lag rc
#
# Usage: nix run .#train-lag-report          (or the nightly workflow)
# Exit: 0 = report generated (aligned or lag, advisory mode); 1/3 = real
#        failure or strict lag; 2 = usage/environment
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "train-lag-report: not inside a git repository" >&2
  exit 2
}
cd "$REPO_ROOT" || exit 2

GATE_CMD="${TRAIN_LAG_REPORT_GATE_CMD:-bash scripts/checks/check-release-train.sh --refresh-cache --strict-lag 0}"
STRICT="${TRAIN_LAG_REPORT_STRICT:-0}"

LOG="$(mktemp /tmp/cqrs-htmx-train-lag-XXXXXX.log)"
trap 'rm -f "$LOG"' EXIT

rc=0
eval "$GATE_CMD" >"$LOG" 2>&1 || rc=$?

# gotcha-2 guard: no tally = the gate never really ran.
if ! grep -q "internal requires" "$LOG"; then
  echo "train-lag-report: GUARD MISS — gate output has no 'internal requires' tally (silent no-op refused)."
  echo "Gate output tail:"
  tail -10 "$LOG" | sed 's/^/    /'
  exit 1
fi

echo "=== Train-lag report ($(date -u +%Y-%m-%dT%H:%M:%SZ)) ==="
cat "$LOG"

case "$rc" in
0)
  echo "train-lag-report: aligned — 0 lag (nothing to do)"
  exit 0
  ;;
3)
  if [ "$STRICT" = "1" ]; then
    echo "train-lag-report: lag present and TRAIN_LAG_REPORT_STRICT=1 — propagating rc 3"
    exit 3
  fi
  echo "train-lag-report: EARLY WARNING — train lag present (see recipe above). Advisory only; run the alignment pass before the next push."
  exit 0
  ;;
*)
  echo "train-lag-report: gate FAILED with rc=$rc — propagating (never mask a broken gate)"
  exit "$rc"
  ;;
esac
