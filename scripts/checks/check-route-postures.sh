#!/usr/bin/env bash
# check-route-postures.sh — the L4 route-posture gate (docs/PERMISSIONS.md).
#
# Every UI module's route manifest is pinned by its own golden test
# (adminui TestRoutes_*, dashboardui TestRoutes_*/manifest tests, loginpage
# TestRoutes_*, setup TestRoutePostures_*, root TestPosture*). This gate runs
# them all (workspace mode — see the note in the loop) and REFUSES the
# empty-match trap: a module whose test pattern matches nothing fails loudly
# — a green that verified nothing is the gotcha-2/40 class.
#
# The manifests themselves are generated from the same route tables the muxes
# register from, so "a route added outside the table" cannot happen in
# adminui/dashboardui; setup's aggregate honors its Disable*/path config.
# The golden pins make every manifest change a reviewable diff.
#
# Env: ROUTE_POSTURE_ROOT    override the repo root (fixture self-test)
#      ROUTE_POSTURE_MODULES override the module list (fixture self-test)
#      ROUTE_POSTURE_GO      override the go binary (fixture self-test)
# Exit: 0 all manifest pins green · 1 a pin failed or ran nothing · 2 setup.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${ROUTE_POSTURE_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"
MODULES="${ROUTE_POSTURE_MODULES:-. adminui dashboardui loginpage setup}"
GO_BIN="${ROUTE_POSTURE_GO:-go}"
PATTERN='TestRoutes_|TestRoutePostures|TestPosture'

if [ ! -d "$REPO_ROOT" ]; then
  echo "✗ route-postures: $REPO_ROOT is not a directory" >&2
  exit 2
fi

red=0
for module in $MODULES; do
  dir="$REPO_ROOT/$module"
  if [ ! -d "$dir" ]; then
    echo "✗ route-postures: module dir $dir missing" >&2
    red=1
    continue
  fi

  # Workspace mode ON PURPOSE: the manifest pins check SOURCE truth, and the
  # submodules' go.mod still points at the last published root tag until the
  # next family train — GOWORK=off would compile against the stale tag and
  # fail on symbols that exist only in the workspace.
  out="$(cd "$dir" && GOEXPERIMENT=jsonv2 "$GO_BIN" test -run "$PATTERN" -count=1 -v ./... 2>&1)"
  rc=$?

  passed="$(printf '%s\n' "$out" | grep -c '^--- PASS:' || true)"
  if [ "$rc" -ne 0 ]; then
    echo "✗ route-postures: $module manifest pins FAILED:" >&2
    detail="$(printf '%s\n' "$out" | grep -E '^--- FAIL:|^=== RUN|^    [a-z_]+\.go:[0-9]+:' | head -15)"
    if [ -n "$detail" ]; then
      printf '%s\n' "$detail" >&2
    else
      # The detail grep matched nothing: the go test command itself refused
      # (toolchain floor, GOWORK/env, download failure, missing module) and
      # its stderr is the ONLY diagnostic — never swallow it (A5 class).
      echo "✗ route-postures: detail grep matched nothing; raw go test output head:" >&2
      printf '%s\n' "$out" | head -10 >&2
    fi
    red=1
    continue
  fi
  if [ "$passed" -eq 0 ]; then
    echo "✗ route-postures: $module ran ZERO manifest tests (pattern '$PATTERN' matched nothing) — the empty-match trap; a manifest pin must exist" >&2
    red=1
    continue
  fi

  echo "✓ route-postures: $module ($passed manifest test(s) green)"
done

if [ "$red" -ne 0 ]; then
  echo "✗ route-postures: gate red — fix the manifest or update the golden in the SAME commit as the route change" >&2
  exit 1
fi

echo "✓ route-postures: every module's route manifest is pinned and green"
exit 0
