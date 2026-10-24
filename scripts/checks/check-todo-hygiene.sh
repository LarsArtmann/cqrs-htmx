#!/usr/bin/env bash
# check-todo-hygiene.sh — four mechanical rules that keep TODO_LIST.md honest.
#
# Why: the backlog twice drifted into a session diary — an 8,104-character
# header, 23 `[x]` rows in a file whose own legend forbids them, and dead rows
# whose titles were already struck. Each defect was invisible to every existing
# gate, so the file lied for weeks. These rules make each class a red build.
#
# Rules (all applied to TODO_LIST.md):
#   1. No `[x]` checklist rows. Done work moves to CHANGELOG.md.
#   2. No struck-title rows. A row whose title starts with `~~` is dead; delete
#      it after verifying the receipt (CHANGELOG.md or an inline hash).
#   3. No section more than 30% dead. Dead = `[x]` rows + struck-title rows.
#      Authoring debt is capped per section, not only in total.
#   4. Header discipline. The preamble before the first `##` heading must stay
#      under 1,500 characters in total, and no single paragraph in it may
#      exceed 300 characters. This kills the session-diary class without
#      forbidding a short pointer block.
#
# Usage: bash scripts/checks/check-todo-hygiene.sh [--verbose]
# Env:   TODO_HYGIENE_ROOT  override the scan root (fixture self-test).
# Exit:  0 clean, 1 rule violation, 2 usage/missing file.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${TODO_HYGIENE_ROOT:-$REPO_ROOT}"
TODO="$ROOT/TODO_LIST.md"

VERBOSE=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --verbose) VERBOSE=1 ;;
  *)
    echo "usage: check-todo-hygiene.sh [--verbose]" >&2
    exit 2
    ;;
  esac
  shift
done

if [ ! -f "$TODO" ]; then
  echo "✗ todo-hygiene: $TODO not found" >&2
  exit 2
fi

out=$(TODO_FILE="$TODO" awk -v verbose="$VERBOSE" '
function trim(s) { sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s); return s }

function looks_dead_title(rest) {
  rest = trim(rest)
  while (substr(rest, 1, 2) == "**") rest = trim(substr(rest, 3))
  return substr(rest, 1, 2) == "~~"
}

/^## / {
  # close the previous section and evaluate rule 3
  if (section != "") {
    if (dead * 100 > 30 * rows) {
      printf "RULE3 %s|%d|%d\n", section, dead, rows
      violations++
    }
    if (verbose == 1) printf "SECTION %s rows=%d dead=%d\n", section, rows, dead
  }
  section = trim(substr($0, 4))
  rows = 0
  dead = 0
  preamble_done = 1
  next
}

{
  if (preamble_done == 0) {
    preamble_total += length($0) + 1
    if (trim($0) == "") {
      if (para != "") { paras[++nparas] = para; para = "" }
    } else {
      if (para != "") para = para "\n" $0
      else para = $0
    }
    next
  }

  if ($0 ~ /^[[:space:]]*- \[[ x~]\]/) {
    rows++
    grows++
    rest = $0
    sub(/^[[:space:]]*- \[[ x~]\]/, "", rest)
    is_x = ($0 ~ /^[[:space:]]*- \[x\]/)
    is_dead = looks_dead_title(rest)
    if (is_x) {
      printf "RULE1 %d|%s\n", NR, trim($0)
      violations++
      dead++
      gdead++
    } else if (is_dead) {
      printf "RULE2 %d|%s\n", NR, trim($0)
      violations++
      dead++
      gdead++
    }
  }
}

END {
  if (para != "") paras[++nparas] = para
  for (i = 1; i <= nparas; i++) {
    if (length(paras[i]) > 300) {
      printf "RULE4-PARA %d|%s\n", i, substr(paras[i], 1, 110)
      violations++
    }
  }
  if (preamble_total > 1500) {
    printf "RULE4-TOTAL %d\n", preamble_total
    violations++
  }
  if (section != "") {
    if (dead * 100 > 30 * rows) {
      printf "RULE3 %s|%d|%d\n", section, dead, rows
      violations++
    }
    if (verbose == 1) printf "SECTION %s rows=%d dead=%d\n", section, rows, dead
  }
  printf "TOTAL rows=%d dead=%d violations=%d preamble=%d\n", grows, gdead, violations, preamble_total
  exit (violations > 0 ? 1 : 0)
}
' "$TODO")

rc=$?
summary=$(printf '%s\n' "$out" | grep '^TOTAL ' | tail -1)
printf '%s\n' "$out" | grep -v -E '^TOTAL ' | sed 's/^/  /'

case "$rc" in
0)
  echo "✓ todo-hygiene: clean ($summary)"
  exit 0
  ;;
1)
  cat >&2 <<'EOF'
✗ todo-hygiene: violations above.
  RULE1 — delete the `[x]` row; move the completed item to CHANGELOG.md.
  RULE2 — delete the struck-title row after confirming its receipt.
  RULE3 — that section is >30% dead; delete the dead rows.
  RULE4 — shorten the header: total <1,500 chars, no paragraph >300 chars.
EOF
  exit 1
  ;;
*)
  echo "✗ todo-hygiene: internal parser failure (rc=$rc)" >&2
  exit 2
  ;;
esac
