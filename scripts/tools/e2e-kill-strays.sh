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
# Usage: bash scripts/tools/e2e-kill-strays.sh [--force]   (from anywhere in the repo)
# Exit:  0 all ports clear (after killing); 1 a port is STILL occupied;
#        3 a playwright battery appears LIVE (etiquette refusal, see below)
#
# Session etiquette (f#41, 2026-11-24): the battery ports are SHARED between
# concurrent sessions on this worktree — killing the servers under another
# session's live battery corrupts its run mid-flight (the gotcha-47 class,
# hit twice during the 2026-11-24 dual-wave). When a `playwright test`
# process is running, this script REFUSES by default; coordinate with the
# other session first, then re-run with --force (or E2E_KILL_STRAYS_FORCE=1).
# E2E_KILL_STRAYS_FAKE_LIVE=1 simulates a live battery for the offline
# fixture self-test.

set -uo pipefail

PORTS="18923 18930 18931 18932 18933 18940 18941"

if [ "${E2E_KILL_STRAYS_FORCE:-0}" != "1" ] && [ "${1:-}" != "--force" ]; then
  live="$(pgrep -f 'playwright test' 2>/dev/null | grep -vx "$$" || true)"
  if [ -n "$live" ] || [ "${E2E_KILL_STRAYS_FAKE_LIVE:-0}" = "1" ]; then
    echo "✗ e2e-kill-strays: a playwright battery appears LIVE (pid(s): ${live:-simulated})" >&2
    echo "  killing the battery servers would corrupt its run (shared-worktree etiquette)" >&2
    echo "  coordinate with the other session first, then re-run with --force" >&2
    exit 3
  fi
fi

echo "==> e2e-kill-strays: clearing battery ports: $PORTS"

# mvdan/sh has no kill builtin and /bin/kill does not exist on this host —
# pkill patterns are the portable shape here.
#
# NEVER kill by the bare name "server": it matches every process of that
# name the user can signal (another worktree's mid-battery server, a dev
# server), and the sweep below already targets exactly what holds a battery
# port. A copied bash named server on no battery port is the stand-in the
# self-test uses (scripts/selftests/test-e2e-kill-strays.sh, case 5).
pkill -f '/tmp/e2e-server' 2>/dev/null || true
pkill -f 'cqrs-htmx-e2e-server' 2>/dev/null || true

# Anything holding a battery port: target via ss + the pid (skip the ss
# header line and the script itself).
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
