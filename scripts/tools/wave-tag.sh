#!/usr/bin/env bash
# wave-tag.sh — the release tag rhythm as ONE command (TODO S51).
#
# The 2026-10-13 adminui→setup cascade ran this sequence manually 3× in one
# day and every hand-step was a daemon/TTL race window (status 2026-10-13
# 22-33 §f10). This tool runs the rhythm with refuse-on-any-red-leg
# semantics:
#
#   leg 0  guards   verify-tag <dir> <version> --dry-run
#                    (all verify-tag guards: uncommitted go.mod, existing
#                     tag, replace hygiene, resolvable requires)
#   leg 1  push     git push            — commits reach origin FIRST
#   leg 2  tag      verify-tag <dir> <version> --push
#                    — signed annotated tag at HEAD + pushed + ls-remote
#                      verified (the poisoned-tag protocol)
#   leg 3  train    check-release-train --refresh-cache --strict-lag 0
#                    — gotcha 27 mechanized: the fresh-tag cache race is
#                      impossible to skip, and 866 requires must resolve
#   leg 4  sweep    bump-dep '<module-path>$' <version> --commit --lint
#                    — consumers align in ONE commit (daemon-proof), with
#                      the hermetic lint leg (S47)
#   leg 5  carriers report + optionally cascade
#                    — every go.mod the sweep changed that belongs to a
#                      PUBLISHED module needs its own re-tag; each carrier
#                      prints as the exact next wave-tag invocation. With
#                      --cascade, they run (max 3 per wave — beyond that,
#                      stop and look at your life choices).
#
# Usage: scripts/tools/wave-tag.sh <module-dir> <version> [--execute]
#          [--cascade] [--skip-sweep] [--message <tpl>]
#
#   DRY-RUN (default): leg 0 runs for real (it touches nothing), legs 1-5
#   print as the exact commands they would run. --execute runs them.
#   --skip-sweep  leaf modules with no consumers: skip legs 4-5.
#   --cascade     after a green sweep, re-run the full rhythm for each
#                 detected carrier at the next patch version.
#   --message <tpl> passed to bump-dep ({version} substituted); default
#                 'deps: align {module} to {version} (wave-tag sweep)'.
#
# Exit: 0 = rhythm completed (or dry-run plan sound); 1 = a leg refused;
#       2 = usage/environment.
#
# Env (fixture self-test): WAVE_TAG_VERIFY_TAG_CMD, WAVE_TAG_TRAIN_CMD,
# WAVE_TAG_BUMP_DEP_CMD, WAVE_TAG_GIT_CMD (defaults: the real tools/git).
# Stubs receive exactly the args the real tool would.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Tool commands as ARRAYS: the env overrides may be multi-word ("bash /path");
# array expansion splits them correctly where a quoted string would become
# one bogus program name and a bare string would trip shellcheck (SC2086).
read -r -a VERIFY_TAG_CMD <<<"${WAVE_TAG_VERIFY_TAG_CMD:-bash $REPO_ROOT/scripts/tools/verify-tag.sh}"
read -r -a TRAIN_CMD <<<"${WAVE_TAG_TRAIN_CMD:-bash $REPO_ROOT/scripts/checks/check-release-train.sh}"
read -r -a BUMP_DEP_CMD <<<"${WAVE_TAG_BUMP_DEP_CMD:-bash $REPO_ROOT/scripts/tools/bump-dep.sh}"
read -r -a GIT_CMD <<<"${WAVE_TAG_GIT_CMD:-git}"

usage() {
  echo "usage: $0 <module-dir> <version> [--execute] [--cascade] [--skip-sweep] [--message <tpl>]" >&2
  exit 2
}

[ $# -ge 2 ] || usage
MOD_DIR="$1"
VER="$2"
shift 2
EXECUTE=0
CASCADE=0
SKIP_SWEEP=0
MSG_TPL='deps: align {module} to {version} (wave-tag sweep)'
while [ $# -gt 0 ]; do
  case "$1" in
  --execute) EXECUTE=1 ;;
  --cascade) CASCADE=1 ;;
  --skip-sweep) SKIP_SWEEP=1 ;;
  --message)
    [ $# -ge 2 ] || usage
    MSG_TPL="$2"
    shift
    ;;
  *) usage ;;
  esac
  shift
done

cd "$REPO_ROOT" 2>/dev/null || {
  echo "wave-tag: cannot cd to $REPO_ROOT" >&2
  exit 2
}

if [ ! -f "$MOD_DIR/go.mod" ]; then
  echo "wave-tag: $MOD_DIR/go.mod not found (module dir is relative to the repo root, e.g. usermgmt/totp)" >&2
  exit 2
fi

MOD_PATH="$(awk '/^module /{print $2; exit}' "$MOD_DIR/go.mod")"
[ -n "$MOD_PATH" ] || {
  echo "wave-tag: no module directive in $MOD_DIR/go.mod" >&2
  exit 2
}

