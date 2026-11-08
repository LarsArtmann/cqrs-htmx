#!/usr/bin/env bash
# check-version-drift.sh — Detect sibling modules referencing different versions
# Adapted from go-cqrs-lite's CI version drift detection.
#
# DIVISION OF LABOR vs scripts/checks/check-release-train.sh: THIS script answers
# "is every internal dependency referenced at ONE version repo-wide, and do
# those tags exist?" (--strict = both legs blocking). check-release-train
# answers "is each require on the LATEST published version (train lag), and
# which requires are excused by local replaces?" Run both (nix run
# .#check-modules runs drift; --report runs both).
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
# Usage: ./scripts/checks/check-version-drift.sh [--strict] [--refresh-cache] [--no-cache]
# Exit: 0 = no drift + all tags verified, 1 = drift or missing tag (strict),
#       4 = tags UNVERIFIABLE (ls-remote starved/offline, outside CI) — not a verdict
# Requires: network access to github.com (git ls-remote) on cache miss.
#
# Starvation defense (2026-11-08, the multi-session 403 incident): the ~25-repo
# tag burst rode GitHub's anonymous smart-HTTP quota, which 403s under
# concurrent gate runs — and git only attempts credentials on 401, so the
# configured libsecret/gh credentials never engaged. Three layers:
#   (1) pre-auth: when `git credential fill` resolves github.com credentials,
#       ls-remote uses https://larsartmann@github.com/... so the burst rides
#       the authenticated quota (GIT_TERMINAL_PROMPT=0 keeps credential-less
#       hosts on the anonymous path — never a prompt hang);
#   (2) shared tag cache: reads/writes the release-train gate's 900s cache
#       (TRAIN_TAG_CACHE_DIR) — a push-hook run makes O(new-repos) requests,
#       not O(all); a cached MISS is re-verified LIVE before a tag is declared
#       missing (the just-pushed-tag race, gotcha 27's class);
#   (3) starvation exit: unfetchable tags are UNVERIFIED, never MISSING —
#       exit 4 outside CI (the pre-push hook warns and proceeds; CI backstops),
#       downgraded to a warning under CI=true (the train gate's shape).

set -euo pipefail

# --strict: exit 1 on drift/missing tags (for blocking CI). Default: warn only.
# --refresh-cache: bypass cache READS, re-fetch every repo, rewrite the cache.
# --no-cache: bypass the tag cache entirely (reads and writes).
STRICT=false
REFRESH=0
NO_CACHE=0
for arg in "${@:-}"; do
  case "$arg" in
    --strict) STRICT=true ;;
    --refresh-cache) REFRESH=1 ;;
    --no-cache) NO_CACHE=1 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# TEST HOOK: self-tests override the scanned root with a fixture tree.
REPO_ROOT="${DRIFT_ROOT:-$REPO_ROOT}"
cd "$REPO_ROOT"

# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../lib/replace-exemption.sh"

# Shared tag cache with check-release-train.sh (same env names, same files).
CACHE_DIR="${TRAIN_TAG_CACHE_DIR:-/tmp/cqrs-htmx-tag-cache}"
CACHE_TTL="${TRAIN_TAG_CACHE_TTL:-900}"

# Credential probe (once per run): if a helper can fill github.com credentials,
# ls-remote rides the authenticated quota instead of the starvation-prone
# anonymous one. Prompt-disabled so credential-less hosts fail fast, not hang.
CRED_OK=false
if printf 'protocol=https\nhost=github.com\n\n' | GIT_TERMINAL_PROMPT=0 git credential fill >/dev/null 2>&1; then
  CRED_OK=true
fi

ls_remote_tags() { # <repo> -> sorted unique tags on stdout
  local repo="$1" url="https://github.com/larsartmann/${1}.git"
  if [[ $CRED_OK == "true" ]]; then
    url="https://larsartmann@github.com/larsartmann/${repo}.git"
  fi
  GIT_TERMINAL_PROMPT=0 git ls-remote --tags "$url" 2>/dev/null |
    sed -e 's|.*refs/tags/||' -e 's/\^{}$//' | sort -u
}

# Collect all internal module paths (from go.mod module declarations)
declare -A MODULE_VERSIONS
ALL_PAIRS=""

echo "=== Version Drift Check ==="
echo ""

# Find all go.mod files
for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -not -path '*/testdata/*' | sort); do
  moddir=$(dirname "$modfile")
  # Extract all require entries that reference larsartmann modules
  while IFS= read -r line; do
    # Parse "github.com/larsartmann/.../vX vY.Z.W"
    mod_path=$(echo "$line" | awk '{print $1}')
    version=$(echo "$line" | awk '{print $2}')

    if [[ $mod_path =~ ^github\.com/larsartmann/ ]]; then
      # A local replace satisfies this require locally — the replaced source
      # wins at build time, so the recorded version string is cosmetic and
      # cannot drift against siblings (same exemption the tag check applies;
      # single-sourced in scripts/lib/replace-exemption.sh).
      if replace_exemption_applies "$moddir" "$mod_path"; then
        continue
      fi
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
declare -A REPO_TAGS_FROM_CACHE

