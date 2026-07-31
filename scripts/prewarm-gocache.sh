#!/usr/bin/env bash
# Pre-warm the Go build cache (GOCACHE) to prevent concurrent compilation races.
#
# ROOT CAUSE:
#   encoding/json/v2 and other GOEXPERIMENT=jsonv2 stdlib packages are NOT
#   pre-compiled in the Go installation (they are experimental). When multiple
#   buildflow module-fanout processes (govalid, golangci-lint, go-auto-upgrade)
#   compile them concurrently to the same GOCACHE, they race, producing
#   cascading "markers: failed prerequisites" / "could not import encoding/json/v2"
#   errors that disappear on re-run. This is the #1 flakiest buildflow step
#   (20% per-attempt failure rate over 1867 runs).
#
# FIX:
#   Compile all workspace modules once (serialized) BEFORE concurrent analysis
#   tools start. This converts all subsequent go/packages.Load calls from WRITE
#   (compile) to READ (cache hit). Go's build cache is safe for concurrent READS;
#   the race only occurs on concurrent WRITES.
#
# PERFORMANCE:
#   Warm cache: <1s total (all 18 modules hit cache, no compilation needed).
#   Cold cache: ~5s (one-time cost, amortized across all subsequent tool runs).

set -uo pipefail
export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"

# All workspace modules from go.work (kept in sync manually).
modules=(
  .
  identity-model
  usermgmt
  usermgmt/totp
  usermgmt/webauthn
  usermgmt/oauth2
  adminui
  loginpage
  dashboardui
  integration_test
  e2e/server
  examples/datastar-demo
  examples/admin-demo
  examples/basic
  examples/dashboard-demo
  examples/middleware-demo
  examples/observability-demo
  examples/catalog-demo
)

for mod in "${modules[@]}"; do
  (cd "$mod" && go build ./... 2>/dev/null) || {
    echo "prewarm-gocache: build failed in $mod (buildflow will report the real error)" >&2
  }
done
