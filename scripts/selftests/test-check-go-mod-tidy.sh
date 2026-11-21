#!/usr/bin/env bash
# test-check-go-mod-tidy.sh — offline fixture self-test for check-go-mod-tidy.sh.
#
# Builds a throwaway tree with a two-member go.work: one clean module (no
# requires) and one DRIFT module (a stale unimported require that tidy would
# remove). Expects: drift named + exit 1; then the clean-only variant exits 0.
# No network: the drift require is never imported, so tidy only ever REMOVES.
set -uo pipefail

GATE="$(cd "$(dirname "${BASH_SOURCE[0]}")/../checks" && pwd)/check-go-mod-tidy.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$tmp/cleanmod" "$tmp/driftmod"
cat > "$tmp/go.work" <<'EOF'
go 1.27

use (
	./cleanmod
	./driftmod
)
EOF

cat > "$tmp/cleanmod/go.mod" <<'EOF'
module example.com/cleanmod

go 1.27
EOF
printf 'package cleanmod\n\nfunc Hello() string { return "hi" }\n' > "$tmp/cleanmod/hello.go"

cat > "$tmp/driftmod/go.mod" <<'EOF'
module example.com/driftmod

go 1.27

require github.com/someone/unused v1.2.3
EOF
printf 'package driftmod\n\nfunc Hello() string { return "hi" }\n' > "$tmp/driftmod/hello.go"

fail=0

# Case 1: drift module is named, gate exits 1.
out="$(TIDY_GATE_ROOT="$tmp" bash "$GATE" 2>&1)"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "driftmod" && printf '%s' "$out" | grep -q "DRIFT"; then
  echo "  ok 1: drift module named, rc=1"
else
  echo "  FAIL 1: expected rc=1 naming driftmod, got rc=$rc:"; printf '%s\n' "$out" | sed 's/^/    /'
  fail=1
fi

# Case 2: clean-only tree passes.
rm "$tmp/driftmod/go.mod"
cat > "$tmp/go.work" <<'EOF'
go 1.27

use (
	./cleanmod
)
EOF
out="$(TIDY_GATE_ROOT="$tmp" bash "$GATE" 2>&1)"
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "tidy-clean"; then
  echo "  ok 2: clean tree passes, rc=0"
else
  echo "  FAIL 2: expected rc=0 tidy-clean, got rc=$rc:"; printf '%s\n' "$out" | sed 's/^/    /'
  fail=1
fi

# Case 3: missing go.work is usage (rc=2).
out="$(TIDY_GATE_ROOT="$tmp/nowhere" bash "$GATE" 2>&1)"
rc=$?
if [ "$rc" -eq 2 ]; then
  echo "  ok 3: missing go.work refused, rc=2"
else
  echo "  FAIL 3: expected rc=2, got rc=$rc"; printf '%s\n' "$out" | sed 's/^/    /'
  fail=1
fi

# Case 4: a go.work member without go.mod is a named failure (rc=1).
cat > "$tmp/go.work" <<'EOF'
go 1.27

use (
	./cleanmod
	./ghostmod
)
EOF
mkdir -p "$tmp/ghostmod"
out="$(TIDY_GATE_ROOT="$tmp" bash "$GATE" 2>&1)"
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "ghostmod"; then
  echo "  ok 4: go.mod-less member named, rc=1"
else
  echo "  FAIL 4: expected rc=1 naming ghostmod, got rc=$rc:"; printf '%s\n' "$out" | sed 's/^/    /'
  fail=1
fi

if [ "$fail" -ne 0 ]; then
  echo "test-check-go-mod-tidy: FAILURE(S)"
  exit 1
fi
echo "test-check-go-mod-tidy: all cases green"
