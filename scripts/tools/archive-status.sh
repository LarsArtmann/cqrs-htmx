#!/usr/bin/env bash
# archive-status.sh — one atomic, gated archive pass over docs/status reports.
#
# Hand-running this ritual has failed twice on ORDER (annotate → gate → move →
# re-base links → manifest → lower the baseline): the 2026-10-15 pass gated
# only after `git mv`, hiding shape damage, and round-22 broke 23 links because
# `git mv` changes relative-link depth. This script fixes the order in code.
#
# Per file (in this order):
#   1. `check-status-rows.py` on the LIVE path  (gate BEFORE the move)
#   2. `git mv` into docs/status/archived/
#   3. prepend the dated `> ANNOTATED … SUPERSEDED` blockquote
#   4. re-base every relative link for the new depth (rebase_links.py)
# After every file: the two status gates + `check-docs-links.sh`, and
# `--update-baseline` lowers docs/status/TAIL_BASELINE to the measured tail.
#
# Dry run is the DEFAULT: it validates the spec and prints the plan, mutating
# nothing. Nothing is ever deleted from git history — the move is reversible
# with `git mv docs/status/archived/<file> docs/status/<file>`.
#
# Usage:
#   bash scripts/tools/archive-status.sh <spec.tsv>                    # dry run
#   bash scripts/tools/archive-status.sh --apply <spec.tsv>            # archive
#   bash scripts/tools/archive-status.sh --apply --update-baseline <spec.tsv>
#
# Options:
#   --date YYYY-MM-DD   annotation date (default: today)
#   --pass NAME         manifest pass label (default: "round-23 (<date>)")
#   --keep-baseline     do not lower TAIL_BASELINE even with --update-baseline
#
# Spec: TSV, one file per line, `#` comments and blank lines ignored:
#   <report-basename>	<successor>	<reason>
# `<successor>` is relative to docs/status/ (live report or `archived/…`).
# `<reason>` must name what the successor covers; a generic "same series"
# reason is the docs-health skill's #1 failure mode — do not pass one.
#
# Env: ARCHIVE_STATUS_ROOT  override the repo root (fixture self-test).
# Exit: 0 ok, 1 refusal/gate failure, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${ARCHIVE_STATUS_ROOT:-$REPO_ROOT}"
STATUS_DIR="$ROOT/docs/status"
ARCHIVE_DIR="$STATUS_DIR/archived"

APPLY=0
UPDATE_BASELINE=0
KEEP_BASELINE=0
SKIP_GATES="${ARCHIVE_STATUS_SKIP_GATES:-0}"
DATE="$(date +%Y-%m-%d)"
PASS=""
SPEC=""

while [ "$#" -gt 0 ]; do
  case "$1" in
  --apply) APPLY=1 ;;
  --update-baseline) UPDATE_BASELINE=1 ;;
  --keep-baseline) KEEP_BASELINE=1 ;;
  --date)
    shift
    DATE="${1:-}"
    ;;
  --pass)
    shift
    PASS="${1:-}"
    ;;
  --*)
    echo "unknown option: $1" >&2
    exit 2
    ;;
  *)
    SPEC="$1"
    ;;
  esac
  shift
done

[ -z "$PASS" ] && PASS="round-23 ($DATE)"
if [ -z "$SPEC" ]; then
  echo "usage: archive-status.sh [--apply] [--update-baseline] [--date YYYY-MM-DD] [--pass NAME] <spec.tsv>" >&2
  exit 2
fi
case "$DATE" in
[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]) ;;
*)
  echo "✗ archive-status: --date must be YYYY-MM-DD (got '$DATE')" >&2
  exit 2
  ;;
esac
if [ ! -f "$SPEC" ]; then
  echo "✗ archive-status: spec not found: $SPEC" >&2
  exit 1
fi

reports=()
successors=()
reasons=()
verdicts=()
lineno=0
while IFS=$'\t' read -r report successor reason verdict || [ -n "${report:-}" ]; do
  lineno=$((lineno + 1))
  case "$report" in '' | '#'*) continue ;; esac
  if [ -z "${successor:-}" ] || [ -z "${reason:-}" ]; then
    echo "✗ archive-status: $SPEC:$lineno needs 3 tab-separated fields (report, successor, reason[, verdict])" >&2
    exit 1
  fi
  verdict="${verdict:-superseded}"
  case "$verdict" in
  superseded | resolved) ;;
  *)
    echo "✗ archive-status: $SPEC:$lineno verdict must be 'superseded' or 'resolved' (got '$verdict')" >&2
    exit 1
    ;;
  esac
  reports+=("$report")
  successors+=("$successor")
  reasons+=("$reason")
  verdicts+=("$verdict")
