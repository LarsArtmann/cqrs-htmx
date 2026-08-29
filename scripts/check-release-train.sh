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
# Usage: ./scripts/check-release-train.sh
# Exit:  0 = no UNPUBLISHED requires, 1 = at least one UNPUBLISHED require
#        (or ls-remote failure outside CI).
# Requires: network access to github.com (git ls-remote). CI=true downgrades
# ls-remote failures to warnings (private-repo auth is unverified there).

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "=== Release Train Check ==="
echo ""

declare -A REPO_TAGS_OK
declare -A REPO_TAGS_LIST

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

for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' | sort); do
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
        echo "  WARN: cannot list tags of $repo (ls-remote failed; CI advisory)"
        continue
      fi
      echo "  ERROR: cannot list tags of github.com/larsartmann/$repo (ls-remote failed)"
      exit 1
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
        echo "  EXEMPT (local replace): $mod_path@$version (no published tags for this path) — required by $local_label"
        exempted=$((exempted + 1))
      else
        echo "  UNPUBLISHED: $mod_path@$version (no published tags at all) — required by $local_label"
        unpublished=$((unpublished + 1))
      fi
      continue
    fi

    maxv="${max#"${prefix}"}"
    if version_gt "$version" "$maxv"; then
      if [[ $replace_exempt == "true" ]]; then
        echo "  EXEMPT (local replace): $mod_path@$version > published $maxv — required by $local_label"
        exempted=$((exempted + 1))
      else
        echo "  UNPUBLISHED: $mod_path@$version > published $maxv — required by $local_label"
        unpublished=$((unpublished + 1))
      fi
    elif [[ $version != "$maxv" ]]; then
      echo "  TRAIN LAG: $mod_path@$version but $maxv is published — required by $local_label"
      lags=$((lags + 1))
    fi
  done < <(cd "$moddir" && awk '
        /^require \(/ { in_req=1; next }
        /^\)/ { in_req=0 }
        in_req && /^\t/ && /github\.com\/larsartmann\// { print $0 }
        /^require[ \t]+github\.com\/larsartmann\// { $1=""; print "\t" $0 }
    ' go.mod 2>/dev/null)
done

echo ""
echo "Checked $checked internal requires: $unpublished unpublished, $exempted replace-exempted, $lags train lag."

if [[ $unpublished -gt 0 ]]; then
  echo "✗ UNPUBLISHED requires present — cut/push the missing tags or pin back to published versions."
  echo "  (Workspace mode masks these; hermetic GOWORK=off builds fail.)"
  exit 1
fi

echo "✓ No unpublished requires. Train-lag entries above are the next family train's alignment list."
