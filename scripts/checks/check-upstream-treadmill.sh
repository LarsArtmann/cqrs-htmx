#!/usr/bin/env bash
# check-upstream-treadmill.sh — how far behind is every larsartmann UPSTREAM?
#
# The release-train gate answers "are cqrs-htmx's OWN module tags published?";
# this gate answers the mirror question for every NON-cqrs-htmx larsartmann
# dependency (go-cqrs-lite, httputil, templ-components, go-aichat, ...):
# for each (repo, tag-prefix) pin in the workspace's go.mod files, what is the
# latest published version and how many releases is the pin behind?
#
# Why this script exists (TODO S50): the 2026-10-12 A5 ad-hoc sweep died
# silently — a bad awk swallowed stderr and the sqliteengine/ tag-prefix
# surprise (tags live under metaengine/sqliteengine/, NOT sqliteengine/)
# produced a wrong-zero instead of an error. This gate makes every failure
# class loud and the whole sweep one command:
#   - unknown prefix (repo reachable, but zero tags under the derived prefix)
#     = structural ERROR, not a silent zero — the sqliteengine trap class;
#   - a fetch failure = exit 4 with the repo named — stderr is never swallowed;
#   - a non-exempt pseudo-version pin (consuming an untagged upstream commit)
#     = structural ERROR (replace-exempted pins are cosmetic, not checked).
#
# Verdict semantics (mirrors check-release-train.sh):
#   advisory (default): lag rows report, exit 0 — the nightly/CI surface;
#   --strict-lag N:     exit 3 when ANY prefix lags more than N releases
#                       (planning-time / deliberate-alignment surface).
#
# Tag cache: SHARED with check-release-train.sh (TRAIN_TAG_CACHE_DIR/TTL,
# same "<repo>.tags" file format) — a warm train run warms this gate.
#
# Usage: scripts/checks/check-upstream-treadmill.sh [--strict-lag N]
#          [--no-cache] [--refresh-cache]
# Exit: 0 = report generated (advisory) / within budget
#       1 = structural error (unknown prefix, pseudo-version pin)
#       3 = lag exceeds the --strict-lag budget
#       4 = cannot verify (a repo's tags could not be fetched)
#
# Env (fixture self-test):
#   TREADMILL_ROOT           module-discovery root (default: the git toplevel)
#   TREADMILL_FETCH_TAGS_CMD fetch stub (default: git ls-remote --tags <url>);
#                            receives the repo URL as $1, must print tags
#                            one per line (refs stripped by the caller)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${TREADMILL_ROOT:-$(git rev-parse --show-toplevel 2>/dev/null)}" || {
  echo "check-upstream-treadmill: not inside a git repository (set TREADMILL_ROOT for fixtures)" >&2
  exit 2
}

# shellcheck disable=SC1091
source "$SCRIPT_DIR/../lib/replace-exemption.sh"

STRICT_LAG=-1 # -1 = advisory
NO_CACHE=0
REFRESH=0
while [ $# -gt 0 ]; do
  case "$1" in
  --strict-lag)
    shift
    STRICT_LAG="${1:?--strict-lag requires a number}"
    [[ $STRICT_LAG =~ ^[0-9]+$ ]] || {
      echo "check-upstream-treadmill: --strict-lag must be a non-negative integer, got '$STRICT_LAG'" >&2
      exit 2
    }
    ;;
  --no-cache) NO_CACHE=1 ;;
  --refresh-cache) REFRESH=1 ;;
  *)
    echo "check-upstream-treadmill: unknown option '$1'" >&2
    exit 2
    ;;
  esac
  shift
done

CACHE_DIR="${TRAIN_TAG_CACHE_DIR:-/tmp/cqrs-htmx-tag-cache}"
CACHE_TTL="${TRAIN_TAG_CACHE_TTL:-900}"

echo "=== Upstream Treadmill ==="
echo ""

declare -A REPO_TAGS_OK
declare -A REPO_TAGS_LIST
declare -A FETCH_FAILED
# Seed-and-unset so empty associative arrays are BOUND under set -u (bash
# 5.x still treats a declared-but-never-assigned assoc as unbound when
# expanded — the "line 292: FETCH_FAILED: unbound variable" class).
for _arr in REPO_TAGS_OK REPO_TAGS_LIST FETCH_FAILED PIN PINS_ALL PIN_WHERE PIN_DISAGREE; do
  declare -A "$_arr[__seed__]=1"
  unset "$_arr[__seed__]"
done
CACHE_MAX_AGE=-1

