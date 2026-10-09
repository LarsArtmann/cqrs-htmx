"""Shared markdown-status-table primitives.

Single source of truth for the PARTIAL-row gate (scripts/checks/check-status-rows.py)
and the fixer (scripts/tools/normalize-status-rows.py). The 2026-10-08 split
brain (fixer required a well-formed separator; gate flagged rows in any
pipe-block) existed because both tools carried private copies of this logic
that had drifted. Rule: block/row semantics may only change HERE, and the
equivalence meta-test (scripts/selftests/test-status-table-equivalence.sh)
pins both tools to identical verdicts.
"""

import re

DEFAULT_DIR_ARG_HELP = "default: docs/status/archived/*.md"


def outside_code_spans(text: str) -> str:
    """Remove inline code spans so `~~` inside backticks is not strike markup."""
    without_double = re.sub(r"``[^`]+``", "", text)

    return re.sub(r"`[^`]*`", "", without_double)


def cells_of(line: str) -> list[str]:
    """Whitespace-stripped pipe-table cells (canonical cell shape)."""
    return [c.strip() for c in line.strip().strip("|").split("|")]


def is_separator(line: str) -> bool:
    """True for the `| --- | :---: |` alignment row of a pipe table."""
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
    """Yield contiguous pipe-line blocks as (1-based lineno, line) tuples.

    A block starts at the first line whose stripped form begins with `|` and
    ends at the first line that does not. The separator row's well-formedness
    is irrelevant here — the gate flags rows in ANY pipe-block, so the fixer
    must normalize in ANY pipe-block too.
    """
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
