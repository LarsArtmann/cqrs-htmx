#!/usr/bin/env bash
# check-root-clusters.sh — generate and verify the ROOT PACKAGE's per-cluster
# table from a manifest, so the plan's table is produced, not hand-summed.
#
# WHY: the 2026-11-06 re-export plan's §1.2 cluster table was summed by hand and
# did not reconcile with the root total (~8,030 across clusters vs 7,594 claimed
# total, against 8,547 measured), and it had no row at all for the 702-LOC
# dispatch-gate file that landed the same morning. A hand sum cannot be checked;
# a manifest plus a gate can. This gate proves the manifest is a PARTITION of the
# root package (each production file mapped exactly once, no stale rows) and
# prints the table the docs quote, with LOC measured from disk every run.
#
# Manifest: scripts/checks/root-clusters.tsv  (one row: <cluster><TAB><file>)
#
# Usage: check-root-clusters.sh [--emit]
#   --emit   print ONLY the markdown table (docs paste this; nothing else)
#
# Env: ROOT_CLUSTERS_ROOT / ROOT_CLUSTERS_MANIFEST / ROOT_CLUSTERS_BASELINE
#      override paths (fixture self-test)
# Exit: 0 partition holds and totals reconcile · 1 drift · 2 the measurement,
#       manifest or baseline itself failed (never a false green)
set -uo pipefail

MODE=check
case "${1:-}" in
'') ;;
--emit) MODE=emit ;;
*)
  echo "usage: $(basename "$0") [--emit]" >&2
  exit 2
  ;;
esac

ROOT="${ROOT_CLUSTERS_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
MANIFEST="${ROOT_CLUSTERS_MANIFEST:-$ROOT/scripts/checks/root-clusters.tsv}"
BASELINE="${ROOT_CLUSTERS_BASELINE:-$ROOT/scripts/checks/root-size.baseline}"

if [ ! -d "$ROOT" ]; then
  echo "✗ root-clusters: $ROOT is not a directory (rc=2)" >&2
  exit 2
fi
if [ ! -f "$MANIFEST" ]; then
  echo "✗ root-clusters: no manifest at $MANIFEST (rc=2)" >&2
  exit 2
fi

work="$(mktemp -d /tmp/cqrs-htmx-root-clusters-XXXXXX)"
trap 'rm -rf "$work"' EXIT

files="$work/files"
mapped="$work/mapped"
find "$ROOT" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' ! -name '*_templ.go' \
  | sed "s|^$ROOT/||" | LC_ALL=C sort >"$files"
count="$(wc -l <"$files" | tr -d ' ')"
if [ "$count" -eq 0 ]; then
  echo "✗ root-clusters: 0 production .go files under $ROOT — the measurement lost its target (rc=2)" >&2
  exit 2
fi

cut -f2 "$MANIFEST" | LC_ALL=C sort >"$mapped"

unmapped="$(comm -23 "$files" "$mapped")"
stale="$(comm -13 "$files" "$mapped")"
dupes="$(uniq -d "$mapped")"
badrows="$(awk -F'\t' 'NF != 2 || $1 == "" || $2 == "" {print NR}' "$MANIFEST")"

fail=0
if [ -n "$badrows" ]; then
  echo "✗ root-clusters: malformed manifest row(s) — each row must be exactly '<cluster>TAB<file>' (line $(printf '%s' "$badrows" | tr '\n' ' '))" >&2
  fail=1
fi
if [ -n "$dupes" ]; then
  echo "✗ root-clusters: file(s) mapped more than once (a partition cannot double-count):" >&2
  printf '%s\n' "$dupes" | sed 's/^/    /' >&2
  fail=1
fi
if [ -n "$unmapped" ]; then
  echo "✗ root-clusters: production file(s) with NO cluster row (add them to $MANIFEST):" >&2
  printf '%s\n' "$unmapped" | sed 's/^/    /' >&2
  fail=1
fi
if [ -n "$stale" ]; then
  echo "✗ root-clusters: manifest row(s) naming a file that no longer exists (stale rows hide deletions):" >&2
  printf '%s\n' "$stale" | sed 's/^/    /' >&2
  fail=1
fi
if [ "$fail" -ne 0 ]; then
  exit 1
fi

# Per-cluster file count and LOC, measured from disk.
rows="$work/rows"
: >"$rows"
while IFS=$'\t' read -r cluster file; do
  loc="$(wc -l <"$ROOT/$file" | tr -d ' ')"
  printf '%s\t%s\t%s\n' "$cluster" 1 "$loc" >>"$rows"
done <"$MANIFEST"

summary="$work/summary"
awk -F'\t' '{files[$1] += $2; loc[$1] += $3} END {for (c in files) printf "%s\t%d\t%d\n", c, files[c], loc[c]}' \
  "$rows" | LC_ALL=C sort >"$summary"

total_files="$(awk -F'\t' '{s += $2} END {print s}' "$summary")"
total_loc="$(awk -F'\t' '{s += $3} END {print s}' "$summary")"

# Reconcile with the ONE size baseline so the two gates cannot disagree.
reconcile=0
if [ -f "$BASELINE" ]; then
  read -r bfiles bloc extra <"$BASELINE" || true
  case "${bfiles:-}${bloc:-}" in
  *[!0-9]*) reconcile=2 ;;
  '') reconcile=2 ;;
  *)
    if [ "$total_files" -ne "$bfiles" ] || [ "$total_loc" -ne "${bloc:-0}" ]; then
      reconcile=1
    fi
    ;;
  esac
fi

if [ "$MODE" = emit ]; then
  printf '| Cluster | Files | LOC |\n| --- | --- | --- |\n'
  while IFS=$'\t' read -r cluster n loc; do
    printf '| %s | %s | %s |\n' "$cluster" "$n" "$loc"
  done <"$summary"
  printf '| **Total** | **%s** | **%s** |\n' "$total_files" "$total_loc"
  exit 0
fi

echo "✓ root-clusters: manifest is a partition of the root package ($total_files files / $total_loc LOC across $(wc -l <"$summary" | tr -d ' ') clusters)"
while IFS=$'\t' read -r cluster n loc; do
  printf '    %-24s %2s files %6s LOC\n' "$cluster" "$n" "$loc"
done <"$summary"

case "$reconcile" in
0) ;;
1)
  echo "✗ root-clusters: totals disagree with $BASELINE ($total_files files / $total_loc LOC)" >&2
  echo "  Re-run: bash scripts/checks/check-root-size.sh --update-baseline, then regenerate the" >&2
  echo "  cluster table with: bash scripts/checks/check-root-clusters.sh --emit" >&2
  exit 1
  ;;
*)
  echo "✗ root-clusters: $BASELINE is missing or malformed — cannot reconcile totals (rc=2)" >&2
  exit 2
  ;;
esac

exit 0
