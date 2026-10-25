#!/usr/bin/env bash
# test-check-templ-codegen-staged.sh — fixture self-test for the staged-templ
# codegen-drift gate (S48/D03).
#
# A throwaway git repo with an adminui/ module pair (x.templ + x_templ.go) and
# STUB templ/gofmt binaries on PATH (the stub regenerates _templ.go as a
# deterministic function of the .templ content — offline, no toolchain):
#   1. no staged templ files  → fast skip, rc=0;
#   2. staged .templ edit with a STALE _templ.go → rc=1, names the module + fix;
#   3. staged .templ edit WITH the regenerated _templ.go staged too → rc=0;
#   4. hand-edited _templ.go (no .templ change) → rc=1 (regen overwrites it).
#
# Usage: bash scripts/selftests/test-check-templ-codegen-staged.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-templ-codegen-staged.sh"

WORK=$(mktemp -d /tmp/test-templ-staged-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

# Stub toolchain: `templ generate` maps every *.templ to a *_templ.go that
# embeds the .templ content (deterministic); `gofmt` is an identity pass.
mkdir -p "$WORK/bin"
cat >"$WORK/bin/templ" <<'EOF'
#!/usr/bin/env bash
# stub templ: "generate" copies each .templ to its _templ.go mirror
[ "${1:-}" = "generate" ] || exit 0
shopt -s nullglob
for f in *.templ; do
  out="${f%.templ}_templ.go"
  {
    echo "// generated from $f (stub)"
    cat "$f"
  } >"$out"
done
exit 0
EOF
cat >"$WORK/bin/gofmt" <<'EOF'
#!/usr/bin/env bash
# stub gofmt: identity (files are already in final form)
exit 0
EOF
chmod +x "$WORK/bin/templ" "$WORK/bin/gofmt"

# Fixture repo with a committed baseline pair.
git init -q "$WORK/repo"
cd "$WORK/repo"
git config user.email fixture@test.local
git config user.name fixture
mkdir adminui
printf 'component X v1\n' >adminui/x.templ
{
  echo "// generated from x.templ (stub)"
  echo "component X v1"
} >adminui/x_templ.go
git add -A
git commit -qm baseline

run_gate() {
  PATH="$WORK/bin:$PATH" bash "$GATE" 2>&1
}

pass=0
fail=0

# Case 1: staged non-templ file → fast skip.
echo "unrelated" >notes.txt
git add notes.txt
out=$(run_gate)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "no staged .templ/_templ.go changes — skip"; then
  echo "  ok 1: non-templ staging is a fast skip"
  pass=$((pass + 1))
else
  echo "  FAIL 1 (rc=$rc): $out" >&2
  fail=$((fail + 1))
fi

# Case 2: staged .templ edit with STALE staged _templ.go → red, names module + fix.
printf 'component X v2\n' >adminui/x.templ
git add adminui/x.templ
out=$(run_gate)
rc=$?
if [ "$rc" -eq 1 ] &&
  printf '%s' "$out" | grep -q "codegen drift in adminui" &&
  printf '%s' "$out" | grep -q "adminui/x_templ.go" &&
  printf '%s' "$out" | grep -q "nix run .#gen"; then
  echo "  ok 2: stale _templ.go next to a staged .templ edit fails with the fix hint"
  pass=$((pass + 1))
else
  echo "  FAIL 2 (rc=$rc): $out" >&2
  fail=$((fail + 1))
fi

# Case 3: regenerate + stage the pair → green.
git add adminui/x_templ.go
out=$(run_gate)
rc=$?
if [ "$rc" -eq 0 ] && printf '%s' "$out" | grep -q "adminui — no drift"; then
  echo "  ok 3: regenerated pair committed together passes"
  pass=$((pass + 1))
else
  echo "  FAIL 3 (rc=$rc): $out" >&2
  fail=$((fail + 1))
fi
git commit -qm "pair v2"

# Case 4: hand-edit the committed _templ.go only → red (regen overwrites).
printf 'hand edit\n' >>adminui/x_templ.go
git add adminui/x_templ.go
out=$(run_gate)
rc=$?
if [ "$rc" -eq 1 ] && printf '%s' "$out" | grep -q "codegen drift in adminui"; then
  echo "  ok 4: hand-edited _templ.go is caught (regeneration is canonical)"
  pass=$((pass + 1))
else
  echo "  FAIL 4 (rc=$rc): $out" >&2
  fail=$((fail + 1))
fi

echo ""
if [ "$fail" -gt 0 ]; then
  echo "test-check-templ-codegen-staged: $fail case(s) FAILED"
  exit 1
fi
echo "test-check-templ-codegen-staged: all cases green"
