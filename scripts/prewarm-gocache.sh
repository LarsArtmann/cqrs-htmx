#!/usr/bin/env bash
# Pre-warm the Go build cache (GOCACHE) to prevent concurrent compilation races.
#
# ROOT CAUSE:
#   encoding/json/v2 and other GOEXPERIMENT=jsonv2 stdlib packages are NOT
#   pre-compiled in the Go installation (they are experimental). When multiple
#   buildflow module-fanout processes (govalid, golangci-lint, go-auto-upgrade)
#   compile them concurrently to the same GOCACHE, they race, producing
#   cascading "markers: failed prerequisites" / "could not import encoding/json/v2"
#   errors that disappear on re-run. Historical failure rate: 16.6% in pre-commit
#   mode, 12.0% in full mode (buildflow SQLite timing DB, 6,072 records).
#
#   Why pre-commit is worse: buildflow skips `workspace-build-verify` (which runs
#   `go build ./...`) in pre-commit mode AND in check mode (phantom-skip). So the
#   GOCACHE is never pre-warmed before concurrent analysis tools start.
#
# FIX:
#   Compile all workspace modules once (serialized) BEFORE concurrent analysis
#   tools start. This converts all subsequent go/packages.Load calls from WRITE
#   (compile) to READ (cache hit). Go's build cache is safe for concurrent READS;
#   the race only occurs on concurrent WRITES.
#
# PERFORMANCE:
#   Warm cache: ~3s total (all modules hit cache, no compilation needed).
#   Cold cache: ~5-10s (one-time cost, amortized across all subsequent tool runs).

set -uo pipefail
export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"

# Auto-discover workspace modules from go.work (no manual list to maintain).
mapfile -t modules < <(go work edit -json 2>/dev/null | grep DiskPath | sed 's/.*: *"*//;s/".*//')

if [ ${#modules[@]} -eq 0 ]; then
  echo "prewarm-gocache: WARNING: no workspace modules found in go.work" >&2
  exit 0
fi

for mod in "${modules[@]}"; do
  (cd "$mod" && go build ./... 2>/dev/null) || {
    echo "prewarm-gocache: build failed in $mod (buildflow will report the real error)" >&2
  }
done
