#!/usr/bin/env bash
# e2e-fast-slice.sh — the ~3-minute chromium slice REQUIRED for render-touching
# family bumps (templ-components etc.), the 2026-11-21 §g.1 process fix.
#
# The full battery (127 tests, ~2.5 min local / ~6 min CI) is the shipped-state
# gate; this slice is the FAST proof a render-touching bump needs BEFORE any
# push: the render golden (loginpage), the interactive admin surfaces with
# their pagination/toast contracts (admin-behavior), and the console guards
# that catch CSP/inline-style breaks on the setup bundle (setup-bundle).
# The v1.27.2 accent sweep shipped without this slice once and cost the next
# session a full triage pass — the slice is the cheap insurance.
#
# Usage:
#   nix run .#e2e-fast-slice          # the slice (chromium, 3 specs)
#   nix run .#e2e-fast-slice -- --headed   (extra args forward to playwright)
#
# Exit: playwright's rc (0 green / 1 failures).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

bash "$REPO_ROOT/scripts/tools/e2e-kill-strays.sh" >/dev/null 2>&1 || true

cd "$REPO_ROOT" || exit 1
echo "e2e-fast-slice: loginpage-golden + admin-behavior + setup-bundle (chromium)"
exec nix run .#e2e-firefox -- --project=chromium \
  loginpage-golden admin-behavior setup-bundle "$@"
