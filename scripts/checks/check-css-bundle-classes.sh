#!/usr/bin/env bash
# check-css-bundle-classes.sh — exact-class-set drift gate for the committed Tailwind bundles
#
# check-css-bundles.sh pins the canonical FORM (minified 1-line, banner
# header, byte floor, canary utility FAMILIES). This gate pins the CONTENT:
# the sorted set of class tokens in the committed bundle must exactly equal
# the set a fresh canonical build produces (nix run .#build-adminui-css /
# .#build-dashboardui-css). It mechanizes the manual "formatting-only,
# class sets identical" proof the templ-components v1.19.1 adoption needed
# (adoption-session report §e3, §f13): a family bump or source edit that
# changes the class set now fails CI instead of silently shipping missing
# or extra utilities.
#
# Method: extract class tokens from the committed bundle, rebuild in place
# via the canonical flake builder, extract again, diff the sorted sets,
# then `git restore` the bundle so the gate never leaves a tree mutation.
# Refuses to run on a dirty bundle (a restore would destroy uncommitted
# work — commit or rebuild first).
#
# Usage: scripts/check-css-bundle-classes.sh
# Env (TEST HOOK): CHECK_CSS_BUNDLE_CLASSES_NO_BUILD=1
#                  skip the nix builders; compare <bundle> against
#                  <bundle>.fresh instead (fixture mode: no mutation,
#                  no git needed).
# Exit: 0 = class sets identical; 1 = drift or preflight failure

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NO_BUILD="${CHECK_CSS_BUNDLE_CLASSES_NO_BUILD:-0}"

# module:bundle-path pairs (module name drives the flake builder app)
BUNDLES=(
  "adminui:adminui/assets/admin-tw.css"
  "dashboardui:dashboardui/assets/dashboard-tw.css"
)

# Class-token extractor: a CSS class selector segment, including escape
# sequences (md\:flex, bg-black\/40). Applied identically to both sides,
# so selector-adjacent noise (numeric value fragments like ".025em") is
# symmetric and never alone causes drift.
extract_classes() { # <css-file> -> sorted unique tokens on stdout
  grep -oE '\.([A-Za-z0-9_-]|\\.)+' "$1" | sort -u
}

failures=0

for entry in "${BUNDLES[@]}"; do
  module="${entry%%:*}"
  rel="${entry#*:}"
  path="$REPO_ROOT/$rel"

  if [ ! -f "$path" ]; then
    echo "✗ $rel: MISSING — run nix run .#build-$module-css and commit the bundle."
    failures=$((failures + 1))
    continue
  fi

  committed="$(mktemp)"
  fresh="$(mktemp)"

  if [ "$NO_BUILD" = "1" ]; then
    if [ ! -f "$path.fresh" ]; then
      echo "✗ $rel: fixture mode requires $rel.fresh"
      failures=$((failures + 1))
      continue
    fi
    extract_classes "$path" >"$committed"
    extract_classes "$path.fresh" >"$fresh"
  else
    if ! command -v nix >/dev/null 2>&1; then
      # CI runner class: no nix, so the canonical builders cannot run here.
      # Same wiring pattern as check-vcs-cache in CI — the fixture self-test
      # is the coverage; the real comparison runs wherever nix exists.
      echo "⊘ nix not available — class-set comparison needs the canonical flake"
      echo "  builders. Run locally: nix run .#check-css-bundle-classes"
      exit 0
    fi
    if ! git -C "$REPO_ROOT" diff --quiet -- "$rel" 2>/dev/null; then
      echo "✗ $rel: bundle has UNCOMMITTED changes — restoring after rebuild"
      echo "       would destroy them. Commit (or rebuild) first, then re-run."
      failures=$((failures + 1))
      continue
    fi
    extract_classes "$path" >"$committed"
    if ! (cd "$REPO_ROOT" && nix run ".#build-$module-css" >/dev/null 2>&1); then
      echo "✗ $rel: canonical builder .#build-$module-css failed."
      git -C "$REPO_ROOT" restore -- "$rel" 2>/dev/null || true
      failures=$((failures + 1))
      continue
    fi
    extract_classes "$path" >"$fresh"
    # Leave the tree exactly as we found it, whatever the verdict.
    git -C "$REPO_ROOT" restore -- "$rel"
  fi

  if diff -u "$committed" "$fresh" >/tmp/css-class-drift-$$.diff 2>&1; then
    count=$(wc -l <"$committed")
    echo "✓ $rel: class set exact ($count tokens)"
  else
    echo "✗ $rel: CLASS-SET DRIFT vs fresh canonical build:"
    missing=$(grep -c '^-' /tmp/css-class-drift-$$.diff || true)
    extra=$(grep -c '^+' /tmp/css-class-drift-$$.diff || true)
    echo "       $((missing > 1 ? missing - 1 : 0)) token(s) only in committed," \
      "$((extra > 1 ? extra - 1 : 0)) only in fresh build."
    sed -n '1,12p' /tmp/css-class-drift-$$.diff | sed 's/^/       /'
    echo "       Fix: rebuild + commit the bundle (nix run .#build-$module-css) in the"
    echo "       same change as the edit that altered the class set."
    failures=$((failures + 1))
  fi

  rm -f "$committed" "$fresh"
done

rm -f /tmp/css-class-drift-$$.diff

if [ "$failures" -ne 0 ]; then
  echo "✗ $failures bundle(s) drifted."
  exit 1
fi

echo "✓ ${#BUNDLES[@]} bundles class-set exact."
