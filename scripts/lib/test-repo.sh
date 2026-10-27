#!/usr/bin/env bash
# test-repo.sh — shared git-fixture helper for self-tests.
#
# WHY: machine-global git config (user.email, commit.gpgSign, tag.gpgSign)
# leaks into fixture repos and breaks self-tests nondeterministically —
# test-verify-remote-tags needed an explicit `tag.gpgsign false` after
# exactly that leak (2026-10-13). One helper, one set of guarantees; new
# self-tests source this instead of re-rolling `git init` blocks.
#
# Usage:
#   . "$SCRIPT_DIR/../lib/test-repo.sh"
#   fixture_repo "$tmp/myrepo"        # init + identity + signing off + initial commit
#   fixture_git_config "$tmp/myrepo"  # config only (caller owns init + commits)
#
# Sourced, not executed.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  echo "test-repo.sh: source this file, do not execute it" >&2
  exit 2
fi

fixture_git_config() { # <dir>
  git -C "$1" config user.email test@example.com
  git -C "$1" config user.name test
  git -C "$1" config commit.gpgsign false
  git -C "$1" config tag.gpgsign false
}

fixture_repo() { # <dir> [branch]
  local dir="$1" branch="${2:-main}"
  git init -q -b "$branch" "$dir" || return 1
  fixture_git_config "$dir"
  git -C "$dir" commit -q --allow-empty -m initial
}
