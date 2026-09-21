#!/usr/bin/env bash
# docs-import-paths.sh — the /v4-suffix rule for cqrs-htmx import paths in
# living docs (docs-health B2).
#
# The bug class (README, 2026-09-20): documented import paths like
# `github.com/larsartmann/cqrs-htmx/htmx` compile nowhere — the module is
# `github.com/larsartmann/cqrs-htmx/v4`, so every SUBPACKAGE import must
# carry the `/v4` major suffix. Bare repo mentions (no trailing slash) are
# legitimate URLs and are not flagged.
#
# Usage (sourced):
#   check_import_paths <file>...
#     prints one "STALE:" line per offender and sets CHECK_FAILED=1.
# shellcheck shell=bash

check_import_paths() {
  # shellcheck disable=SC2034
  CHECK_FAILED="${CHECK_FAILED:-0}"
  local f
  for f in "$@"; do
    [ -f "$f" ] || continue
    while IFS=: read -r lineno match; do
      [ -n "$match" ] || continue
      echo "  STALE: $f:$lineno cqrs-htmx subpackage path missing /v4: ...${match: -60}"
      CHECK_FAILED=1
    done < <(grep -nP 'github\.com/larsartmann/cqrs-htmx/(?!v4\b)' "$f" 2>/dev/null || true)
  done
  return 0
}
