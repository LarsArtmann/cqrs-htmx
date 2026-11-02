#!/usr/bin/env bash
# check-permission-surface.sh — fail on the four authorization anti-patterns
# that made the 2026-11-02 consumer audit expensive to read.
#
# The audit found that authorization in this ecosystem is written ad hoc:
# grants written into the in-memory enforcer, policy engines called with raw
# arguments instead of the contract, and denial sentinels re-coded per module so
# no shared classifier could recognise them. Each rule below is one of those
# defects, detected statically.
#
# Usage:
#   check-permission-surface.sh                 # scan this repository
#   check-permission-surface.sh --path DIR      # scan another checkout
#
# Exit codes: 0 clean · 1 violations found · 2 usage error.
#
# Suppressing a finding: put `permission-surface-allow` in a comment on the
# offending line, with a reason. Suppressions are counted and printed, so they
# stay visible in review instead of rotting silently.

set -uo pipefail

repo_root="$(cd "$(dirname "$0")/../.." && pwd)"
scan_root="$repo_root"

while [ $# -gt 0 ]; do
  case "$1" in
  --path)
    if [ $# -lt 2 ]; then
      echo "usage: $0 [--path DIR]" >&2
      exit 2
    fi
    scan_root="$2"
    shift 2
    ;;
  --path=*)
    scan_root="${1#--path=}"
    shift
    ;;
  -h | --help)
    sed -n '2,20p' "$0"
    exit 0
    ;;
  *)
    echo "usage: $0 [--path DIR]" >&2
    exit 2
    ;;
  esac
done

if [ ! -d "$scan_root" ]; then
  echo "FAIL: scan root does not exist: $scan_root" >&2
  exit 2
fi

cd "$scan_root" || exit 2

# Go sources, excluding tests and vendored/third-party trees. Tests are excluded
# on purpose: a test that writes a policy directly is often the point.
excludes=(--exclude-dir=.git --exclude-dir=vendor --exclude-dir=node_modules
  --exclude-dir=testdata --exclude='*_test.go' --exclude-dir=archived)

fail=0
suppressed=0
report() { printf '  %s:%s: %s\n' "$1" "$2" "$3"; }

# is_suppressed FILE LINE TEXT — a finding is suppressed when the marker sits on
# the line itself or on the line directly above (the nolint placement every Go
# developer already uses).
is_suppressed() {
  local file="$1" lnum="$2" text="$3"
  if printf '%s' "$text" | grep -q 'permission-surface-allow'; then
    return 0
  fi
  local prev
  prev="$(sed -n "$((lnum - 1))p" "$file" 2>/dev/null || true)"
  printf '%s' "$prev" | grep -q 'permission-surface-allow'
}

scan() {
  local rule="$1" pattern="$2" description="$3"
  shift 3
  local skip_paths=("$@")

  local hits
  hits="$(grep -rnE "$pattern" --include='*.go' "${excludes[@]}" . 2>/dev/null || true)"
  [ -z "$hits" ] && return 0

  while IFS= read -r line; do
    [ -z "$line" ] && continue
    local file="${line%%:*}"
    local rest="${line#*:}"
    local lnum="${rest%%:*}"
    local text="${rest#*:}"
    local clean="${file#./}"

    if is_suppressed "$file" "$lnum" "$text"; then
      suppressed=$((suppressed + 1))
      continue
    fi

    local skip=0
    for allowed in "${skip_paths[@]}"; do
      case "$clean" in
      $allowed)
        skip=1
        break
        ;;
      esac
    done
    if [ "$skip" -eq 1 ]; then
      continue
    fi

    if [ "$fail" -eq 0 ]; then
      echo "FAIL: permission-surface violations"
    fi
    fail=1
    report "$clean" "$lnum" "$rule — $description"
  done <<<"$hits"
}

# Rule 1: grants written into the in-memory enforcer are not events, so they do
# not survive a restart, a projection rebuild, or a second instance.
scan "ephemeral-grant" \
  'AddGroupPolicy\(' \
  "grants must go through Service.GrantRoles / AddMember; ApplyGroupPolicy is the projection-only write path" \
  'identity-model/*'

# Rule 2: the untyped escape hatch bypasses the authorization contract entirely.
scan "untyped-enforce" \
  'EnforceAny\(' \
  "EnforceAny takes untyped arguments; use Decide/Check so the question is auditable" \
  'identity-model/*'

# Rule 3: calling the engine directly re-derives arity, domain, and error
# handling at each site — the class that produced 503-instead-of-403.
scan "raw-enforce" \
  '\.Enforce\(' \
  "call the engine through Decide/Check/EnforceIn, not directly" \
  'identity-model/*' \
  'authz.go'

# Rule 4: a denial sentinel whose code is not the shared one is invisible to
# cqrshtmx.IsForbidden and to every cross-module classifier.
while IFS= read -r line; do
  [ -z "$line" ] && continue
  file="${line%%:*}"
  rest="${line#*:}"
  lnum="${rest%%:*}"
  text="${rest#*:}"
  clean="${file#./}"

  if is_suppressed "$file" "$lnum" "$text"; then
    suppressed=$((suppressed + 1))
    continue
  fi

  # Allowed: the canonical codes themselves (with or without the constants).
  if printf '%s' "$text" | grep -qE 'NewRejection\((cqrshtmx\.)?Code(Forbidden|Unauthorized)|NewRejection\("(forbidden|unauthorized)"'; then
    continue
  fi

  if [ "$fail" -eq 0 ]; then
    echo "FAIL: permission-surface violations"
  fi
  fail=1
  report "$clean" "$lnum" \
    "split-sentinel — denial codes must be exactly \"forbidden\" / \"unauthorized\" (use CodeForbidden/CodeUnauthorized) so IsForbidden sees them"
done <<<"$(grep -rnE 'NewRejection\("[^"]*(forbidden|unauthorized)' --include='*.go' "${excludes[@]}" . 2>/dev/null || true)"

if [ "$fail" -ne 0 ]; then
  echo
  echo "Suppressed findings: $suppressed (each needs a reason next to permission-surface-allow)"
  exit 1
fi

echo "OK: no permission-surface anti-patterns (suppressed: $suppressed)"
