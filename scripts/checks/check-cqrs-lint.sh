#!/usr/bin/env bash
# check-cqrs-lint.sh — cqrs-lint --strict over every gate module.
#
# Extracted from the inline flake app (2026-10-01) so the flag surface has one
# home and the fixture self-test can pin it. Runs the SYSTEM cqrs-lint binary
# from PATH (local-only gate: CI has no cqrs-lint until the Go-installable
# distribution lands — see TODO P2/P3 "cqrs-lint distribution").
#
# Flags: --strict (hard-fails on package LOAD errors — a broken build must
# never look green) and --fail-on-stale-suppressions (a suppression whose rule
# no longer fires is an ERROR: stale suppressions are lies that accrete; act
# on the tool's "safe to remove" hints instead of keeping them, gotcha 13).
#
# Findings posture (escalated 2026-11-17, the BuildFlow-incident closure):
# the gate fails on ANY finding output — warning/info included — not only on
# error-severity exit codes. The old exit-code-only verdict stayed green for
# weeks while 200 warning/info findings accumulated invisibly behind this
# script's /dev/null; with the v4.22.0 per-module truth + V003 silence the
# tree's honest residual is ZERO findings, so any future finding is a
# triage-now signal (disable-with-reason config or a fix), never background
# noise. Findings are detected from the output shape
# (`<SEVERITY> <file>:<line>:<col> ...`), so the verdict no longer depends on
# the binary's exit-code severity policy.
#
# cqrs-lint shells out to `go list`: goPkg (the pinned flake toolchain) +
# GOTOOLCHAIN=local are REQUIRED or every module fails with load errors (the
# ambient go cannot load the workspace floor).
#
# Usage: bash scripts/checks/check-cqrs-lint.sh [module-dir...]   (default: the gate list)
set -uo pipefail

export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOTOOLCHAIN=local

command -v cqrs-lint >/dev/null 2>&1 || {
  echo "check-cqrs-lint: FAILED — cqrs-lint not in PATH (local-only gate; install it or run via the flake app)" >&2
  exit 1
}

# Stale-binary guard: the findings-blocking posture is only sound on
# cqrs-lint >= v4.22.0. Older binaries carry the fleet-constant V003 false
# positives (systemcatalog v4.0.0 mislabeled "3 behind v4.3.x" under
# GOPRIVATE) and the workspace-root phantom walk — an escalated gate on such
# a binary would red on noise instead of truth. Pseudo-version suffixes
# (4.20.1-0.<date>-<sha>) sort below the plain release, which is the
# direction we want to refuse.
GATE_MIN_VERSION="4.22.1"
GOT_VERSION="$(cqrs-lint version 2>/dev/null | awk '{print $2}')"
if [ -z "$GOT_VERSION" ]; then
  echo "check-cqrs-lint: FAILED — 'cqrs-lint version' printed nothing (stale or broken binary)" >&2
  exit 1
fi
OLDEST="$(printf '%s\n%s\n' "$GATE_MIN_VERSION" "$GOT_VERSION" | sort -V | head -1)"
if [ "$OLDEST" != "$GATE_MIN_VERSION" ]; then
  echo "check-cqrs-lint: FAILED — cqrs-lint $GOT_VERSION is older than the required $GATE_MIN_VERSION." >&2
  echo "  The findings-blocking posture needs the per-module truth wave (v4.22.0):" >&2
  echo "  older binaries fabricate version-lag findings (fleet-constant V003) and" >&2
  echo "  walk monorepo roots in workspace mode (the 143-findings phantom class)." >&2
  echo "  Install: go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4@v4.22.0" >&2
  exit 1
fi

MODULES=("$@")
if [ "${#MODULES[@]}" -eq 0 ]; then
  MODULES=(. identity-model usermgmt usermgmt/totp usermgmt/webauthn usermgmt/oauth2 adminui loginpage dashboardui datastar huma systemadapter health auditlog)
fi

echo "=== cqrs-lint strict check (ANY finding fails; stale suppressions fail) ==="
echo "candidates=${#MODULES[@]}"
fail=0
for mod in "${MODULES[@]}"; do
  echo "==> $mod"
  module_out="$(cd "$mod" && cqrs-lint --strict --fail-on-stale-suppressions . 2>&1)"
  module_rc=$?
  # Finding lines carry a position: "<SEVERITY> <file>:<line>:<col> ..." at
  # column 0. The verdict banner's bare "ERROR" is indented, so it never
  # matches the anchored pattern — only real findings do.
  if [ "$module_rc" -ne 0 ] || printf '%s\n' "$module_out" | grep -qE '^(INFO|WARNING|ERROR) .+:[0-9]+:[0-9]+'; then
    echo "FAIL: cqrs-lint findings in $mod (rc=$module_rc, full report below):"
    printf '%s\n' "$module_out" | tail -60
    fail=1
  fi
done
if [ "$fail" -eq 0 ]; then
  echo "All modules pass cqrs-lint strict (zero findings)."
  exit 0
fi
exit 1
