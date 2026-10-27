#!/usr/bin/env bash
# check-train-consumers-hermetic.sh — pre-publish hermetic consumer catch
# (gotcha-30 mechanized, plan T04).
#
# WHY THIS EXISTS: a sub-module require must resolve to a root/family tag that
# CONTAINS the symbols and BEHAVIOR it uses. While go.work carries local-path
# replaces, every workspace-mode gate is blind to gaps — workspace builds
# resolve `../` replaces and stay green — while every hermetic consumer
# (GOWORK=off, the proxy, CI's tag-level jobs) resolves PUBLISHED tags. The
# class fired twice: symbols (asset_serve.go vs root v4.13.1, 2026-10-07) and
# behavior (the login-page CSP test asserting root code that existed only
# unpushed, 2026-10-08 — `nix run .#test` is hermetic, so setup resolved root
# v4.13.2 from the proxy and the test failed ONLY there).
#
# WHAT IT DOES: finds every module whose go.mod/go.sum changed vs the base
# ref, then for each runs the HERMETIC consumer pipeline —
#   GOWORK=off go build ./... && go vet ./... && go test ./...
# — exactly what a consumer resolving published tags experiences. Wired into
# the release checklist BEFORE `verify-tag --push`: publishing happens only
# after this is green.
#
# Zero changed modules is an EXPLICIT pass with the count printed (gotcha 2:
# silent-zero is a false green). If the base ref cannot be resolved (shallow
# CI checkout), the gate self-skips with a loud message rather than guessing.
#
# Environment overrides (fixture self-test + scoped runs):
#   TRAIN_HERMETIC_BASE     base ref (default origin/master)
#   TRAIN_HERMETIC_ROOT     tree under test (default: this repo)
#   TRAIN_HERMETIC_PIPELINE command run per changed module
#                           (default: the GOWORK=off build+vet+test pipeline)
#
# Usage: nix run .#check-train-consumers-hermetic
# Exit: 0 = all changed modules pass hermetically (or none changed);
#       1 = at least one module fails; 2 = usage/environment
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="${TRAIN_HERMETIC_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}"
[ -n "$REPO_ROOT" ] || {
  echo "train-consumers-hermetic: not inside a git repository" >&2
  exit 2
}
cd "$REPO_ROOT" || exit 2

BASE="${TRAIN_HERMETIC_BASE:-origin/master}"
export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"
# shellcheck disable=SC1091
source "$(dirname "${BASH_SOURCE[0]}")/../lib/go-cache-env.sh"

PIPELINE="${TRAIN_HERMETIC_PIPELINE:-GOWORK=off go build ./... && GOWORK=off go vet ./... && GOWORK=off go test ./... -count=1}"

if ! git rev-parse --verify --quiet "$BASE" >/dev/null; then
  echo "train-consumers-hermetic: SKIP — base ref '$BASE' not resolvable (shallow checkout?)."
  echo "  The hermetic catch needs the diff base; run it locally where origin/master exists."
  exit 0
fi

# Changed module discovery: committed + staged + unstaged go.mod/go.sum paths
# under the diff base, mapped to their module directories. scripts/testdata is
# excluded (deliberately-untidy fixtures, gotcha 34).
changed_files=$(
  {
    git diff --name-only "$BASE" -- '*.mod' '*.sum'
    git diff --cached --name-only -- '*.mod' '*.sum'
    git diff --name-only -- '*.mod' '*.sum'
  } | sort -u
)

modules=""
while IFS= read -r f; do
  [ -n "$f" ] || continue
  case "$f" in scripts/testdata/*) continue ;; esac
  dir="$(dirname "$f")"
  [ "$dir" = "." ] && dir="Root"
  modules+="$dir"$'\n'
done <<<"$changed_files"

modules="$(printf '%s' "$modules" | sort -u)"
count="$(printf '%s' "$modules" | grep -c . || true)"
count="${count//[!0-9]/}"
count="${count:-0}"

if [ "$count" -eq 0 ]; then
  echo "train-consumers-hermetic: PASS — 0 modules with go.mod/go.sum changes vs $BASE; nothing to verify hermetically."
  exit 0
fi

echo "train-consumers-hermetic: $count module(s) with dependency changes vs $BASE — hermetic pipeline per module:"
echo "  pipeline: $PIPELINE"

failed=0
while IFS= read -r dir; do
  [ -n "$dir" ] || continue
  echo ""
  echo "=== $dir ==="
  if [ "$dir" = "Root" ]; then
    target="."
  else
    target="$dir"
  fi
  if (cd "$target" && eval "$PIPELINE") >/tmp/cqrs-htmx-hermetic-$$.log 2>&1; then
    echo "  ✓ hermetic build+vet+test GREEN"
  else
    rc=$?
    echo "  ✗ hermetic pipeline FAILED (rc=$rc) — consumers resolving PUBLISHED tags will see this."
    echo "  Output tail:"
    tail -20 "/tmp/cqrs-htmx-hermetic-$$.log" | sed 's/^/      /'
    failed=1
  fi
  rm -f "/tmp/cqrs-htmx-hermetic-$$.log"
done <<<"$modules"

echo ""
if [ "$failed" -eq 0 ]; then
  echo "✓ train-consumers-hermetic: $count/$count module(s) green — safe to publish"
  exit 0
else
  echo "✗ train-consumers-hermetic: red module(s) above — publish AFTER this is green (verify-tag --push waits)"
  exit 1
fi