fetch_repo_tags() { # <repo> — populate REPO_TAGS_* ; never exits on failure
  local repo="$1"
  [[ -v "REPO_TAGS_OK[$repo]" ]] && return 0

  local cache_file="$CACHE_DIR/$repo.tags"
  if [ "$NO_CACHE" = 0 ] && [ "$REFRESH" = 0 ] &&
    [ -f "$cache_file" ] &&
    [ -z "$(find "$cache_file" -mmin "+$((CACHE_TTL / 60))" 2>/dev/null)" ]; then
    REPO_TAGS_OK[$repo]=true
    REPO_TAGS_LIST[$repo]="$(cat "$cache_file")"
    local age
    age=$(($(date +%s) - $(stat -c %Y "$cache_file" 2>/dev/null || date +%s)))
    [ "$age" -gt "$CACHE_MAX_AGE" ] && CACHE_MAX_AGE=$age
    return 0
  fi

  local url="https://github.com/larsartmann/${repo}.git"
  # Intentional word-splitting into an array so the default AND the
  # TREADMILL_FETCH_TAGS_CMD override both split into command + flags.
  # shellcheck disable=SC2206
  local fetch_cmd=(${TREADMILL_FETCH_TAGS_CMD:-git ls-remote --tags})
  local tags
  if ! tags=$("${fetch_cmd[@]}" "$url" 2>/dev/null |
    sed -e 's|.*refs/tags/||' -e 's/\^{}$//' | sort -u); then
    REPO_TAGS_OK[$repo]=false
    REPO_TAGS_LIST[$repo]=""
    FETCH_FAILED[$repo]=1
    return 0
  fi

  REPO_TAGS_OK[$repo]=true
  REPO_TAGS_LIST[$repo]="$tags"
  if [ "$NO_CACHE" = 0 ]; then
    mkdir -p "$CACHE_DIR"
    printf '%s\n' "$tags" >"$cache_file" 2>/dev/null || true
  fi
}

# Map an upstream module path to (repo, tag-prefix). Same derivation rule as
# check-release-train.sh's tag_prefix_for, kept byte-similar on purpose:
#   github.com/larsartmann/go-cqrs-lite/v4                     -> (go-cqrs-lite, "")
#   github.com/larsartmann/go-cqrs-lite/event/v4                -> (go-cqrs-lite, "event/")
#   github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4
#                                                               -> (go-cqrs-lite, "metaengine/sqliteengine/")
upstream_repo_for() {
  local rest="${1#github.com/larsartmann/}"
  echo "${rest%%/*}"
}

