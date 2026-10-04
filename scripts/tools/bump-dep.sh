#!/usr/bin/env bash
# bump-dep.sh — sweep a family dependency to one version across the workspace
# (docs-health D3; runbook: docs/runbooks/dependency-train-bump.md).
#
# For every go.mod that requires a module matching <module-pattern>, set the
# require to <version>, then run hermetic tidy + `go mod verify` + build + vet
# per module. Digit-safe pattern matching (the `[a-z/-]+` class once silently
# skipped usermgmt/oauth2). Matches BOTH the block-require form
# (`\tgithub.com/x v1.0.0`) AND the single-line form
# (`require github.com/x v1.0.0`) — the 9b3c2e18 gap that needed a hand-fix
# during the v4.13.x train. Ends with the absence assertion the runbook mandates.
#
# Usage:
#   scripts/tools/bump-dep.sh <module-substring-or-regex> <version> \
#       [--dry-run] [--commit] [--no-verify]
# Example:
#   scripts/tools/bump-dep.sh 'larsartmann/go-cqrs-lite' v4.14.0
#   scripts/tools/bump-dep.sh 'larsartmann/httputil$' v1.3.0   # $ = exact module only
# A trailing $ anchors the END of the module path — without it the pattern
# is a prefix and also sweeps sibling submodules with their own trains
# (httputil/server_timing and go-cqrs-lite/storage/memory both bit this way:
# each publishes tags on its own schedule, and bumping them to a sibling's
# version produces "unknown revision" download failures).
#
# Options:
#   --dry-run     list the modules that would change (touch nothing, no network)
#   --commit      after a green sweep, stage the go.mod/go.sum changes and
#                 commit in-process — sweep + verification land as ONE commit
#                 (beats the auto-commit daemon shredding them apart)
#   --no-verify   skip the per-module `go mod verify` step (default: on)
#
# Env:
#   BUMP_DEP_ROOT  scan root override (fixture self-test). When set, the git
#                  dirty-check and --commit are skipped (fixtures are not repos).
#
# Exit: 0 = all touched modules green; 1 = any failure (names the module)
#
# shellcheck disable=SC2317

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

DRY=0
COMMIT=0
VERIFY=1
positional=()
for arg in "$@"; do
  case "$arg" in
  --dry-run) DRY=1 ;;
  --commit) COMMIT=1 ;;
  --no-verify) VERIFY=0 ;;
  *) positional+=("$arg") ;;
  esac
done

if [ "${#positional[@]}" -lt 2 ]; then
  echo "usage: scripts/tools/bump-dep.sh <module-substring-or-regex> <version> [--dry-run] [--commit] [--no-verify]" >&2
  exit 2
fi

PATTERN="${positional[0]}"
VERSION="${positional[1]}"

if [ -n "${BUMP_DEP_ROOT:-}" ]; then
  SCAN_ROOT="$BUMP_DEP_ROOT"
  cd "$SCAN_ROOT" || exit 1
else
  SCAN_ROOT="$REPO_ROOT"
  cd "$REPO_ROOT" || exit 1
  if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "bump-dep: REFUSING to run on a dirty tree — the auto-commit daemon makes interleaved sweeps unreviewable. Commit or stash first." >&2
    exit 1
  fi
fi

# shellcheck disable=SC1091
source "$REPO_ROOT/scripts/lib/go-cache-env.sh" || true

if [ "$COMMIT" -eq 1 ] && [ -n "${BUMP_DEP_ROOT:-}" ]; then
  echo "bump-dep: --commit ignored under BUMP_DEP_ROOT (fixture is not a repo)" >&2
  COMMIT=0
fi

# Digit-safe: match the full module path up to whitespace, no [a-z/-] traps.
# A trailing $ in PATTERN means exact-module: drop the subpath class so
# prefix patterns like 'httputil' cannot also match 'httputil/server_timing'.
# The leading alternation also accepts the single-line `require <mod> <ver>`
# form (column-0 keyword), which the original `^[[:space:]]*` regex missed.
TAIL_CLASS='[A-Za-z0-9/._-]*'
case "$PATTERN" in
*'$')
  TAIL_CLASS=''
  PATTERN="${PATTERN%$}"
  ;;
esac
MATCH_RE="^([[:space:]]*|require[[:space:]]+)github\.com/${PATTERN}${TAIL_CLASS}[[:space:]]+v[0-9][^[:space:]]*"

# Extract the module path from a matched require line: the single-line form
# leads with the `require` keyword, the block form is just the path.
require_path() {
  awk '{ if ($1 == "require") print $2; else print $1 }'
}

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
      path="$(echo "$req" | require_path)"
      GOWORK=off go mod edit "-require=${path}@${VERSION}" || exit 2
      changed=1
    done < <(grep -E "$MATCH_RE" go.mod)
    [ "$changed" = 1 ] || exit 0
    GOWORK=off go mod tidy || exit 3
    if [ "$VERIFY" = 1 ]; then
      GOWORK=off go mod verify || exit 6
    fi
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
echo "bump-dep: OK — $PATTERN at $VERSION everywhere, all modules tidy+verify+build+vet green"

if [ "$COMMIT" -eq 1 ]; then
  cd "$REPO_ROOT" || exit 1
  git add -u
  git commit -m "chore(deps): bump $PATTERN to $VERSION"
  echo "bump-dep: committed the sweep as one commit"
else
  echo "  next: check-release-train --refresh-cache --strict-lag 0, then commit"
fi
