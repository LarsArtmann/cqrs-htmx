#!/usr/bin/env bash
# check-nested-retag-reminder.sh — P01's rule, mechanized (2026-11-21).
#
# A dependency fix that landed AFTER a nested module's latest tag means every
# consumer pinning that nested module alone is stranded on the vulnerable
# closure until a nested tag is cut — the oauth2/v4.13.0 class (the go-jose
# DoS fix rode the root train for three weeks with no nested tag carrying
# it). This gate walks the workspace's nested modules and prints one
# reminder per module whose latest tag predates the fix date.
#
# Usage:
#   check-nested-retag-reminder.sh <dep-module-path> <fix-iso-date>
#   e.g. check-nested-retag-reminder.sh github.com/larsartmann/go-jose/v4 2026-10-28
#
# The dep must appear as a DIRECT require in the module's go.mod (a closure
# match needs a full module-graph walk; the direct require is the shape the
# oauth2 incident had and the one a nested tag fixes deterministically).
#
# Exit: 0 = no stranded module; 1 = at least one reminder (advisory —
#       callers decide whether to block); 2 = usage error.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${REPO_ROOT_OVERRIDE:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

dep="${1:-}"
fix_date="${2:-}"
if [[ -z $dep || -z $fix_date ]]; then
  echo "usage: $0 <dep-module-path> <fix-iso-date>  (e.g. github.com/larsartmann/go-jose/v4 2026-10-28)" >&2
  exit 2
fi
if [[ ! $fix_date =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]]; then
  echo "usage: fix date must be ISO (YYYY-MM-DD), got: $fix_date" >&2
  exit 2
fi

cd "$REPO_ROOT"

reminders=0
escaped_dep=${dep//./\\.}
# Every nested module = a directory with its own go.mod below the root,
# excluding the never-published workspace-only dirs (examples, integration_test
# and e2e are consumers/test scaffolding, never pinned by consumers — the
# "published" set is everything git tags name).
while IFS= read -r gomod; do
  case $gomod in
  ./examples/* | ./integration_test/* | ./e2e/*) continue ;;
  esac
  moddir="$(dirname "$gomod")"
  moddir="${moddir#./}"
  [[ $moddir == "." ]] && continue # the root module has no nested tag to strand

  # Direct require in either form: block body (tab + path) or single-line
  # ("require path version").
  if ! grep -Eq "^[[:space:]]*(require[[:space:]]+)?${escaped_dep}[[:space:]]+v[0-9]+\.[0-9]+\.[0-9]+" "$gomod" 2>/dev/null; then
    continue
  fi

  # Tag pattern mirrors the module layout: usermgmt/oauth2 -> usermgmt/oauth2/v*.
  pattern="${moddir}/v*"
  latest_date="$(git for-each-ref --sort=-creatordate --count=1 --format='%(creatordate:short)' "refs/tags/${pattern}" 2>/dev/null || true)"

  if [[ -z $latest_date ]]; then
    echo "RE-TAG REMINDER: $moddir requires $dep directly but has NEVER been tagged — nested importers cannot consume any fix of it."
    reminders=$((reminders + 1))
  elif [[ $latest_date < $fix_date ]]; then
    echo "RE-TAG REMINDER: $moddir's latest tag is $latest_date, older than the $dep fix ($fix_date) — cut a nested tag so importers pinned to $moddir alone resolve the fix."
    reminders=$((reminders + 1))
  fi
done < <(find . -name go.mod -not -path './.git/*' -not -path '*/node_modules/*' -not -path './e2e/*' | sort)

if [[ $reminders -eq 0 ]]; then
  echo "✓ nested-retag-reminder: every nested module requiring $dep is tagged at or after the fix date ($fix_date)"
  exit 0
fi

echo "✗ nested-retag-reminder: $reminders stranded nested module(s) — cut their tags (the P01/oauth2-v4.13.1 class)."
exit 1
