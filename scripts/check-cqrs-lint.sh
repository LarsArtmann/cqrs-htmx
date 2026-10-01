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
# cqrs-lint shells out to `go list`: goPkg (1.27.1) + GOTOOLCHAIN=local are
# REQUIRED or every module fails with load errors (the ambient go cannot load
# the 1.27.1 workspace).
#
# Usage: bash scripts/check-cqrs-lint.sh [module-dir...]   (default: the gate list)
set -uo pipefail

export GOWORK=off
export GOEXPERIMENT=jsonv2
export GOTOOLCHAIN=local

command -v cqrs-lint >/dev/null 2>&1 || {
  echo "check-cqrs-lint: FAILED — cqrs-lint not in PATH (local-only gate; install it or run via the flake app)" >&2
  exit 1
}

MODULES=("$@")
if [ "${#MODULES[@]}" -eq 0 ]; then
  MODULES=(. identity-model usermgmt usermgmt/totp usermgmt/webauthn usermgmt/oauth2 adminui loginpage dashboardui datastar systemadapter health auditlog)
fi

echo "=== cqrs-lint strict check (stale suppressions fail) ==="
echo "candidates=${#MODULES[@]}"
fail=0
for mod in "${MODULES[@]}"; do
  echo "==> $mod"
  if ! (cd "$mod" && cqrs-lint --strict --fail-on-stale-suppressions . >/dev/null 2>&1); then
    echo "FAIL: cqrs-lint findings in $mod (run 'cqrs-lint --strict --fail-on-stale-suppressions .' for details)"
    fail=1
  fi
done
if [ "$fail" -eq 0 ]; then
  echo "All modules pass cqrs-lint strict."
  exit 0
fi
exit 1
