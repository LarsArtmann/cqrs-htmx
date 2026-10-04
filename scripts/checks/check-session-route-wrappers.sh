#!/usr/bin/env bash
# check-session-route-wrappers.sh — mechanize the ADR-0055 invariant:
# every /auth/* route whose handler reads the authenticated user from the
# request context (directly or through the withAuthContext/authContext/
# importExportContext funnels, or the sessionUserID/requireSessionUserTarget
# gates) must be registered behind h.withSession(...) inside the
# Register*Routes functions. A context-only handler behind a bare mount is a
# dead route (401 forever); an ungated ceremony is an account-takeover hole.
#
# Both failure classes regress the same way: someone drops the wrapper. This
# gate makes that a hard red. The scan is transitive (handler bodies are
# followed through h.* method calls to a fixpoint) so helper-funnelled
# context reads (handleTOTPSetupVerify -> handleTOTPCode -> withAuthContext)
# are caught too.
#
# Env: SESSION_ROUTE_WRAPPERS_ROOT — repo root override (fixture self-tests).

set -uo pipefail

ROOT="${SESSION_ROUTE_WRAPPERS_ROOT:-$(cd "$(dirname "$0")/../.." && pwd)}"
PKG="$ROOT/usermgmt"

route_files=(
  "$PKG/http.go"
  "$PKG/verification_totp_http.go"
  "$PKG/oauth2_http.go"
)

for f in "${route_files[@]}"; do
  if [ ! -f "$f" ]; then
    echo "FAIL: route file missing: $f"
    exit 1
  fi
done

# Context chokepoints: reaching ANY of these from a handler means the
# handler depends on the request context being session-populated.
markers=(
  'h.currentUser('
  'h.authContext('
  'h.withAuthContext('
  'h.importExportContext('
  'h.sessionUserID('
  'h.requireSessionUserTarget('
  'UserFromContext('
)

# body_of <method-name> — print the func body (package usermgmt, non-test).
body_of() {
  local name="$1" f
  for f in "$PKG"/*.go; do
    case "$f" in *_test.go) continue ;; esac
    awk -v fn="$name" '
      $0 ~ "^func \\(h \\*AuthHandler\\) " fn "\\(" { infn = 1 }
      infn { print; if ($0 ~ /^}/) { infn = 0 } }
    ' "$f"
  done
}

# reaches_marker <method-name> <depth> — transitive chokepoint scan.
reaches_marker() {
  local name="$1" depth="$2" m body callee
  [ "$depth" -le 0 ] && return 1
  body="$(body_of "$name")"
  [ -z "$body" ] && return 1
  for m in "${markers[@]}"; do
    if printf '%s' "$body" | grep -qF "$m"; then
      return 0
    fi
  done
  while IFS= read -r callee; do
    [ -n "$callee" ] || continue
    if reaches_marker "$callee" "$((depth - 1))"; then
      return 0
    fi
  done < <(printf '%s\n' "$body" | grep -oE 'h\.[A-Za-z][A-Za-z0-9]*\(' | sed 's/^h\.//;s/($//' | sort -u)
  return 1
}

total=0
wrapped=0
bare_public=0
fail=0

check_registration() { # <route-label> <expr>
  local route="$1" expr="$2" handler
  total=$((total + 1))
  if printf '%s' "$expr" | grep -qF 'h.withSession('; then
    wrapped=$((wrapped + 1))
    return
  fi
  handler="$(printf '%s' "$expr" | grep -oE 'h\.[A-Za-z][A-Za-z0-9]*' | head -1 | sed 's/^h\.//')"
  if [ -z "$handler" ]; then
    echo "FAIL: $route — bare registration with no h.* handler (inline func?): classify it explicitly"
    fail=1
    return
  fi
  if reaches_marker "$handler" 5; then
    echo "FAIL: $route — handler $handler reads the session context but is NOT wrapped in h.withSession(...): dead route on a bare mount (ADR-0055)"
    fail=1
    return
  fi
  bare_public=$((bare_public + 1))
}

for f in "${route_files[@]}"; do
  while IFS= read -r line; do
    [ -n "$line" ] || continue
    route="$(printf '%s' "$line" | grep -oE '"[A-Z]+ /[^"]*"' | tr -d '"')"
    check_registration "$route" "$line"
  done < <(grep -E 'mux\.HandleFunc\("' "$f")
done

echo "registrations: $total  wrapped: $wrapped  bare-public: $bare_public"

# False-green guards (gotcha: any sweep must fail loudly on zero candidates).
if [ "$total" -lt 10 ]; then
  echo "FAIL: expected at least 10 route registrations, found $total — scanner broken?"
  exit 1
fi
if [ "$wrapped" -lt 8 ]; then
  echo "FAIL: expected at least 8 wrapped session routes, found $wrapped — scanner broken?"
  exit 1
fi
if [ "$bare_public" -lt 5 ]; then
  echo "FAIL: expected at least 5 bare public routes, found $bare_public — scanner broken?"
  exit 1
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "OK: every context-dependent /auth/* route is wrapped (ADR-0055 invariant holds)"
