# Benchmarks

Artifacts for the setup spike benchmark (`BenchmarkSpikeBaselineVsAppkit`,
`setup/run_appkit_test.go`).

## Files

| File | Role | Comparable across machines? |
| --- | --- | --- |
| `setup-baseline.raw.txt` | **The gate baseline.** Pinned by `nix run .#bench-spike -- --save-baseline`. Env-guarded (goos/goarch/pkg/cpu): the gate refuses to compare across machines. | NO — per-machine |
| `setup-baseline-2026-08-17.txt` | Human markdown record of the original pinning session. | NO — Xeon numbers, historical only |

## Sub-benchmarks

- `baseline-httputil` — ping handler behind `RunHandler` (httputil server).
- `appkit-service` — same handler behind `RunWithAppkit` (appkit spike, ADR-001).
- `json-roundtrip` — no HTTP: encode+decode of an envelope-shaped payload in
  isolation, so stack deltas can be attributed to middleware rather than
  codec work (~1µs / 6 allocs vs ~20-25µs for the stacks).

## Workflow

```sh
# Gate (default 5×2s, 10% median-regression threshold):
nix run .#bench-spike

# Tighten for a specific investigation:
BENCH_SPIKE_THRESHOLD=5 nix run .#bench-spike

# Per-bench override (name uppercased, non-alnum -> _; suffix match works):
BENCH_THRESHOLD_JSON_ROUNDTRIP=15 nix run .#bench-spike

# Re-pin (REQUIRED in the same change as any bench rename/handler change):
nix run .#bench-spike -- --save-baseline
git add docs/benchmarks/setup-baseline.raw.txt
```

## Load guard + re-pin policy

The app refuses to measure when the 1-minute load average is at or above
`BENCH_MAX_LOAD` (default: cores/4; `0` disables). Two sessions in a row
lost bench runs to concurrent builds (~8.7 load produced ±25% garbage)
before anyone checked `uptime` — the guard moves that check into the app.

Re-pin policy (write it down instead of re-deciding it ad hoc):

- Re-pin when this machine's **quiet-state medians drift >10%** from the
  pinned baseline, or after any machine/hardware/OS change.
- `--save-baseline` records a `load1:` context header — never pin under
  contention (the guard enforces this).
- A stale-but-conservative baseline (pinned slower than current speed)
  halves regression sensitivity; prefer re-pinning once per machine-speed
  change over hoarding headroom.

## Why 10%?

Measured run-to-run median noise on the pinned machine (AMD Ryzen AI MAX+ 395):
~9% on the ~1µs `json-roundtrip` sub-bench and ~5% on the ~20µs HTTP
sub-benches. A threshold below machine noise fails spuriously, which trains
people to ignore the gate; real regressions (the 2.8× per-request-logging bug
class) clear 10% by an order of magnitude. Sub-2µs benches flap near any
fixed percentage; give one its own budget via `BENCH_THRESHOLD_<NAME>`
(NAME = bench name uppercased, non-alnum → `_`; a distinctive suffix such
as `JSON_ROUNDTRIP` matches) instead of loosening the global threshold.

## CI status (decision note)

`bench-spike` is deliberately NOT a CI gate: it is meaningful only on the
machine the raw baseline was pinned on, and CI runners are a different
population entirely (noisy shared cores; the env guard would exit 2 anyway).
Locally it is part of the change checklist for anything touching the
request path. If a future runner class gets a pinned baseline of its own,
drop its raw file here and point a separate app at it — do not reuse
another machine's file.
