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
#   3. Raise GOTOOLCHAIN to the workspace floor when the ambient toolchain is
#      older (bare shells run go1.26.7 with GOTOOLCHAIN=local while go.work
#      demands 1.27.1 — the root cause of pre-commit-hook failures from
#      non-devShell shells). Never downgrades; a no-op when ambient is current.
#
# shellcheck shell=bash

if ! mkdir -p "${GOCACHE:-$HOME/.cache/go-build}" 2>/dev/null ||
  ! mkdir -p "${GOLANGCI_LINT_CACHE:-$HOME/.cache/golangci-lint}" 2>/dev/null; then
  GOCACHE="/tmp/go-build-cache"
  GOMODCACHE="/tmp/go-mod-cache"
  GOLANGCI_LINT_CACHE="/tmp/golangci-cache"
  export GOCACHE GOMODCACHE GOLANGCI_LINT_CACHE
  mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOLANGCI_LINT_CACHE"
fi

# Pin GOCACHE to the default when the ambient env carries none (CI
# runners) — the disk-space guard below reads it and `set -u` sourcing
# contexts would otherwise abort with "GOCACHE: unbound variable" (seen
# on the module-architecture CI job, 2026-09-01).
export GOCACHE="${GOCACHE:-$HOME/.cache/go-build}"

_cache_min_free_mb="${GO_CACHE_MIN_FREE_MB:-2048}"
case "$_cache_min_free_mb" in
'' | *[!0-9]*) _cache_min_free_mb="2048" ;;
esac

if [ "$_cache_min_free_mb" != "0" ] && command -v df >/dev/null 2>&1; then
  # GOMODCACHE is checked alongside GOCACHE: module sources typically dwarf
  # the build cache and the same "full disk fails late and confusingly"
  # class applies (cold /tmp re-downloads everything at once).
  for _cache_dir in "$GOCACHE" "${GOMODCACHE:-$HOME/go/pkg/mod}"; do
    _cache_avail_kb="$(df -Pk "$_cache_dir" 2>/dev/null | awk 'NR == 2 { print $4 }' || true)"
    if [ -n "$_cache_avail_kb" ]; then
      _cache_avail_mb=$((_cache_avail_kb / 1024))
      if [ "$_cache_avail_mb" -lt "$_cache_min_free_mb" ]; then
        echo "go-cache-env: LOW DISK SPACE: ${_cache_avail_mb}MB free on the filesystem holding $_cache_dir (minimum: ${_cache_min_free_mb}MB)." >&2
        echo "  Why: a full cache disk fails late and confusingly — 'no space left on device', corrupted module cache, phantom 'missing go.sum entry' (2026-08-29 /tmp incident; see AGENTS.md)." >&2
        echo "  Fix: free space — the cache directories are safe to delete, Go rebuilds them on demand — or set GO_CACHE_MIN_FREE_MB=<mb> (0 disables this check)." >&2
        # return covers the source context; exit covers direct execution
        # (unreachable when sourced, hence the directive).
        # shellcheck disable=SC2317
        return 1 2>/dev/null || exit 1
      fi
    fi
  done
fi

# --- Toolchain floor alignment (2026-09-22) ---
# go.work's `go` directive demands a minimum toolchain. Ambient shells on this
# fleet run an older go with GOTOOLCHAIN=local, which cannot load the
# workspace — buildflow's Go steps then fail, so commits from bare shells
# bypassed the hook entirely (the heuristic-commit pile of 2026-09-21/22).
# Only RAISE to the floor, never downgrade a newer ambient toolchain. The
# required toolchain resolves from the local module cache — no network.
if [ -z "${GOTOOLCHAIN:-}" ] || [ "${GOTOOLCHAIN:-}" = "local" ]; then
  _repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
  _ws_floor="$(awk '$1 == "go" { print $2; exit }' "${_repo_root}/go.work" 2>/dev/null || true)"
  _ws_floor="${_ws_floor#go}"
  _ambient_go="$(go version 2>/dev/null | awk '{ print $3 }' | sed 's/^go//')"
  if [ -n "$_ws_floor" ] && [ -n "$_ambient_go" ]; then
    _older_of="$(printf '%s\n%s\n' "$_ws_floor" "$_ambient_go" | sort -V | head -1)"
    if [ "$_older_of" = "$_ambient_go" ] && [ "$_ws_floor" != "$_ambient_go" ]; then
      export GOTOOLCHAIN="go$_ws_floor"
    fi
  fi
  unset _repo_root _ws_floor _ambient_go _older_of
fi
