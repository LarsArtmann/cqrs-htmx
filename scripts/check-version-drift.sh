#!/usr/bin/env bash
# check-version-drift.sh — Detect sibling modules referencing different versions
# Adapted from go-cqrs-lite's CI version drift detection.
#
# Catches: module A requires sibling at v3.3.0 while module B requires v3.4.0.
# In go.work mode this is masked; under GOWORK=off it causes build failures.
#
# Also verifies (--tag-existence, always on) that every github.com/larsartmann/*
# require resolves to a tag that actually exists on the module's remote repo —
# catches phantom version bumps to never-published tags (e.g. a tool rewriting
# a require to v4.8.0 before that tag is cut). Requires referencing a module
# are exempt when that go.mod carries a local `replace` for it (the replace
# satisfies the build locally; the require version is then not proxy-resolved).
#
# Usage: ./scripts/check-version-drift.sh [--strict]
# Exit: 0 = no drift + all tags exist, 1 = drift or missing tag (strict), 0 + warning otherwise
# Requires: network access to github.com (git ls-remote).

set -euo pipefail

# --strict: exit 1 on drift/missing tags (for blocking CI). Default: warn only.
STRICT=false
if [[ ${1:-} == "--strict" ]]; then
  STRICT=true
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Collect all internal module paths (from go.mod module declarations)
declare -A MODULE_VERSIONS
ALL_PAIRS=""

echo "=== Version Drift Check ==="
echo ""

# Find all go.mod files
for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | sort); do
  moddir=$(dirname "$modfile")
  # Extract all require entries that reference larsartmann modules
  while IFS= read -r line; do
    # Parse "github.com/larsartmann/.../vX vY.Z.W"
    mod_path=$(echo "$line" | awk '{print $1}')
    version=$(echo "$line" | awk '{print $2}')

    if [[ $mod_path =~ ^github\.com/larsartmann/ ]]; then
      key="${mod_path} ${version}"
      MODULE_VERSIONS["$key"]+="${moddir},"
      ALL_PAIRS+="${moddir}|${mod_path}|${version}
