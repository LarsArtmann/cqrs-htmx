#!/usr/bin/env bash
# check-libs-report-count.sh — fleet-series integrity gate (plan B11, 2026-11-15).
#
# The 2026-11-14 fleet refresh dropped the go-etag report exactly this way:
# the per-lib report file existed, the index.html table row did not, and the
# omission survived to publication because nothing compared the two sides.
# This gate makes the divergence impossible: the set of report files under
# docs/review/libs/ (FORMAT.md excluded — conventions, not a report) must
# equal, in BOTH directions, the set of `>md</a>` report rows in index.html,
# and BOTH counts must equal the pinned baseline (ratchet: a deliberate add
# or remove bumps it in the same, visible commit via --update-baseline).
#
# Usage:
#   bash scripts/checks/check-libs-report-count.sh                # report (advisory)
#   bash scripts/checks/check-libs-report-count.sh --strict       # blocking (CI)
#   bash scripts/checks/check-libs-report-count.sh --update-baseline
#
# Env: LIBS_COUNT_GATE_ROOT  override the repo root (fixture self-test).
#      LIBS_COUNT_BASELINE   override the baseline file (fixture self-test).
# Exit: 0 ok / advisory, 1 drift under --strict or missing inputs, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${LIBS_COUNT_GATE_ROOT:-$REPO_ROOT}"
LIBS_DIR="$ROOT/docs/review/libs"
INDEX_FILE="$LIBS_DIR/index.html"
BASELINE="${LIBS_COUNT_BASELINE:-$SCRIPT_DIR/libs-report-count.baseline}"

STRICT=0
UPDATE=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --strict) STRICT=1 ;;
  --update-baseline) UPDATE=1 ;;
  *)
    echo "usage: check-libs-report-count.sh [--strict] [--update-baseline]" >&2
    exit 2
    ;;
  esac
  shift
done

if [ ! -d "$LIBS_DIR" ]; then
  echo "✗ libs-report-count: missing series directory $LIBS_DIR" >&2
  exit 1
fi
if [ ! -f "$INDEX_FILE" ]; then
  echo "✗ libs-report-count: missing fleet index $INDEX_FILE" >&2
  exit 1
fi

# Report files: *.md in the series dir minus the conventions file.
mapfile -t report_files < <(find "$LIBS_DIR" -maxdepth 1 -name '*.md' ! -name 'FORMAT.md' -printf '%f\n' | sed 's/\.md$//' | LC_ALL=C sort)

# Report rows: index table entries linking a report as their `md` cell.
# Unique + raw occurrence count so a duplicated row is caught too.
mapfile -t row_names < <(grep -oE 'href="[a-z0-9-]+\.md">md</a>' "$INDEX_FILE" | sed 's/^href="//; s/\.md">md<\/a>$//' | LC_ALL=C sort -u)
row_raw=$(grep -cE 'href="[a-z0-9-]+\.md">md</a>' "$INDEX_FILE" || true)

files_count=${#report_files[@]}
rows_count=${#row_names[@]}

if [ "$UPDATE" -eq 1 ]; then
  if [ "$files_count" -ne "$rows_count" ] || [ "$rows_count" -ne "$row_raw" ]; then
    echo "✗ libs-report-count: refuse --update-baseline while the two sides disagree (files=$files_count rows=$rows_count raw=$row_raw) — fix the divergence first" >&2
    exit 1
  fi
  printf '%s\n' "$files_count" >"$BASELINE"
  echo "✓ libs-report-count: baseline written: $files_count"
  exit 0
fi

echo "libs-report-count: $files_count report file(s), $rows_count index row(s) (raw $row_raw)"

if [ "$rows_count" -ne "$row_raw" ]; then
  echo "✗ libs-report-count: duplicate report row(s) in the index ($row_raw raw vs $rows_count unique)" >&2
  exit 1
fi

drift=0
if diff_out=$(diff <(printf '%s\n' "${report_files[@]}") <(printf '%s\n' "${row_names[@]}")); then
  :
else
  drift=1
  echo "✗ libs-report-count: report files and index rows disagree (left = file without row, right = row without file):" >&2
  printf '%s\n' "$diff_out" | sed 's/^/    /' >&2
fi

if [ ! -f "$BASELINE" ]; then
  echo "✗ libs-report-count: missing baseline $BASELINE (run with --update-baseline on a green tree to create it)" >&2
  exit 1
fi

pinned=$(tr -d '[:space:]' <"$BASELINE")
if [ "$files_count" -ne "$pinned" ]; then
  drift=1
  echo "✗ libs-report-count: series size $files_count != pinned baseline $pinned (grow or shrink it deliberately, bump the baseline in the SAME commit)" >&2
fi

if [ "$drift" -eq 0 ]; then
  echo "✓ libs-report-count: series intact at $files_count"
  exit 0
fi

if [ "$STRICT" -eq 0 ]; then
  echo "libs-report-count: advisory — re-run with --strict to block"
  exit 0
fi

echo "libs-report-count: every report must be table-linked and every link must resolve (the go-etag drop class); fix the index or the series, then bump the baseline same-commit if the size deliberately changed: bash scripts/checks/check-libs-report-count.sh --update-baseline" >&2
exit 1
