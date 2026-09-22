#!/usr/bin/env bash
# check-vcs-cache.sh — verify the Go module VCS cache is resolvable.
#
# WHY (2026-09-22, round6-tail §f10/f11): GOPRIVATE family deps (the
# `github.com/larsartmann/*` repos on this fleet) resolve through BARE git
# repos under $GOMODCACHE/cache/vcs/*/. An entry that loses its
# `remote.origin.url` (observed on /mnt/buildcache) breaks EVERY hermetic
# GOWORK=off build with load-error-shaped failures (`invalid package name:
# ""` — looks like a code bug, is actually cache corruption). The repair:
#   git -C <vcs-dir> remote add origin https://github.com/larsartmann/<repo>.git
# (full narrative: AGENTS.md gotcha 12). This gate surfaces the corruption
# BEFORE a build eats it.
#
# Behavior:
#   - Sweeps every directory under $GOMODCACHE/cache/vcs/ that looks like a
#     bare git repo (has a HEAD file); asserts remote.origin.url is set.
#   - No GOMODCACHE, no go, or no cache/vcs directory => nothing to check
#     (exit 0). CI runners fetch via the proxy and have no VCS entries.
#   - A repo missing its origin remote => FAIL with the repair command.
#   - TEST HOOK: VCS_CACHE_ROOT overrides the cache/vcs path (self-test).
#
# Usage: ./scripts/check-vcs-cache.sh
# Exit: 0 = cache healthy or absent, 1 = at least one VCS entry is broken

set -uo pipefail

VCS_DIR="${VCS_CACHE_ROOT:-}"
if [ -z "$VCS_DIR" ]; then
  MODCACHE="${GOMODCACHE:-}"
  if [ -z "$MODCACHE" ] && command -v go >/dev/null 2>&1; then
    MODCACHE="$(go env GOMODCACHE 2>/dev/null || true)"
  fi
  [ -z "$MODCACHE" ] && {
    echo "✓ no GOMODCACHE to check (no go, no env) — nothing to do"
    exit 0
  }
  VCS_DIR="$MODCACHE/cache/vcs"
fi

if [ ! -d "$VCS_DIR" ]; then
  echo "✓ no VCS cache at $VCS_DIR (proxy-only resolution) — nothing to do"
  exit 0
fi

broken=0
checked=0
for dir in "$VCS_DIR"/*/; do
  [ -d "$dir" ] || continue
  # Only bare-repo-shaped entries (go creates one per GOPRIVATE repo).
  [ -f "$dir/HEAD" ] || continue
  checked=$((checked + 1))
  url="$(git -C "$dir" config --get remote.origin.url 2>/dev/null || true)"
  if [ -z "$url" ]; then
    echo "✗ VCS cache entry $dir has NO remote.origin.url" >&2
    echo "  Every GOPRIVATE resolution of this repo will fail with" >&2
    echo "  load-error-shaped failures (invalid package name: \"\")." >&2
    echo "  Repair: git -C $dir remote add origin https://github.com/larsartmann/<repo>.git" >&2
    broken=$((broken + 1))
  fi
done

if [ "$broken" -gt 0 ]; then
  echo ""
  echo "✗ $broken of $checked VCS cache entries are broken"
  exit 1
fi

echo "✓ All $checked VCS cache entries carry remote.origin.url"
exit 0