cache_fresh() { # <cache_file>
  [ "$NO_CACHE" = 0 ] && [ "$REFRESH" = 0 ] && [ -f "$1" ] &&
    [ -z "$(find "$1" -mmin "+$((CACHE_TTL / 60))" 2>/dev/null)" ]
}

write_cache() { # <repo> <tags>
  if [[ $NO_CACHE -eq 0 ]]; then
    mkdir -p "$CACHE_DIR"
    printf '%s\n' "$2" >"$CACHE_DIR/$1.tags" 2>/dev/null || true
  fi
}

# Fetch the tag list of a larsartmann repo once per run (cache-first).
# Sets REPO_TAGS_LIST[repo] + REPO_TAGS_FROM_CACHE[repo]; on failure leaves
# REPO_TAGS_OK[repo]=false (the caller counts it UNVERIFIED, never MISSING).
fetch_repo_tags() {
  local repo="$1"
  if [[ -v "REPO_TAGS_OK[$repo]" ]]; then
    return 0
  fi
  local cache_file="$CACHE_DIR/$repo.tags"
  if cache_fresh "$cache_file"; then
    REPO_TAGS_LIST[$repo]="$(cat "$cache_file")"
    REPO_TAGS_FROM_CACHE[$repo]=true
    REPO_TAGS_OK[$repo]=true
    return 0
  fi
  local tags
  if ! tags=$(ls_remote_tags "$repo"); then
    sleep 2 # starvation/403 class: one retry with backoff before giving up
    if ! tags=$(ls_remote_tags "$repo"); then
      REPO_TAGS_OK[$repo]=false
      REPO_TAGS_LIST[$repo]=""
      return 0
    fi
  fi
  REPO_TAGS_OK[$repo]=true
  REPO_TAGS_LIST[$repo]="$tags"
  REPO_TAGS_FROM_CACHE[$repo]=false
  write_cache "$repo" "$tags"
}

# Live refetch (bypasses the cache read, rewrites it) — the stale-cache guard
# calls this before any tag is declared MISSING.
refresh_repo_tags() {
  local repo="$1" tags
  if ! tags=$(ls_remote_tags "$repo"); then
    REPO_TAGS_OK[$repo]=false
    REPO_TAGS_LIST[$repo]=""
    return 0
  fi
  REPO_TAGS_OK[$repo]=true
  REPO_TAGS_LIST[$repo]="$tags"
  REPO_TAGS_FROM_CACHE[$repo]=false
  write_cache "$repo" "$tags"
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
unverified=0
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
    echo "  WARN: cannot list tags of github.com/larsartmann/$repo (ls-remote failed — UNVERIFIED, not missing)"
    unverified=$((unverified + 1))
    continue
  fi

  expected=$(expected_tag_for "$mod_path" "$version")
  checked=$((checked + 1))
  if ! grep -qxF "$expected" <<<"${REPO_TAGS_LIST[$repo]}"; then
    # Stale-cache guard: a tag pushed after this repo's cache entry was
    # written would read as missing. Refetch LIVE before declaring it.
    if [[ ${REPO_TAGS_FROM_CACHE[$repo]:-false} == "true" ]]; then
      refresh_repo_tags "$repo"
      if [[ ${REPO_TAGS_OK[$repo]} != "true" ]]; then
        echo "  WARN: cannot verify $mod_path@$version — $repo unfetchable on refetch (ls-remote failed — UNVERIFIED)"
        unverified=$((unverified + 1))
        continue
      fi
    fi
    if ! grep -qxF "$expected" <<<"${REPO_TAGS_LIST[$repo]}"; then
      echo "  MISSING TAG: $mod_path@$version needs tag '$expected' in larsartmann/$repo"
      echo "               (required by $origin_dir)"
      missing=$((missing + 1))
    else
      echo "  NOTE: '$expected' absent from stale cache — live refetch confirmed it (cache rewritten)"
    fi
  fi
done <<<"$ALL_PAIRS"

if [[ $missing -eq 0 && $unverified -eq 0 ]]; then
  echo "✓ All $checked larsartmann requires resolve to published tags ($skipped skipped: pseudo-version or locally replaced)"
else
  if [[ $missing -gt 0 ]]; then
    echo ""
    echo "✗ $missing larsartmann require(s) reference tags that do not exist upstream"
    echo "  Cut+push the tag, or pin back to a published version."
    echo "  $skipped skipped (pseudo-version or locally replaced)."
    if [[ $STRICT == "true" ]]; then
      exit 1
    fi
  fi
  if [[ $unverified -gt 0 ]]; then
    echo ""
    echo "⚠ $unverified repo(s) UNVERIFIED (ls-remote starved or offline) — not a verdict."
    echo "  Retry shortly; the shared tag cache may already be warm on the next run."
    if [[ ${CI:-false} != "true" ]]; then
      exit 4
    fi
    echo "  (CI: downgraded to warning — the train gate's advisory shape.)"
  fi
fi
