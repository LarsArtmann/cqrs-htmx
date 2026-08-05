#!/usr/bin/env bash
# check-domain-counts.sh — Canonical event/command counts from identity-model source
#
# The "21 events / 20 commands" numbers are hardcoded across 5+ docs
# (AGENTS.md, FEATURES.md, ROADMAP.md, TODO_LIST.md, guides). When a new event
# or command is added, these docs drift. This script:
#   1. Counts the real structs from identity-model source (the source of truth)
#   2. Checks that hardcoded TOTAL counts in key docs match the real counts
#
# Only checks total-count phrasings ("NN events / MM commands", "NN event
# payload structs", "NN command structs") — NOT aggregate subset counts
# (e.g. "12 events" for the User aggregate is intentionally a subset).
#
# Usage: ./scripts/check-domain-counts.sh
# Exit:  0 = all docs match, 1 = drift detected

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IDM="$REPO_ROOT/identity-model"

# Count event payload structs (suffix "Payload") and command structs (suffix "Cmd")
events=$(grep -rhoP 'type \w+Payload struct' --include='*.go' --exclude='*_test.go' "$IDM" | sort -u | wc -l)
commands=$(grep -rhoP 'type \w+Cmd struct' --include='*.go' --exclude='*_test.go' "$IDM" | sort -u | wc -l)

echo "=== Domain model counts (from identity-model source) ==="
echo "Events:   ${events} (structs ending in 'Payload')"
echo "Commands: ${commands} (structs ending in 'Cmd')"
echo ""

# Check TOTAL-count phrasings only (not aggregate subsets like "12 events").
drift=0

check_paired() {
    # Matches "NN events / MM commands" — the canonical total shorthand.
    local file="$1" expected_e="$2" expected_c="$3"
    [ -f "$file" ] || return 0
    while IFS= read -r line; do
        local e c
        e=$(echo "$line" | grep -oiP '\d+(?=\s*events?\b)' | head -1)
        c=$(echo "$line" | grep -oiP '\d+(?=\s*commands?\b)' | head -1)
        if [ -n "$e" ] && [ "$e" != "$expected_e" ]; then
            echo "DRIFT: $(basename "$file") — paired line says ${e} events (source: ${expected_e})"
            echo "       > $(echo "$line" | head -c 120)"
            drift=1
        fi
        if [ -n "$c" ] && [ "$c" != "$expected_c" ]; then
            echo "DRIFT: $(basename "$file") — paired line says ${c} commands (source: ${expected_c})"
            echo "       > $(echo "$line" | head -c 120)"
            drift=1
        fi
    done < <(grep -iP '\d+\s*events?\s*/\s*\d+\s*commands?' "$file" 2>/dev/null || true)
}

check_phrase() {
    # Matches "NN event payload" or "NN command struct" — total phrasings.
    local file="$1" kind="$2" expected="$3" pattern="$4"
    [ -f "$file" ] || return 0
    while IFS= read -r line; do
        local n
        n=$(echo "$line" | grep -oiP '\d+(?=\s*'"$pattern"')' | head -1)
        if [ -n "$n" ] && [ "$n" != "$expected" ]; then
            echo "DRIFT: $(basename "$file") — says ${n} ${kind} (source: ${expected})"
            echo "       > $(echo "$line" | head -c 120)"
            drift=1
        fi
    done < <(grep -iP '\d+\s*'"$pattern" "$file" 2>/dev/null || true)
}

for doc in "$REPO_ROOT/AGENTS.md" "$REPO_ROOT/FEATURES.md" "$REPO_ROOT/ROADMAP.md" "$REPO_ROOT/TODO_LIST.md"; do
    check_paired  "$doc" "$events" "$commands"
    check_phrase  "$doc" "events"   "$events"   'event payload'
    check_phrase  "$doc" "commands" "$commands" 'command struct'
done

if [ "$drift" -eq 0 ]; then
    echo "OK: No total-count drift detected."
else
    echo ""
    echo "Fix the drifted counts above to match the source of truth (${events} events / ${commands} commands)."
    exit 1
fi
