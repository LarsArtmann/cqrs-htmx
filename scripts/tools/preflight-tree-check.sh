#!/usr/bin/env bash
# preflight-tree-check.sh — abort-on-surprise gate before ANY tree-mutating
# batch step in a shared tree (gotcha 4: the auto-commit daemon plus
# concurrent sessions make unverified assumptions about tree state unsafe).
#
# Checks (each independently reported):
#   D  dirty tree (staged or unstaged changes) — someone's work is in flight;
#      mutating on top makes the sweep unreviewable and the authorship unknown
#   F  foreign fresh commit — a NON-daemon commit newer than
#      PREFLIGHT_RECENCY_SECONDS (default 300) means another session may be
#      mid-operation (the 2026-09-22 templ-components tidy-interference class)
#   R  rapid commit velocity — more than PREFLIGHT_MAX_RECENT_COMMITS
#      (default 3) commits of any kind within the recency window suggests an
#      active batch (daemon bursts are exempt only if all-daemon)
#
# Daemon commits (`chore: auto-commit ... (heuristic)`) are expected churn and
# never trip F on their own; they DO count toward R.
#
# Usage:
#   scripts/preflight-tree-check.sh              # execute checks, exit 0/1
#   PREFLIGHT_RECENCY_SECONDS=0 scripts/...      # disable recency window
# Exit: 0 = safe to mutate; 1 = abort (reasons listed); 2 = usage/environment
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)" || {
  echo "preflight: not inside a git repository" >&2
  exit 2
}
cd "$REPO_ROOT" || exit 2

RECENCY="${PREFLIGHT_RECENCY_SECONDS:-300}"
MAX_RECENT="${PREFLIGHT_MAX_RECENT_COMMITS:-3}"

# Uncommitted = staged + unstaged + untracked (ignored files excluded).
tree_dirty() {
  ! git diff --quiet 2>/dev/null || ! git diff --cached --quiet 2>/dev/null ||
    [ -n "$(git ls-files --others --exclude-standard 2>/dev/null)" ]
}

abort=0

# D: dirty tree?
if tree_dirty; then
  echo "preflight: ABORT — dirty tree; uncommitted changes present:" >&2
  git status --short | head -10 | sed 's/^/    /' >&2
  abort=1
fi

if [ "$RECENCY" -gt 0 ]; then
  now=$(date +%s)
  # commit-ts<newline>is-daemon<newline>subject<newline> — read 3 lines per commit
  foreign=""
  recent_total=0
  while IFS=$'\t' read -r cts subject; do
    age=$((now - cts))
    [ "$age" -gt "$RECENCY" ] && continue
    recent_total=$((recent_total + 1))
    case "$subject" in
    "chore: auto-commit"*) : ;;
    *) foreign+="${subject}"$'\n' ;;
    esac
  done < <(git log --since="@-${RECENCY} seconds" --format='%ct%x09%s' 2>/dev/null)

  # F: foreign fresh commit?
  if [ -n "$foreign" ]; then
    echo "preflight: ABORT — non-daemon commit(s) within the last ${RECENCY}s (possible concurrent session in flight):" >&2
    printf '%s\n' "$foreign" | sed '/^$/d' | head -5 | sed 's/^/    /' >&2
    abort=1
  fi

  # R: commit velocity?
  if [ "$recent_total" -gt "$MAX_RECENT" ]; then
    echo "preflight: ABORT — ${recent_total} commits within the last ${RECENCY}s (max ${MAX_RECENT}); an active batch may be running" >&2
    abort=1
  fi
fi

if [ "$abort" -ne 0 ]; then
  echo "preflight: resolve the above (commit, coordinate, or wait), then retry" >&2
  exit 1
fi

echo "preflight: OK — tree clean, no foreign in-flight activity (window ${RECENCY}s)"
exit 0
