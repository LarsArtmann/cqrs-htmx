#!/usr/bin/env bash
# go-cache-env.sh — shared guard for Go build caches on machines where the
# ambient cache location is unwritable (e.g. the dead /mnt/buildcache sda1,
# documented in AGENTS.md + TODO_LIST).
#
# Source this in any gate that runs go tooling:
#
#   source "$(dirname "${BASH_SOURCE[0]}")/../lib/go-cache-env.sh"
#
# Behavior:
#   1. If GOCACHE (or its default $HOME/.cache/go-build) cannot be created,
#      fall back to /tmp caches and export them.
#   2. Fail fast when the filesystem holding the resolved GOCACHE has less
#      than GO_CACHE_MIN_FREE_MB (default 2048) free. A full cache disk does
#      not fail cleanly — it surfaces minutes later as phantom "no space left
#      on device" errors, corrupted module-cache files, and bogus "missing
#      go.sum entry" messages (the 2026-08-29 /tmp tmpfs incident). Set
#      GO_CACHE_MIN_FREE_MB=0 to disable the check.
#
# shellcheck shell=bash

if ! mkdir -p "${GOCACHE:-$HOME/.cache/go-build}" 2>/dev/null; then
  GOCACHE="/tmp/go-build-cache"
  GOMODCACHE="/tmp/go-mod-cache"
  GOLANGCI_LINT_CACHE="/tmp/golangci-cache"
  export GOCACHE GOMODCACHE GOLANGCI_LINT_CACHE
  mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOLANGCI_LINT_CACHE"
fi

_cache_min_free_mb="${GO_CACHE_MIN_FREE_MB:-2048}"
case "$_cache_min_free_mb" in
'' | *[!0-9]*) _cache_min_free_mb="2048" ;;
esac

if [ "$_cache_min_free_mb" != "0" ] && command -v df >/dev/null 2>&1; then
  _cache_avail_kb="$(df -Pk "$GOCACHE" 2>/dev/null | awk 'NR == 2 { print $4 }' || true)"
  if [ -n "$_cache_avail_kb" ]; then
    _cache_avail_mb=$((_cache_avail_kb / 1024))
    if [ "$_cache_avail_mb" -lt "$_cache_min_free_mb" ]; then
      echo "go-cache-env: LOW DISK SPACE: ${_cache_avail_mb}MB free on the filesystem holding $GOCACHE (minimum: ${_cache_min_free_mb}MB)." >&2
      echo "  Why: a full cache disk fails late and confusingly — 'no space left on device', corrupted module cache, phantom 'missing go.sum entry' (2026-08-29 /tmp incident; see AGENTS.md)." >&2
      echo "  Fix: free space — the cache directories are safe to delete, Go rebuilds them on demand — or set GO_CACHE_MIN_FREE_MB=<mb> (0 disables this check)." >&2
      # return covers the source context; exit covers direct execution
      # (unreachable when sourced, hence the directive).
      # shellcheck disable=SC2317
      return 1 2>/dev/null || exit 1
    fi
  fi
fi
