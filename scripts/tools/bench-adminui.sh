#!/usr/bin/env bash
# bench-adminui.sh — the adminui 200-user page render benchmark as one command
# (extracted from the bench-adminui flake app so check-modules can run it as
# an advisory record; the app delegates here).
#
# ADVISORY by design: the adminui bench is a directional instrument, not a
# release gate. Load guard identical to bench-spike (BENCH_MAX_LOAD, default
# cores/4; 0 disables) — concurrent load makes ns/op garbage; a refusal is a
# SKIP (exit 0 with a loud marker), never a composite failure. A bench that
# genuinely cannot run (compile error) still exits 1: a broken instrument is
# a real defect. Env: BENCH_COUNT, BENCHTIME, BENCH_MAX_LOAD.
#
# Usage: bench-adminui.sh [--save-baseline [path]]
#   --save-baseline re-pins docs/benchmarks/adminui-baseline.raw.txt with a
#   load1 context header; re-pin ONLY on an idle machine (bench canon).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT" || exit 1

baseline="docs/benchmarks/adminui-baseline.raw.txt"
count="${BENCH_COUNT:-5}"
benchtime="${BENCHTIME:-2s}"

# shellcheck disable=SC1091
source scripts/lib/go-cache-env.sh

cores="$(nproc)"
max_load="${BENCH_MAX_LOAD:-$((cores / 4))}"
load1="$(cut -d' ' -f1 /proc/loadavg)"
if [ "$max_load" != "0" ]; then
  ok="$(awk -v l="$load1" -v m="$max_load" 'BEGIN{print (l < m) ? 1 : 0}')"
  if [ "$ok" != "1" ]; then
    echo "bench-adminui: SKIP — 1-min load $load1 >= BENCH_MAX_LOAD $max_load (cores=$cores); refusing to measure (advisory)." >&2
    echo "  Wait for a quiet machine, or override: BENCH_MAX_LOAD=<n> (0 disables the guard)." >&2
    exit 0
  fi
  echo "bench-adminui: load $load1 (max $max_load, cores $cores) — OK"
fi

bench_args=(
  -run xxx
  -bench "^BenchmarkUsersPageRender$"
  -benchtime="$benchtime"
  -benchmem
  -count="$count"
  -timeout=120s
)

if [ "${1:-}" = "--save-baseline" ]; then
  out="${2:-$baseline}"
  echo "== bench-adminui: pinning baseline to $out ($count x $benchtime) =="
  { echo "load1: $load1"; (cd adminui && go test "${bench_args[@]}" .); } | tee "$out"
  echo "Saved $out — commit it in the same change as any bench-path edit."
  exit 0
fi

current="$(mktemp /tmp/bench-adminui-current.XXXXXX)"
trap 'rm -f "$current"' EXIT

echo "== bench-adminui: $count x $benchtime (ADVISORY) =="
if ! (cd adminui && go test "${bench_args[@]}" .) | tee "$current"; then
  echo "bench-adminui: FAIL — the benchmark could not run (compile or test error above)." >&2
  exit 1
fi

if [ -f "$baseline" ]; then
  echo
  echo "== benchstat: baseline -> current =="
  if command -v benchstat >/dev/null 2>&1; then
    benchstat "$baseline" "$current" || true
  else
    echo "bench-adminui: benchstat not on PATH — raw current output above is the record." >&2
  fi
else
  echo "bench-adminui: no baseline at $baseline (advisory run only)."
  echo "  Pin one on an idle machine: nix run .#bench-adminui -- --save-baseline" >&2
fi
echo "bench-adminui: OK (advisory — no verdict gate)"
