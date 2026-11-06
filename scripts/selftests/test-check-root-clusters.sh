#!/usr/bin/env bash
# test-check-root-clusters.sh — fixture self-test for check-root-clusters.sh.
#
# Cases:
#   1. a complete partition at the baseline passes
#   2. a production file with no cluster row fails and is named
#   3. a manifest row for a vanished file fails (stale rows hide deletions)
#   4. a file mapped twice fails (a partition cannot double-count)
#   5. a malformed row (one column) fails
#   6. totals that disagree with the size baseline fail
#   7. a missing manifest exits 2
#   8. a missing baseline exits 2
#   9. an empty root exits 2
#  10. --emit prints the markdown table including the reconciled total
#  11. an unknown argument exits 2
#
# Usage: bash scripts/selftests/test-check-root-clusters.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TOOL="$SCRIPT_DIR/../checks/check-root-clusters.sh"

WORK=$(mktemp -d /tmp/test-check-root-clusters-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

pass=0
fail=0
ok() {
  echo "  ok $1"
  pass=$((pass + 1))
}
bad() {
  echo "  FAIL $1: $2" >&2
  fail=$((fail + 1))
}

run() {
  local r="$1"
  shift
  ROOT_CLUSTERS_ROOT="$r" ROOT_CLUSTERS_MANIFEST="$r/manifest.tsv" \
    ROOT_CLUSTERS_BASELINE="$r/baseline" bash "$TOOL" "$@" 2>&1
}

# mkfixture <dir> — two production files of 10 lines each, mapped, baseline 2 20
mkfixture() {
  local r="$1"
  mkdir -p "$r"
  yes 'package root' | head -n 10 >"$r/alpha.go"
  yes 'package root' | head -n 10 >"$r/beta.go"
  printf 'core\talpha.go\ncore\tbeta.go\n' >"$r/manifest.tsv"
  echo '2 20' >"$r/baseline"
}

# --- 1. valid partition passes --------------------------------------------
r="$WORK/valid"
mkfixture "$r"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q 'partition of the root package (2 files / 20 LOC'; then
  ok "a complete partition passes and reports the totals"
else
  bad "valid partition" "rc=$rc out=$out"
fi

# --- 2. unmapped production file fails ------------------------------------
r="$WORK/unmapped"
mkfixture "$r"
yes 'package root' | head -n 5 >"$r/gamma.go"
echo '2 25' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'gamma.go'; then
  ok "an unmapped production file fails and is named"
else
  bad "unmapped file" "rc=$rc out=$out"
fi

# --- 3. stale manifest row fails ------------------------------------------
r="$WORK/stale"
mkfixture "$r"
printf 'core\talpha.go\ncore\tbeta.go\ncore\tvanished.go\n' >"$r/manifest.tsv"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'vanished.go'; then
  ok "a manifest row for a vanished file fails"
else
  bad "stale row" "rc=$rc out=$out"
fi

# --- 4. duplicate mapping fails ------------------------------------------
r="$WORK/dupe"
mkfixture "$r"
printf 'core\talpha.go\ncore\talpha.go\ncore\tbeta.go\n' >"$r/manifest.tsv"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'more than once'; then
  ok "a double-mapped file fails"
else
  bad "duplicate row" "rc=$rc out=$out"
fi

# --- 5. malformed row fails ----------------------------------------------
r="$WORK/malformed"
mkfixture "$r"
printf 'core\ncore\talpha.go\ncore\tbeta.go\n' >"$r/manifest.tsv"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'malformed manifest row'; then
  ok "a one-column row fails as malformed"
else
  bad "malformed row" "rc=$rc out=$out"
fi

# --- 6. totals disagreeing with the baseline fail ------------------------
r="$WORK/totals"
mkfixture "$r"
echo '2 999' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q 'disagree with'; then
  ok "totals that disagree with the size baseline fail"
else
  bad "totals disagree" "rc=$rc out=$out"
fi

# --- 7. missing manifest exits 2 -----------------------------------------
r="$WORK/no-manifest"
mkfixture "$r"
rm -f "$r/manifest.tsv"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "missing manifest exits 2"; else bad "missing manifest" "rc=$rc out=$out"; fi

# --- 8. missing baseline exits 2 -----------------------------------------
r="$WORK/no-baseline"
mkfixture "$r"
rm -f "$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "missing baseline exits 2"; else bad "missing baseline" "rc=$rc out=$out"; fi

# --- 9. empty root exits 2 -----------------------------------------------
r="$WORK/empty"
mkdir -p "$r"
printf 'core\tnothing.go\n' >"$r/manifest.tsv"
echo '0 0' >"$r/baseline"
out="$(run "$r")"
rc=$?
if [ "$rc" -eq 2 ]; then ok "an empty root exits 2"; else bad "empty root" "rc=$rc out=$out"; fi

# --- 10. --emit prints the markdown table --------------------------------
r="$WORK/emit"
mkfixture "$r"
out="$(run "$r" --emit)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q '| \*\*Total\*\* | \*\*2\*\* | \*\*20\*\* |'; then
  ok "--emit prints the markdown table with the reconciled total"
else
  bad "emit" "rc=$rc out=$out"
fi

# --- 11. unknown argument exits 2 ----------------------------------------
r="$WORK/bad-arg"
mkfixture "$r"
out="$(run "$r" --bogus)"
rc=$?
if [ "$rc" -eq 2 ]; then ok "unknown argument exits 2"; else bad "unknown argument" "rc=$rc out=$out"; fi

echo
if [ "$fail" -eq 0 ]; then
  echo "✓ root-clusters self-test: $pass/$pass cases passed"
  exit 0
fi
echo "✗ root-clusters self-test: $fail case(s) failed ($pass passed)" >&2
exit 1
