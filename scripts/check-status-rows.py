#!/usr/bin/env python3
"""Row-integrity gate for annotated status reports (docs-health Gate 2).

Repo policy is defined in docs/status/README.md:

  * Every data row in an archived report must be *internally consistent*:
    either all of its cells carry strikethrough or none do. A row that mixes
    struck and unstruck cells is a PARTIAL row — an aborted or hand-edited
    annotation. PARTIAL rows are the only hard failure here (the 2026-09-16
    F12.3 miss shipped through this hole).
  * *Mixed tables* (some rows struck = done, others unstruck = open) are
    first-class by convention: struck means resolved, unstruck means still
    open. They are counted and printed for the reader, never failed.

Header and separator rows are exempt. Tildes inside inline code spans (a
literal `~~...~~` in a cell) never count as strikethrough.

This is the repo-owned counterpart to the docs-health skill's authoring-time
check-rows.py: that asset reports mixed tables as INCOMPLETE for the annotator
to adjudicate; this gate encodes the adjudicated policy so a clean checkout
can run it green.

Usage:
  python3 scripts/check-status-rows.py [path...]   # dirs expand to DIR/*.md
  (no arguments: docs/status/archived/*.md)

Exit: 0 = no PARTIAL rows; 1 = at least one PARTIAL row.
"""

import re
import sys
from pathlib import Path

DEFAULT_DIR = Path("docs/status/archived")


def outside_code_spans(text: str) -> str:
    without_double = re.sub(r"``[^`]+``", "", text)

    return re.sub(r"`[^`]*`", "", without_double)


def cells_of(line: str) -> list[str]:
    return [c.strip() for c in line.strip().strip("|").split("|")]


def is_separator(line: str) -> bool:
    cells = cells_of(line)

    return bool(cells) and all(re.fullmatch(r":?-{3,}:?", c) for c in cells)


def classify_row(line: str) -> str:
    """STRUCK / CLEAN / PARTIAL for one table data row."""
    states = ["~~" in outside_code_spans(cell) for cell in cells_of(line)]

    if all(states):
        return "STRUCK"
    if not any(states):
        return "CLEAN"

    return "PARTIAL"


def table_blocks(lines: list[str]):
    block: list[tuple[int, str]] = []

    for i, line in enumerate(lines, start=1):
        if line.lstrip().startswith("|"):
            block.append((i, line))
            continue
        if block:
            yield block
            block = []
    if block:
        yield block


def check_file(path: Path) -> tuple[int, int]:
    """Returns (partial_row_count, mixed_table_count)."""
    problems: list[str] = []
    mixed_tables = 0

    partial_rows = 0

    for block in table_blocks(path.read_text().splitlines()):
        data = [(no, line) for no, line in block if not is_separator(line)]
        if len(data) < 2:  # header only — nothing to check
            continue

        rows = [(no, line, classify_row(line)) for no, line in data[1:]]
        partial = [(no, line) for no, line, state in rows if state == "PARTIAL"]
        struck = sum(1 for _, _, state in rows if state == "STRUCK")
        partial_rows += len(partial)

        if partial:
            problems.append(
                f"{path.name}: table at line {data[0][0]} has "
                f"{len(partial)} PARTIAL row(s)"
            )
            for no, line in partial:
                problems.append(f"  line {no}: {line[:100]}")
        elif 0 < struck < len(rows):
            mixed_tables += 1

    for problem in problems:
        print(problem)

    return partial_rows, mixed_tables


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
    files = resolve_args(sys.argv[1:])
    if not files:
        print(f"✗ row gate: no markdown files found (looked at {DEFAULT_DIR})")

        return 1

    partial_rows = 0
    missing = 0
    mixed = 0
    for path in files:
        if not path.is_file():
            print(f"  ✗ MISSING: {path}")
            missing += 1
            continue
        file_partials, file_mixed = check_file(path)
        partial_rows += file_partials
        mixed += file_mixed

    print("")
    if partial_rows:
        print(
            f"✗ row gate: {partial_rows} PARTIAL row(s) across "
            f"{len(files)} file(s)"
        )

        return 1

    if missing:
        print(f"✗ row gate: {missing} file(s) not found")

        return 1

    print(
        f"✓ row gate: {len(files)} file(s) free of PARTIAL rows "
        f"({mixed} deliberately-mixed table(s) reported, first-class by convention)"
    )

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
