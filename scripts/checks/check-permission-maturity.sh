#!/usr/bin/env bash
# check-permission-maturity.sh — the docs/PERMISSIONS.md ratchet.
#
# The inventory in docs/PERMISSIONS.md is the single source of truth for what
# authorization posture each service claims. This gate makes sure the claim can
# only improve: a service whose level drops below the committed baseline fails.
#
# Levels are ordered L0 < L1 < L2 < L3 < L4. The `?` level means "not
# inspected" and is neither ratcheted nor compared — an unverified row must not
# masquerade as a level.
#
# Usage:
#   check-permission-maturity.sh                  # advisory report (exit 0)
#   check-permission-maturity.sh --ratchet        # fail on a dropped level
#   check-permission-maturity.sh --update-baseline  # rewrite the baseline
#
# Exit codes: 0 ok · 1 regression · 2 usage/parse error.

set -uo pipefail

cd "$(dirname "$0")/../.." || exit 2

doc="docs/PERMISSIONS.md"
baseline="docs/PERMISSIONS_BASELINE"

mode="advisory"
case "${1:-}" in
"") ;;
--ratchet) mode="ratchet" ;;
--update-baseline) mode="update" ;;
-h | --help)
  sed -n '2,18p' "$0"
  exit 0
  ;;
*)
  echo "usage: $0 [--ratchet|--update-baseline]" >&2
  exit 2
  ;;
esac

if [ ! -f "$doc" ]; then
  echo "FAIL: $doc missing" >&2
  exit 2
fi

# Extract "<service> TAB <level>" rows from the inventory table. The service cell
# is the first cell and may be backtick-quoted; the level cell is the second.
# shellcheck disable=SC2016 # the backticks are literal markdown table syntax, not substitutions
current="$(sed -n 's/^| *`\{0,1\}\([^`|]*[^`| ]\)`\{0,1\} *| *\(L[0-4]\|[?]\) *|.*/\1\t\2/p' "$doc")"

if [ -z "$current" ]; then
  echo "FAIL: no inventory rows parsed from $doc" >&2
  exit 2
fi

if [ "$mode" = "update" ]; then
  {
    echo "# docs/PERMISSIONS_BASELINE — the levels docs/PERMISSIONS.md may not drop below."
    echo "# Format: <service> <TAB> <level>. '?' rows are unverified and not ratcheted."
    echo "# Regenerate: bash scripts/checks/check-permission-maturity.sh --update-baseline"
    printf '%s\n' "$current" | grep -v $'\t?$' | sort
  } >"$baseline"
  echo "baseline written: $baseline"
  exit 0
fi

if [ ! -f "$baseline" ]; then
  echo "FAIL: $baseline missing — create it with --update-baseline" >&2
  exit 2
fi

rank() {
  case "$1" in
  L0) echo 0 ;;
  L1) echo 1 ;;
  L2) echo 2 ;;
  L3) echo 3 ;;
  L4) echo 4 ;;
  *) echo -1 ;;
  esac
}

regressions=0
checked=0

while IFS=$'\t' read -r service level; do
  [ -z "$service" ] && continue
  [ "$level" = "?" ] && continue

  base_level="$(grep -F "$service"$'\t' "$baseline" | cut -f2 | head -1)"
  if [ -z "$base_level" ]; then
    # A new service is an improvement, not a regression: it joined the
    # inventory. Report it so the baseline gets refreshed deliberately.
    echo "NEW: $service -> $level (not in baseline; refresh with --update-baseline)"
    continue
  fi

  checked=$((checked + 1))
  cur_rank="$(rank "$level")"
  base_rank="$(rank "$base_level")"

  if [ "$cur_rank" -lt 0 ] || [ "$base_rank" -lt 0 ]; then
    echo "FAIL: unparseable level for $service (doc=$level baseline=$base_level)" >&2
    exit 2
  fi

  if [ "$cur_rank" -lt "$base_rank" ]; then
    echo "REGRESSION: $service dropped from $base_level to $level"
    regressions=$((regressions + 1))
  fi
done <<<"$current"

while IFS=$'\t' read -r service level; do
  [ -z "$service" ] && continue
  case "$service" in \#*) continue ;; esac
  if ! printf '%s\n' "$current" | grep -qF "$service"$'\t'; then
    echo "REGRESSION: $service vanished from $doc (baseline says $level)"
    regressions=$((regressions + 1))
  fi
done <"$baseline"

if [ "$regressions" -gt 0 ]; then
  echo
  echo "$regressions maturity regression(s); $checked service(s) compared"
  if [ "$mode" = "ratchet" ]; then
    exit 1
  fi
  echo "(advisory mode: exit 0 — pass --ratchet to enforce)"
  exit 0
fi

echo "OK: $checked service(s) at or above their baseline level"
