#!/usr/bin/env bash
# check-workspace-build.sh — workspace-mode `go build ./...` for EVERY module
# in go.work.
#
# WHY THIS EXISTS: the per-module gates run hermetically (GOWORK=off), so they
# are BLIND to go.work-level breakage. The 2026-09-30 incident (BuildFlow's
# go-work-sync auto-dropped the load-bearing `go-etag => v0.6.0` resolution
# shield from go.work, commit 373209a7) broke every workspace build with an
# ambiguous-import load error while every per-module gate stayed green — the
# union module graph only assembles in workspace mode. This gate is the
# complement: workspace mode deliberately ON, `go build ./...` executed inside
# each workspace module so version selection, `use` coverage, and `replace`
# resolution are proven against the union graph everywhere, not just at root.
#
# WORKSPACE_BUILD_ROOT overrides the tree under test (default: this repo's
# root) — the fixture self-test uses it to build throwaway workspaces
# offline (scripts/test-check-workspace-build.sh).
#
# Usage: nix run .#check-workspace-build  OR  bash scripts/check-workspace-build.sh
set -uo pipefail

export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"
# Workspace mode deliberately ON — that is the entire point (no GOWORK=off).
# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/lib/go-cache-env.sh"

PROJECT_ROOT="${WORKSPACE_BUILD_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$PROJECT_ROOT" || exit 1

if [ ! -f go.work ]; then
  echo "check-workspace-build: FAILED — no go.work at $PROJECT_ROOT" >&2
  exit 1
fi

LOG=$(mktemp /tmp/check-workspace-build-XXXXXX.log)
trap 'rm -f "$LOG"' EXIT

# Workspace members from go.work itself — the artifact whose integrity this
# gate protects. Handles both `use ./x` and block `use (...)` forms.
MODULE_DIRS=$(awk '
  /^use \(/ {inblock=1; next}
  inblock && /^\)/ {inblock=0; next}
  inblock {gsub(/\t/, "", $1); print $1}
  /^use [^([]/ {print $2}
' go.work)
if [ -z "$MODULE_DIRS" ]; then
  echo "check-workspace-build: FAILED — go.work declares no use entries at $PROJECT_ROOT" >&2
  exit 1
fi

count=0
while IFS= read -r moddir; do
  [ -z "$moddir" ] && continue
  count=$((count + 1))
  if ! (cd "$PROJECT_ROOT/$moddir" && go build ./... >>"$LOG" 2>&1); then
    echo "check-workspace-build: FAILED — workspace-mode go build ./... broke in $moddir (the 373209a7 class: go.work use/replace mangling only surfaces in workspace mode)" >&2
    tail -30 "$LOG" >&2
    exit 1
  fi
done <<<"$MODULE_DIRS"

echo "check-workspace-build: OK — workspace-mode go build ./... green in $count modules"
