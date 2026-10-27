#!/usr/bin/env bash
# check-readme-quickstart.sh — prove the README's quick start actually compiles.
#
# The single biggest documented failure class of the old README was silently
# lying about the code (the architecture tree claimed root implements CSRF;
# it is a deprecated httputil re-export — 2026-10-17 rewrite session). This
# gate mechanizes the honest claim: extract the first ```go fenced block
# under `## Quick start`, drop it into a throwaway module that requires the
# REAL root module via a path replace, and build it hermetically (GOWORK=off,
# published deps from the proxy/module cache). A README that lies about its
# quick start is now a red gate, not a review-time discovery.
#
# Guards against the silent-false-green classes:
#   - candidate-count print (gotcha 2): a README whose Quick start section
#     holds zero go fences FAILS loudly instead of vacuously passing;
#   - unique mktemp dir + trap cleanup + timeout (no /tmp collisions between
#     concurrent sessions — gotcha 3);
#   - rc captured outside pipes.
#
# Usage: bash scripts/checks/check-readme-quickstart.sh [readme-path]
# Env:   README_QUICKSTART_ROOT  override the module the fence is compiled
#                                against (fixture self-test; default: repo root).
#        README_FENCE_TIMEOUT    go build timeout in seconds (default 600).
# Exit:  0 = fence extracted and compiled; 1 = any failure; 2 = usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

README="${1:-$REPO_ROOT/README.md}"
LIB_ROOT="${README_QUICKSTART_ROOT:-$REPO_ROOT}"
TIMEOUT_SECS="${README_FENCE_TIMEOUT:-600}"

if [ ! -f "$README" ]; then
  echo "✗ readme-quickstart: $README not found"
  exit 1
fi
if [ ! -f "$LIB_ROOT/go.mod" ]; then
  echo "✗ readme-quickstart: $LIB_ROOT has no go.mod (README_QUICKSTART_ROOT?)"
  exit 1
fi

# Extract the section between `## Quick start` and the next heading of the
# same level, then the first ```go fence inside it. awk is deliberate: no
# toolchain dependency, and the anchor rules stay inspectable.
fence="$(awk '
  /^## Quick start/ { in_section = 1; next }
  in_section && /^## / { in_section = 0 }
  in_section && /^```go$/ { in_fence = 1; next }
  in_section && in_fence && /^```$/ { in_fence = 0; found = 1; exit }
  in_section && in_fence { print }
' "$README")"

if [ -z "$fence" ]; then
  echo "✗ readme-quickstart: no go fence found under '## Quick start' in $README"
  echo "  (candidate count: 0 — the gate refuses to pass vacuously; either the"
  echo "   section heading or the fenced example was lost)"
  exit 1
fi
lines="$(printf '%s\n' "$fence" | wc -l | tr -d ' ')"
echo "· readme-quickstart: extracted 1 go fence (${lines} lines) from $README"

TMP=$(mktemp -d /tmp/readme-quickstart-XXXXXX)
trap 'rm -rf "$TMP"' EXIT

# Toolchain floor: the fence compiles against the REAL root module, whose
# go.mod demands the fleet floor (e.g. 1.27.2). Ambient shells can carry an
# older binary with GOTOOLCHAIN=local (the go-cache-env class) — raise
# deterministically from the target module so the consumer-view build never
# dies on "requires go >= X (running older)". The toolchain resolves from
# the local module cache; never downgrades a newer ambient setting.
floor="$(awk '$1 == "toolchain" { print $2; exit }' "$LIB_ROOT/go.mod")"
floor="${floor#go}"
floor="${floor:-$(awk '$1 == "go" { print $2; exit }' "$LIB_ROOT/go.mod")}"
ambient_go="$(go version 2>/dev/null | awk '{ print $3 }' | sed 's/^go//')"
if [ -n "$floor" ] && [ -n "$ambient_go" ] &&
  { [ -z "${GOTOOLCHAIN:-}" ] || [ "${GOTOOLCHAIN:-}" = "local" ]; }; then
  older_of="$(printf '%s\n%s\n' "$floor" "$ambient_go" | sort -V | head -1)"
  if [ "$older_of" = "$ambient_go" ] && [ "$floor" != "$ambient_go" ]; then
    export GOTOOLCHAIN="go$floor"
  fi
fi

printf 'module example.com/readmequickstart\n\ngo %s\n' "$floor" >"$TMP/go.mod"
printf '%s\n' "$fence" >"$TMP/main.go"

cat >>"$TMP/go.mod" <<GOMOD
require github.com/larsartmann/cqrs-htmx/v4 v4.0.0
replace github.com/larsartmann/cqrs-htmx/v4 => $LIB_ROOT
GOMOD

# Hermetic: the workspace (and its local sibling replaces) must not leak in —
# the fence must compile the way a consumer's module would.
export GOWORK=off
export GOEXPERIMENT="${GOEXPERIMENT:-jsonv2}"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/go-cache-env.sh" >/dev/null 2>&1 || true

build_log="$TMP/build.log"
{
  (cd "$TMP" && timeout "$TIMEOUT_SECS" go mod tidy) &&
    (cd "$TMP" && timeout "$TIMEOUT_SECS" go build .)
} >"$build_log" 2>&1
rc=$?

if [ "$rc" -ne 0 ]; then
  echo "✗ readme-quickstart: extracted quick start FAILED to compile (rc=$rc)"
  echo "--- build log (first 40 lines) ---"
  head -40 "$build_log"
  echo "--- fix the README example or the code it demonstrates ---"
  exit 1
fi

echo "✓ readme-quickstart: the README quick start compiles (GOWORK=off, real module graph)"
exit 0
