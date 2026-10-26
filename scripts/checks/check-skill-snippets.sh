#!/usr/bin/env bash
# check-skill-snippets.sh — every Go fence in the skill docs must PARSE.
#
# The skill's Go snippets are the first code a consumer copies. A fence
# mangled by an edit (unbalanced braces, stray ellipsis introduced by a doc
# pass) fails at the consumer's desk, not here — this gate moves that
# failure to commit time. Fragments are first-class: a fence passes if it
# parses as-is, as a package body (declaration fragments), or wrapped in a
# function (statement fragments). Build-level compilation stays the
# check-readme-quickstart pattern (one full consumer module); this is the
# cheap parse floor for EVERY fence.
#
# Usage: bash scripts/checks/check-skill-snippets.sh
# Env: SKILL_SNIPPETS_ROOT  override the repo root (fixture self-test).
# Exit: 0 all fences parse, 1 a fence is mangled, 2 usage.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${SKILL_SNIPPETS_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

SKILLS_DIR="$REPO_ROOT/.agents/skills"
if [ ! -d "$SKILLS_DIR" ]; then
  echo "✗ skill-snippets: $SKILLS_DIR not found"
  exit 1
fi

# extracts a fence's body between ```go and the closing ``` line.
extract_go_fences() {
  awk '
    /^```go$/ { infence = 1; next }
    in_fence && /^```$/ { infence = 0; printf "\n===FENCE===\n"; next }
    in_fence { print }
  ' "$1"
}

# parses_ok <body>: as-is | package-wrapped | func-wrapped.
parses_ok() {
  local body="$1"
  if printf '%s' "$body" | gofmt -e >/dev/null 2>&1; then return 0; fi
  if printf 'package snippets\n\n%s\n' "$body" | gofmt -e >/dev/null 2>&1; then return 0; fi
  if printf 'package snippets\n\nfunc snippet() {\n%s\n}\n' "$body" | gofmt -e >/dev/null 2>&1; then return 0; fi
  return 1
}

fail=0
checked=0

shopt -s globstar
for doc in "$SKILLS_DIR"/**/*.md; do
  [ -f "$doc" ] || continue

  body=""
  while IFS= read -r line; do
    if [ "$line" = "===FENCE===" ]; then
      checked=$((checked + 1))
      if ! parses_ok "$body"; then
        echo "✗ skill-snippets: a Go fence in $doc does not parse:" >&2
        printf '%s\n' "$body" | head -8 | sed 's/^/    /' >&2
        fail=1
      fi
      body=""
    else
      body+="$line"$'\n'
    fi
  done < <(extract_go_fences "$doc")
done

if [ "$checked" -eq 0 ]; then
  echo "✗ skill-snippets: zero Go fences found — the gate scanned nothing"
  exit 1
fi

if [ "$fail" -eq 0 ]; then
  echo "✓ skill-snippets: $checked Go fence(s) all parse"
fi
exit "$fail"
