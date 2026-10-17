#!/usr/bin/env bash
# test-smoke-harness.sh — self-test for scripts/selftests/lib/smoke.sh.
#
# Spawns a stdlib-only Go server on a unique high port, proves the full
# harness contract: readiness wait, GET + POST probes, the log-first dead
# server rule (a failing spawn surfaces its log instead of a probe timeout),
# and idempotent cleanup. Offline (GOPROXY-safe: stdlib only, GOWORK=off).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/smoke.sh"

TMP=$(mktemp -d /tmp/test-smoke-harness-XXXXXX)
failures=0

ok() { echo "ok   $1"; }
bad() {
  echo "FAIL $1"
  failures=$((failures + 1))
}

cat >"$TMP/server.go" <<'EOF'
package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--fail" {
		fmt.Fprintln(os.Stderr, "deliberate startup failure")
		os.Exit(3)
	}
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s done", r.Method)
	})
	_ = http.ListenAndServe(":"+os.Getenv("SMOKE_PORT"), nil)
}
EOF
printf 'module smokeserver\n\ngo 1.21\n' >"$TMP/go.mod"
if ! (cd "$TMP" && GOWORK=off GOTOOLCHAIN=local go build -o server .); then
  echo "FAIL prerequisite: fixture server did not compile"
  exit 1
fi

# --- 1: pick a free port ---
port="$(smoke_pick_port)" || {
  bad "pick_port"
  exit 1
}
ok "pick_port ($port)"

# --- 2: spawn + readiness ---
export SMOKE_PORT="$port"
smoke_start "$TMP/server.log" "$TMP/server"
if smoke_wait_http "$port" /healthz 20; then
  ok "spawn + wait_http readiness"
else
  bad "spawn + wait_http readiness"
fi

# --- 3: GET + POST probes ---
get="$(smoke_http GET "http://127.0.0.1:$port/healthz")"
case "$get" in
"STATUS 200") ok "GET probe ($get)" ;;
*) bad "GET probe (got: $get)" ;;
esac
post="$(smoke_http POST "http://127.0.0.1:$port/echo" '{"x":1}')"
case "$post" in
"STATUS 200") ok "POST probe ($post)" ;;
*) bad "POST probe (got: $post)" ;;
esac

# --- 4: log-first dead-server rule ---
smoke_stop
smoke_start "$TMP/fail.log" "$TMP/server" --fail
if smoke_wait_http "$port" /healthz 5 2>"$TMP/dead.err"; then
  bad "dead server must fail wait_http"
else
  if grep -q "deliberate startup failure" "$TMP/dead.err"; then
    ok "dead server fails fast WITH its log (log-first rule)"
  else
    bad "dead-server failure did not surface the server log"
  fi
fi

# --- 5: idempotent cleanup ---
smoke_stop
smoke_stop
ok "cleanup idempotent"

if [ "$failures" -ne 0 ]; then
  echo "FAIL: $failures case(s) failed (of 5)"
  exit 1
fi
echo "PASS: 5/5 cases"