upstream_prefix_for() {
  local rest="${1#github.com/larsartmann/}"
  local repo sub
  repo="${rest%%/*}"
  sub=""
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

# Bare published versions (prefix STRIPPED) under a prefix, sorted. Comparing
# bare versions — never full tags — is the whole game: sort -V on
# "watermill/v4.7.0" vs pin "v4.7.0" ranks the prefixed tag above the pin and
# fabricates lag; nested-module tags (event/v4/eventtest/vX) can never match
# the anchored form. Empty when the prefix has no published tags.
versions_for() {
  local prefix="$1" tags="$2" t
  while IFS= read -r t; do
    [ -n "$t" ] || continue
    echo "${t#"$prefix"}"
  done < <({ grep -E "^${prefix}v[0-9]+\.[0-9]+\.[0-9]+$" <<<"$tags" || true; })
}

# Latest bare version under a prefix ("" when none).
max_published_for() {
  versions_for "$1" "$2" | sort -V | tail -1
}

# Releases strictly greater than $pin under $prefix (the honest lag count).
lag_count_for() {
  local v n=0
  while IFS= read -r v; do
    [ -n "$v" ] || continue
    if [[ "$(printf '%s\n%s\n' "$v" "$pin" | sort -V | tail -1)" == "$v" && $v != "$pin" ]]; then
      n=$((n + 1))
    fi
  done < <(versions_for "$1" "$2")
  echo "$n"
}

# pin[repo/prefix]="vX.Y.Z" (the workspace MAX pin) · pins[repo/prefix]="all pins"
declare -A PIN
declare -A PINS_ALL
declare -A PIN_WHERE
declare -A PIN_DISAGREE

structural=0
warnings=0
worst_lag=0
worst_row=""
rows=0

for modfile in $(find "$REPO_ROOT" -name go.mod -not -path '*/vendor/*' -not -path '*/.git/*' -not -path '*/testdata/*' | sort); do
  moddir=$(dirname "$modfile")

  while IFS= read -r line; do
    # Normalize: block form ("\tpath vX.Y.Z // indirect") and single-line form
    # ("require path vX.Y.Z") both yield path in $1 — the 9b3c2e18 class
    # bump-dep fixed; keep both here. Replace/comment lines fail the regexes
    # below and are skipped.
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line#require }"
    mod_path=$(echo "$line" | awk '{print $1}')
    version=$(echo "$line" | awk '{print $2}')
    [ -n "${mod_path:-}" ] || continue

    # Upstream lane only: larsartmann deps that are NOT this repo (the
    # cqrs-htmx self-train is check-release-train.sh's lane).
    [[ $mod_path =~ ^github\.com/larsartmann/ ]] || continue
    [[ $mod_path =~ ^github\.com/larsartmann/cqrs-htmx(/|$) ]] && continue

    # Indirect pins are real pins too (the comment suffix never reaches $2).

    if [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      :
    elif [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+-[0-9a-zA-Z.-]+$ ]]; then
      # Pseudo/pre-release pin of an upstream — an untagged commit, unless a
      # local replace exempts it (replaced source wins; the pin is cosmetic).
      if replace_exemption_applies "$moddir" "$mod_path"; then
        continue
      fi
      echo "✗ PSEUDO  $moddir pins $mod_path at pseudo-version $version (untagged upstream commit; replace-exempt pins are skipped)" >&2
      structural=$((structural + 1))
      continue
    else
      # Malformed/unparseable version on a larsartmann require — the bad-awk
      # class. Loud, never dropped.
      echo "✗ PARSE   $moddir: unparseable require '$mod_path $version'" >&2
      structural=$((structural + 1))
      continue
    fi

    repo=$(upstream_repo_for "$mod_path")
    prefix=$(upstream_prefix_for "$mod_path")
    key="$repo/$prefix"

    if [[ -v "PIN[$key]" ]]; then
      PINS_ALL[$key]="${PINS_ALL[$key]} $version"
      if [[ "$(printf '%s\n%s\n' "${PIN[$key]}" "$version" | sort -V | tail -1)" == "$version" && $version != "${PIN[$key]}" ]]; then
        PIN[$key]="$version"
        PIN_WHERE[$key]="$moddir"
      fi
    else
      PIN[$key]="$version"
      PINS_ALL[$key]="$version"
      PIN_WHERE[$key]="$moddir"
    fi
  done < <(grep 'github.com/larsartmann/' "$modfile" || true)
done

# Report per repo, prefix-sorted for stable output.
repos="$(for k in "${!PIN[@]}"; do echo "${k%%/*}"; done | sort -u)"
for repo in $repos; do
  fetch_repo_tags "$repo"

  if [ "${REPO_TAGS_OK[$repo]:-false}" != true ]; then
    if [ "${CI:-false}" = "true" ]; then
      # House rule (check-release-train): ls-remote failures downgrade to
      # warnings under CI — a runner network blip must not spuriously red
      # master. Locally they stay exit-4 loud.
      echo "  WARN: cannot list tags of $repo (ls-remote failed; CI advisory)"
      unset 'FETCH_FAILED[$repo]'
      continue
    fi
    echo "repo $repo: CANNOT VERIFY — tag fetch failed (cache ${CACHE_DIR}/$repo.tags absent or stale)" >&2
    continue # totals will surface the failure
  fi

  echo "repo $repo"
  while IFS= read -r key; do
    [ -n "$key" ] || continue
    prefix="${key#*/}"
    pin="${PIN[$key]}"
    latest=$(max_published_for "$prefix" "${REPO_TAGS_LIST[$repo]}")
    rows=$((rows + 1))

    if [ -z "$latest" ]; then
      echo "  ✗ UNKNOWN $prefix — repo reachable but zero tags under this prefix (wrong prefix derivation or unpublished module; pinned $pin)" >&2
      structural=$((structural + 1))
      continue
    fi

    lag=$(lag_count_for "$prefix" "${REPO_TAGS_LIST[$repo]}" "$pin")

    disagree=""
    if [ "$(echo "${PINS_ALL[$key]}" | tr ' ' '\n' | sort -u | grep -c .)" -gt 1 ]; then
      disagree="  [DISAGREE: ${PINS_ALL[$key]}]"
      warnings=$((warnings + 1))
    fi

    mark="aligned"
    [ "$lag" -gt 0 ] && mark="LAG $lag"
    printf '  %-45s pinned %-10s latest %-10s %-9s%s\n' "$prefix" "$pin" "$latest" "[$mark]" "$disagree"

    if [ "$lag" -gt "$worst_lag" ]; then
      worst_lag=$lag
      worst_row="$repo/$prefix ($pin → $latest)"
    fi
  done < <(for k in "${!PIN[@]}"; do [[ "${k%%/*}" == "$repo" ]] && echo "$k"; done | sort)
  echo ""
done

echo "Totals: $(echo "$repos" | grep -c .) repos · $rows upstream prefixes · worst lag $worst_lag${worst_row:+ ($worst_row)} · $warnings disagreement warning(s)"
[ "$CACHE_MAX_AGE" -ge 0 ] && echo "  (tag cache age ${CACHE_MAX_AGE}s, TTL ${CACHE_TTL}s — shared with check-release-train)"

if [ "${#FETCH_FAILED[@]}" -gt 0 ]; then
  echo "✗ upstream treadmill: cannot verify ${#FETCH_FAILED[@]} repo(s): ${!FETCH_FAILED[*]}" >&2
  exit 4
fi

if [ "$structural" -gt 0 ]; then
  echo "✗ upstream treadmill: $structural structural error(s) — unknown prefixes / pseudo pins / parse failures (see stderr)" >&2
  exit 1
fi

if [ "$STRICT_LAG" -ge 0 ] && [ "$worst_lag" -gt "$STRICT_LAG" ]; then
  echo "✗ upstream treadmill: worst lag $worst_lag exceeds budget $STRICT_LAG ($worst_row)" >&2
  exit 3
fi

if [ "$STRICT_LAG" -ge 0 ]; then
  echo "✓ upstream treadmill within budget (strict-lag $STRICT_LAG)"
else
  echo "✓ upstream treadmill report generated (advisory — pass --strict-lag N to gate)"
fi
