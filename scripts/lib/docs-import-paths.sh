#!/usr/bin/env bash
# docs-import-paths.sh — the /v4-suffix rule for cqrs-htmx import paths in
# living docs (docs-health B2).
#
# The bug class (README, 2026-09-20): documented import paths like
# `github.com/larsartmann/cqrs-htmx/htmx` compile nowhere — the module is
# `github.com/larsartmann/cqrs-htmx/v4`, so every SUBPACKAGE import must
# carry the `/v4` major suffix. Nested submodule modules carry it after the
# submodule path (`cqrs-htmx/usermgmt/webauthn/v4`). Deliberately NOT flagged:
# bare repo mentions (no trailing slash — legitimate URLs), GitHub web paths
# (`/commits/`, `/tree/`, …), and workspace-only modules (`integration_test`,
# `e2e`, `examples`) whose module paths genuinely have no major suffix.
#
# Usage (sourced):
#   check_import_paths <file>...
#     prints one "STALE:" line per offender; returns 1 if any found, else 0.
#     (Signals via exit status — a variable would be lost to the command
#     substitution subshell.)
# shellcheck shell=bash

check_import_paths() {
  local f found=0
  for f in "$@"; do
    [ -f "$f" ] || continue
    while IFS=: read -r lineno match; do
      [ -n "$match" ] || continue
      echo "  STALE: $f:$lineno cqrs-htmx subpackage path missing /v4: ...${match: -60}"
      found=1
    done < <(grep -nP 'github\.com/larsartmann/cqrs-htmx/(?!v\d+\b)(?!([a-z0-9-]+/)+v\d+\b)(?!(integration_test|e2e|examples)\b)(?!(commits|tree|blob|releases|issues|pull|compare|wiki|actions)/)' "$f" 2>/dev/null || true)
  done
  return "$found"
}
