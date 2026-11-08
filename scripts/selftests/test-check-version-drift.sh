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
#   6-12. starvation defense (2026-11-08 incident; hermetic stub `git`):
#     6. starved ls-remote --strict exits 4 (UNVERIFIED, not MISSING)
#     7. starved + CI=true exits 0 with the downgraded warning
#     8. genuine MISSING TAG (tags fetchable, tag absent) still exits 1
#     9. warm cache satisfies verification with a DEAD network (0 ls-remotes)
#    10. stale-cache guard: cached miss re-verified LIVE, confirmed, cache rewritten
#    11. stale-cache guard: starved refetch counts UNVERIFIED (exit 4)
#    12. credential probe selects the pre-auth URL (larsartmann@github.com)
#
# Usage: ./scripts/selftests/test-check-version-drift.sh
# Exit:  0 = all tests pass, 1 = at least one fails
# Requires: network for the fixture tag-existence leg (real
# github.com/larsartmann/templ-components tags are used as fixtures).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DRIFT="$SCRIPT_DIR/../checks/check-version-drift.sh"
LIB="$SCRIPT_DIR/../lib/replace-exemption.sh"

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
# shellcheck source=lib/replace-exemption.sh disable=SC1091
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

# Stub `git` for the hermetic starvation cases: ls-remote/credential are
# faked per $STUB_DIR/.mode (starve | serve | serve_with); every ls-remote
# invocation is logged to $STUB_DIR/.calls.
STUB_BIN="$SCRATCH/stubbin"
STUB_DIR="$SCRATCH/stubctl"
mkdir -p "$STUB_BIN" "$STUB_DIR"
cat >"$STUB_BIN/git" <<'STUB'
#!/usr/bin/env bash
case "${1:-}" in
  credential)
    if [ -f "$STUB_DIR/.cred-ok" ]; then
      printf 'protocol=https\nhost=github.com\nusername=larsartmann\npassword=stub-token\n'
      exit 0
    fi
    exit 1
    ;;
  ls-remote)
    printf 'ls-remote %s\n' "$*" >>"$STUB_DIR/.calls"
    mode="$(cat "$STUB_DIR/.mode")"
    case "$mode" in
      starve) exit 128 ;;
      serve)
        printf '0000000000000000000000000000000000000000\trefs/tags/v1.0.0\n'
        printf '0000000000000000000000000000000000000000\trefs/tags/v1.0.1\n'
        ;;
      serve_with) cat "$STUB_DIR/.serve-tags" ;;
      *) exit 128 ;;
    esac
    ;;
  *) exit 1 ;;
esac
STUB
chmod +x "$STUB_BIN/git"

