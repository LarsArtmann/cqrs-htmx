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
# offline (scripts/selftests/test-check-workspace-build.sh).
#
# CONSUMER VIEW: the tracked go.work carries machine-local replace targets
# (the fleet's sibling checkouts under /home/lars/projects/...) that cannot
# exist on a CI runner and are NOT what consumers resolve. The gate builds
# through a filtered go.work that drops absolute/relative-path replaces;
# version-to-version replaces (the 373209a7 dropped-pin class) survive the
# filter and still fire. First CI run (2026-10-01, run 36824785928) proved
# the unfiltered form fails exactly there.
#
# Usage: nix run .#check-workspace-build  OR  bash scripts/checks/check-workspace-build.sh
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
WORK_GO_WORK=".go.work.check-workspace-build"
# Drop machine-local replace targets (absolute /paths and relative ../paths);
# keep every version-to-version replace line.
grep -v -E '^[[:space:]]*replace[[:space:]]+[^[:space:]]+[[:space:]]+=>[[:space:]]+(\.\./|/)' go.work >"$WORK_GO_WORK" || true
if [ ! -s "$WORK_GO_WORK" ]; then
  echo "check-workspace-build: FAILED — filtered go.work is empty (go.work malformed?)" >&2
  rm -f "$WORK_GO_WORK"
  exit 1
fi
trap 'rm -f "$LOG" "$(pwd)/$WORK_GO_WORK"' EXIT
export GOWORK="$PWD/$WORK_GO_WORK"

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
