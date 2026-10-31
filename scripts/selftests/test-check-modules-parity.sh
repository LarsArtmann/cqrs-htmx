#!/usr/bin/env bash
# test-check-modules-parity.sh — M11 lock: the check-modules DEFAULT path must
# run the SAME stage set --report runs (one stages array, report-then-fail),
# and the --sequential fast path must be a strict SUBSET of it. The pre-M11
# default silently skipped ~24 stages (the gotcha-40-local class: a local
# green that verified only part of the gate bundle); this fixture fails if
# the flake ever drifts back into two divergent lists.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
FLAKE="$REPO_ROOT/flake.nix"

fail=0
pass=0
ok() { echo "  ok $1: $2"; pass=$((pass + 1)); }
bad() { echo "  FAIL $1: $2" >&2; fail=$((fail + 1)); }

# Extract the check-modules app block.
block="$(awk '/^            check-modules = \{/{flag=1} flag{print} flag && /^            \};$/{exit}' "$FLAKE")"
[ -n "$block" ] || { bad 0 "check-modules block not found in flake.nix"; exit 1; }

# Stage array entries: quoted "name:cmd" lines between stages=( and the ).
stage_entries="$(awk '/stages=\(/{flag=1;next} flag && /^\s*\)/{exit} flag' <<<"$block" | grep -oE '"[a-z0-9-]+:(bash|python3) [^"]+"')"
[ -n "$stage_entries" ] || { bad 1 "stage array not found"; exit 1; }

# Sequential branch: `bash <cmd>` lines between the --sequential guard and its exit 0.
seq_cmds="$(awk '/"--sequential"/{flag=1;next} flag && /exit 0/{exit} flag' <<<"$block" | grep -oE '(bash|python3) scripts[^"]*$' | sed -E 's/^(bash|python3) //')"
[ -n "$seq_cmds" ] || { bad 2 "sequential branch not found"; exit 1; }

# Check 1: stage names are unique.
dups="$(cut -d: -f1 <<<"$stage_entries" | sort | uniq -d)"
if [ -z "$dups" ]; then
  ok 3 "stage names unique"
else
  bad 3 "duplicate stage names: $dups"
fi

# Check 2: every sequential command appears verbatim in the stage array.
stage_cmds="$(sed -E 's/^[^:]+:(bash|python3) //' <<<"$stage_entries" | tr -d '"')"
missing=0
while IFS= read -r sc; do
  [ -n "$sc" ] || continue
  if ! grep -qxF "$sc" <<<"$stage_cmds"; then
    bad 4 "sequential command not in stage array: $sc"
    missing=1
  fi
done <<<"$seq_cmds"
[ "$missing" -eq 0 ] && ok 4 "every --sequential command is a named stage (subset holds)"

# Check 3: the default path is report-then-fail — the stages array + report
# loop exist OUTSIDE the --sequential guard, and a summary/exit-1 tail is present.
if grep -q 'stages=(' <<<"$block" &&
  grep -q 'for stage in' <<<"$block" &&
  grep -q 'exit 1' <<<"$block"; then
  ok 5 "default path runs every stage (report-then-fail)"
else
  bad 5 "default path lost the report-then-fail shape"
fi

# Check 4: the description still names the default contract.
if grep -q 'report-then-fail' <<<"$block"; then
  ok 6 "flake description documents report-then-fail default"
else
  bad 6 "description no longer documents the default contract"
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-check-modules-parity: $fail case(s) FAILED"
  exit 1
fi
echo "test-check-modules-parity: all cases green"
exit 0
