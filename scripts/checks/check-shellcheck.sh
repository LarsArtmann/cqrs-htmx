#!/usr/bin/env bash
# check-shellcheck.sh — shellcheck every gate script under scripts/, the SAME
# file set CI's "Shellcheck gate scripts" step lints (koalaman/shellcheck:stable
# over scripts/**/*.sh). Until this gate existed (2026-11-03) shellcheck ran
# ONLY in CI — a local check-modules green could never see the finding class
# that reddened CI run 41675571500 (an unused --gate flag + two
# literal-backtick SC2016s). This closes the CI/local parity gap.
#
# Binary resolution: $SHELLCHECK_BIN overrides; else a shellcheck on PATH
# (the check-modules app provides one via runtimeInputs); else `nix run
# nixpkgs#shellcheck` when nix exists (local nixpkgs tracks stable closely —
# the SC2016/SC2034 classes this gate exists for are stable across both).
#
# Env: SHELLCHECK_SCRIPTS_DIR  override the linted tree (fixture self-test).
# Exit: 0 clean · 1 findings · 2 shellcheck unavailable.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SCRIPTS="${SHELLCHECK_SCRIPTS_DIR:-$REPO_ROOT/scripts}"

if [ ! -d "$SCRIPTS" ]; then
  echo "✗ shellcheck: $SCRIPTS is not a directory" >&2
  exit 2
fi

sc_bin="${SHELLCHECK_BIN:-$(command -v shellcheck || true)}"
if [ -z "$sc_bin" ]; then
  if command -v nix >/dev/null 2>&1; then
    sc_bin="nix-shell-nixpkgs" # sentinel: resolve per-invocation below
  else
    echo "✗ shellcheck: no shellcheck on PATH and no nix fallback — run via 'nix run .#check-modules' or set SHELLCHECK_BIN" >&2
    exit 2
  fi
fi

# Same enumeration as CI: container-relative there, repo-relative here.
mapfile -t files < <(cd "$SCRIPTS" && find . -name '*.sh' | sort | sed 's|^\./||')
if [ "${#files[@]}" -eq 0 ]; then
  echo "✗ shellcheck: no .sh files found under $SCRIPTS — the empty-file-list trap (CI comment, ci.yml Shellcheck step)" >&2
  exit 2
fi

rc=0
if [ "$sc_bin" = "nix-shell-nixpkgs" ]; then
  (cd "$SCRIPTS" && nix run nixpkgs#shellcheck -- "${files[@]}") || rc=$?
else
  (cd "$SCRIPTS" && "$sc_bin" "${files[@]}") || rc=$?
fi

if [ "$rc" -ne 0 ]; then
  echo "✗ shellcheck: findings in ${#files[@]} script(s) under $SCRIPTS (exit $rc) — CI's docker step lints the same set" >&2
  exit 1
fi
echo "✓ shellcheck: ${#files[@]} script(s) under $SCRIPTS clean"
exit 0
