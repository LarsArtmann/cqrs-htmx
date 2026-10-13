#!/usr/bin/env bash
# push-and-watch.sh — push, then watch THIS push's CI run to terminal state.
#
# WHY: the 2026-10-13 wave session pushed and watched `gh run list --limit 1`
# WITHOUT a workflow filter — grabbed the Dependabot config-graph run, read
# it green in 1m34s, and the real ci.yml run stayed red for hours. A push is
# not done when `git push` returns; it is done when the CI run FOR THIS HEAD
# SHA reaches terminal state.
#
# Flow:
#   1. release-train tag-cache refresh (gotcha 39: the 15-min TTL cache
#      races EVERY fresh tag — including cross-repo upstream tags published
#      between sweeps; refresh IMMEDIATELY before every push)
#   2. git push (args pass through: -u, --force-with-lease, ...)
#   3. locate the ci.yml run for THIS branch + HEAD SHA (workflow-filtered
#      poll — "the newest run" is the Dependabot graph update every time)
#   4. gh run watch --exit-status
#   5. on red: print the failed job names + the triage hint
#
# STUB SURFACE (fixture self-test, offline):
#   PAW_TRAIN_CMD     (default: bash scripts/checks/check-release-train.sh --refresh-cache)
#   PAW_GIT_PUSH_CMD  (default: git push)
#   PAW_GH_CMD        (default: gh)
#   PAW_POLL_ATTEMPTS / PAW_POLL_SLEEP (run-appearance poll; self-test tightens)
#
# Usage: nix run .#push-and-watch [-- <git-push-args>...]
# Exit: 0 = push + CI terminal-green; 1 = refresh/push/watch failure or run
#       not found; 2 = usage/environment

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "push-and-watch: not inside a git repository" >&2
  exit 2
}
cd "$REPO_ROOT" || exit 2

TRAIN_CMD="${PAW_TRAIN_CMD:-bash scripts/checks/check-release-train.sh --refresh-cache}"
PUSH_CMD="${PAW_GIT_PUSH_CMD:-git push}"
GH_CMD="${PAW_GH_CMD:-gh}"
POLL_ATTEMPTS="${PAW_POLL_ATTEMPTS:-24}"
POLL_SLEEP="${PAW_POLL_SLEEP:-5}"

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
SHA="$(git rev-parse HEAD)"
echo "push-and-watch: branch=$BRANCH sha=${SHA:0:12}"

echo "==> refreshing the release-train tag cache (gotcha 39: it races every fresh tag)"
if ! $TRAIN_CMD; then
  echo "push-and-watch: tag-cache refresh FAILED — resolve BEFORE pushing" >&2
  echo "  (a stale cache green-lights UNPUBLISHED requires that red CI later;" >&2
  echo "   rc=3 after a fresh tag usually means a NEW UPSTREAM tag: look for it," >&2
  echo "   bump-dep sweep, then retry)." >&2
  exit 1
fi

echo "==> git push $*"
if ! $PUSH_CMD "$@"; then
  echo "push-and-watch: push FAILED" >&2
  exit 1
fi

echo "==> locating the ci.yml run for $BRANCH @ ${SHA:0:12} (workflow-filtered poll)"
RUN_ID=""
for _ in $(seq 1 "$POLL_ATTEMPTS"); do
  RUN_ID="$($GH_CMD run list --workflow ci.yml --branch "$BRANCH" --limit 10 \
    --json databaseId,headSha \
    --jq ".[] | select(.headSha==\"$SHA\") | .databaseId" 2>/dev/null | head -1)"
  [ -n "$RUN_ID" ] && break
  sleep "$POLL_SLEEP"
done
if [ -z "$RUN_ID" ]; then
  echo "push-and-watch: no ci.yml run appeared for ${SHA:0:12} — check Actions" >&2
  echo "  manually. ALWAYS filter by --workflow ci.yml: the newest run without" >&2
  echo "  the filter is the Dependabot graph update (2026-10-13 wrong-run watch)." >&2
  exit 1
fi
echo "==> watching run $RUN_ID"
if $GH_CMD run watch "$RUN_ID" --exit-status; then
  echo "✓ push-and-watch: $BRANCH @ ${SHA:0:12} CI GREEN (run $RUN_ID)"
  exit 0
fi

echo "✗ push-and-watch: run $RUN_ID RED — failed job(s):" >&2
SLUG="$($GH_CMD repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)"
if [ -n "$SLUG" ]; then
  $GH_CMD api "repos/$SLUG/actions/runs/$RUN_ID/jobs" \
    --jq '.jobs[] | select(.conclusion=="failure") | .name' 2>/dev/null || true
  echo "" >&2
  echo "Triage: gh api repos/$SLUG/actions/runs/$RUN_ID/jobs --jq '.jobs[] | select(.conclusion==\"failure\") | .name'" >&2
else
  echo "(could not resolve the repo slug for the job listing — run gh api manually)" >&2
fi
exit 1
