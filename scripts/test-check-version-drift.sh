#!/usr/bin/env bash
# test-check-version-drift.sh — self-test for check-version-drift.sh and the
# shared replace-exemption helper (scripts/lib/replace-exemption.sh).
#
# Coverage (the 2026-09-09 exemption change had NO test — this closes that):
#   1. helper unit: replace/no-replace/tab-indented/old-format go.mods
#   2. drift detection fires across a fixture root (v1.14.0 vs v1.16.0)
#   3. --strict exits 1 on drift; advisory mode exits 0 with a DRIFT line
#   4. a replace-satisfied require is exempt from drift (no finding)
#   5. uniform versions are clean in both modes
#
# Usage: ./scripts/test-check-version-drift.sh
# Exit:  0 = all tests pass, 1 = at least one fails
# Requires: network for the fixture tag-existence leg (real
# github.com/larsartmann/templ-components tags are used as fixtures).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRIFT="$SCRIPT_DIR/check-version-drift.sh"
LIB="$SCRIPT_DIR/lib/replace-exemption.sh"

pass=0
fail=0

report() {
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

SCRATCH="$(mktemp -d)"
defer_cleanup() { rm -rf "$SCRATCH"; }
trap defer_cleanup EXIT

# --- fixture builders ------------------------------------------------------

make_root() { # <name>
  mkdir -p "$SCRATCH/$1"
  echo "$SCRATCH/$1"
}

make_module() { # <root> <dir> <module-path> <version> [replace=>]
  local d="$1/$2"
  mkdir -p "$d"
  {
    echo "module example.com/$2"
    echo ""
    echo "require $3 $4"
  } >"$d/go.mod"
  if [ -n "${5:-}" ]; then
    printf 'replace %s v1.14.0 => ../replaced\n' "$3" >>"$d/go.mod"
  fi
}

# --- 1. helper unit tests --------------------------------------------------

echo "== replace-exemption helper"
# shellcheck source=lib/replace-exemption.sh
source "$LIB"

H="$SCRATCH/helper"
mkdir -p "$H/a" "$H/b" "$H/c" "$H/d"
printf 'module m\n\nrequire github.com/larsartmann/x v1.0.0\n\nreplace github.com/larsartmann/x v1.0.0 => ../x\n' >"$H/a/go.mod"
printf 'module m\n\nrequire github.com/larsartmann/x v1.0.0\n' >"$H/b/go.mod"
printf 'module m\n\nrequire github.com/larsartmann/x v1.0.0\n\n\treplace github.com/larsartmann/x => ../x\n' >"$H/c/go.mod"
printf 'module m\n\nrequire github.com/larsartmann/x v1.0.0\n\nreplace github.com/larsartmann/other => ../other\n' >"$H/d/go.mod"

replace_exemption_applies "$H/a" "github.com/larsartmann/x"
report $? "plain replace line exempts the require"
if replace_exemption_applies "$H/b" "github.com/larsartmann/x"; then
  report 1 "no replace line must not exempt"
else
  report 0 "no replace line must not exempt"
fi
replace_exemption_applies "$H/c" "github.com/larsartmann/x"
report $? "tab-indented replace line exempts"
if replace_exemption_applies "$H/d" "github.com/larsartmann/x"; then
  report 1 "replace of a DIFFERENT module must not exempt"
else
  report 0 "replace of a DIFFERENT module must not exempt"
fi

# --- 2-5. end-to-end drift behavior via DRIFT_ROOT fixtures ----------------

R_DRIFT="$(make_root drift_root)"
make_module "$R_DRIFT" mod_a github.com/larsartmann/templ-components v1.14.0
make_module "$R_DRIFT" mod_b github.com/larsartmann/templ-components v1.16.0

R_EXEMPT="$(make_root exempt_root)"
make_module "$R_EXEMPT" mod_a github.com/larsartmann/templ-components v1.14.0 replaced
make_module "$R_EXEMPT" mod_b github.com/larsartmann/templ-components v1.16.0

R_UNIFORM="$(make_root uniform_root)"
make_module "$R_UNIFORM" mod_a github.com/larsartmann/templ-components v1.16.0
make_module "$R_UNIFORM" mod_b github.com/larsartmann/templ-components v1.16.0

run_drift() { # <root> <extra args...>
  DRIFT_ROOT="$1" bash "$DRIFT" "${@:2}" 2>&1
}

echo "== drift detection"
OUT="$(run_drift "$R_DRIFT" --strict)" || RC=$?
if [ "${RC:-0}" -ne 0 ] && grep -q "DRIFT" <<<"$OUT"; then report 0 "--strict exits non-zero with a DRIFT finding (${RC:-0})"; else report 1 "--strict should fail on drift (rc=${RC:-0})"; fi
unset RC

OUT="$(run_drift "$R_DRIFT")" || RC=$?
if [ "${RC:-0}" -eq 0 ] && grep -q "DRIFT" <<<"$OUT"; then report 0 "advisory mode exits 0 but reports DRIFT"; else report 1 "advisory mode should warn-only on drift (rc=${RC:-0})"; fi
unset RC

echo "== replace exemption end-to-end"
OUT="$(run_drift "$R_EXEMPT" --strict)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && ! grep -q "DRIFT" <<<"$OUT"; then report 0 "replace-satisfied require is exempt (strict green)"; else report 1 "replace exemption should suppress drift (rc=${RC:-0}): $OUT"; fi
unset RC

echo "== uniform versions"
OUT="$(run_drift "$R_UNIFORM" --strict)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && ! grep -q "DRIFT" <<<"$OUT"; then report 0 "uniform versions are clean (strict green)"; else report 1 "uniform versions should be green (rc=${RC:-0}): $OUT"; fi
unset RC

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
