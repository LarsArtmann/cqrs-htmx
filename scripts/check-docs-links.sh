#!/usr/bin/env bash
# check-docs-links.sh — Verify all file-path links in markdown files resolve correctly
#
# Scans all .md files for relative file links and checks they exist.
# Filters out:
#   - URLs (http://, https://, mailto:)
#   - Links inside fenced code blocks (``` ... ```)
#   - Links inside inline code (`...`)
#
# Usage: ./scripts/check-docs-links.sh
# Exit: 0 = all links valid, 1 = at least one broken link found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

broken=0
checked=0

find_md_files() {
    find . -name '*.md' -not -path './.git/*' -not -path './vendor/*' -not -path './node_modules/*' | sort
}

# Extract markdown links from a file, skipping code blocks
extract_links() {
    local file="$1"
    local in_codeblock=0

    while IFS= read -r line; do
        # Track fenced code blocks
        if echo "$line" | grep -qP '^\s*```'; then
            in_codeblock=$((1 - in_codeblock))
            continue
        fi

        # Skip lines inside code blocks
        [ "$in_codeblock" -eq 1 ] && continue

        # Extract [text](path) links, excluding URLs and inline-code links
        # Use grep -oP to find all link targets on the line
        echo "$line" | grep -oP '\[[^\]]*\]\(\K[^)]+' | while IFS= read -r target; do
            # Skip URLs and email links
            case "$target" in
                http://*|https://*|mailto:*) continue ;;
            esac
            # Strip anchor fragments and query strings
            target="${target%%#*}"
            target="${target%%\?*}"
            # Skip empty targets after stripping
            [ -z "$target" ] && continue
            echo "$target"
        done
    done < "$file"
}

echo "=== Markdown Link Check ==="
echo "Scanning all .md files for broken file-path links..."
echo ""

while IFS= read -r md_file; do
    md_dir="$(dirname "$md_file")"

    while IFS= read -r link; do
        [ -z "$link" ] && continue
        checked=$((checked + 1))

        # Resolve relative to the markdown file's directory
        resolved="$md_dir/$link"

        if [ ! -e "$resolved" ]; then
            echo "  BROKEN: $md_file -> $link"
            broken=$((broken + 1))
        fi
    done < <(extract_links "$md_file")
done < <(find_md_files)

echo ""
echo "Checked $checked links across all markdown files."

if [ "$broken" -eq 0 ]; then
    echo "OK: All markdown links resolve correctly."
else
    echo "FAIL: $broken broken link(s) found."
    exit 1
fi
