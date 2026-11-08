#!/usr/bin/env bash
# e2e-kill-strays.sh — the stale-server kill ritual as one command.
#
# WHY (2026-11-08): playwright's `reuseExistingServer` (on locally) silently
# reuses a leaked server, so a spec run can test YESTERDAY'S binary — the
# class that hid regressions for a whole session before the ritual was
# written down in e2e/README. This mechanizes it: kill anything listening on
# the battery ports, then prove they are clear (fail loudly if not).
#
# Battery ports (playwright.config.ts + ad-hoc probe ports):
#   18923 e2e/server · 18930 admin-demo · 18931 datastar-demo
#   18932 basic · 18933 setup-demo · 18940/18941 ad-hoc probes
#
# Usage: bash scripts/tools/e2e-kill-strays.sh   (from anywhere in the repo)
# Exit:  0 all ports clear (after killing); 1 a port is STILL occupied.

set -uo pipefail

PORTS="18923 18930 18931 18932 18933 18940 18941"

echo "==> e2e-kill-strays: clearing battery ports: $PORTS"

# mvdan/sh has no kill builtin and /bin/kill does not exist on this host —
# pkill patterns are the portable shape here.
pkill -x server 2>/dev/null || true
pkill -f '/tmp/e2e-server' 2>/dev/null || true
pkill -f 'cqrs-htmx-e2e-server' 2>/dev/null || true

# Anything still holding a battery port after the name-based kills: target
# via ss + pkill on the pid (skip the ss header line and the script itself).
sleep 0.5
STILL=""
for port in $PORTS; do
  pid="$(ss -tlnpH "sport = :$port" 2>/dev/null | grep -o 'pid=[0-9]*' | head -1 | cut -d= -f2 || true)"
  if [ -n "$pid" ]; then
    pkill -TERM -F "/proc/$pid/stat" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
    STILL="$STILL $port(pid $pid)"
  fi
done

if [ -n "$STILL" ]; then
  sleep 1
  LEFT=""
  for port in $PORTS; do
    ss -tlnH "sport = :$port" 2>/dev/null | grep -q . && LEFT="$LEFT $port"
  done
  if [ -n "$LEFT" ]; then
    echo "✗ e2e-kill-strays: ports still occupied after TERM:$LEFT" >&2
    echo "  refusing to reuse a possibly-stale server; clear them by hand" >&2
    exit 1
  fi
fi

echo "✓ e2e-kill-strays: all battery ports clear"
