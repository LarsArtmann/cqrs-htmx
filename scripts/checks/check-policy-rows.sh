#!/usr/bin/env bash
# check-policy-rows.sh — fail on a policy row that matches no declared dispatch.
#
# A permission row is a promise: "this subject may execute command/X". If no
# command/X is ever declared, the row is either a typo (it grants nothing, and
# somebody believes it does) or a leftover from a command that was deleted. Both
# are silent — a stale row never fails a request, it just stops protecting one.
#
# The two inputs are explicit: this is not a repository scan. Declarations come
# from the dispatch gate's manifest (one object per line, "command/CreateItem"),
# rows from Casbin CSV files ("p, subject, domain, object, action, eft").
#
# Usage:
#   check-policy-rows.sh --declarations FILE [--policies FILE ...]
#
# Exit codes: 0 clean · 1 orphan row(s) · 2 usage.
#
# Suppressing a row: put `policy-rows-allow` on the row or the line above it,
# with a reason. Suppressions are counted and printed.

set -uo pipefail

declarations=""
policies=()

while [ $# -gt 0 ]; do
  case "$1" in
  --declarations)
    [ $# -ge 2 ] || {
      echo "usage: $0 --declarations FILE [--policies FILE ...]" >&2
      exit 2
    }
    declarations="$2"
    shift 2
    ;;
  --policies)
    [ $# -ge 2 ] || {
      echo "usage: $0 --declarations FILE [--policies FILE ...]" >&2
      exit 2
    }
    policies+=("$2")
    shift 2
    ;;
  -h | --help)
    sed -n '2,22p' "$0"
    exit 0
    ;;
  *)
    echo "usage: $0 --declarations FILE [--policies FILE ...]" >&2
    exit 2
    ;;
  esac
done

if [ -z "$declarations" ]; then
  echo "usage: $0 --declarations FILE [--policies FILE ...]" >&2
  exit 2
fi

if [ ! -f "$declarations" ]; then
  echo "FAIL: declarations file does not exist: $declarations" >&2
  exit 2
fi

declared=()
while IFS= read -r line; do
  line="${line%%#*}"
  line="$(printf '%s' "$line" | tr -d '[:space:]')"
  [ -z "$line" ] && continue
  declared+=("$line")
done <"$declarations"

if [ "${#declared[@]}" -eq 0 ]; then
  echo "FAIL: no declarations parsed from $declarations" >&2
  exit 2
fi

# keymatch PATTERN CANDIDATE — Casbin's keyMatch semantics: the pattern may carry
# one "*", which matches anything from that point on. Mirrors
# casbin/util.KeyMatch so the linter agrees with the engine it checks.
keymatch() {
  local pattern="$1" candidate="$2" prefix
  case "$pattern" in
  *"*"*)
    prefix="${pattern%%\**}"
    case "$candidate" in
    "$prefix"*) return 0 ;;
    *) return 1 ;;
    esac
    ;;
  esac
  [ "$candidate" = "$pattern" ]
}

# matches_declared OBJECT — a row is orphaned only when no declared object
# matches it; "*" and prefix patterns that cover something are legitimate.
matches_declared() {
  local pattern="$1" candidate
  for candidate in "${declared[@]}"; do
    if keymatch "$pattern" "$candidate"; then
      return 0
    fi
  done
  return 1
}

# suggest PATTERN — the nearest declared objects, for an actionable message.
suggest() {
  local pattern="$1" prefix out=""
  prefix="${pattern%%/*}/"

  for candidate in "${declared[@]}"; do
    case "$candidate" in
    "$prefix"*)
      out="$out $candidate"
      ;;
    esac
  done

  [ -n "$out" ] && printf ' (declared under %s:%s)' "$prefix" "$out"
  return 0
}

fail=0
rows=0
suppressed=0

for file in "${policies[@]+"${policies[@]}"}"; do
  [ -f "$file" ] || {
    echo "FAIL: policy file does not exist: $file" >&2
    exit 2
  }

  lineno=0
  prev=""
  while IFS= read -r raw; do
    lineno=$((lineno + 1))
    # `above` is the line before this one; keeping it in a variable avoids
    # re-reading the file we are iterating (SC2094).
    above="$prev"
    prev="$raw"

    line="$(printf '%s' "$raw" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    case "$line" in
    "" | "#"*) continue ;;
    esac

    # Only policy rows carry an object; g/g2 grouping rows have no object.
    case "$line" in
    p,* | p\ *) ;;
    *) continue ;;
    esac

    rows=$((rows + 1))

    # The row shape is "p, subject, domain, object, action, eft" — the object
    # is the fourth field.
    object="$(printf '%s' "$line" | cut -d, -f4 | tr -d '[:space:]')"
    if [ -z "$object" ]; then
      continue
    fi

    if matches_declared "$object"; then
      continue
    fi

    if printf '%s' "$raw" | grep -q 'policy-rows-allow'; then
      suppressed=$((suppressed + 1))
      continue
    fi

    if printf '%s' "$above" | grep -q 'policy-rows-allow'; then
      suppressed=$((suppressed + 1))
      continue
    fi

    if [ "$fail" -eq 0 ]; then
      echo "FAIL: orphan policy rows"
    fi
    fail=1
    printf '  %s:%s: object %s matches no declared dispatch%s\n' \
      "$file" "$lineno" "$object" "$(suggest "$object")"
  done <"$file"
done

if [ "$fail" -ne 0 ]; then
  echo
  echo "Suppressed rows: $suppressed (each needs a reason next to policy-rows-allow)"
  exit 1
fi

echo "OK: $rows policy row(s) match a declared dispatch (suppressed: $suppressed)"
