#!/usr/bin/env bash
# erraudit-inventory.sh — per-module critical-findings inventory for the whole
# go.work workspace, so site counts stop living in /tmp working notes.
#
# WHY: the erraudit program (closed 2026-10-01) needed a repeatable answer to
# "how many erraudit criticals exist, in which module, at which site". Ad-hoc
# sweeps lived in /tmp and one of them silently matched zero modules and
# reported a false green. This script is the durable form of that sweep with
# the false-green guard built in: it PRINTS ITS CANDIDATE COUNT and fails
# loudly when the go.work decomposition finds zero modules.
#
# Output: `candidates=N` first, then `  <module>: <count>` per module, then
# `TOTAL=<n>`. Exit codes: 1 on zero candidates or erraudit failure;
# with --gate also 1 when TOTAL > 0. Without --gate it is a report tool
# (gating stays BuildFlow's erraudit step; fail_on: critical).
#
# ERRAUDIT_BIN overrides the binary (default: `erraudit` from PATH) — the
# fixture self-test uses it to run offline with a stub.
#
# Usage: bash scripts/checks/erraudit-inventory.sh [--gate] [repo-root]
set -uo pipefail

GATE=0
ROOT=""
for arg in "$@"; do
  case "$arg" in
  --gate) GATE=1 ;;
  *) ROOT="$arg" ;;
  esac
done
ROOT="${ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
cd "$ROOT" || exit 1

ERRAUDIT_BIN="${ERRAUDIT_BIN:-erraudit}"
command -v "$ERRAUDIT_BIN" >/dev/null 2>&1 || {
  echo "erraudit-inventory: FAILED — erraudit binary not found ('$ERRAUDIT_BIN')." >&2
  echo "  Install erraudit (local tool, not in the devShell) or point ERRAUDIT_BIN at it." >&2
  exit 1
}

[ -f go.work ] || {
  echo "erraudit-inventory: FAILED — no go.work at $ROOT" >&2
  exit 1
}

# Workspace members from go.work itself (same decomposition as
# check-workspace-build.sh — the artifact whose integrity matters here).
MODULE_DIRS=$(awk '
  /^use \(/ {inblock=1; next}
  inblock && /^\)/ {inblock=0; next}
  inblock {gsub(/\t/, "", $1); print $1}
  /^use [^([]/ {print $2}
' go.work)

candidates=$(printf '%s\n' "$MODULE_DIRS" | grep -c . || true)
echo "candidates=$candidates"
if [ "$candidates" -eq 0 ]; then
  echo "erraudit-inventory: FAILED — go.work decomposition found 0 modules (false-green guard; fix the awk, do not trust an empty sweep)" >&2
  exit 1
fi

total=0
failed=0
while IFS= read -r moddir; do
  [ -z "$moddir" ] && continue
  csv=$("$ERRAUDIT_BIN" "$moddir" --violations-only --severity critical --format csv 2>/dev/null)
  rc=$?
  if [ "$rc" -ne 0 ]; then
    echo "erraudit-inventory: FAILED — erraudit rc=$rc in $moddir" >&2
    failed=$((failed + 1))
    continue
  fi
  count=$(printf '%s\n' "$csv" | tail -n +2 | grep -c . || true)
  total=$((total + count))
  printf '  %s: %d\n' "$moddir" "$count"
done <<<"$MODULE_DIRS"

[ "$failed" -gt 0 ] && exit 1

echo "TOTAL=$total"
if [ "$GATE" -eq 1 ] && [ "$total" -gt 0 ]; then
  echo "erraudit-inventory: FAILED — $total erraudit critical(s) across the workspace (gate mode)" >&2
  exit 1
fi
exit 0
