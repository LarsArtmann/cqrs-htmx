#!/usr/bin/env bash
# cqrs-htmx batch release — strips internal replaces, resolves pseudo-versions, tags.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

modules=(
  "."
  "identity-model"
  "usermgmt"
  "usermgmt/oauth2"
  "usermgmt/totp"
  "usermgmt/webauthn"
  "adminui"
  "loginpage"
  "dashboardui"
)
versions=(
  "v4.5.0"
  "v4.1.0"
  "v4.5.0"
  "v4.5.0"
  "v4.5.0"
  "v4.5.0"
  "v4.5.0"
  "v4.5.0"
  "v4.0.0"
)
descriptions=(
  "Event catalog, projection status, SSE broadcasting, typed handlers, dashboardui module"
  "Upcaster registry, enhanced fold operations"
  "Projection health monitoring, event catalog, identity-model integration"
  "Lockstep v4.5.0 alignment"
  "Lockstep v4.5.0 alignment"
  "Lockstep v4.5.0 alignment"
  "Explicit root+usermgmt dependencies, lockstep alignment"
  "Explicit root+usermgmt dependencies, lockstep alignment"
  "First release: CQRS/ES observability dashboard with SSE real-time updates"
)

tags=()
for i in "${!modules[@]}"; do
  mod="${modules[$i]}"
  ver="${versions[$i]}"
  if [ "$mod" = "." ]; then
    tag="$ver"
  else
    tag="${mod}/${ver}"
  fi
  if git tag -l "$tag" | grep -q .; then
    echo "ERROR: tag $tag already exists"
    exit 1
  fi
  tags+=("$tag")
done

echo "Releasing ${#tags[@]} modules:"
for i in "${!tags[@]}"; do
  echo "  ${tags[$i]}: ${descriptions[$i]}"
done

# Verify clean tree
if ! git diff-index --quiet HEAD --; then
  echo "ERROR: working tree has uncommitted changes"
  git status --short
  exit 1
fi

# --- Strip internal cqrs-htmx replace directives ---
echo "Stripping internal replace directives..."
backup_dir="$(mktemp -d)"
trap 'rm -rf "$backup_dir"' EXIT

find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  mkdir -p "$backup_dir/$dir"
  cp "$gomod" "$backup_dir/$gomod"
  # Also back up go.sum so it can be restored after tagging
  if [ -f "$dir/go.sum" ]; then
    cp "$dir/go.sum" "$backup_dir/$dir/go.sum"
  fi
  while IFS= read -r replace_line; do
    replace_path=$(echo "$replace_line" | grep -oP 'github\.com/larsartmann/\S+' | head -1 || true)
    if [ -n "$replace_path" ]; then
      (cd "$dir" && go mod edit "-dropreplace=${replace_path}" 2>/dev/null || true)
    fi
  done < <(grep '=>' "$gomod" 2>/dev/null || true)
done

# Re-resolve requires
echo "Re-resolving requires (go mod tidy with replaces stripped)..."
find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -not -path './examples/*' -not -path './integration_test/*' | while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  echo "  tidying $dir..."
  (cd "$dir" && GOWORK=off GOEXPERIMENT=jsonv2 go mod tidy -e 2>/dev/null || true)
done

# Verify no pseudo-versions in tagged modules
echo "Verifying no pseudo-versions remain..."
for mod in "${modules[@]}"; do
  if grep -q "00010101000000" "${mod}/go.mod" 2>/dev/null; then
    echo "WARNING: ${mod}/go.mod still has pseudo-versions"
    grep "00010101000000" "${mod}/go.mod"
  fi
done

# Create temporary commit
git add -A
git commit -m "chore(release): strip replace directives for batch release" --no-verify 2>/dev/null || true

# Create tags
for i in "${!tags[@]}"; do
  tag="${tags[$i]}"
  desc="${descriptions[$i]}"
  echo "Creating tag: $tag"
  git tag -a "$tag" -m "${tag}: ${desc}"
  echo "  ✓ $tag"
done

# Restore originals
echo "Restoring original go.mod files..."
find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  if [ -f "$backup_dir/$gomod" ]; then
    cp "$backup_dir/$gomod" "$gomod"
  fi
done
find . -name go.sum -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gosum; do
  if [ -f "$backup_dir/$gosum" ]; then
    cp "$backup_dir/$gosum" "$gosum"
  fi
done

# Undo the temporary release commit without touching the restored working tree.
# --mixed resets both HEAD and the index to the parent, leaving working tree
# files (restored from backup above) intact.  Do NOT use --soft + checkout
# here — that would overwrite the restored originals with stripped-replace
# versions from the index.
git reset --mixed HEAD~1

echo ""
echo "Created ${#tags[@]} tags. To push:"
echo "  git push origin ${tags[*]}"
