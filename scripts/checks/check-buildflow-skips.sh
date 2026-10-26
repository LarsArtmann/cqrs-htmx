#!/usr/bin/env bash
# check-buildflow-skips.sh — canary (c): the documented BuildFlow skip_steps
# must stay listed in .buildflow.yml.
#
# Gotcha 35: a daemon-captured .buildflow.yml deletion resurrected
# documented-destructive steps and the next --fix run re-damaged the tree
# within the hour. Every skip below is load-bearing (reasons in the
# .buildflow.yml comments); this gate fails the moment one disappears.
# A NEW skip is reported as a warning — add it here with a justification.
#
# Usage: bash scripts/checks/check-buildflow-skips.sh
# Env: BUILDFLOW_SKIPS_FILE  override the config (fixture self-test).
# Exit: 0 all documented skips present, 1 a documented skip is missing, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
BUILDFLOW="${BUILDFLOW_SKIPS_FILE:-$REPO_ROOT/.buildflow.yml}"

if [ ! -f "$BUILDFLOW" ]; then
  echo "✗ buildflow-skips: $BUILDFLOW not found" >&2
  exit 1
fi

# The documented skips. Order does not matter; each entry mirrors a
# load-bearing comment block in .buildflow.yml.
expected=(
  templ-generate   # bundled templ binary writes directory-prefixed FileName fields
  tailwind-build   # consumer-compiled CSS bundles; the stale-bundle class
  jest-test        # Go library, no JS test framework
  vitest-test      # Go library, no JS test framework
  pnpm-audit       # no JS lockfile
  deadnix          # not installed
  dprint-format    # dprint.json removed 2026-10-18; tombstone vs unscoped defaults
)

# skip_steps entries: lines "- name" after "skip_steps:" until the next
# top-level key.
actual=$(awk '
  /^skip_steps:/ { in_block = 1; next }
  in_block && /^[^#[:space:]][^:]*:/ { in_block = 0 }
  in_block && /^- / { sub(/^- */, ""); print }
' "$BUILDFLOW")

if [ -z "$actual" ]; then
  echo "✗ buildflow-skips: no skip_steps entries parsed from $BUILDFLOW — the block may have been deleted entirely (gotcha 35)" >&2
  exit 1
fi

missing=0
for skip in "${expected[@]}"; do
  if ! printf '%s\n' "$actual" | grep -qx "$skip"; then
    echo "✗ buildflow-skips: documented skip '$skip' is MISSING — the gotcha-35 deletion class; restore it with its comment" >&2
    missing=1
  fi
done

while IFS= read -r skip; do
  [ -n "$skip" ] || continue
  if ! printf '%s\n' "${expected[@]}" | grep -qx "$skip"; then
    echo "⚠ buildflow-skips: undocumented skip '$skip' — add it to the expected set in check-buildflow-skips.sh with a justification"
  fi
done <<<"$actual"

if [ "$missing" -eq 0 ]; then
  echo "✓ buildflow-skips: all ${#expected[@]} documented skips present"
fi
exit "$missing"
