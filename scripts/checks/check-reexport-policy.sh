#!/usr/bin/env bash
# check-reexport-policy.sh — ADR-0062 "Direct Imports Only" enforcement.
#
# The root package grew three permanent alias files because a move without a
# shim felt like a breaking change. It is not: a move IS breaking only for the
# published spelling, so the sanctioned bridge is `// Deprecated:` plus a codemod
# plus exactly one breaking window per major. This gate makes that monotone.
#
# Three rules, against scripts/checks/reexport-baseline.tsv:
#   R1  no alias file outside the baseline, and no file may hold MORE aliases
#       than its baseline row (a ratchet: aliases shrink, never grow)
#   R2  every alias in a `reexport`-class file is `// Deprecated:` AND names the
#       package that actually owns the symbol
#   R3  every file holding an alias is classified in the baseline
#
# Usage:
#   bash scripts/checks/check-reexport-policy.sh              # advisory report
#   bash scripts/checks/check-reexport-policy.sh --strict     # blocking (CI)
#
# Env (fixture self-test):
#   REEXPORT_POLICY_ROOT      repo root to scan (default: this repo)
#   REEXPORT_POLICY_BASELINE  baseline TSV (default: scripts/checks/reexport-baseline.tsv)
#   REEXPORT_POLICY_DECLS     pre-baked declaration TSV (skips the Go scanner)
# Exit: 0 ok / advisory, 1 violations under --strict, 2 usage or missing input.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ROOT="${REEXPORT_POLICY_ROOT:-$REPO_ROOT}"
BASELINE="${REEXPORT_POLICY_BASELINE:-$SCRIPT_DIR/reexport-baseline.tsv}"
SCANNER="$SCRIPT_DIR/reexport_scanner.go"

STRICT=0
while [ "$#" -gt 0 ]; do
  case "$1" in
  --strict) STRICT=1 ;;
  *)
    echo "usage: check-reexport-policy.sh [--strict]" >&2
    exit 2
    ;;
  esac
  shift
done

if [ ! -f "$BASELINE" ]; then
  echo "✗ reexport-policy: missing baseline $BASELINE" >&2
  exit 2
fi

DECLS_FILE="${REEXPORT_POLICY_DECLS:-}"
if [ -z "$DECLS_FILE" ]; then
  DECLS_FILE="$(mktemp)"
  trap 'rm -f "$DECLS_FILE"' EXIT
  if ! (cd "$ROOT" && go run "$SCANNER") >"$DECLS_FILE" 2>"$DECLS_FILE.err"; then
    echo "✗ reexport-policy: inventory scan failed:" >&2
    sed 's/^/    /' "$DECLS_FILE.err" >&2
    rm -f "$DECLS_FILE.err"
    exit 2
  fi
  rm -f "$DECLS_FILE.err"
fi

violations=0
fail() {
  echo "✗ $*" >&2
  violations=$((violations + 1))
}

# --- baseline: path -> class, path -> expected alias count -------------------
declare -A baseline_class=()
declare -A baseline_count=()
while IFS=$'\t' read -r path class count note; do
  case "$path" in '' | '#'*) continue ;; esac
  case "$class" in
  reexport | exempt:unexported | exempt:internal | exempt:domain-vocabulary | exempt:pending) ;;
  *) fail "R3 unclassified: $path declares class '$class' (known: reexport, exempt:unexported, exempt:internal, exempt:domain-vocabulary, exempt:pending)" ;;
  esac
  baseline_class["$path"]="$class"
  baseline_count["$path"]="$count"
done <"$BASELINE"

# --- live declarations ------------------------------------------------------
declare -A live_count=()
total_decls=0
total_files=0

declare -A reported_unclassified=()

while IFS=$'\t' read -r module file name _kind target deprecated names_target _pure; do
  case "$module" in '' | '#'*) continue ;; esac
  total_decls=$((total_decls + 1))
  if [ -z "${live_count[$file]+x}" ]; then
    total_files=$((total_files + 1))
  fi
  live_count["$file"]=$((${live_count[$file]:-0} + 1))

  class="${baseline_class[$file]:-}"

  # R3: every alias-bearing file must be classified (report each file once).
  if [ -z "$class" ]; then
    if [ -z "${reported_unclassified[$file]:-}" ]; then
      reported_unclassified["$file"]=1
      fail "R3 unclassified: $file holds aliases but has no baseline row"
    fi
    continue
  fi

  # R2 applies to compatibility layers only, and only to exported names:
  # an unexported alias is package-private plumbing, not a second import path.
  if [ "$class" != "reexport" ]; then
    continue
  fi
  case "$name" in
  [A-Z]*) ;;
  *) continue ;;
  esac
  if [ "$deprecated" != "true" ]; then
    fail "R2 not-deprecated: $file: $name (= $target.$name) has no // Deprecated: paragraph"
    continue
  fi
  if [ "$names_target" != "true" ]; then
    fail "R2 unnamed-target: $file: $name is deprecated but never names the owning package '$target'"
  fi
done <"$DECLS_FILE"

# --- R1: ratchet + no new alias file ---------------------------------------
for file in "${!live_count[@]}"; do
  expected="${baseline_count[$file]:-}"
  if [ -z "$expected" ]; then
    if [ -n "${baseline_class[$file]:-}" ]; then
      continue
    fi
    fail "R1 new-alias-file: $file is not in the baseline (a move ships repointed consumers, not a shim — ADR-0062)"
    continue
  fi
  actual="${live_count[$file]}"
  if [ "$actual" -gt "$expected" ]; then
    fail "R1 alias-growth: $file holds $actual aliases, baseline froze $expected (aliases may shrink, never grow)"
  elif [ "$actual" -lt "$expected" ]; then
    echo "  note: $file shrank to $actual (baseline $expected) — lower the baseline in the same commit"
  fi
done

# A *_reexport.go with zero alias declarations never shows up in the scan, so
# check the filenames themselves.
while IFS= read -r reexport_file; do
  rel="${reexport_file#"$ROOT"/}"
  if [ -z "${baseline_class[$rel]+x}" ]; then
    fail "R1 new-alias-file: $rel exists but has no baseline row"
  fi
done < <(find "$ROOT" -name '*_reexport.go' -type f \
  -not -path '*/vendor/*' -not -path '*/testdata/*' -not -path '*/.git/*' | LC_ALL=C sort)

# --- verdict ----------------------------------------------------------------
echo "reexport-policy: $total_files alias file(s), $total_decls alias declaration(s), ${#baseline_class[@]} baselined path(s)"

if [ "$violations" -gt 0 ]; then
  echo "✗ reexport-policy: $violations violation(s) of ADR-0062" >&2
  if [ "$STRICT" -eq 0 ]; then
    echo "reexport-policy: advisory — re-run with --strict to block" >&2
    exit 0
  fi
  exit 1
fi

echo "✓ reexport-policy: ADR-0062 holds (no new alias file, no growth, every re-export deprecated and pointed)"
exit 0