done <"$SPEC"

if [ "${#reports[@]}" -eq 0 ]; then
  echo "✗ archive-status: spec has no entries" >&2
  exit 1
fi

# Phase A: validate every entry and gate every report BEFORE moving anything.
stale=0
moved_target=0
for i in "${!reports[@]}"; do
  report="${reports[$i]}"
  successor="${successors[$i]}"
  case "$report" in
  */* | README.md)
    echo "✗ archive-status: '$report' must be a bare report basename in docs/status/" >&2
    stale=$((stale + 1))
    continue
    ;;
  esac
  if [ -e "$ARCHIVE_DIR/$report" ]; then
    echo "✗ archive-status: already archived: $report" >&2
    moved_target=$((moved_target + 1))
    continue
  fi
  if [ ! -f "$STATUS_DIR/$report" ]; then
    echo "✗ archive-status: report not found: docs/status/$report" >&2
    stale=$((stale + 1))
    continue
  fi
  successor_path="$STATUS_DIR/$successor"
  if [ ! -f "$successor_path" ]; then
    successor_path="$ARCHIVE_DIR/$(basename "$successor")"
  fi
  if [ ! -f "$successor_path" ]; then
    echo "✗ archive-status: successor not found for $report: $successor" >&2
    stale=$((stale + 1))
    continue
  fi
  case "${reasons[$i]}" in
  *'|'*)
    echo "✗ archive-status: reason for $report contains '|' (breaks the manifest table)" >&2
    stale=$((stale + 1))
    continue
    ;;
  esac
  if [ -e "$ARCHIVE_DIR/$report" ]; then
    echo "✗ archive-status: already archived: $report" >&2
    moved_target=$((moved_target + 1))
    continue
  fi
  if [ "$SKIP_GATES" -eq 0 ] && ! python3 "$REPO_ROOT/scripts/checks/check-status-rows.py" "$STATUS_DIR/$report" >/dev/null 2>&1; then
    echo "✗ archive-status: PARTIAL rows in $report — run 'python3 scripts/checks/check-status-rows.py $STATUS_DIR/$report' and normalize first" >&2
    stale=$((stale + 1))
    continue
  fi
done

if [ "$stale" -gt 0 ] || [ "$moved_target" -gt 0 ]; then
  echo "✗ archive-status: $((stale + moved_target)) problem(s); nothing was moved" >&2
  exit 1
fi

echo "archive-status: ${#reports[@]} report(s), pass '$PASS', date $DATE"
for i in "${!reports[@]}"; do
  echo "  - ${reports[$i]}  ->  ${successors[$i]}  [${verdicts[$i]}]"
done

if [ "$APPLY" -eq 0 ]; then
  echo "dry run: nothing moved. Re-run with --apply to archive ${#reports[@]} file(s)."
  exit 0
fi

# Phase B: move, annotate, re-base.
changed=()
for i in "${!reports[@]}"; do
  report="${reports[$i]}"
  successor="${successors[$i]}"
  reason="${reasons[$i]}"
  verdict="${verdicts[$i]}"
  # Both the blockquote and the manifest row live in docs/status/archived/, so a
  # live successor is one level up; an archived successor is a sibling.
  if [ -f "$STATUS_DIR/$successor" ]; then
    successor_rel="../$(basename "$successor")"
  else
    successor_rel="$(basename "$successor")"
  fi

  if ! git -C "$ROOT" mv "docs/status/$report" "docs/status/archived/$report"; then
    echo "✗ archive-status: git mv failed for $report" >&2
    exit 1
  fi
  new_path="$ARCHIVE_DIR/$report"

  if ! python3 "$SCRIPT_DIR/rebase_links.py" "$ROOT" "$new_path" "docs/status" "docs/status/archived"; then
    echo "✗ archive-status: link re-base failed for $report (moved; finish by hand)" >&2
    exit 1
  fi

  if ! REPORT="$report" SUCCESSOR="$successor_rel" REASON="$reason" VERDICT="$verdict" WHEN="$DATE" PASSNAME="$PASS" python3 - "$new_path" <<'PY'
import os
import sys

path = sys.argv[1]
text = open(path, encoding="utf-8").read()
lines = text.splitlines()
if not lines or not lines[0].startswith("# "):
    print(f"✗ {path}: first line is not an H1; cannot place the blockquote", file=sys.stderr)
    raise SystemExit(1)
if "ANNOTATED " + os.environ["WHEN"] in text:
    print(f"archive-status: {path} already annotated for {os.environ['WHEN']}; skipping blockquote")
    raise SystemExit(0)

rest = "\n".join(lines[1:]).lstrip("\n")
successor = os.environ["SUCCESSOR"]
link = os.path.basename(successor)
reason = os.environ["REASON"]
when = os.environ["WHEN"]
passname = os.environ["PASSNAME"]
if os.environ["VERDICT"] == "resolved":
    block = (
        f"> **ANNOTATED {when}** — ~~every open item in this report is resolved~~ "
        f"**ARCHIVED {when} ({passname}).**\n"
        f"> The next session in the same arc is [`{link}`]({successor}), which {reason}.\n"
        "> Nothing here is a live work item: it is kept as the session's evidence.\n"
    )
else:
    block = (
        f"> **ANNOTATED {when}** — ~~this report~~ **SUPERSEDED** by "
        f"[`{link}`]({successor}), which {reason}. Strike depth:\n"
        "> supersession verdict (round-18 precedent for series-internal reports); nothing here is a\n"
        f"> unique open item. **ARCHIVED {when} ({passname}).**\n"
    )
out = lines[0] + "\n\n" + block + "\n" + rest
if text.endswith("\n"):
    out += "\n"
open(path, "w", encoding="utf-8").write(out)
PY
  then
    echo "✗ archive-status: annotate failed for $report (moved; finish by hand)" >&2
    exit 1
  fi

  changed+=("$report")
  if [ "$verdict" = "resolved" ]; then
    class_label="ARCHIVE — resolved"
  else
    class_label="ARCHIVE — superseded"
  fi
  # shellcheck disable=SC2016 # the backticks are literal markdown, not substitutions
  printf '| %s | `%s` | %s | [`%s`](%s) %s |\n' \
    "$PASS" "$report" "$class_label" "$(basename "$successor")" "$successor_rel" "$reason" >>"$ARCHIVE_DIR/README.md"
done

# Phase B2: re-point INBOUND links to the moved files across the whole repo.
# Moving one file breaks every citation of it elsewhere (live docs, archived
# siblings, the manifest rows just written). rebase_links.py only fixes the
# moved file's own links, so this pass closes the class that round-22 left with
# 20 broken links.
move_pairs=()
for i in "${!reports[@]}"; do
  move_pairs+=("docs/status/${reports[$i]}" "docs/status/archived/${reports[$i]}")
done
if ! python3 "$SCRIPT_DIR/fix_moved_links.py" "$ROOT" "${move_pairs[@]}"; then
  echo "✗ archive-status: inbound-link re-point failed (files moved; fix links by hand)" >&2
  exit 1
fi

# Phase C: gates over the mutated tree. Skipped only by the fixture self-test
# (a fixture root has no scripts/checks; production runs always gate).
gate_failed=0
if [ "$SKIP_GATES" -eq 0 ]; then
  for cmd in \
    "bash $REPO_ROOT/scripts/checks/check-status-annotations.sh" \
    "python3 $REPO_ROOT/scripts/checks/check-status-rows.py" \
    "bash $REPO_ROOT/scripts/checks/check-docs-links.sh"; do
    output=$($cmd 2>&1)
    rc=$?
    if [ "$rc" -ne 0 ]; then
      echo "✗ archive-status: gate failed: ${cmd%% *} …" >&2
      printf '%s\n' "$output" | tail -25 >&2
      gate_failed=1
    fi
  done
fi
if [ "$gate_failed" -ne 0 ]; then
  echo "✗ archive-status: ${#changed[@]} file(s) moved but a gate is red — fix forward; the move is reversible with 'git mv docs/status/archived/<file> docs/status/<file>'" >&2
  exit 1
fi

# Phase D: tail count + optional baseline ratchet-down.
count=$(find "$STATUS_DIR" -maxdepth 1 -name '*.md' ! -name 'README.md' | wc -l | tr -d '[:space:]')
baseline_file="$STATUS_DIR/TAIL_BASELINE"
echo "archive-status: tail is now $count report(s)"
if [ "$UPDATE_BASELINE" -eq 1 ] && [ "$KEEP_BASELINE" -eq 0 ] && [ -f "$baseline_file" ]; then
  baseline=$(tr -d '[:space:]' <"$baseline_file")
  if [ "$count" -lt "$baseline" ]; then
    printf '%s\n' "$count" >"$baseline_file"
    echo "archive-status: TAIL_BASELINE lowered $baseline -> $count"
  else
    echo "archive-status: TAIL_BASELINE kept at $baseline (tail $count)"
  fi
fi
echo "✓ archive-status: archived ${#changed[@]} report(s) in pass '$PASS'"
