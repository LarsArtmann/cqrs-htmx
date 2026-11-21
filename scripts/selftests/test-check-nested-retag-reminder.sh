#!/usr/bin/env bash
# Fixture self-test for check-nested-retag-reminder.sh — offline: builds a
# throwaway git repo with tagged/untagged nested modules and dates under the
# test's control (GIT_COMMITTER_DATE), one planted failure per rule.
set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$SCRIPT_DIR/../checks/check-nested-retag-reminder.sh"
pass=0
fail=0
report() {
  if [ "$1" -eq 0 ]; then
    pass=$((pass + 1))
    echo "  PASS: ${*:2}"
  else
    fail=$((fail + 1))
    echo "  FAIL: ${*:2}"
  fi
}

tree=$(mktemp -d)
trap 'rm -rf "$tree"' EXIT
cd "$tree"
git init -q .
git config user.email t@t.dev
git config user.name t

# Module layout: root + one nested module tagged BEFORE the fix + one tagged
# AFTER + one never tagged + one that does not require the dep.
mkdir -p usermgmt/oauth2 usermgmt/totp
cat >go.mod <<'EOF'
module github.com/larsartmann/cqrs-htmx/v4
go 1.27
EOF
cat >usermgmt/oauth2/go.mod <<'EOF'
module github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4
go 1.27
require github.com/larsartmann/go-jose/v4 v4.1.5
EOF
cat >usermgmt/totp/go.mod <<'EOF'
module github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4
go 1.27
require github.com/someone-else/dep v1.0.0
EOF
git add -A
git commit -qm init

# oauth2 tagged 2026-10-11 (BEFORE the fix), totp tagged 2026-11-01 (AFTER).
GIT_COMMITTER_DATE="2026-10-11T00:00:00" git tag -a usermgmt/oauth2/v4.13.0 -m t
GIT_COMMITTER_DATE="2026-11-01T00:00:00" git tag -a usermgmt/totp/v4.12.1 -m t

echo "== test-check-nested-retag-reminder.sh"

# F1 the stranded class: oauth2's tag predates the fix -> reminder, exit 1.
out=$(cd "$tree" && git config --global --add safe.directory "$tree" >/dev/null 2>&1; REPO_ROOT_OVERRIDE="$tree" bash "$GATE" github.com/larsartmann/go-jose/v4 2026-10-28)
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
echo "$out" | grep -q "usermgmt/oauth2's latest tag is 2026-10-11, older than the" || ok=1
echo "$out" | grep -q "usermgmt/totp" && ok=1 # totp's tag is AFTER the fix -> no reminder
report $ok "F1 stranded nested module reminds, clean one stays silent (got rc=$rc)"

# F2 never-tagged nested module requiring the dep -> reminder.
mkdir -p usermgmt/webauthn
cat >usermgmt/webauthn/go.mod <<'EOF'
module github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4
go 1.27
require github.com/larsartmann/go-jose/v4 v4.1.6
EOF
git add -A
git commit -qm webauthn
out=$(cd "$tree" && REPO_ROOT_OVERRIDE="$tree" bash "$GATE" github.com/larsartmann/go-jose/v4 2026-10-28)
rc=$?
ok=0
[ "$rc" -eq 1 ] || ok=1
echo "$out" | grep -q "usermgmt/webauthn requires.*but has NEVER been tagged" || ok=1
report $ok "F2 never-tagged nested module reminds (got rc=$rc)"

# F3 clean tree: fix date before every tag -> exit 0 with the green line.
out=$(cd "$tree" && REPO_ROOT_OVERRIDE="$tree" bash "$GATE" github.com/larsartmann/go-jose/v4 2026-10-01)
rc=$?
ok=0
[ "$rc" -eq 0 ] || ok=1
echo "$out" | grep -q "every nested module requiring.*is tagged at or after" || ok=1
report $ok "F3 all tags at/after the fix date exits 0 (got rc=$rc)"

# F4 usage errors -> exit 2.
(cd "$tree" && REPO_ROOT_OVERRIDE="$tree" bash "$GATE" only-one-arg >/dev/null 2>&1); [ $? -eq 2 ]; report $? "F4 missing arg exits 2"
(cd "$tree" && REPO_ROOT_OVERRIDE="$tree" bash "$GATE" github.com/x/y not-a-date >/dev/null 2>&1); [ $? -eq 2 ]; report $? "F4 bad date exits 2"

echo "== $pass passed, $fail failed"
[ "$fail" -eq 0 ]
