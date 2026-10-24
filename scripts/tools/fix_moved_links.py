#!/usr/bin/env python3
"""Re-point every markdown link that pointed at a file which has just moved.

`git mv` only rewrites the moved file's own relative links (see
rebase_links.py). Inbound links are the other half: a live doc that cited
`../status/<report>.md`, or an archived doc that cited it as `../<report>.md`,
now points at nothing. Round-22 left 20 of these broken.

This tool scans every markdown file under the repo root and, for each relative
link whose RESOLVED target equals one of the moved files' OLD paths, rewrites
the link to the correct relative path to the NEW location. Code fences, absolute
URLs, absolute paths and pure anchors are skipped.

Usage:
  python3 scripts/tools/fix_moved_links.py <repo-root> <old> <new> [<old> <new> ...]

Paths are repo-relative. `--dry-run` prints what would change.

Exit: 0 = done (changed or not), 1 = bad usage / path does not exist.
"""

from __future__ import annotations

import os
import re
import sys
from pathlib import Path

LINK = re.compile(r"\]\(([^)\s]+)\)")
ABSOLUTE = re.compile(r"^(?:[a-zA-Z][a-zA-Z0-9+.-]*:|/|#)")
SKIP_DIRS = {".git", "node_modules", ".direnv", "result"}


def split_target(target: str) -> tuple[str, str]:
    for i, ch in enumerate(target):
        if ch in "?#":
            return target[:i], target[i:]
    return target, ""


def rewrite_line(line: str, file_dir: Path, oldmap: dict[str, str], root: Path) -> tuple[str, int]:
    changed = 0

    def repl(match: re.Match[str]) -> str:
        nonlocal changed
        target = match.group(1)
        if not target or ABSOLUTE.match(target):
            return match.group(0)
        path, suffix = split_target(target)
        if not path:
            return match.group(0)
        resolved = os.path.normpath(file_dir / path)
        new_abs = oldmap.get(resolved)
        if new_abs is None:
            return match.group(0)
        changed += 1
        return "](" + os.path.relpath(new_abs, file_dir) + suffix + ")"

    return LINK.sub(repl, line), changed


def main(argv: list[str]) -> int:
    dry_run = "--dry-run" in argv
    args = [a for a in argv[1:] if a != "--dry-run"]
    if len(args) < 3 or len(args) % 2 == 0:
        print("usage: fix_moved_links.py <repo-root> <old> <new> [<old> <new> ...] [--dry-run]", file=sys.stderr)
        return 1
    root = Path(args[0]).resolve()
    oldmap: dict[str, str] = {}
    for i in range(1, len(args), 2):
        old = os.path.normpath(root / args[i])
        new = os.path.normpath(root / args[i + 1])
        if not (root / args[i + 1]).exists():
            print(f"✗ fix-moved-links: new path does not exist: {args[i + 1]}", file=sys.stderr)
            return 1
        oldmap[old] = new

    files_changed = 0
    links_changed = 0
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP_DIRS]
        for name in filenames:
            if not name.endswith(".md"):
                continue
            path = Path(dirpath) / name
            text = path.read_text(encoding="utf-8")
            out: list[str] = []
            in_code = False
            changed_here = 0
            for line in text.splitlines(keepends=True):
                if line.lstrip().startswith("```"):
                    in_code = not in_code
                    out.append(line)
                    continue
                if in_code:
                    out.append(line)
                    continue
                new_line, n = rewrite_line(line, Path(dirpath), oldmap, root)
                changed_here += n
                out.append(new_line)
            if changed_here:
                files_changed += 1
                links_changed += changed_here
                rel = os.path.relpath(path, root)
                if dry_run:
                    print(f"  would fix {changed_here} link(s) in {rel}")
                else:
                    path.write_text("".join(out), encoding="utf-8")
                    print(f"fix-moved-links: {changed_here} link(s) in {rel}")

    verb = "would fix" if dry_run else "fixed"
    print(f"fix-moved-links: {verb} {links_changed} link(s) across {files_changed} file(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
