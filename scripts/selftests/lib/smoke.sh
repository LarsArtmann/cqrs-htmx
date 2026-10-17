#!/usr/bin/env bash
# smoke.sh — reusable server smoke-test harness (plan M17, 2026-10-17).
#
# The pattern proven by the 2026-10-17 README session (extract → build → run
# → probe, 4 iterations): spawn a server on a UNIQUE high port, READ THE
# SERVER LOG before interpreting client output (a client error with an empty
# server log means the server never started), probe with a Go HTTP client
# (curl is banned in this fleet), and always clean up.
#
# Usage (source this file):
#   source scripts/selftests/lib/smoke.sh
#   smoke_pick_port                       # echoes an unused high port
#   smoke_start <logfile> <cmd...>        # spawns cmd, sets $SMOKE_PID/$SMOKE_LOG
#   smoke_wait_http <port> [path] [secs]  # blocks until an HTTP probe answers
#   smoke_http <method> <url> [body]      # prints "STATUS <n>" (Go client)
#   smoke_stop                            # terminates + waits (idempotent)
#
# Ports 8080 (SigNoz) and 8099 (InboxClean) are permanently occupied on this
# fleet — smoke_pick_port starts above 24100 and probes the port is free.
# The HTTP prober is stdlib-only Go, compiled once per test run into a temp
# binary (GOWORK=off; no module deps — works offline).
set -uo pipefail

SMOKE_PORT_BASE="${SMOKE_PORT_BASE:-24128}"
SMOKE_PID=""
SMOKE_LOG=""
_SMOKE_TMP=""
_SMOKE_PROBE=""

smoke__cleanup() {
  smoke_stop
  [ -n "$_SMOKE_TMP" ] && rm -rf "$_SMOKE_TMP"
}
trap smoke__cleanup EXIT

smoke_pick_port() {
  local port=$((SMOKE_PORT_BASE + (RANDOM % 2000)))
  local tries=0
  while [ "$tries" -lt 25 ]; do
    if ! (exec 3<>"/dev/tcp/127.0.0.1/$port") 2>/dev/null; then
      echo "$port"
      return 0
    fi
    port=$((port + 1))
    tries=$((tries + 1))
  done
  echo "smoke: no free port found above $SMOKE_PORT_BASE" >&2
  return 1
}

smoke_start() { # smoke_start <logfile> <cmd...>
  SMOKE_LOG="$1"
  shift
  : >"$SMOKE_LOG"
  "$@" >"$SMOKE_LOG" 2>&1 &
  SMOKE_PID=$!
}

smoke_stop() {
  if [ -n "$SMOKE_PID" ] && kill -0 "$SMOKE_PID" 2>/dev/null; then
    kill "$SMOKE_PID" 2>/dev/null || true
    local waited=0
    while kill -0 "$SMOKE_PID" 2>/dev/null && [ "$waited" -lt 50 ]; do
      sleep 0.1
      waited=$((waited + 1))
    done
    kill -9 "$SMOKE_PID" 2>/dev/null || true
    wait "$SMOKE_PID" 2>/dev/null || true
  fi
  SMOKE_PID=""
}

smoke__probe_bin() { # compiles the stdlib prober once; echoes its path
  if [ -n "$_SMOKE_PROBE" ] && [ -x "$_SMOKE_PROBE" ]; then
    printf '%s\n' "$_SMOKE_PROBE"
    return 0
  fi
  _SMOKE_TMP="$(mktemp -d /tmp/smoke-harness-XXXXXX)"
  printf 'module smokeprobe\n\ngo 1.21\n' >"$_SMOKE_TMP/go.mod"
  cat >"$_SMOKE_TMP/main.go" <<'EOF'
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: probe <method> <url> [body]")
		os.Exit(2)
	}
	method, url := os.Args[1], os.Args[2]
	body := ""
	if len(os.Args) > 3 {
		body = os.Args[3]
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(&http.Request{
		Method: method,
		URL:    mustParse(url),
		Body:   io.NopCloser(strings.NewReader(body)),
		Header: http.Header{"Content-Type": []string{"application/json"}},
	})
	if err != nil {
		fmt.Printf("ERROR %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	fmt.Printf("STATUS %d\n", resp.StatusCode)
}

func mustParse(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
EOF
  _dir="$_SMOKE_TMP"
  if ! (cd "$_dir" && GOWORK=off GOTOOLCHAIN=local GOFLAGS=-mod=mod go build -o smokeprobe .); then
    echo "smoke: probe compile failed" >&2
    return 1
  fi
  _SMOKE_PROBE="$_dir/smokeprobe"
  printf '%s\n' "$_SMOKE_PROBE"
}

smoke_http() { # smoke_http <method> <url> [body] -> "STATUS <n>" | "ERROR ..."
  local bin
  bin="$(smoke__probe_bin)" || return 1
  "$bin" "$@"
}

smoke_wait_http() { # smoke_wait_http <port> [path] [secs]
  local port="$1" path="${2:-/}" secs="${3:-15}"
  local waited=0
  while [ "$waited" -lt $((secs * 10)) ]; do
    if smoke_http GET "http://127.0.0.1:$port$path" 2>/dev/null | grep -q '^STATUS'; then
      return 0
    fi
    # Log-first rule: if the server DIED, fail now with its log — do not
    # keep probing a corpse.
    if [ -n "$SMOKE_PID" ] && ! kill -0 "$SMOKE_PID" 2>/dev/null; then
      echo "smoke: server exited before answering; log:" >&2
      tail -5 "$SMOKE_LOG" >&2 2>/dev/null || true
      return 1
    fi
    sleep 0.1
    waited=$((waited + 1))
  done
  echo "smoke: no HTTP answer on :$port within ${secs}s; server log tail:" >&2
  tail -5 "$SMOKE_LOG" >&2 2>/dev/null || true
  return 1
}
