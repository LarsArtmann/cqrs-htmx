#!/usr/bin/env python3
"""Re-base relative markdown links after a file moved between directories.

Archiving a report adds one directory level, so every relative link inside it
silently points at the wrong place. Hand-rebasing is where the last two archive
passes lost links (23, then 4, then 12 broken in the round-22 sweep). This tool
computes the change instead of guessing it.

Rules:
  * Only relative links are touched. Absolute URLs (`scheme://`), absolute
    paths (`/...`), pure anchors (`#...`) and `mailto:` are left alone.
  * A target's `#anchor` / `?query` is preserved; only the path is rewritten.
  * The rewrite preserves the RESOLVED absolute target: what the link pointed
    at before the move, it still points at after.
  * Links whose pre-move target does not exist are reported as warnings and
    left untouched. A pre-existing broken link is not this tool's job; the
    repository's `check-docs-links.sh` gate owns that verdict.
  * Fenced code blocks are skipped, exactly like the link gate.

Usage:
  python3 scripts/tools/rebase_links.py <repo-root> <file> <old-dir> <new-dir>

`<old-dir>` / `<new-dir>` are repo-relative directories (the file's location
before and after the move). Dry-run with `--dry-run`.

Exit: 0 = rewrote or nothing to do, 1 = file missing / bad usage.
"""

from __future__ import annotations

import os
import re
import sys
from pathlib import Path

LINK = re.compile(r"\]\(([^)\s]+)\)")
ABSOLUTE = re.compile(r"^(?:[a-zA-Z][a-zA-Z0-9+.-]*:|/|#)")


def split_target(target: str) -> tuple[str, str]:
    for i, ch in enumerate(target):
        if ch in "?#":
            return target[:i], target[i:]
    return target, ""


def rebase_line(
    line: str, old_dir: Path, new_dir: Path, root: Path, file: str
) -> tuple[str, list[str]]:
    warnings: list[str] = []

    def repl(match: re.Match[str]) -> str:
        target = match.group(1)
        if not target or ABSOLUTE.match(target):
            return match.group(0)
        path, suffix = split_target(target)
        if not path:
            return match.group(0)
        old_target = Path(os.path.normpath(old_dir / path))
        if not (root / old_target).exists():
            warnings.append(f"{file}: unresolved pre-move link target: {target}")
            return match.group(0)
        new_path = os.path.relpath(root / old_target, root / new_dir)
        return "](" + new_path + suffix + ")"

    return LINK.sub(repl, line), warnings


def rebase(
    file_path: Path, old_dir: Path, new_dir: Path, root: Path, dry_run: bool
) -> int:
    if not file_path.is_file():
        print(f"✗ rebase-links: not a file: {file_path}", file=sys.stderr)
        return 1

    lines = file_path.read_text(encoding="utf-8").splitlines(keepends=True)
    out: list[str] = []
    warnings: list[str] = []
    in_code = False
    changed = 0
    rel_file = os.path.relpath(file_path, root)

    for line in lines:
        stripped = line.lstrip()
        if stripped.startswith("```"):
            in_code = not in_code
            out.append(line)
            continue
        if in_code:
            out.append(line)
            continue
        new_line, warns = rebase_line(line, old_dir, new_dir, root, rel_file)
        if new_line != line:
            changed += 1
            for w in warns:
                warnings.append(w)
            if dry_run:
                print(f"  would rewrite: {line.strip()} -> {new_line.strip()}")
        else:
            warnings.extend(warns)
        out.append(new_line)

    for warning in warnings:
        print(f"⚠ rebase-links: {warning}")

    if changed and not dry_run:
        file_path.write_text("".join(out), encoding="utf-8")
    verb = "would rewrite" if dry_run else "rewrote"
    print(f"rebase-links: {verb} {changed} line(s) in {rel_file}")
    return 0


def main(argv: list[str]) -> int:
    args = [a for a in argv[1:] if a != "--dry-run"]
    dry_run = "--dry-run" in argv
    if len(args) != 4:
        print(
            "usage: rebase_links.py <repo-root> <file> <old-dir> <new-dir> [--dry-run]",
            file=sys.stderr,
        )
        return 1
    root = Path(args[0]).resolve()
    file_path = Path(args[1]).resolve()
    old_dir = Path(args[2])
    new_dir = Path(args[3])
    return rebase(file_path, old_dir, new_dir, root, dry_run)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
