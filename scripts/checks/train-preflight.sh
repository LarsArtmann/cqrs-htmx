#!/usr/bin/env bash
# train-preflight.sh — ONE command before a release train (or any push):
# is the shared tree safe to mutate, and would the current state survive
# the push-boundary gates?
#
# WHY THIS EXISTS: the 2026-10-08 push-unblock session discovered two
# pre-existing CI reds (status-row gate, auditlog go.sum residue) and one
# hermetic-only test red (root CSP img-src vs published root v4.13.2,
# gotcha-30 test form) MID-VERIFICATION instead of pre-flight — hours of
# forensics that one command would have surfaced before the first mutation.
#
# Stages (each independently reported; aggregate rc):
#   1. quiescence   — wait-tree-quiet: no dirty tree, stable HEAD for a full
#                     window (bounded; env WAIT_TREE_QUIET_MAX_WAIT honored)
#   2. surprise     — preflight-tree-check: dirty tree / foreign fresh commit /
#                     commit-velocity burst
#   3. lint         — nix run .#lint (workspace golangci-lint)
#   4. tests        — nix run .#test (hermetic race battery, CI-parity:
#                     flake goApp exports GOWORK=off)
#   5. train        — check-release-train --refresh-cache --strict-lag 0
#                     (the exact CI/pre-push flags)
#
# CANDIDATE-COUNT GUARDS (gotcha 2: a gate that scans zero candidates is a
# FALSE green): the lint/test logs must contain at least one per-module
# marker ("==> <dir>"), the train log must contain the "internal requires"
# tally line. A guard miss fails the run loudly even when the stage rc was 0.
#
# STUB SURFACE (fixture self-test): every stage command is env-overridable —
#   TRAIN_PREFLIGHT_QUIET_CMD / _PREFLIGHT_CMD / _LINT_CMD / _TEST_CMD / _TRAIN_CMD
# The self-test (scripts/selftests/test-train-preflight.sh) injects stubs to
# exercise pass, fail, and guard-miss paths offline in seconds.
#
# Usage: nix run .#train-preflight  OR  bash scripts/checks/train-preflight.sh
# Exit: 0 = all stages green; 1 = at least one stage or guard red (first red
#        aborts unless TRAIN_PREFLIGHT_KEEP_GOING=1, which runs every stage
#        for a full red/green report); 2 = usage/environment
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "train-preflight: not inside a git repository" >&2
  exit 2
}
cd "$REPO_ROOT" || exit 2

QUIET_CMD="${TRAIN_PREFLIGHT_QUIET_CMD:-bash scripts/tools/wait-tree-quiet.sh}"
PREFLIGHT_CMD="${TRAIN_PREFLIGHT_PREFLIGHT_CMD:-bash scripts/tools/preflight-tree-check.sh}"
LINT_CMD="${TRAIN_PREFLIGHT_LINT_CMD:-nix run .#lint}"
TEST_CMD="${TRAIN_PREFLIGHT_TEST_CMD:-nix run .#test}"
TRAIN_CMD="${TRAIN_PREFLIGHT_TRAIN_CMD:-bash scripts/checks/check-release-train.sh --refresh-cache --strict-lag 0}"
KEEP_GOING="${TRAIN_PREFLIGHT_KEEP_GOING:-0}"

LOGDIR="$(mktemp -d /tmp/cqrs-htmx-train-preflight-XXXXXX)"
trap 'rm -rf "$LOGDIR"' EXIT

failed=0

run_stage() { # <name> <cmd> <guard-regex-or-empty> <min-hits>
  local name="$1" cmd="$2" guard="$3" minhits="$4"
  local log="$LOGDIR/$name.log"
  echo ""
  echo "=== train-preflight: $name ==="
  echo "  cmd: $cmd"
  if eval "$cmd" >"$log" 2>&1; then
    if [ -n "$guard" ]; then
      local hits
      # grep -c always prints the count (0 included); no `|| echo 0` here —
      # that would append a second line and corrupt the count.
      hits=$(grep -cE "$guard" "$log" 2>/dev/null)
      hits=${hits//[!0-9]/}
      if [ "${hits:-0}" -lt "$minhits" ]; then
        echo "  ✗ $name: GUARD MISS — rc was 0 but candidate count is $hits (< $minhits)."
        echo "    A zero-iteration stage is a FALSE green (gotcha 2); refusing to trust it."
        echo "    Last lines of output:"
        tail -5 "$log" | sed 's/^/      /'
        failed=1
        return 1
      fi
      echo "  ✓ $name (guard: $hits candidate marker(s) ≥ $minhits)"
    else
      echo "  ✓ $name"
    fi
    return 0
  else
    local rc=$?
    echo "  ✗ $name FAILED (rc=$rc). Output tail:"
    tail -15 "$log" | sed 's/^/      /'
    failed=1
    return 1
  fi
}

# Stage 1+2 share the quiescence/surprise semantics; neither needs a guard.
run_stage "quiescence" "$QUIET_CMD" "" 0 || [ "$KEEP_GOING" = "1" ] || exit 1
run_stage "surprise" "$PREFLIGHT_CMD" "" 0 || [ "$KEEP_GOING" = "1" ] || exit 1
# Lint/test iterate every (non-excluded) workspace module: the per-module
# marker lines are the candidate count. The workspace has ~20 lintable
# modules; the floor of 1 only catches the silent-zero class while the
# exact count is printed for eyeballing.
run_stage "lint" "$LINT_CMD" '^==> ' 1 || [ "$KEEP_GOING" = "1" ] || exit 1
run_stage "tests" "$TEST_CMD" '^==> ' 1 || [ "$KEEP_GOING" = "1" ] || exit 1
# Train gate prints "Checked N internal requires" on every complete run.
run_stage "train" "$TRAIN_CMD" 'internal requires' 1 || [ "$KEEP_GOING" = "1" ] || exit 1

echo ""
if [ "$failed" -eq 0 ]; then
  echo "✓ train-preflight: ALL STAGES GREEN — safe to start the train"
  exit 0
else
  echo "✗ train-preflight: $failed stage(s)/guard(s) red — resolve before the first mutation or push"
  exit 1
fi