"
    fi
  done < <(cd "$moddir" && awk '
        /^require \(/ { in_req=1; next }
        /^\)/ { in_req=0 }
        in_req && /^\t/ && /github\.com\/larsartmann\// { print $0 }
        /^require[ \t]+github\.com\/larsartmann\// { $1=""; print "\t" $0 }
    ' go.mod 2>/dev/null)
done

# Check for drift: same module path at different versions
declare -A MODULE_ALL_VERSIONS
for key in "${!MODULE_VERSIONS[@]}"; do
  mod_path=$(echo "$key" | awk '{print $1}')
  version=$(echo "$key" | awk '{print $2}')
  MODULE_ALL_VERSIONS["$mod_path"]+="${version},"
done

failed=0

for mod_path in "${!MODULE_ALL_VERSIONS[@]}"; do
  versions=$(echo "${MODULE_ALL_VERSIONS[$mod_path]}" | tr ',' '\n' | grep -v '^$' | sort -u)
  version_count=$(echo "$versions" | wc -l)

  if [[ $version_count -gt 1 ]]; then
    echo "  DRIFT: $mod_path referenced at multiple versions:"
    echo "$versions" | while read -r v; do
      echo "    - $v"
    done
    failed=1
  fi
done

if [[ $failed -eq 0 ]]; then
  echo "✓ No version drift detected"
else
  echo ""
  if [[ $STRICT == "true" ]]; then
    echo "✗ Version drift detected — siblings reference different versions"
    echo "  Fix: ensure all modules reference the same version of internal deps"
    exit 1
  else
    echo "⚠ Version drift detected (advisory mode — use --strict to fail)"
    echo "  Fix: ensure all modules reference the same version of internal deps"
  fi
fi

# ---------------------------------------------------------------------------
# Published-tag existence check
# ---------------------------------------------------------------------------

echo ""
echo "=== Published Tag Existence Check ==="

declare -A REPO_TAGS_OK
declare -A REPO_TAGS_LIST

# Fetch the tag list of a larsartmann repo once per run.
# Sets REPO_TAGS_LIST[repo]; on failure leaves REPO_TAGS_OK[repo]=false.
fetch_repo_tags() {
  local repo="$1"
  if [[ -v "REPO_TAGS_OK[$repo]" ]]; then
    return 0
  fi
  local tags
  if ! tags=$(git ls-remote --tags "https://github.com/larsartmann/${repo}.git" 2>/dev/null |
    sed -e 's|.*refs/tags/||' -e 's/\^{}$//' | sort -u); then
    REPO_TAGS_OK[$repo]=false
    REPO_TAGS_LIST[$repo]=""
    return 0
  fi
  REPO_TAGS_OK[$repo]=true
  REPO_TAGS_LIST[$repo]="$tags"
}

# Map a module path + version to the expected git tag in its repo.
#   github.com/larsartmann/<repo>               vX.Y.Z      -> vX.Y.Z
#   github.com/larsartmann/<repo>/<sub>/vN      vX.Y.Z      -> <sub>/vX.Y.Z
#   github.com/larsartmann/<repo>/<sub>         vX.Y.Z      -> <sub>/vX.Y.Z
expected_tag_for() {
  local mod_path="$1" version="$2"
  local rest="${mod_path#github.com/larsartmann/}"
  local repo="${rest%%/*}"
  local sub=""
  if [[ $rest == *"/"* && $rest != "$repo" ]]; then
    sub="${rest#"${repo}"/}"
  fi
  if [[ $sub =~ /v[0-9]+$ ]]; then
    sub="${sub%/v[0-9]*}"
  fi
  # Bare major suffix (root module of a /vN module path, e.g. cqrs-htmx/v4): no subpath.
  if [[ $sub =~ ^v[0-9]+$ ]]; then
    sub=""
  fi
  version="${version%+incompatible}"
  if [[ -z $sub ]]; then
    echo "$version"
  else
    echo "$sub/$version"
  fi
}

missing=0
checked=0
skipped=0
while IFS='|' read -r origin_dir mod_path version; do
  [[ -z $origin_dir ]] && continue

  # Pseudo-versions (vX.Y.Z-<timestamp>-<hash>) resolve via commits, not tags — skip.
  if [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+-[0-9]{14}-[0-9a-f]{12}$ ]]; then
    skipped=$((skipped + 1))
    continue
  fi

  # A local replace in the requiring go.mod satisfies the build locally — exempt.
  if [[ -f "$origin_dir/go.mod" ]] && grep -qE "^[[:space:]]*replace[[:space:]]+${mod_path}([[:space:]]|v[0-9])" "$origin_dir/go.mod"; then
    skipped=$((skipped + 1))
    continue
  fi

  rest="${mod_path#github.com/larsartmann/}"
  repo="${rest%%/*}"
  fetch_repo_tags "$repo"
  if [[ ${REPO_TAGS_OK[$repo]} != "true" ]]; then
    echo "  ERROR: cannot list tags of github.com/larsartmann/$repo (ls-remote failed)"
    missing=$((missing + 1))
    continue
  fi

  expected=$(expected_tag_for "$mod_path" "$version")
  checked=$((checked + 1))
  if ! grep -qxF "$expected" <<<"${REPO_TAGS_LIST[$repo]}"; then
    echo "  MISSING TAG: $mod_path@$version needs tag '$expected' in larsartmann/$repo"
    echo "               (required by $origin_dir)"
    missing=$((missing + 1))
  fi
done <<<"$ALL_PAIRS"

if [[ $missing -eq 0 ]]; then
  echo "✓ All $checked larsartmann requires resolve to published tags ($skipped skipped: pseudo-version or locally replaced)"
else
  echo ""
  echo "✗ $missing larsartmann require(s) reference tags that do not exist upstream"
  echo "  (or their repo was unreachable). Cut+push the tag, or pin back to a"
  echo "  published version. $skipped skipped (pseudo-version or locally replaced)."
  if [[ $STRICT == "true" ]]; then
    exit 1
  fi
fi
