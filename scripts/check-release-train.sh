#!/usr/bin/env bash
# check-release-train.sh — verify internal requires against PUBLISHED tags,
# with release-train planning output.
#
# Complements check-version-drift.sh:
#   - drift script: cross-module consistency (same module referenced at the
#     same version everywhere) + tag existence (exit 1 on missing tag).
#   - THIS script: version-vs-reality per internal dependency, classified:
#       UNPUBLISHED — require points at a tag that does not exist on the
#                     module's remote repo (the examples/admin-demo
#                     `usermgmt/totp/v4 v4.8.0` class that poisoned the
#                     workspace graph 2026-08-17..29). HARD FAIL unless the
#                     requiring go.mod carries a local replace for it.
#       TRAIN LAG   — a NEWER published tag exists for that module path;
#                     advisory list for the next coordinated family train.
#
# Exemptions (same rule as the drift script): a local `replace` for the
# module in the requiring go.mod satisfies the build locally — UNPUBLISHED
# is downgraded to a warning (the require is annotated, e.g. setup's
# DEV-ONLY usermgmt replace); TRAIN LAG is still reported for planning.
#
# Usage: ./scripts/check-release-train.sh [--json] [--strict-lag N]
#                                         [--no-cache] [--refresh-cache]
#   --json           machine-readable summary (stable shape, for tooling)
#   --strict-lag N   exit 3 when train lag exceeds N (default: lag advisory)
#   --no-cache       bypass the persistent ls-remote tag cache entirely
#   --refresh-cache  force re-fetch every repo's tags, then rewrite cache
#
# Tag cache: git ls-remote costs ~30s across the family repos on a cold
# run, which made the check unusable as a pre-commit gate. Tag lists are
# cached under TRAIN_TAG_CACHE_DIR (default /tmp/cqrs-htmx-tag-cache) for
# TRAIN_TAG_CACHE_TTL seconds (default 900); a warm run costs ~1s. Cache
# misses only ever widen what is checked (a stale cache reports MORE lag,
# never fewer UNPUBLISHED), so the fail-closed property is preserved.
#
# Exit: 0 = no UNPUBLISHED (and lag within --strict-lag)
#       1 = at least one UNPUBLISHED require (or ls-remote failure off-CI)
#       2 = usage error
#       3 = lag exceeds --strict-lag N
#       4 = ls-remote failure outside CI (cannot check, not a verdict)
# Requires: network access to github.com (git ls-remote) on cache miss.
# CI=true downgrades ls-remote failures to warnings (advisory mode there).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

JSON=0
STRICT_LAG=-1
NO_CACHE=0
REFRESH=0
while [ $# -gt 0 ]; do
  case "$1" in
  --json) JSON=1 ;;
  --strict-lag)
    shift
    STRICT_LAG="${1:?--strict-lag requires a number}"
    [[ $STRICT_LAG =~ ^[0-9]+$ ]] || {
      echo "check-release-train: --strict-lag must be a non-negative integer, got '$STRICT_LAG'" >&2
      exit 2
    }
    ;;
  --no-cache) NO_CACHE=1 ;;
  --refresh-cache) REFRESH=1 ;;
  *)
    echo "check-release-train: unknown option '$1'" >&2
    exit 2
    ;;
  esac
  shift
done

CACHE_DIR="${TRAIN_TAG_CACHE_DIR:-/tmp/cqrs-htmx-tag-cache}"
CACHE_TTL="${TRAIN_TAG_CACHE_TTL:-900}"

# Classification lines are the human report; in --json mode stdout must be
# pure JSON, so they move to stderr.
emit() {
  if [ "$JSON" = 1 ]; then
    echo "$*" >&2
  else
    echo "$*"
  fi
}

if [ "$JSON" = 0 ]; then
  echo "=== Release Train Check ==="
  echo ""
fi

declare -A REPO_TAGS_OK
declare -A REPO_TAGS_LIST
CACHE_MAX_AGE=-1 # seconds; oldest cache entry actually served this run (-1 = cache unused)

