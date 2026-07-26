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
  "v4.6.0"
  "v4.1.0"
  "v4.6.0"
  "v4.6.0"
  "v4.6.0"
  "v4.6.0"
  "v4.6.0"
  "v4.6.0"
  "v4.1.0"
)
descriptions=(
  "HTMX redirect helpers, dedup sweep, dashboardui SSE bridge, dashboard-demo pseudo-version fix"
  "Upcaster registry, enhanced fold operations"
  "Projection health monitoring, event catalog, identity-model integration"
  "Lockstep v4.6.0 alignment"
  "Lockstep v4.6.0 alignment"
  "Lockstep v4.6.0 alignment"
  "Explicit root+usermgmt dependencies, lockstep alignment"
  "Explicit root+usermgmt dependencies, lockstep alignment"
  "Dead code cleanup (notImplemented, renderStatCardsTempl), SSE bridge improvements"
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

# --- Resolve internal cqrs-htmx requires to target versions ---
# We deliberately do NOT run `go mod tidy` here: with GOWORK=off it silently
# removes dependencies whose published tags are broken (go-cqrs-lite publishing
# bug). Instead, we surgically update each internal require to its target
# version using `go mod edit -require=`. This fixes the root cause of
# inter-module version drift: replace directives mask stale require versions
# during development, and stripping replaces without re-resolving exposes them.
echo "Resolving internal requires to target versions..."

# Build module-path → version mapping from the modules/versions arrays.
mapping_file="$(mktemp)"
for i in "${!modules[@]}"; do
  mod="${modules[$i]}"
  ver="${versions[$i]}"
  mod_path=$(head -1 "${mod}/go.mod" | awk '{print $2}')
  if [ -n "$mod_path" ]; then
    printf '%s\t%s\n' "$mod_path" "$ver" >> "$mapping_file"
  fi
done

# For every go.mod, update internal cqrs-htmx requires to their target versions.
find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | while IFS= read -r gomod; do
  dir="$(dirname "$gomod")"
  own_path=$(head -1 "$gomod" | awk '{print $2}')
  while IFS= read -r req_path; do
    # Skip self-references (the module's own path)
    [ "$req_path" = "$own_path" ] && continue
    target_ver=$(grep -P "^${req_path}\t" "$mapping_file" | cut -f2)
    if [ -n "$target_ver" ]; then
      (cd "$dir" && go mod edit "-require=${req_path}@${target_ver}")
      echo "  ${dir}/go.mod: ${req_path} → ${target_ver}"
    fi
  done < <(grep -oP 'github\.com/larsartmann/cqrs-htmx/\S+' "$gomod" | sort -u)
done
rm -f "$mapping_file"

# Verify no pseudo-versions remain in ANY go.mod (not just tagged modules).
# Catches both zero-date (00010101000000) and date-based (YYYYMMDDHHMMSS) forms.
echo "Verifying no pseudo-versions remain..."
while IFS= read -r gomod; do
  if grep -qP 'v[0-9]+\.[0-9]+\.[0-9]+-[0-9]{14}' "$gomod" 2>/dev/null; then
    echo "ERROR: ${gomod} still has pseudo-versions"
    grep -nP 'v[0-9]+\.[0-9]+\.[0-9]+-[0-9]{14}' "$gomod"
    exit 1
  fi
done < <(find . -name go.mod -not -path './vendor/*' -not -path './.git/*')

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
