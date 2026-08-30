#!/usr/bin/env bash
# install-git-hooks.sh — keep .git/hooks from drifting.
#
# The repo's pre-commit hook = buildflow's generated baseline + 3 MANUAL
# EDIT blocks (large-file guard, release-train gate, GOCACHE prewarm).
# The manual blocks live ONLY in the installed hook (untracked), so a
# `buildflow precommit install` regeneration silently deletes them —
# that is how gates stop gating. This script restores them.
#
# The template of truth is scripts/hooks/pre-commit.template. The
# installed hook must byte-match it; anything else is refuse-and-diff
# (buildflow's own baseline is recoverable via `buildflow precommit
# install`, so overwriting from the template is always safe).
#
# Discovered 2026-08-30: the global gitconfig sets core.hooksPath=.githooks
# and .githooks/ did not exist — the repo's manual gates were silently
# dead. The ACTIVE hook path is resolved from core.hooksPath (repo, then
# global) with .git/hooks as fallback; a relative hooksPath resolves
# against the repo root.
#
# Usage:
#   scripts/install-git-hooks.sh            install template if the hook
#                                           is missing; refuse with a diff
#                                           if it exists but differs
#   scripts/install-git-hooks.sh --force    overwrite the hook with the
#                                           template wholesale
#   scripts/install-git-hooks.sh --verify   exit 0 if byte-match, 1 with a
#                                           diff if not, 2 if missing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE="$SCRIPT_DIR/hooks/pre-commit.template"

ROOT="$(git -C "$SCRIPT_DIR/.." rev-parse --show-toplevel)"
HOOKPATH="$(git -C "$ROOT" config core.hooksPath || true)"
if [ -n "$HOOKPATH" ]; then
  case "$HOOKPATH" in
  /*) ;;
  *) HOOKPATH="$ROOT/$HOOKPATH" ;;
  esac
else
  HOOKPATH="$(git -C "$ROOT" rev-parse --git-path hooks)"
fi
HOOK="$HOOKPATH/pre-commit"

usage() {
  echo "usage: $0 [--force|--verify]" >&2
  exit 2
}

MODE=default
for arg in "$@"; do
  case "$arg" in
  --force) MODE=force ;;
  --verify) MODE=verify ;;
  *) usage ;;
  esac
done

[ -f "$TEMPLATE" ] || {
  echo "install-git-hooks: template missing: $TEMPLATE" >&2
  exit 1
}

case "$MODE" in
verify)
  if [ ! -f "$HOOK" ]; then
    echo "install-git-hooks: VERIFY FAIL — $HOOK does not exist (run: $0)" >&2
    exit 2
  fi
  if cmp -s "$TEMPLATE" "$HOOK"; then
    echo "install-git-hooks: VERIFY OK — $HOOK byte-matches the template"
    exit 0
  fi
  echo "install-git-hooks: VERIFY FAIL — $HOOK differs from template:" >&2
  diff "$TEMPLATE" "$HOOK" >&2 || true
  echo "install-git-hooks: re-run with --force to restore the manual gate blocks" >&2
  exit 1
  ;;
force)
  mkdir -p "$HOOKPATH"
  cp "$TEMPLATE" "$HOOK"
  chmod +x "$HOOK"
  echo "install-git-hooks: $HOOK overwritten with template"
  cmp -s "$TEMPLATE" "$HOOK" # sanity; cp cannot diverge
  echo "install-git-hooks: VERIFY OK — byte-match confirmed"
  ;;
default)
  if [ ! -f "$HOOK" ]; then
    mkdir -p "$HOOKPATH"
    cp "$TEMPLATE" "$HOOK"
    chmod +x "$HOOK"
    echo "install-git-hooks: no hook existed — installed template at $HOOK"
    exit 0
  fi
  if cmp -s "$TEMPLATE" "$HOOK"; then
    echo "install-git-hooks: hook already byte-matches template — nothing to do"
    exit 0
  fi
  echo "install-git-hooks: $HOOK differs from the template — refusing to guess:" >&2
  diff "$TEMPLATE" "$HOOK" >&2 || true
  echo "install-git-hooks: if the diff is only a buildflow regeneration (manual gate blocks missing), run with --force to restore them" >&2
  exit 1
  ;;
esac
