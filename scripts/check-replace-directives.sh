#!/usr/bin/env bash
# check-replace-directives.sh — Verify replace directives point at real modules
# Adapted from go-cqrs-lite's CI portability check.
#
# Two checks:
# 1. No absolute paths in go.mod replace directives (breaks portability:
#    they work on one machine but fail on CI, Docker, and other machines).
#    go.work is exempt — its absolute go-cqrs-lite paths are the established
#    local-dev pattern and go.work never ships to consumers.
# 2. Every local-filesystem replace target (in go.mod AND go.work) contains a
#    go.mod. Catches dead replaces after an upstream module is deleted, moved,
#    or extracted (e.g. the ADR-0128 go-cqrs-lite extraction deleted in-repo
#    shim modules whose directories still existed — only the go.mod was gone,
#    so a `-d` check would have passed; this broke every workspace build).
#    Targets INSIDE this repo fail hard everywhere (CI included). Sibling
#    targets (outside the repo, e.g. ../go-cqrs-lite) are machine-local and
#    skipped on CI runners (CI=true); the local `nix run .#check-modules`
#    gate verifies them.
# Usage: ./scripts/check-replace-directives.sh
# Exit: 0 = all replaces valid, 1 = invalid replace found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=== Replace Directive Check ==="
echo "Checking for absolute paths in go.mod replace directives..."
echo ""

failed=0

for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | sort); do
  # Check for absolute paths (starting with / on Unix)
  if grep -E 'replace.*=> */' "$modfile" 2>/dev/null; then
    echo "  ABSOLUTE PATH in $modfile"
    failed=1
  fi
done

if [[ $failed -eq 0 ]]; then
  echo "✓ All go.mod replace directives use relative paths"
else
  echo ""
  echo "✗ Absolute paths found in go.mod replace directives"
  echo "  Fix: use relative paths like '../sibling' instead of '/home/user/sibling'"
fi
echo ""
echo "Checking that every local replace target contains a go.mod..."

dead=0
check_targets() {
  local file="$1"
  local file_dir
  file_dir="$(dirname "$file")"
  # Extract local-path targets (start with '/', './', or '../') from both
  # single-line `replace A => target` and block `replace ( A => target )` forms.
  while IFS= read -r target; do
    [[ -z $target ]] && continue
    local resolved="$target"
    [[ $target == /* ]] || resolved="$file_dir/$target"
    local norm
    norm="$(realpath -m "$resolved")"
    if [[ $norm == "$REPO_ROOT" || $norm == "$REPO_ROOT"/* ]]; then
      # Target inside this repo: must have a go.mod everywhere (CI included).
      if [[ ! -f "$norm/go.mod" ]]; then
        echo "  DEAD REPLACE in $file: target '$target' has no go.mod (deleted/moved module?)"
        dead=1
      fi
    elif [[ ${CI:-} != "true" ]]; then
      # Target outside the repo (sibling checkout like ../go-cqrs-lite):
      # machine-local by design — siblings are not present on CI runners.
      # Verified by the local `nix run .#check-modules` gate instead.
      if [[ ! -f "$resolved/go.mod" ]]; then
        echo "  DEAD REPLACE in $file: sibling target '$target' has no go.mod (clone the sibling, or remove/fix the replace)"
        dead=1
      fi
    fi
  done < <(awk '
        /^replace \(/ { in_rep=1; next }
        /^\)/ { in_rep=0; next }
        in_rep && /=>/ { print $NF; next }
        /^replace .*=>/ { print $NF }
    ' "$file" | grep -E '^(\./|\.\./|/)' || true)
}

for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | sort); do
  check_targets "$modfile"
done
[[ -f go.work ]] && check_targets "go.work"

if [[ $dead -eq 0 ]]; then
  echo "✓ All local replace targets contain a go.mod"
else
  echo ""
  echo "✗ Dead replace targets found (point at directories without go.mod)"
  echo "  Fix: remove the replace, or repoint it at the module's new location"
  exit 1
fi

exit "$failed"
