#!/usr/bin/env bash
# dep-footprint.sh — generate the README dependency-footprint table.
#
# Per Go module: direct-require count, module-graph closure size, and the
# go directive. The README table is GENERATED from this script, never
# hand-maintained (the check-root-size precedent) — re-run and paste whenever
# any go.mod changes (release-checklist step; the "The modules" README table
# carries the regeneration note).
#
# Output: TSV on stdout — header row `module<TAB>direct<TAB>closure<TAB>go`,
# one row per go.mod under the repo root, sorted by module directory.
#
# Direct + go directive come from `go mod edit -json` (offline). Closure size
# is the module graph `GOWORK=off go list -m all` line count (module cache;
# first run per module may hit the network for missing sums). Modules whose
# closure cannot be resolved print `n/a` instead of a number and are the
# operator's to investigate — no silent zeros.
#
# Usage: scripts/checks/dep-footprint.sh [repo-root]
# Exit: 0 table generated, 1 usage/unrecoverable.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${1:-$SCRIPT_DIR/../..}" && pwd)"

command -v go >/dev/null || { echo "dep-footprint: go not in PATH" >&2; exit 1; }
command -v jq >/dev/null || { echo "dep-footprint: jq not in PATH" >&2; exit 1; }
[[ -f "$REPO_ROOT/go.work" || -f "$REPO_ROOT/go.mod" ]] || {
	echo "dep-footprint: $REPO_ROOT is not a Go workspace root" >&2
	exit 1
}

printf 'module\tdirect\tclosure\tgo\n'

find "$REPO_ROOT" -name go.mod \
	-not -path '*/node_modules/*' \
	-not -path '*/testdata/*' \
	-not -path '*/.git/*' | sort | while read -r gomod; do
	dir="$(dirname "$gomod")"
	label="${dir#"$REPO_ROOT"/}"
	[[ "$label" == "$dir" ]] && label="."

	json="$(cd "$dir" && GOWORK=off go mod edit -json 2>/dev/null)" || {
		echo "dep-footprint: go mod edit -json failed in $label" >&2
		continue
	}
	direct="$(printf '%s' "$json" | jq '[.Require[]? | select(.Indirect != true)] | length')"
	gover="$(printf '%s' "$json" | jq -r '.Go // "n/a"')"
	closure="$(cd "$dir" && GOWORK=off go list -m all 2>/dev/null | wc -l | tr -d ' ')"
	if [[ "$closure" -eq 0 ]]; then
		closure="n/a"
		reason="$(cd "$dir" && GOWORK=off go list -m all 2>&1 >/dev/null | head -1)"
		echo "dep-footprint: closure unresolvable for $label: ${reason:-go list -m all failed}" >&2
	fi

	printf '%s\t%s\t%s\t%s\n' "$label" "$direct" "$closure" "$gover"
done
