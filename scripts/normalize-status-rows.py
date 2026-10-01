#!/usr/bin/env python3
"""Fixer for PARTIAL table rows in annotated status reports (docs-health Gate 2).

The checker (`check-status-rows.py`) flags a PARTIAL row: a data row whose
cells disagree on strikethrough (some struck, some not). In practice these come
from an annotator striking only the item cell of a DONE row and leaving the
Effort/Category/Route cells unstruck — an aborted annotation, invisible while
the file lives outside `docs/status/archived/` (the gate scans that dir only)
and surfaced when the file is archived.

Policy (docs/status/README.md; established by the 2026-10-01 round-13 batch
that normalized 72 rows): a PARTIAL row is treated as DONE and completed by
striking every remaining cell — the "whole-row-strike" normalizer. CLEAN rows
and fully-STRUCK rows are never touched; mixed tables stay mixed.

Strikethrough already present inside an inline code span never counts (the
checker ignores it), so a cell holding only `~~x~~` literal text is still
wrapped once.

Usage:
  python3 scripts/normalize-status-rows.py [--dry-run] [path...]
  (no paths: docs/status/archived/*.md)

Exit: 0 = normalized (or nothing to do); 1 = a file could not be read.
"""

import re
import sys
from pathlib import Path

DEFAULT_DIR = Path("docs/status/archived")


def outside_code_spans(text: str) -> str:
    without_double = re.sub(r"``[^`]+``", "", text)

    return re.sub(r"`[^`]*`", "", without_double)


def is_separator(line: str) -> bool:
    cells = [c.strip() for c in line.strip().strip("|").split("|")]

    return bool(cells) and all(re.fullmatch(r":?-{3,}:?", c) for c in cells)


def strike_cell(cell: str) -> str:
    """Wrap one raw table cell's content in ~~ unless it is already struck."""
    if "~~" in outside_code_spans(cell):
        return cell
    stripped = cell.strip()
    if not stripped:
        return cell

    return cell.replace(stripped, f"~~{stripped}~~", 1)


def classify_row(line: str) -> str:
    cells = line.strip().strip("|").split("|")
    states = ["~~" in outside_code_spans(c) for c in cells]

    if all(states):
        return "STRUCK"
    if not any(states):
        return "CLEAN"

    return "PARTIAL"


def normalize_row(line: str) -> str:
    parts = line.split("|")
    if len(parts) < 3:
        return line
    # parts[0] and parts[-1] are the empty edges around the leading/trailing |.
    parts[1:-1] = [strike_cell(c) for c in parts[1:-1]]

    return "|".join(parts)


def normalize_text(text: str) -> tuple[str, int]:
    """Returns (new_text, changed_row_count)."""
    lines = text.splitlines(keepends=True)
    in_table = False
    header_seen = False
    changed = 0

    for i, raw in enumerate(lines):
        line = raw.rstrip("\n")
        if line.lstrip().startswith("|"):
            if not in_table:
                in_table = True
                header_seen = False
            elif not header_seen:
                # First line after the header is the separator row.
                if is_separator(line):
                    header_seen = True
                    continue
            if not header_seen or is_separator(line):
                continue
            if classify_row(line) == "PARTIAL":
                lines[i] = normalize_row(line) + ("\n" if raw.endswith("\n") else "")
                changed += 1
        else:
            in_table = False
            header_seen = False

    return "".join(lines), changed


def resolve_args(argv: list[str]) -> list[Path]:
    if not argv:
        return sorted(DEFAULT_DIR.glob("*.md"))
    files: list[Path] = []
    for arg in argv:
        path = Path(arg)
        if path.is_dir():
            files.extend(sorted(path.glob("*.md")))
        else:
            files.append(path)

    return files


def main() -> int:
    argv = sys.argv[1:]
    dry_run = "--dry-run" in argv
    argv = [a for a in argv if a != "--dry-run"]
    files = resolve_args(argv)
    if not files:
        print(f"✗ normalize rows: no markdown files found (looked at {DEFAULT_DIR})")

        return 1

    total = 0
    for path in files:
        if not path.is_file():
            print(f"  ✗ MISSING: {path}")

            return 1
        original = path.read_text()
        updated, changed = normalize_text(original)
        if changed and not dry_run:
            path.write_text(updated)
        if changed:
            print(f"  {'would normalize' if dry_run else 'normalized'} {changed} row(s) in {path}")
            total += changed

    if total == 0:
        print(f"✓ normalize rows: nothing to fix across {len(files)} file(s)")
    elif dry_run:
        print(f"· normalize rows (dry-run): {total} row(s) would be fixed")
    else:
        print(f"✓ normalize rows: {total} row(s) normalized across {len(files)} file(s)")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
