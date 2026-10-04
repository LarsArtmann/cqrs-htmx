#!/usr/bin/env bash
# test-check-session-route-wrappers.sh — self-test for
# check-session-route-wrappers.sh (offline fixture trees):
#   F1  pristine copies of the real route files     -> PASS, 20/13/7 counts
#   F2  one session route unwrapped                 -> FAIL, names the route
#   F3  deep helper chain unwrapped (TOTP funnel)   -> FAIL, transitive catch
#   F4  ceremony gate dropped from wrapper set      -> FAIL (gate is a marker)
#   F5  empty route file                            -> FAIL (false-green guard)
#
# Usage: ./scripts/selftests/test-check-session-route-wrappers.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/check-session-route-wrappers.sh"
REPO="$(cd "$SCRIPT_DIR/../.." && pwd)"

pass=0
fail=0

report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

make_fixture() { # <dir> — pristine copy of the real usermgmt package
  mkdir -p "$1/usermgmt"
  cp "$REPO"/usermgmt/*.go "$1/usermgmt/"
}

expect_pass() { # <dir> <label>
  OUT="$(SESSION_ROUTE_WRAPPERS_ROOT="$1" bash "$CHECKER" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ]; then
    report 0 "$2"
  else
    report 1 "$2 (rc=$rc): $OUT"
  fi
}

expect_fail() { # <dir> <label> <must-mention>
  OUT="$(SESSION_ROUTE_WRAPPERS_ROOT="$1" bash "$CHECKER" 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ] && printf '%s' "$OUT" | grep -qF "$3"; then
    report 0 "$2"
  else
    report 1 "$2 (rc=$rc, wanted mention of '$3'): $OUT"
  fi
}

echo ""
echo "=== Test Suite: check-session-route-wrappers.sh ==="
echo ""

# --- F1: pristine real files pass --------------------------------------------
F1="$(mktemp -d)"
make_fixture "$F1"
expect_pass "$F1" "F1 pristine route files exit 0"

# --- F2: one direct session route unwrapped ----------------------------------
F2="$(mktemp -d)"
make_fixture "$F2"
sed -i 's|h.withSession(h.handleListCredentials)|h.handleListCredentials|' "$F2/usermgmt/http.go"
expect_fail "$F2" "F2 unwrapped credentials route fails" "GET /auth/credentials"

# --- F3: transitive helper chain unwrapped -----------------------------------
F3="$(mktemp -d)"
make_fixture "$F3"
sed -i 's|h.withSession(h.handleTOTPSetupVerify)|h.handleTOTPSetupVerify|' "$F3/usermgmt/verification_totp_http.go"
expect_fail "$F3" "F3 helper-funnelled TOTP route fails transitively" "POST /auth/totp/setup/verify"

# --- F4: ceremony registration loses its wrapper ------------------------------
F4="$(mktemp -d)"
make_fixture "$F4"
sed -i 's|h.withSession(h.handleWebAuthnBeginRegistration)|h.handleWebAuthnBeginRegistration|' "$F4/usermgmt/http.go"
expect_fail "$F4" "F4 ungated enrollment ceremony fails" "POST /auth/webauthn/register/begin"

# --- F5: empty route files trip the false-green guard -------------------------
F5="$(mktemp -d)"
make_fixture "$F5"
: >"$F5/usermgmt/http.go"
: >"$F5/usermgmt/verification_totp_http.go"
OUT="$(SESSION_ROUTE_WRAPPERS_ROOT="$F5" bash "$CHECKER" 2>&1)"
rc=$?
if [ "$rc" -ne 0 ]; then
  report 0 "F5 empty registrations fail loudly (rc=$rc)"
else
  report 1 "F5 empty registrations must fail (got rc=0)"
fi

echo ""
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ] || exit 1
echo "OK: all fixtures behaved"
