#!/usr/bin/env bash
# wait-tree-quiet.sh — block until the shared tree has been quiet for a full
# window: no uncommitted changes and no new commit for --quiet seconds
# (gotcha 4 quiescence gate before push retries, release trains, or any
# step that must not interleave with the auto-commit daemon or a concurrent
# session).
#
# A new commit OR any dirty-tree observation restarts the quiet window.
# Exits 0 once one full window passes clean; exits 1 on --max-wait timeout.
#
# Usage:
#   scripts/wait-tree-quiet.sh [--quiet 90] [--poll 5] [--max-wait 600]
# Environment overrides: WAIT_TREE_QUIET_SECONDS / _POLL / _MAX_WAIT
# Exit: 0 = quiet window completed; 1 = timeout; 2 = usage/environment
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "wait-tree-quiet: not inside a git repository" >&2
  exit 2
}
cd "$REPO_ROOT" || exit 2

QUIET="${WAIT_TREE_QUIET_SECONDS:-90}"
POLL="${WAIT_TREE_QUIET_POLL:-5}"
MAX_WAIT="${WAIT_TREE_QUIET_MAX_WAIT:-600}"

while [ "$#" -gt 0 ]; do
  case "$1" in
  --quiet)
    QUIET="${2:?}"
    shift 2
    ;;
  --poll)
    POLL="${2:?}"
    shift 2
    ;;
  --max-wait)
    MAX_WAIT="${2:?}"
    shift 2
    ;;
  *)
    echo "wait-tree-quiet: unknown argument: $1" >&2
    exit 2
    ;;
  esac
done

started=$(date +%s)
quiet_since=""
head_sha=""

# Uncommitted = staged + unstaged + untracked (ignored files excluded).
tree_dirty() {
  ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null ||
    [ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]
}

while :; do
  now=$(date +%s)
  elapsed=$((now - started))
  if [ "$elapsed" -ge "$MAX_WAIT" ]; then
    echo "wait-tree-quiet: TIMEOUT after ${elapsed}s — tree still not quiet (dirty or committing)" >&2
    exit 1
  fi

  if tree_dirty; then
    dirty=1
  else
    dirty=0
  fi

  new_sha=$(git rev-parse HEAD 2>/dev/null || echo "")
  if [ "$dirty" -eq 0 ] && [ -n "$new_sha" ] && [ "$new_sha" = "$head_sha" ]; then
    span=$((now - quiet_since))
    if [ "$span" -ge "$QUIET" ]; then
      echo "wait-tree-quiet: OK — clean tree, HEAD stable for ${span}s (sha ${new_sha:0:12})"
      exit 0
    fi
  else
    quiet_since=$now
    head_sha=$new_sha
  fi

  sleep "$POLL"
done