run_drift_stub() { # <root> <cache_dir> <extra args...>
  DRIFT_ROOT="$1" STUB_DIR="$STUB_DIR" TRAIN_TAG_CACHE_DIR="$2" \
    PATH="$STUB_BIN:$PATH" bash "$DRIFT" "${@:3}" 2>&1
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

# --- 6-12. starvation defense (hermetic stub git) ---------------------------

echo "== starvation exit semantics"
R_STUB="$(make_root stub_root)"
make_module "$R_STUB" mod_a github.com/larsartmann/stubrepo v1.0.0
rm -rf "$SCRATCH/cache6"
echo starve >"$STUB_DIR/.mode"
OUT="$(run_drift_stub "$R_STUB" "$SCRATCH/cache6" --strict)" || RC=$?
if [ "${RC:-0}" -eq 4 ] && grep -q "UNVERIFIED" <<<"$OUT"; then report 0 "starved --strict exits 4 with UNVERIFIED (not MISSING)"; else report 1 "starved strict should exit 4 (rc=${RC:-0}): $OUT"; fi
unset RC

rm -rf "$SCRATCH/cache7"
OUT="$(CI=true run_drift_stub "$R_STUB" "$SCRATCH/cache7" --strict)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && grep -q "UNVERIFIED" <<<"$OUT"; then report 0 "starved + CI=true exits 0 (downgraded warning)"; else report 1 "starved CI run should warn-and-pass (rc=${RC:-0}): $OUT"; fi
unset RC

echo "== genuine missing tag still blocks"
R_MISS="$(make_root miss_root)"
make_module "$R_MISS" mod_a github.com/larsartmann/stubrepo v9.9.9
rm -rf "$SCRATCH/cache8"
echo serve >"$STUB_DIR/.mode"
OUT="$(run_drift_stub "$R_MISS" "$SCRATCH/cache8" --strict)" || RC=$?
if [ "${RC:-0}" -eq 1 ] && grep -q "MISSING TAG" <<<"$OUT"; then report 0 "fetchable-but-absent tag exits 1 with MISSING TAG"; else report 1 "genuine missing should exit 1 (rc=${RC:-0}): $OUT"; fi
unset RC

echo "== warm cache survives a dead network"
rm -rf "$SCRATCH/cache9"
mkdir -p "$SCRATCH/cache9"
printf 'v1.0.0\nv1.0.1\n' >"$SCRATCH/cache9/stubrepo.tags"
rm -f "$STUB_DIR/.calls"
echo starve >"$STUB_DIR/.mode"
OUT="$(run_drift_stub "$R_STUB" "$SCRATCH/cache9" --strict)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && [ ! -s "$STUB_DIR/.calls" ]; then report 0 "cache hit verifies with ZERO ls-remotes (starvation-decoupled)"; else report 1 "cache hit should need no network (rc=${RC:-0}, calls=$(cat "$STUB_DIR/.calls" 2>/dev/null | wc -l))"; fi
unset RC

echo "== stale-cache live re-verify"
R_STALE="$(make_root stale_root)"
make_module "$R_STALE" mod_a github.com/larsartmann/stubrepo v1.0.1
rm -rf "$SCRATCH/cache10"
mkdir -p "$SCRATCH/cache10"
printf 'v1.0.0\n' >"$SCRATCH/cache10/stubrepo.tags"
printf '0000000000000000000000000000000000000000\trefs/tags/v1.0.1\n' >"$STUB_DIR/.serve-tags"
echo serve_with >"$STUB_DIR/.mode"
rm -f "$STUB_DIR/.calls"
OUT="$(run_drift_stub "$R_STALE" "$SCRATCH/cache10" --strict)" || RC=$?
if [ "${RC:-0}" -eq 0 ] && grep -q "live refetch confirmed" <<<"$OUT" && grep -q "v1.0.1" "$SCRATCH/cache10/stubrepo.tags"; then report 0 "cached miss re-verified live, confirmed, cache rewritten"; else report 1 "stale-cache happy path failed (rc=${RC:-0}): $OUT"; fi
unset RC

rm -rf "$SCRATCH/cache11"
mkdir -p "$SCRATCH/cache11"
printf 'v1.0.0\n' >"$SCRATCH/cache11/stubrepo.tags"
echo starve >"$STUB_DIR/.mode"
OUT="$(run_drift_stub "$R_STALE" "$SCRATCH/cache11" --strict)" || RC=$?
if [ "${RC:-0}" -eq 4 ] && grep -q "UNVERIFIED" <<<"$OUT"; then report 0 "starved refetch of a cached miss counts UNVERIFIED (exit 4)"; else report 1 "starved refetch should exit 4 (rc=${RC:-0}): $OUT"; fi
unset RC

echo "== pre-auth URL selection"
rm -rf "$SCRATCH/cache12"
echo serve >"$STUB_DIR/.mode"
rm -f "$STUB_DIR/.calls"
OUT="$(run_drift_stub "$R_STUB" "$SCRATCH/cache12" --strict)" || RC=$?
if grep -q "https://github.com/larsartmann/stubrepo.git" "$STUB_DIR/.calls" 2>/dev/null && ! grep -q "larsartmann@" "$STUB_DIR/.calls"; then report 0 "credential-less host keeps the anonymous URL"; else report 1 "anonymous URL expected without credentials: $(cat "$STUB_DIR/.calls" 2>/dev/null)"; fi
unset RC

touch "$STUB_DIR/.cred-ok"
rm -rf "$SCRATCH/cache12b"
rm -f "$STUB_DIR/.calls"
OUT="$(run_drift_stub "$R_STUB" "$SCRATCH/cache12b" --strict)" || RC=$?
if grep -q "https://larsartmann@github.com/larsartmann/stubrepo.git" "$STUB_DIR/.calls" 2>/dev/null; then report 0 "credential-filling host rides the pre-auth URL"; else report 1 "pre-auth URL expected with credentials: $(cat "$STUB_DIR/.calls" 2>/dev/null)"; fi
unset RC
rm -f "$STUB_DIR/.cred-ok"

echo ""
echo "Results: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
