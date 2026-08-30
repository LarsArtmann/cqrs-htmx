#!/usr/bin/env bash
# verify-tag.sh — tag a module the SAFE way (the protocol from the
# setup/v4.8.1+v4.8.2 poisoned-tag incident).
#
# Usage: scripts/verify-tag.sh <module-dir> <version> [--push] [--dry-run] [--allow-replace-exempt]
#   e.g. scripts/verify-tag.sh setup v4.8.4 --push
#
# Steps:
#   1. Guard: <module-dir>/go.mod must exist and have NO uncommitted changes
#      (tags point at commits, never the working tree — the v4.8.2 incident).
#   2. Guard: tag must not already exist locally or on origin.
#   3. Content assertions on `git show HEAD:<module-dir>/go.mod` (HEAD is
#      exactly what the tag will point at, per guard 1):
#      - any local-path `replace` (family `../`, sibling `../../x`, absolute)
#        is refused — consumers IGNORE replaces and silently resolve the
#        plain require, so the published tag would NOT contain the content
#        verified here (setup/v4.8.1 family class; systemadapter go-cqrs-lite
#        sibling class). `--allow-replace-exempt` overrides for documented
#        spike-only modules;
#      - no pseudo-version requires of larsartmann modules;
#      - the `go` directive is present;
#      - every cqrs-htmx require resolves to a tag on origin.
#   4. With --dry-run: run ALL guards, then stop WITHOUT creating the tag
#      (this is what the fixture self-test exercises — no signing key needed).
#   5. Create a signed annotated tag at HEAD.
#   6. With --push: push the tag, then verify it exists on origin via
#      ls-remote (proxy-cache rule: never re-push the same version name).
#
# Exits non-zero on any failed guard — a tag that fails here would poison a
# published version. There is no repair for a pushed bad tag except a new
# version (and ideally a retract directive in the next release).

set -euo pipefail

usage() {
  echo "usage: $0 <module-dir> <version> [--push] [--dry-run] [--allow-replace-exempt]" >&2
  exit 2
}

[ $# -ge 2 ] || usage
MOD="$1"
VER="$2"
shift 2
PUSH=0
DRY=0
ALLOW_LOCAL_REPLACE=0
for arg in "$@"; do
  case "$arg" in
  --push) PUSH=1 ;;
  --dry-run) DRY=1 ;;
  --allow-replace-exempt) ALLOW_LOCAL_REPLACE=1 ;;
  *) usage ;;
  esac
done

fail() {
  echo "verify-tag: FAIL: $*" >&2
  echo "verify-tag: nothing was pushed. Fix the tree, then re-run." >&2
  exit 1
}

[ -f "$MOD/go.mod" ] || fail "$MOD/go.mod does not exist (is the module dir correct?)"

TAGNAME="$MOD/$VER"
# Root-module tags have no directory prefix (cqrs-htmx/v4@v4.8.0 -> v4.8.0).
case "$MOD" in
. | ./) TAGNAME="$VER" ;;
esac

if [ -n "$(git status --porcelain --untracked-files=no -- "$MOD")" ]; then
  echo "verify-tag: uncommitted changes inside $MOD:" >&2
  git status --porcelain --untracked-files=no -- "$MOD" | sed 's/^/    /' >&2
  fail "tags point at commits, not the working tree — commit (and push) the module first"
fi

if git rev-parse -q --verify "refs/tags/$TAGNAME" >/dev/null; then
  fail "tag $TAGNAME already exists locally"
fi
if git ls-remote --exit-code origin "refs/tags/$TAGNAME" >/dev/null 2>&1; then
  fail "tag $TAGNAME already exists on origin — NEVER re-push a version name (the module proxy caches it); cut a new version"
fi

# Content assertions run against HEAD's tree — the module files are
# committed-and-clean (guard above), so HEAD is exactly what the tag will
# point at. Asserting BEFORE tagging keeps a failed verification from
# littering the repo with tags.
SHOW="$(git show "HEAD:$MOD/go.mod" 2>/dev/null || git show "HEAD:go.mod" 2>/dev/null || true)"
[ -n "$SHOW" ] || fail "could not read go.mod from HEAD — wrong module dir for this tag?"

# Normalized larsartmann require list — BOTH syntaxes:
#   require github.com/... vX.Y.Z        (single-line require)
#   github.com/... vX.Y.Z                (line inside a require block)
REQ_LINES="$(echo "$SHOW" | awk '
  /^[[:space:]]*require[[:space:]]+github\.com\/larsartmann\// {print $2, $3}
  /^[[:space:]]*github\.com\/larsartmann\// {print $1, $2}
')"