MSG="${MSG_TPL//\{module\}/$(basename "$MOD_DIR")}"
MSG="${MSG//\{version\}/$VER}"

leg() { # <leg-name> <cmd...> — run (or print in dry-run); refuse on red
  local name="$1"
  shift
  if [ "$EXECUTE" = 0 ]; then
    echo "  [dry-run] $name: $*"
    return 0
  fi
  echo "  → $name: $*"
  if ! "$@"; then
    echo "✗ wave-tag: leg '$name' REFUSED — rhythm stopped. Completed legs are safe to keep; fix and re-run." >&2
    exit 1
  fi
  echo "  ✓ $name"
}

echo "=== wave-tag: $MOD_DIR $VER (${MOD_PATH}) ==="

# leg 0 — guards (always real, even in dry-run: verify-tag --dry-run touches nothing)
if ! "${VERIFY_TAG_CMD[@]}" "$MOD_DIR" "$VER" --dry-run; then
  echo "✗ wave-tag: leg 'guards' REFUSED — verify-tag --dry-run failed. Nothing was pushed or tagged." >&2
  exit 1
fi
echo "  ✓ guards (verify-tag --dry-run)"

# leg 1 — push the commits the tag will point at
leg push "${GIT_CMD[@]}" push

# leg 2 — tag + push the tag (the poisoned-tag protocol)
leg tag "${VERIFY_TAG_CMD[@]}" "$MOD_DIR" "$VER" --push

# leg 3 — fresh-tag cache refresh + full train strictness (gotcha 27)
leg train "${TRAIN_CMD[@]}" --refresh-cache --strict-lag 0

# leg 4 — consumer sweep in one commit with the lint leg
if [ "$SKIP_SWEEP" = 1 ]; then
  echo "  [skip] sweep (--skip-sweep: leaf module, no consumers)"
else
  leg sweep "${BUMP_DEP_CMD[@]}" "${MOD_PATH}$" "$VER" --commit --lint --message "$MSG"
fi

# leg 5 — carriers: changed go.mods belonging to PUBLISHED modules need re-tags
detect_carriers() { # echoes "<dir> <latest-tag-version>" per carrier
  local base
  base="$(${GIT_CMD[@]} rev-parse HEAD~1 2>/dev/null)"
  [ -n "$base" ] || return 0
  local changed t dir prefix latest
  changed="$(${GIT_CMD[@]} diff --name-only "$base" HEAD 2>/dev/null | grep 'go\.mod$' || true)"
  [ -n "$changed" ] || return 0
  while IFS= read -r t; do
    [ -n "$t" ] || continue
    dir="$(dirname "$t")"
    [ "$dir" = "." ] && dir=""
    prefix="$dir"
    [ -n "$prefix" ] && prefix="$prefix/"
    # the tags cache lives in TRAIN_TAG_CACHE_DIR (shared with the train gate)
    latest="$({ grep -E "^${prefix}v[0-9]+\.[0-9]+\.[0-9]+$" "${TRAIN_TAG_CACHE_DIR:-/tmp/cqrs-htmx-tag-cache}/cqrs-htmx.tags" 2>/dev/null || true; } | sort -V | tail -1)"
    [ -n "$latest" ] || continue # unpublished dirs are not carriers
    [ "$dir" = "$MOD_DIR" ] && continue # the module we just tagged
    echo "$dir ${latest#"$prefix"}"
  done <<<"$changed"
}

if [ "$SKIP_SWEEP" = 1 ] || [ "$EXECUTE" = 0 ]; then
  if [ "$EXECUTE" = 0 ]; then
    echo "  [dry-run] carriers: git diff --name-only HEAD~1 HEAD (post-sweep go.mod changes of published modules)"
  fi
else
  carriers="$(detect_carriers)"
  if [ -n "$carriers" ]; then
    echo "  carriers detected (published modules whose go.mod the sweep changed):"
    n=0
    while IFS=' ' read -r cdir clatest; do
      [ -n "$cdir" ] || continue
      n=$((n + 1))
      next="$(echo "$clatest" | awk -F. -v OFS=. '{$NF = $NF + 1; print}')"
      echo "    $cdir (latest $clatest → next $next): $0 $cdir $next --execute"
      if [ "$CASCADE" = 1 ] && [ "$n" -le 3 ]; then
        echo "  ↪ cascade: $0 $cdir $next --execute"
        if ! "$0" "$cdir" "$next" --execute; then
          echo "✗ wave-tag: cascade into $cdir $next REFUSED — continue manually from the list above." >&2
          exit 1
        fi
      fi
    done <<<"$carriers"
    [ "$CASCADE" = 1 ] || echo "  (re-run with --cascade to chain the carrier re-tags automatically; max 3)"
  else
    echo "  ✓ no carriers (no published module's go.mod changed in the sweep)"
  fi
fi

echo "=== wave-tag: rhythm complete for $MOD_DIR $VER ==="
