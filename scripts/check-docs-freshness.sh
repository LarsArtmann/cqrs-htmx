#!/usr/bin/env bash
# check-docs-freshness.sh — Scan .md files for stale version strings
# Compares version numbers mentioned in docs against actual go.mod versions.
# Usage: nix run .#check-docs-freshness
# Exit: 0 = docs fresh, 1 = stale versions found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

FAILED=0

echo "=== Docs Freshness Check ==="
echo ""

# Extract versions from go.mod files
extract_version() {
  local modpath="$1" dep="$2"
  grep "$dep" "$modpath/go.mod" 2>/dev/null | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo ""
}

# Check AGENTS.md dependency table against go.mod
check_agents_md() {
  local dep="$1" agents_pattern="$2"
  local root_ver

  root_ver=$(extract_version "." "$dep")

  # The version mentioned in AGENTS.md
  local agents_ver
  agents_ver=$(grep -oP "${agents_pattern}\s+v\d+\.\d+\.\d+" AGENTS.md | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo "")

  if [ -n "$agents_ver" ] && [ -n "$root_ver" ]; then
    if [ "$agents_ver" != "$root_ver" ]; then
      echo "  STALE: AGENTS.md has $dep $agents_ver, go.mod has $root_ver"
      FAILED=1
    fi
  fi
}

echo "Checking AGENTS.md version strings..."
check_agents_md "go-webauthn" "go-webauthn"
check_agents_md "pquerna/otp" "pquerna/otp"
check_agents_md "coreos/go-oidc" "coreos/go-oidc"
check_agents_md "a-h/templ" "a-h/templ"

# Check for stale Go version references
echo ""
echo "Checking Go version references..."
GO_MOD_VERSION=$(grep '^go ' go.mod | awk '{print $2}')
AGENTS_GO=$(grep -oP 'Go 1\.\d+\.\d+' AGENTS.md | head -1 | grep -oP '1\.\d+\.\d+' || echo "")
if [ -n "$AGENTS_GO" ] && [ "$AGENTS_GO" != "$GO_MOD_VERSION" ]; then
  echo "  INFO: AGENTS.md says Go $AGENTS_GO, go.mod says $GO_MOD_VERSION (may be intentional)"
fi

# Check for stale HTMX version references
echo ""
echo "Checking HTMX version references..."
HTMX_CONST=$(grep -oP 'htmxVersion\s*=\s*"\K[^"]+' htmx_embed.go 2>/dev/null || echo "")
HTMX_DOCS=$(grep -oP 'HTMX v\d+\.\d+\.\d+' AGENTS.md docs/ 2>/dev/null | head -1 | grep -oP 'v\d+\.\d+\.\d+' || echo "")
if [ -n "$HTMX_CONST" ] && [ -n "$HTMX_DOCS" ]; then
  DOCS_VER=${HTMX_DOCS//v/}
  if [ "$HTMX_CONST" != "$DOCS_VER" ]; then
    echo "  STALE: htmx_embed.go has $HTMX_CONST, docs reference $HTMX_DOCS"
    FAILED=1
  fi
fi

# Check for deprecated API references in docs
echo ""
echo "Checking for deprecated API references..."
if grep -rn 'errors\.New\|fmt\.Errorf' --include="*.md" . 2>/dev/null | grep -v 'banned\|enforced\|Use event\.' | head -5; then
  echo "  INFO: Found stdlib error constructor references in docs (may be documentation of the ban)"
fi

# ---------------------------------------------------------------------------
# Replace-state claims (added 2026-09-10): a living doc claiming a module
# "carries a replace" must find that replace in go.work or some go.mod.
# Both classes (replace-state, uniform-at) were provably wrong while this
# gate stayed green — see the 2026-09-09 docs-health sweep.
# ---------------------------------------------------------------------------
LIVING_DOCS=$(ls AGENTS.md README.md FEATURES.md TODO_LIST.md ROADMAP.md docs/*.md docs/guides/*.md 2>/dev/null || true)

echo ""
echo "Checking replace-state claims in living docs..."
for f in $LIVING_DOCS; do
  while IFS= read -r line; do
    modpath=$(grep -oP 'github\.com/larsartmann/[a-z0-9/.-]+' <<<"$line" | head -1 || true)
    if [ -z "$modpath" ]; then
      continue
    fi
    found=0
    if grep -qE "replace[[:space:]]+$modpath([[:space:]]|v[0-9])" go.work 2>/dev/null; then
      found=1
    fi
    if [ "$found" -eq 0 ]; then
      if grep -rlE "^[[:space:]]*replace[[:space:]]+$modpath([[:space:]]|v[0-9])" --include=go.mod . 2>/dev/null | grep -qv vendor; then
        found=1
      fi
    fi
    if [ "$found" -eq 0 ]; then
      echo "  STALE: $f claims a replace for $modpath but no go.work/go.mod replace exists"
      FAILED=1
    fi
    # Only report the first offending claim per file to keep output readable.
    break
  done < <(grep -iE '(carries|has|keeps?|with|mount|attaches?) (a |an |its )?(local |dev-only |temporary )?replace' "$f" 2>/dev/null |
    grep -viE 'stripped|removed|deleted|no (longer|more)|historical|never')
done

# ---------------------------------------------------------------------------
# "uniform at vX" family claims: every non-replace-satisfied templ-components
# require in the tree must be at the claimed version.
# ---------------------------------------------------------------------------
echo ""
echo "Checking 'uniform at vX' family claims..."
# shellcheck disable=SC1091
source scripts/lib/replace-exemption.sh
while IFS=: read -r f lineno line; do
  claimed=$(grep -oP '(?:uniform at|all \w+ modules at) \Kv[0-9]+\.[0-9]+\.[0-9]+' <<<"$line" | head -1 || true)
  if [ -z "$claimed" ]; then
    continue
  fi
  bad=""
  for modfile in $(find . -name go.mod -not -path './vendor/*' -not -path './.git/*' -not -path '*/testdata/*' | sort); do
    moddir=$(dirname "$modfile")
    while IFS= read -r req; do
      mod_path=$(awk '{print $1}' <<<"$req")
      ver=$(awk '{print $2}' <<<"$req")
      if replace_exemption_applies "$moddir" "$mod_path"; then continue; fi
      if [ "$ver" != "$claimed" ]; then bad="$bad $moddir@$ver"; fi
    done < <(grep -E "github\.com/larsartmann/templ-components" "$modfile" | grep -oP 'github\.com/larsartmann/templ-components[a-z/-]*\s+v[0-9]+\.[0-9]+\.[0-9]+')
  done
  if [ -n "$bad" ]; then bad=$(printf '%s
' "$bad" | sort -u | tr '
' ' ' | sed 's/^ //;s/ $//'); fi
  if [ -n "$bad" ]; then
    echo "  STALE: $f:$lineno claims templ-components uniform at $claimed but tree has:$bad"
    FAILED=1
  fi
done < <(eval grep -nE '"(uniform at|all [a-z]+ modules at) v" $LIVING_DOCS' 2>/dev/null)

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✓ Docs freshness check PASSED"
else
  echo "✗ Docs freshness check found stale references"
  exit 1
fi