fetch_repo_tags() {
  local repo="$1"
  if [[ -v "REPO_TAGS_OK[$repo]" ]]; then
    return 0
  fi

  local cache_file="$CACHE_DIR/$repo.tags"
  if [ "$NO_CACHE" = 0 ] && [ "$REFRESH" = 0 ] &&
    [ -f "$cache_file" ] &&
    [ -z "$(find "$cache_file" -mmin "+$((CACHE_TTL / 60))" 2>/dev/null)" ]; then
    REPO_TAGS_OK[$repo]=true
    REPO_TAGS_LIST[$repo]="$(cat "$cache_file")"
    local age=$(($(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || date +%s)))
    [ "$age" -gt "$CACHE_MAX_AGE" ] && CACHE_MAX_AGE=$age
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
  if [ "$NO_CACHE" = 0 ]; then
    mkdir -p "$CACHE_DIR"
    printf '%s\n' "$tags" >"$cache_file" 2>/dev/null || true
  fi
}

# Map a module path to the tag-prefix its versions live under.
#   github.com/larsartmann/cqrs-htmx/v4                 -> ""
#   github.com/larsartmann/cqrs-htmx/usermgmt/v4        -> "usermgmt/"
#   github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4   -> "usermgmt/totp/"
tag_prefix_for() {
  local mod_path="$1"
  local rest="${mod_path#github.com/larsartmann/}"
  local repo="${rest%%/*}"
  local sub=""
  if [[ $rest == *"/"* && $rest != "$repo" ]]; then
    sub="${rest#"${repo}"/}"
  fi
  if [[ $sub =~ /v[0-9]+$ ]]; then
    sub="${sub%/v[0-9]*}"
  fi
  if [[ $sub =~ ^v[0-9]+$ ]]; then
    sub=""
  fi
  if [[ -z $sub ]]; then
    echo ""
  else
    echo "$sub/"
  fi
}

# Max published vX.Y.Z for a tag prefix ("usermgmt/" -> "usermgmt/v4.8.1").
# sort -V gives correct semver order. Empty output when the prefix has no
# published tags (guarded against pipefail: grep exits 1 on no match).
max_published_for() {
  local prefix="$1" tags="$2"
  { grep -E "^${prefix}v[0-9]+\.[0-9]+\.[0-9]+$" <<<"$tags" || true; } | sort -V | tail -1
}