# Any local-path replace makes the tagged module consume untagged content:
# consumers IGNORE replace directives and silently resolve the plain
# require, so a published tag would NOT contain the content verified here.
# Classes seen in the wild: family dev-replaces (setup/v4.8.1:
# cqrs-htmx/... => ../usermgmt), sibling spike replaces (same file:
# go-appkit => ../../go-appkit), and upstream WIP replaces (systemadapter:
# go-cqrs-lite/metaengine/projectionadapter => ../../go-cqrs-lite/...).
# Version-override replaces (<mod> => <mod> vX.Y.Z) are legitimate and
# still pass. The second pattern also catches replace-BLOCK bodies (inner
# lines carry no 'replace' keyword).
if [ "$ALLOW_LOCAL_REPLACE" = 0 ]; then
  LOCAL_REPLACES="$(echo "$SHOW" |
    grep -E '^[[:space:]]*replace[[:space:]][^=]+=>[[:space:]]*(\.\.?/|/)|^[[:space:]]*[a-zA-Z0-9./_-]+[[:space:]]+=>[[:space:]]*(\.\.?/|/)' || true)"
  if [ -n "$LOCAL_REPLACES" ]; then
    # shellcheck disable=SC2001
    echo "$LOCAL_REPLACES" | sed 's/^/    /' >&2
    fail "go.mod replaces modules with LOCAL PATHS — consumers ignore replaces and resolve the plain require, so the tag would not contain the content verified here. Strip the replaces (publish or require real dependency versions), or pass --allow-replace-exempt for a documented spike-only module"
  fi
else
  echo "verify-tag: --allow-replace-exempt set — local-path replaces accepted (spike-only escape hatch; document it in the release notes)"
fi

# Pseudo-versions of Lars' own modules never happen legitimately (every
# family module publishes real tags); third-party pseudo-versions are fine.
if echo "$REQ_LINES" | grep -Eq -- '-0\.[0-9]{14}-[0-9a-f]+'; then
  echo "$REQ_LINES" | grep -E -- '-0\.[0-9]{14}-' | sed 's/^/    /' >&2
  fail "go.mod contains pseudo-version requires of larsartmann modules"
fi

echo "$SHOW" | grep -Eq '^go [0-9]' || fail "go.mod has no 'go' directive"

# Every internal cqrs-htmx require must resolve to a tag that exists on
# origin — tagging against an unpublished version poisons the release (the
# setup/v4.8.2 class). Content-correctness of a published dep (the stale
# dashboardui/v4.8.1 class) is NOT automatable from go.mod text: verify the
# dep's tag carries the API you ship against (git show <tag>:<file>) BEFORE
# running this script.
UNPUBLISHED=0
while IFS=' ' read -r dep ver; do
  [ -n "$dep" ] || continue
  sub="${dep#github.com/larsartmann/cqrs-htmx}"
  [ "$sub" != "$dep" ] || continue
  sub="${sub#/}"
  remoteref="refs/tags/${sub%/v*}/${ver}"
  case "$sub" in
  v*) remoteref="refs/tags/$ver" ;; # root module: cqrs-htmx/v4 -> plain vX.Y.Z
  *)
    # usermgmt/totp/v4 -> usermgmt/totp/v4.8.1 (strip the /v4 major suffix)
    remoteref="refs/tags/${sub%/v*}/$ver"
    ;;
  esac
  if ! git ls-remote --exit-code origin "$remoteref" >/dev/null 2>&1; then
    echo "verify-tag: require $dep $ver has NO tag at $remoteref on origin" >&2
    UNPUBLISHED=1
  fi
done <<EOF
$(echo "$REQ_LINES" | awk '/^github\.com\/larsartmann\/cqrs-htmx/ {print $1 " " $2}')
EOF
[ "$UNPUBLISHED" = 0 ] || fail "go.mod requires unpublished cqrs-htmx versions — push the dependency tags first"

echo "verify-tag: content assertions passed for $TAGNAME"

if [ "$DRY" = 1 ]; then
  echo "verify-tag: dry-run — ALL guards passed; stopping before tag creation"
  exit 0
fi

git tag -s "$TAGNAME" -m "$MOD $VER" || fail "git tag -s failed (signing key unavailable?); NOT tagging unsigned"

if [ "$PUSH" = 1 ]; then
  git push origin "$TAGNAME" || fail "push failed"
  if ! git ls-remote --exit-code origin "refs/tags/$TAGNAME" >/dev/null 2>&1; then
    fail "push reported success but ls-remote cannot see $TAGNAME — investigate before continuing"
  fi
  echo "verify-tag: $TAGNAME pushed and visible on origin"
else
  echo "verify-tag: $TAGNAME created locally. Inspect, then push: git push origin $TAGNAME"
fi
