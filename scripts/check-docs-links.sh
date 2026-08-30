#!/usr/bin/env bash
# check-docs-links.sh — Verify all file-path links in markdown files resolve correctly
#
# Scans all .md files for relative file links and checks they exist.
# Filters out:
#   - URLs (http://, https://, mailto:)
#   - Links inside fenced code blocks (``` ... ```)
#   - False positives: targets with spaces, commas, or missing file extensions
#
# Usage: ./scripts/check-docs-links.sh
# Exit: 0 = all links valid, 1 = at least one broken link found

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

broken=0
checked=0

echo "=== Markdown Link Check ==="
echo "Scanning all .md files for broken file-path links..."
echo ""

while IFS= read -r md_file; do
  md_dir="$(dirname "$md_file")"

  # Use awk to extract links, skipping fenced code blocks.
  # Only accept targets that look like real file paths:
  #   - Ends with a file extension (.md, .go, .sh, .yml, etc.)
  #   - OR starts with ./ or ../ and contains a path separator
  # This excludes Go generics like [T](mapper), function signatures,
  # and other false positives in embedded code samples.
  links=$(awk '
        /^[[:space:]]*```/ { in_code = !in_code; next }
        in_code { next }
        {
            line = $0
            while (match(line, /\[[^][]*\]\([^)]*\)/)) {
                link_part = substr(line, RSTART, RLENGTH)
                if (match(link_part, /\]\(([^)]*)\)/)) {
                    target = substr(link_part, RSTART+2, RLENGTH-3)
                    # Strip anchor/query before testing
                    path = target
                    sub(/[?#].*$/, "", path)
                    # Accept: file extension at end, or explicit relative path with /
                    # Reject: anything with spaces (Go code, not file paths)
                    if (path !~ / / && (path ~ /\.[a-zA-Z][a-zA-Z0-9]*$/ || path ~ /^\.\.?\//)) {
                        print target
                    }
                }
                line = substr(line, RSTART + RLENGTH)
            }
        }
    ' "$md_file")

  while IFS= read -r target; do
    [ -z "$target" ] && continue

    # Skip URLs and email links
    case "$target" in
    http://* | https://* | mailto:*) continue ;;
    esac

    # Strip anchor fragments and query strings
    path_part="${target%%#*}"
    path_part="${path_part%%\?*}"

    # Skip empty targets after stripping (anchor-only links)
    [ -z "$path_part" ] && continue

    checked=$((checked + 1))

    # Resolve relative to the markdown file's directory
    resolved="$md_dir/$path_part"

    if [ ! -e "$resolved" ]; then
      echo "  BROKEN: $md_file -> $target"
      broken=$((broken + 1))
    fi
  done <<<"$links"
done < <(find . -name '*.md' -not -path './.git/*' -not -path '*/node_modules/*' -not -path './vendor/*' | sort)

echo ""
echo "Checked $checked links across all markdown files."

if [ "$broken" -eq 0 ]; then
  echo "OK: All markdown links resolve correctly."
else
  echo "FAIL: $broken broken link(s) found."
  exit 1
fi
