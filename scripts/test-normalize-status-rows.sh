#!/usr/bin/env bash
# test-normalize-status-rows.sh — fixture self-test for normalize-status-rows.py.
#
# Proves the fixer + the checker agree:
#   1. a PARTIAL row is completed by whole-row strike; CLEAN and STRUCK rows
#      are untouched; a literal `~~x~~` inside an inline code span is wrapped
#      (not mistaken for an existing strike);
#   2. after normalizing, check-status-rows.py reports 0 PARTIAL;
#   3. --dry-run writes nothing;
#   4. the normalizer is idempotent (a second run finds nothing to fix).
#
# Usage: bash scripts/test-normalize-status-rows.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NORMALIZER="$SCRIPT_DIR/normalize-status-rows.py"
CHECKER="$SCRIPT_DIR/check-status-rows.py"

WORK=$(mktemp -d /tmp/test-normalize-status-rows-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0

write_fixture() {
  cat >"$1" <<'EOF'
# Fixture report

| Item | Effort | Status |
| --- | --- | --- |
| ~~already done~~ | ~~S~~ | ~~done~~ |
| partial row | XS | ~~done~~ |
| clean open row | M | open |
| ~~done item~~ | L | `~~literal~~` |
EOF
}

fixture="$WORK/case.md"
write_fixture "$fixture"

# --- Case 1: normalize in place ------------------------------------------
out=$(python3 "$NORMALIZER" "$fixture" 2>&1)
rc=$?
if [ "$rc" -eq 0 ]; then
  echo "  ok 1: normalizer exits 0"
  pass=$((pass + 1))
else
  echo "  FAIL 1: normalizer rc=$rc; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Row 2 (STRUCK) must be byte-identical.
if grep -qF '| ~~already done~~ | ~~S~~ | ~~done~~ |' "$fixture"; then
  echo "  ok 2a: fully-struck row untouched"
  pass=$((pass + 1))
else
  echo "  FAIL 2a: fully-struck row was modified" >&2
  fail=$((fail + 1))
fi

# Row 3 (PARTIAL) must now be fully struck.
if grep -qF '| ~~partial row~~ | ~~XS~~ | ~~done~~ |' "$fixture"; then
  echo "  ok 2b: partial row completed by whole-row strike"
  pass=$((pass + 1))
else
  echo "  FAIL 2b: partial row not completed; got:" >&2
  grep -n 'partial row' "$fixture" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# Row 4 (CLEAN) must be untouched.
if grep -qF '| clean open row | M | open |' "$fixture"; then
  echo "  ok 2c: clean row untouched"
  pass=$((pass + 1))
else
  echo "  FAIL 2c: clean row was modified" >&2
  fail=$((fail + 1))
fi

# Row 5: the code-span literal must NOT be mistaken for an existing strike —
# the whole cell is wrapped, and the checker's code-span stripping still sees a strike.
# shellcheck disable=SC2016  # backticks are literal here (a fixed grep -F pattern)
if grep -qF '| ~~done item~~ | ~~L~~ | ~~`~~literal~~`~~ |' "$fixture"; then
  echo "  ok 2d: code-span literal cell wrapped exactly once"
  pass=$((pass + 1))
else
  echo "  FAIL 2d: code-span cell not wrapped as expected; got:" >&2
  grep -n 'done item' "$fixture" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# --- Case 3: the checker agrees (0 PARTIAL) -------------------------------
if out=$(python3 "$CHECKER" "$fixture" 2>&1); then
  echo "  ok 3: checker reports 0 PARTIAL after normalize"
  pass=$((pass + 1))
else
  echo "  FAIL 3: checker still reports PARTIAL rows; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# --- Case 4: idempotent ---------------------------------------------------
out=$(python3 "$NORMALIZER" "$fixture" 2>&1)
if printf '%s' "$out" | grep -q "nothing to fix"; then
  echo "  ok 4: idempotent (second run finds nothing to fix)"
  pass=$((pass + 1))
else
  echo "  FAIL 4: second run was not a no-op; output:" >&2
  printf '%s\n' "$out" | sed 's/^/      /' >&2
  fail=$((fail + 1))
fi

# --- Case 5: --dry-run writes nothing -------------------------------------
dry="$WORK/dry.md"
write_fixture "$dry"
before=$(cat "$dry")
out=$(python3 "$NORMALIZER" --dry-run "$dry" 2>&1)
after=$(cat "$dry")
if [ "$before" = "$after" ] && printf '%s' "$out" | grep -q "dry-run"; then
  echo "  ok 5: --dry-run leaves the file byte-identical"
  pass=$((pass + 1))
else
  echo "  FAIL 5: --dry-run modified the file or lost the dry-run banner" >&2
  fail=$((fail + 1))
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-normalize-status-rows: $fail case(s) FAILED"
  exit 1
fi
echo "test-normalize-status-rows: all cases green"
