#!/usr/bin/env bash
# go-cache-env.sh — shared guard for Go build caches on machines where the
# ambient cache location is unwritable (e.g. the dead /mnt/buildcache sda1,
# documented in AGENTS.md + TODO_LIST).
#
# Source this in any gate that runs go tooling:
#
#   source "$(dirname "${BASH_SOURCE[0]}")/../lib/go-cache-env.sh"
#
# Behavior: if GOCACHE (or its default $HOME/.cache/go-build) cannot be
# created, fall back to /tmp caches and export them. Also falls back when
# GOLANGCI_LINT_CACHE is set but unwritable, keeping every consumer uniform.
#
# shellcheck shell=bash

if ! mkdir -p "${GOCACHE:-$HOME/.cache/go-build}" 2>/dev/null; then
  GOCACHE="/tmp/go-build-cache"
  GOMODCACHE="/tmp/go-mod-cache"
  GOLANGCI_LINT_CACHE="/tmp/golangci-cache"
  export GOCACHE GOMODCACHE GOLANGCI_LINT_CACHE
  mkdir -p "$GOCACHE" "$GOMODCACHE" "$GOLANGCI_LINT_CACHE"
fi