# semver ordering via sort -V: returns 0 when $1 > $2.
version_gt() {
  [[ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -1)" == "$1" && $1 != "$2" ]]
}

unpublished=0
lags=0
checked=0
exempted=0
LAG_ENTRIES=()

for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -not -path '*/testdata/*' | sort); do
  moddir=$(dirname "$modfile")

  while IFS= read -r line; do
    mod_path=$(echo "$line" | awk '{print $1}')
    version=$(echo "$line" | awk '{print $2}')

    [[ $mod_path =~ ^github\.com/larsartmann/ ]] || continue
    [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || continue # skip pseudo-versions etc.

    rest="${mod_path#github.com/larsartmann/}"
    repo="${rest%%/*}"
    fetch_repo_tags "$repo"

    if [[ ${REPO_TAGS_OK[$repo]} != "true" ]]; then
      if [[ ${CI:-false} == "true" ]]; then
        emit "  WARN: cannot list tags of $repo (ls-remote failed; CI advisory)"
        continue
      fi
      emit "  ERROR: cannot list tags of github.com/larsartmann/$repo (ls-remote failed)"
      exit 4
    fi

    prefix=$(tag_prefix_for "$mod_path")
    max=$(max_published_for "$prefix" "${REPO_TAGS_LIST[$repo]}")
    checked=$((checked + 1))

    # Local replace in the requiring go.mod satisfies the build hermetically.
    replace_exempt=false
    if grep -qE "^[[:space:]]*replace[[:space:]]+${mod_path}([[:space:]]|v[0-9])" "$modfile" 2>/dev/null; then
      replace_exempt=true
    fi

    local_label="${moddir#./}"
    if [[ -z $max ]]; then
      if [[ $replace_exempt == "true" ]]; then
        emit "  EXEMPT (local replace): $mod_path@$version (no published tags for this path) — required by $local_label"
        exempted=$((exempted + 1))
      else
        emit "  UNPUBLISHED: $mod_path@$version (no published tags at all) — required by $local_label"
        unpublished=$((unpublished + 1))
      fi
      continue
    fi

    maxv="${max#"${prefix}"}"
    if version_gt "$version" "$maxv"; then
      if [[ $replace_exempt == "true" ]]; then
        emit "  EXEMPT (local replace): $mod_path@$version > published $maxv — required by $local_label"
        exempted=$((exempted + 1))
      else
        emit "  UNPUBLISHED: $mod_path@$version > published $maxv — required by $local_label"
        unpublished=$((unpublished + 1))
      fi
    elif [[ $version != "$maxv" ]]; then
      emit "  TRAIN LAG: $mod_path@$version but $maxv is published — required by $local_label"
      lags=$((lags + 1))
      LAG_ENTRIES+=("{\"module\":\"$mod_path\",\"required\":\"$version\",\"latest\":\"$maxv\",\"required_by\":\"$local_label\"}")
    fi
  done < <(cd "$moddir" && awk '
        /^require \(/ { in_req=1; next }
        /^\)/ { in_req=0 }
        in_req && /^\t/ && /github\.com\/larsartmann\// { print $0 }
        /^require[ \t]+github\.com\/larsartmann\// { $1=""; print "\t" $0 }
    ' go.mod 2>/dev/null)
done

if [ "$JSON" = 1 ]; then
  lag_json="$(
    IFS=$'\n'
    if [ ${#LAG_ENTRIES[@]} -gt 0 ]; then printf '%s\n' "${LAG_ENTRIES[*]}" | paste -sd, -; fi
  )"
  printf '{"checked":%d,"unpublished":%d,"exempted":%d,"lag":%d,"lag_entries":[%s],"strict_lag":%d,"cache_age_seconds":%d,"ok":%s}\n' \
    "$checked" "$unpublished" "$exempted" "$lags" "$lag_json" "$STRICT_LAG" "$CACHE_MAX_AGE" \
    "$([ "$unpublished" -eq 0 ] && echo true || echo false)"
else
  echo ""
  echo "Checked $checked internal requires: $unpublished unpublished, $exempted replace-exempted, $lags train lag."
  if [ "$CACHE_MAX_AGE" -gt 0 ] && [ "$CACHE_MAX_AGE" -gt $(((CACHE_TTL * 3) / 4)) ]; then
    echo "  NOTE: tag cache is ${CACHE_MAX_AGE}s old (TTL ${CACHE_TTL}s) — consider --refresh-cache before a release call."
  fi
fi

if [ "$REFRESH" = 1 ] && [ "$NO_CACHE" = 0 ]; then
  echo "Tag cache refreshed at $CACHE_DIR (TTL ${CACHE_TTL}s)." >&2
fi

if [[ $unpublished -gt 0 ]]; then
  [ "$JSON" = 0 ] && {
    echo "✗ UNPUBLISHED requires present — cut/push the missing tags or pin back to published versions."
    echo "  (Workspace mode masks these; hermetic GOWORK=off builds fail.)"
  }
  exit 1
fi

if [ "$STRICT_LAG" -ge 0 ] && [ "$lags" -gt "$STRICT_LAG" ]; then
  [ "$JSON" = 0 ] && echo "✗ Train lag $lags exceeds --strict-lag $STRICT_LAG — run the family alignment pass."
  exit 3
fi

[ "$JSON" = 0 ] && echo "✓ No unpublished requires. Train-lag entries above are the next family train's alignment list."
exit 0
