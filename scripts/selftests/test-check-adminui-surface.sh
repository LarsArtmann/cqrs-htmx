#!/usr/bin/env bash
# test-check-adminui-surface.sh — self-test for check-adminui-surface.sh
# (offline fixture trees staged from the real adminui table/nav/templates):
#   F1  pristine copies of routes.go + handler.go + *.templ   -> PASS
#   F2  sidebar nav href retargeted to an undeclared path     -> FAIL, names it
#   F3  templ href typo ("/usres/")                            -> FAIL, file:line
#   F4  row-defining function orphaned from the table assembly -> FAIL
#   F5  raw mux registration outside the route table           -> FAIL
#   F6  empty routes.go                                        -> FAIL (false-green guard)
#
# Usage: ./scripts/selftests/test-check-adminui-surface.sh
# Exit: 0 = all tests pass, 1 = at least one test fails

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHECKER="$SCRIPT_DIR/../checks/check-adminui-surface.sh"
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

make_fixture() { # <dir> — pristine copy of the adminui surface files
  mkdir -p "$1/adminui"
  cp "$REPO"/adminui/routes.go "$REPO"/adminui/handler.go "$1/adminui/"
  cp "$REPO"/adminui/*.templ "$1/adminui/"
}

expect_pass() { # <dir> <label>
  OUT="$(ADMINUI_SURFACE_ROOT="$1" bash "$CHECKER" 2>&1)"
  rc=$?
  if [ "$rc" -eq 0 ]; then
    report 0 "$2"
  else
    report 1 "$2 (rc=$rc): $OUT"
  fi
}

expect_fail() { # <dir> <label> <must-mention>
  OUT="$(ADMINUI_SURFACE_ROOT="$1" bash "$CHECKER" 2>&1)"
  rc=$?
  if [ "$rc" -ne 0 ] && printf '%s' "$OUT" | grep -qF "$3"; then
    report 0 "$2"
  else
    report 1 "$2 (rc=$rc, wanted mention of '$3'): $OUT"
  fi
}

echo ""
echo "=== Test Suite: check-adminui-surface.sh ==="
echo ""

# --- F1: pristine surface files pass ------------------------------------------
F1="$(mktemp -d)"
make_fixture "$F1"
expect_pass "$F1" "F1 pristine surface files exit 0"

# --- F2: sidebar nav href points at an undeclared route -----------------------
F2="$(mktemp -d)"
make_fixture "$F2"
sed -i 's|{Href: "/members", Label: "Members"|{Href: "/memberz", Label: "Members"|' "$F2/adminui/handler.go"
expect_fail "$F2" "F2 undeclared nav href fails" "/memberz"

# --- F3: templ href typo --------------------------------------------------------
F3="$(mktemp -d)"
make_fixture "$F3"
sed -i 's|d.BasePath + "/users/"|d.BasePath + "/usres/"|' "$F3/adminui/dashboard.templ"
expect_fail "$F3" "F3 typo'd templ href fails" "/usres/"

# --- F4: route-row source orphaned from the table assembly --------------------
F4="$(mktemp -d)"
make_fixture "$F4"
# Rename ONLY the definer; the h.superAdminTenantMemberRoutes() call site
# disappears, so the function's rows are served by nothing.
sed -i 's|func (h \*Handler) superAdminTenantMemberRoutes() \[\]routeSpec|func (h *Handler) orphanedTenantMemberRoutes() []routeSpec|' "$F4/adminui/routes.go"
expect_fail "$F4" "F4 orphaned row source fails" "orphanedTenantMemberRoutes"

# --- F5: raw mux registration outside the route table --------------------------
F5="$(mktemp -d)"
make_fixture "$F5"
sed -i 's|func (h \*Handler) registerRoutes(mux \*http.ServeMux) {|func (h *Handler) registerRoutes(mux *http.ServeMux) {\n\tmux.Handle("/admin/seedz", nil)|' "$F5/adminui/routes.go"
expect_fail "$F5" "F5 off-table mux registration fails" "/admin/seedz"

# --- F6: empty route file trips the false-green guard --------------------------
F6="$(mktemp -d)"
make_fixture "$F6"
: >"$F6/adminui/routes.go"
OUT="$(ADMINUI_SURFACE_ROOT="$F6" bash "$CHECKER" 2>&1)"
rc=$?
if [ "$rc" -ne 0 ]; then
  report 0 "F6 empty route table fails loudly (rc=$rc)"
else
  report 1 "F6 empty route table must fail (got rc=0)"
fi

echo ""
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ] || exit 1
echo "OK: all fixtures behaved"
