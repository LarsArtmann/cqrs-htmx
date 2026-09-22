#!/usr/bin/env bash
# bump-dep.sh — sweep a family dependency to one version across the workspace
# (docs-health D3; runbook: docs/runbooks/dependency-train-bump.md).
#
# For every go.mod that requires a module matching <module-pattern>, set the
# require to <version>, then run hermetic tidy + build + vet per module.
# Digit-safe pattern matching (the `[a-z/-]+` class once silently skipped
# usermgmt/oauth2). Ends with the absence assertion the runbook mandates.
#
# Usage:
#   scripts/bump-dep.sh <module-substring-or-regex> <version> [--dry-run]
# Example:
#   scripts/bump-dep.sh 'larsartmann/go-cqrs-lite' v4.14.0
#   scripts/bump-dep.sh 'larsartmann/httputil$' v1.3.0   # $ = exact module only
# A trailing $ anchors the END of the module path — without it the pattern
# is a prefix and also sweeps sibling submodules with their own trains
# (httputil/server_timing and go-cqrs-lite/storage/memory both bit this way:
# each publishes tags on its own schedule, and bumping them to a sibling's
# version produces "unknown revision" download failures).
# Exit: 0 = all touched modules green; 1 = any failure (names the module)
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

if [ "$#" -lt 2 ]; then
  echo "usage: scripts/bump-dep.sh <module-substring-or-regex> <version> [--dry-run]" >&2
  exit 2
fi

PATTERN="$1"
VERSION="$2"
DRY=0
[ "${3:-}" = "--dry-run" ] && DRY=1

# shellcheck disable=SC1091
source scripts/lib/go-cache-env.sh || true

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "bump-dep: REFUSING to run on a dirty tree — the auto-commit daemon makes interleaved sweeps unreviewable. Commit or stash first." >&2
  exit 1
fi

# Digit-safe: match the full module path up to whitespace, no [a-z/-] traps.
# A trailing $ in PATTERN means exact-module: drop the subpath class so
# prefix patterns like 'httputil' cannot also match 'httputil/server_timing'.
TAIL_CLASS='[A-Za-z0-9/._-]*'
case "$PATTERN" in
  *'$')
    TAIL_CLASS=''
    PATTERN="${PATTERN%$}"
    ;;
esac
MATCH_RE="^[[:space:]]*github\.com/${PATTERN}${TAIL_CLASS}[[:space:]]+v[0-9][^[:space:]]*"

mods=()
while IFS= read -r modfile; do
  if grep -qE "$MATCH_RE" "$modfile"; then
    mods+=("$(dirname "$modfile")")
  fi
done < <(find . -name go.mod -not -path './.git/*' -not -path '*/testdata/*' -not -path './.githooks/*' | sort)

if [ "${#mods[@]}" -eq 0 ]; then
  echo "bump-dep: no go.mod requires a module matching '$PATTERN' — nothing to do"
  exit 0
fi

echo "bump-dep: sweeping $PATTERN -> $VERSION across ${#mods[@]} module(s)"
[ "$DRY" -eq 1 ] && printf '  (dry-run) %s\n' "${mods[@]}" && exit 0

fail=0
for mod in "${mods[@]}"; do
  log="/tmp/bump-dep-$(basename "$mod")-$$-$(date +%s).log"
  echo "==> $mod"
  (
    cd "$mod" || exit 1
    changed=0
    while IFS= read -r req; do
      path="$(echo "$req" | awk '{print $1}')"
      GOWORK=off go mod edit "-require=${path}@${VERSION}" || exit 2
      changed=1
    done < <(grep -E "$MATCH_RE" go.mod)
    [ "$changed" = 1 ] || exit 0
    GOWORK=off go mod tidy || exit 3
    GOWORK=off go build ./... || exit 4
    GOWORK=off go vet ./... || exit 5
  ) >"$log" 2>&1
  rc=$?
  if [ "$rc" -eq 0 ]; then
    echo "  PASS"
  else
    echo "  FAIL (rc=$rc) — log: $log"
    tail -5 "$log" | sed 's/^/      /'
    fail=1
  fi
done

echo ""
if grep -rE "$MATCH_RE" --include=go.mod . 2>/dev/null | grep -v '/testdata/' | grep -v "$VERSION" | grep -q .; then
  echo "bump-dep: ABSENCE ASSERTION FAILED — stale requires remain:"
  grep -rE "$MATCH_RE" --include=go.mod . | grep -v '/testdata/' | grep -v "$VERSION" | head -10
  exit 1
fi

if [ "$fail" -ne 0 ]; then
  echo "bump-dep: FAILED for at least one module (see logs above)"
  exit 1
fi
echo "bump-dep: OK — $PATTERN at $VERSION everywhere, all modules tidy+build+vet green"
echo "  next: check-release-train --refresh-cache --strict-lag 0, then commit"
