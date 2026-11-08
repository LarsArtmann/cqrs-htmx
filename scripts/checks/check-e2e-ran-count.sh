#!/usr/bin/env bash
# check-e2e-ran-count.sh — fail loud when Playwright silently did not run tests.
#
# WHY (2026-11-08, the bug-#7 lesson): the browser battery's axe tests
# silently "did not run" for weeks (serial-cascade skips + a load-sensitive
# availability probe), which masked a real shipped a11y violation until the
 # skip class was fixed by accident. A battery that quietly runs fewer tests
# than it lists is not a gate. This script compares the LISTED test count
# against a JSON run report and fails on any spec that produced NO result.
#
# Usage:
#   check-e2e-ran-count.sh <report.json> [--expect-unrun N]
#     <report.json>   a `--reporter=json` output file
#     --expect-unrun N   allow exactly N result-less specs (e.g. a known
#                        serial-cascade) without failing; the default 0.
# Exit codes: 0 = every listed spec produced a result (or exactly N did not);
#             1 = more specs without results than expected; 2 = usage/input.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../../lib/shell-lib.sh
[ -f "$SCRIPT_DIR/../lib/shell-lib.sh" ] && . "$SCRIPT_DIR/../lib/shell-lib.sh" || true

REPORT=""
EXPECT_UNRUN=0

while [ $# -gt 0 ]; do
  case "$1" in
    --expect-unrun)
      EXPECT_UNRUN="${2:?--expect-unrun needs a value}"
      shift 2
      ;;
    -h|--help)
      sed -n '2,16p' "$0"
      exit 0
      ;;
    *)
      if [ -n "$REPORT" ]; then
        echo "check-e2e-ran-count: unexpected extra argument '$1'" >&2
        exit 2
      fi
      REPORT="$1"
      shift
      ;;
  esac
done

if [ -z "$REPORT" ] || [ ! -f "$REPORT" ]; then
  echo "check-e2e-ran-count: report file missing or not found: '${REPORT:-<none>}'" >&2
  echo "  produce one with: playwright test --reporter=json > report.json (or PLAYWRIGHT_JSON_OUTPUT_NAME)" >&2
  exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "check-e2e-ran-count: jq is required" >&2
  exit 2
fi

# A "did not run" spec: listed in the report but every one of its tests has
# ZERO results (an intended test.skip() still writes a skipped RESULT; a
# serial-cascade or interrupted spec writes none).
read -r TOTAL UNRUN <<<"$(jq -r '
  [ .. | objects | select(has("specs")) | .specs[]? ] as $all
  | [ $all[] | select((.tests // []) | all(.results // [] | length == 0)) ] as $unrun
  | "\($all | length) \($unrun | length)"
' "$REPORT")"

echo "check-e2e-ran-count: $TOTAL listed specs, $UNRUN produced no result (expected: $EXPECT_UNRUN)"

if [ "$UNRUN" -gt "$EXPECT_UNRUN" ]; then
  echo "✗ check-e2e-ran-count: $UNRUN result-less specs exceed the expected $EXPECT_UNRUN —" >&2
  echo "  the battery ran FEWER tests than it lists; a silent skip is masking signal" >&2
  jq -r '[ .. | objects | select(has("specs")) | .specs[]? ]
    | .[] | select((.tests // []) | all(.results // [] | length == 0))
    | "  - \(.title) (\(.file))"' "$REPORT" >&2 || true
  exit 1
fi

echo "✓ check-e2e-ran-count: every listed spec produced a result"
