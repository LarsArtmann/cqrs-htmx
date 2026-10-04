#!/usr/bin/env bash
# test-erraudit-inventory.sh — offline fixture self-test for
# erraudit-inventory.sh. Stubs the erraudit binary (ERRAUDIT_BIN) so no real
# tool or network is needed. Covers: healthy sweep (candidates printed, TOTAL
# from CSV rows), violation counting + --gate failure, the zero-candidate
# false-green guard, and the missing-binary guard.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INVENTORY="$SCRIPT_DIR/erraudit-inventory.sh"
TMP=$(mktemp -d /tmp/erraudit-inventory-test-XXXXXX)
trap 'rm -rf "$TMP"' EXIT
failures=0

check() { # check <name> <expected-rc> <actual-rc> <required-grep> <output>
  local name="$1" want_rc="$2" got_rc="$3" pattern="$4" out="$5"
  if [ "$want_rc" != "$got_rc" ]; then
    echo "FAIL $name: rc=$got_rc want=$want_rc"
    echo "$out" | head -5
    failures=$((failures + 1))
    return
  fi
  if [ -n "$pattern" ] && ! printf '%s\n' "$out" | grep -q "$pattern"; then
    echo "FAIL $name: output missing '$pattern'"
    echo "$out" | head -5
    failures=$((failures + 1))
    return
  fi
  echo "ok   $name"
}

# Stub erraudit: emits header + N violation rows; row count driven by which
# module path it is handed (mod-a -> 2 rows, mod-b -> 0).
cat >"$TMP/fake-erraudit" <<'STUB'
#!/usr/bin/env bash
echo "Type,Severity,File,Line,Column,Message"
case "$1" in
  */mod-a)
    echo "context_loss,critical,service.go,10,2,lost x"
    echo "context_loss,critical,service.go,20,2,lost y" ;;
esac
exit 0
STUB
chmod +x "$TMP/fake-erraudit"

# Workspace with two use-block members.
mkdir -p "$TMP/ws/mod-a" "$TMP/ws/mod-b"
for m in mod-a mod-b; do
  printf 'module example.com/%s/v4\n\ngo 1.27\n' "$m" >"$TMP/ws/$m/go.mod"
done
cat >"$TMP/ws/go.work" <<'EOF'
go 1.27

use (
	./mod-a
	./mod-b
)
EOF

out=$(ERRAUDIT_BIN="$TMP/fake-erraudit" bash "$INVENTORY" "$TMP/ws" 2>&1)
check "healthy-sweep candidates+TOTAL" 0 $? 'candidates=2' "$out"
printf '%s\n' "$out" | grep -q '  ./mod-a: 2' || {
  echo "FAIL healthy-sweep: mod-a count wrong"
  echo "$out"
  failures=$((failures + 1))
}
printf '%s\n' "$out" | grep -q 'TOTAL=2' || {
  echo "FAIL healthy-sweep: TOTAL wrong"
  echo "$out"
  failures=$((failures + 1))
}

out=$(ERRAUDIT_BIN="$TMP/fake-erraudit" bash "$INVENTORY" --gate "$TMP/ws" 2>&1)
check "gate-mode fails on TOTAL>0" 1 $? 'gate mode' "$out"

# Zero rows: empty stub must give TOTAL=0 and pass in gate mode.
cat >"$TMP/fake-empty" <<'STUB'
#!/usr/bin/env bash
echo "Type,Severity,File,Line,Column,Message"
exit 0
STUB
chmod +x "$TMP/fake-empty"
out=$(ERRAUDIT_BIN="$TMP/fake-empty" bash "$INVENTORY" --gate "$TMP/ws" 2>&1)
check "gate-mode passes on TOTAL=0" 0 $? 'TOTAL=0' "$out"

# False-green guard: go.work without use entries.
mkdir -p "$TMP/ws-broken"
printf 'go 1.27\n' >"$TMP/ws-broken/go.work"
out=$(ERRAUDIT_BIN="$TMP/fake-erraudit" bash "$INVENTORY" "$TMP/ws-broken" 2>&1)
check "zero-candidate guard fails" 1 $? 'found 0 modules' "$out"

# Missing binary guard.
out=$(ERRAUDIT_BIN="$TMP/definitely-missing" bash "$INVENTORY" "$TMP/ws" 2>&1)
check "missing-binary guard fails" 1 $? 'not found' "$out"

# Erraudit rc!=0 must fail the sweep (no silent partial inventories).
cat >"$TMP/fake-crash" <<'STUB'
#!/usr/bin/env bash
exit 7
STUB
chmod +x "$TMP/fake-crash"
out=$(ERRAUDIT_BIN="$TMP/fake-crash" bash "$INVENTORY" "$TMP/ws" 2>&1)
check "erraudit-failure guard fails" 1 $? 'rc=7' "$out"

if [ "$failures" -gt 0 ]; then
  echo "erraudit-inventory self-test: $failures FAILURE(S)"
  exit 1
fi
echo "erraudit-inventory self-test: all cases green"
