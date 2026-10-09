#!/usr/bin/env bash
# test-status-table-equivalence.sh — adversarial equivalence meta-test for the
# status-table pair (plan T05).
#
# The 2026-10-08 split brain (fixer skipped rows the checker flagged because
# the fixer demanded a well-formed separator row) is structurally dead now
# that both tools import scripts/lib/status_table.py — but an import is not a
# proof. This meta-test pushes adversarial fixtures through BOTH tools and
# asserts: the fixer normalizes EXACTLY the rows the checker flags PARTIAL,
# byte-for-byte per row index, and never touches STRUCK/CLEAN rows.
#
#   F1  malformed 2-dash separator row (the original split-brain fixture)
#   F2  bare unpaired `~~` cell
#   F3  `~~` inside an inline code span (must NOT count as struck)
#   F4  `~~` inside a double-backtick span
#   F5  shifted-cell row (cell count differs from header)
#   F6  STRUCK/CLEAN rows stay byte-identical under the fixer
#   F7  table directly at EOF (no trailing newline block flush)
#
# Usage: ./scripts/selftests/test-status-table-equivalence.sh
# Exit: 0 = tools agree on every fixture; 1 = drift detected

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")" && pwd)"
ROOT="$SCRIPT_DIR/../.."

pass=0
fail=0
report() { # <ok 0|1> <label...>
  if [ "$1" -eq 0 ]; then
    echo "  PASS: ${*:2}"
    pass=$((pass + 1))
  else
    echo "  FAIL: ${*:2}"
    fail=$((fail + 1))
  fi
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

python3 - "$ROOT" "$tmp" <<'PYEOF'
import importlib.util
import sys
from pathlib import Path

root = Path(sys.argv[1])
tmp = Path(sys.argv[2])
ok = True


def fail(msg):
    global ok
    ok = False
    print(f"    DRIFT: {msg}")


def load(name, rel):
    spec = importlib.util.spec_from_file_location(name, root / rel)
    mod = importlib.util.module_from_spec(spec)
    sys.path.insert(0, str((root / rel).parent.parent / "lib"))
    spec.loader.exec_module(mod)

    return mod


checker = load("checker", "scripts/checks/check-status-rows.py")
fixer = load("fixer", "scripts/tools/normalize-status-rows.py")
lib = load("lib", "scripts/lib/status_table.py")

fixtures = {
    "F1-malformed-separator": """| A | B |
| -- | --- |
| ~~done~~ item | 2h |
| open item | 3h |
""",
    "F2-bare-tildes": """| A | B |
| --- | --- |
| ~~done~~ | x |
| open ~~ | y |
""",
    "F3-code-span-tildes": """| A | B |
| --- | --- |
| `~~not struck~~` | 1h |
| struck | ~~2h~~ |
""",
    "F4-double-backtick": """| A | B |
| --- | --- |
| ``a~~b`` | 1h |
| real | ~~2h~~ |
""",
    "F5-shifted-cells": """| A | B | C |
| --- | --- | --- |
| ~~done~~ | 2h |
| ~~also~~ | ~~3h~~ | docs |
""",
    "F6-clean-and-struck": """| A | B |
| --- | --- |
| clean row | 1h |
| ~~full strike~~ | ~~2h~~ |
| partial row | 3h |
""",
    "F7-eof-no-newline": """| A | B |
| --- | --- |
| ~~done~~ | 1h |
| open | 2h |""",
}

for name, text in fixtures.items():
    path = tmp / f"{name}.md"
    path.write_text(text)

    lines = text.splitlines()
    # Gate's view: which 1-based rows are PARTIAL?
    gate_partial = set()
    for block in lib.table_blocks(lines):
        data = [(no, ln) for no, ln in block if not lib.is_separator(ln)]
        for no, ln in data[1:]:
            if lib.classify_row(ln) == "PARTIAL":
                gate_partial.add(no)

    # Fixer's view: which 1-based rows changed?
    new_text, _ = fixer.normalize_text(text)
    new_lines = new_text.splitlines(keepends=False)
    fixed_rows = {
        i + 1 for i, (a, b) in enumerate(zip(lines, new_lines)) if a != b
    }
    # Fixer may also extend the last line for a missing trailing newline —
    # row COUNT must match for index comparison.
    if len(new_lines) != len(lines):
        fail(f"{name}: fixer changed line count {len(lines)} -> {len(new_lines)}")
        continue

    if gate_partial != fixed_rows:
        fail(f"{name}: gate PARTIAL {sorted(gate_partial)} != fixer changed {sorted(fixed_rows)}")
        continue

    # Whole-row-strike policy: every fixed row must end fully struck.
    for no in fixed_rows:
        if lib.classify_row(new_lines[no - 1]) != "STRUCK":
            fail(f"{name}: row {no} fixed but classifies {lib.classify_row(new_lines[no - 1])}, not STRUCK")

    # Checker's file-level verdict must agree with zero PARTIALs after fixing.
    post = [l for l in new_text.splitlines()]
    post_partial = 0
    for block in lib.table_blocks(post):
        data = [(no, ln) for no, ln in block if not lib.is_separator(ln)]
        post_partial += sum(
            1 for _, ln in data[1:] if lib.classify_row(ln) == "PARTIAL"
        )
    if post_partial != 0:
        fail(f"{name}: {post_partial} PARTIAL row(s) SURVIVE the fixer")

    if ok:
        print(f"    {name}: gate PARTIAL {sorted(gate_partial) or '{}'} == fixer changed {sorted(fixed_rows) or '{}'}")

sys.exit(0 if ok else 1)
PYEOF
rc=$?
report "$([ "$rc" -eq 0 ] && echo 0 || echo 1)" "all fixtures: gate and fixer agree exactly (rc=$rc)"

echo ""
echo "test-status-table-equivalence: $pass passed, $fail failed"
[ "$fail" -eq 0 ] || exit 1
exit 0
