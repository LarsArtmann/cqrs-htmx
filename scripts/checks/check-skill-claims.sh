#!/usr/bin/env bash
# check-skill-claims.sh — drift gate (skeleton): every module-qualified
# exported symbol the skill docs name must still exist in the code.
#
# The skill (.agents/skills/cqrs-htmx/) is a PROMISE about the API surface;
# when code moves and docs stay, every session built on the skill inherits
# the lie. This skeleton mechanically pins the cheapest class: a doc writes
# `pkg.Symbol` and the symbol no longer (or never did) exist anywhere in the
# owning module's sources. Semantic claims ("X does Y") stay human work.
#
# Usage: bash scripts/checks/check-skill-claims.sh
# Env: SKILL_CLAIMS_ROOT   override the repo root (fixture self-test).
# Exit: 0 all named symbols resolve, 1 drift or missing inputs, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${SKILL_CLAIMS_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

SKILLS_DIR="$REPO_ROOT/.agents/skills"
if [ ! -d "$SKILLS_DIR" ]; then
  echo "✗ skill-claims: $SKILLS_DIR not found"
  exit 1
fi

# Module alias -> source dir. The alias is how docs qualify symbols
# (cqrshtmx.Foo, usermgmt.Bar); the dir is where the symbol must appear as
# a declaration (func/type/const/var/method-literal).
declare -A MODULE_DIRS=(
  [cqrshtmx]="."
  [cqrs]=.
  [transport]=transport
  [usermgmt]=usermgmt
  [identitymodel]=identity-model
  [setup]=setup
  [adminui]=adminui
  [dashboardui]=dashboardui
  [loginpage]=loginpage
  [datastar]=datastar
  [systemadapter]=systemadapter
  [health]=health
  [auditlog]=auditlog
  [totp]=usermgmt/totp
  [webauthn]=usermgmt/webauthn
  [oauth2]=usermgmt/oauth2
)

# Symbols the docs may name that are NOT Go declarations in this repo
# (upstream re-exports documented for consumers, method values, fixtures).
ALLOWLIST="
cqrshtmx.TemplComponent
"

fail=0
checked=0

shopt -s globstar
for doc in "$SKILLS_DIR"/**/*.md; do
  [ -f "$doc" ] || continue

  # module-qualified ExportedSymbol tokens.
  while IFS= read -r sym; do
    [ -n "$sym" ] || continue

    alias="${sym%%.*}"
    name="${sym#*.}"

    dir="${MODULE_DIRS[$alias]:-}"
    if [ -z "$dir" ]; then
      # Unknown qualifier: not ours to verify (e.g. upstream libs in examples).
      continue
    fi

    if grep -qx "$sym" <<<"$ALLOWLIST"; then
      continue
    fi

    checked=$((checked + 1))

    # Declaration search scope: for the ROOT alias the dir is "." and a
    # recursive grep would leak into the sub-modules (cqrshtmx.ErrValidation
    # "resolved" against usermgmt's sentinel once) — search only the root
    # package files. _test.go files never declare exported API and only add
    # false confidence, so they are excluded everywhere.
    if [ "$dir" = "." ]; then
      search=("$REPO_ROOT"/*.go)
      mapfile -t search < <(printf '%s\n' "${search[@]}" | grep -v '_test\.go$' || true)
    else
      search_args=("$REPO_ROOT/$dir" --include='*.go' --exclude='*_test.go')
    fi

    # Declaration sites: standalone (func/type/const/var Name) or inside a
    # grouped block — the re-export and sentinel files declare
    # "Name = other.Pkg.Name" or "Name SomeType = value" on their own line.
    if [ "$dir" = "." ]; then
      if ! grep -qE "(func|type|const|var)[[:space:]]+(method[[:space:]]+\([^)]*\)[[:space:]]+)?$name\b" "${search[@]}" 2>/dev/null &&
        ! grep -qE "^[[:space:]]*$name([[:space:]]+[A-Za-z0-9_.*]+)?[[:space:]]*=[^=]" "${search[@]}" 2>/dev/null; then
        echo "✗ skill-claims: $doc names $sym but no declaration '$name' exists in the root package" >&2
        fail=1
      fi
    else
      if ! grep -rqE "(func|type|const|var)[[:space:]]+(method[[:space:]]+\([^)]*\)[[:space:]]+)?$name\b" \
        "${search_args[@]}" 2>/dev/null &&
        ! grep -rqE "^[[:space:]]*$name([[:space:]]+[A-Za-z0-9_.*]+)?[[:space:]]*=[^=]" \
          "${search_args[@]}" 2>/dev/null; then
        echo "✗ skill-claims: $doc names $sym but no declaration '$name' exists under $dir/" >&2
        fail=1
      fi
    fi
  done < <(grep -ohE '\b(cqrshtmx|cqrs|transport|usermgmt|identitymodel|setup|adminui|dashboardui|loginpage|datastar|systemadapter|health|auditlog|totp|webauthn|oauth2)\.[A-Z][A-Za-z0-9]*\b' "$doc" | sort -u)
done

if [ "$checked" -eq 0 ]; then
  echo "✗ skill-claims: zero symbols extracted — the gate scanned nothing (fixture or path drift)"
  exit 1
fi

if [ "$fail" -eq 0 ]; then
  echo "✓ skill-claims: $checked documented symbol(s) all resolve in code"
fi
exit "$fail"
