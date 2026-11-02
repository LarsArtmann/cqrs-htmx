#!/usr/bin/env bash
# test-check-permission-surface.sh — fixture self-test for
# check-permission-surface.sh. Offline, no network, no Go build.
#
# Cases:
#   1. a clean tree exits 0
#   2. each of the four rules is detected independently (4 planted violations)
#   3. an inline `permission-surface-allow` suppression is honoured and counted
#   4. test files are exempt by design

set -uo pipefail

script_dir="$(cd "$(dirname "$0")" && pwd)"
checker="$script_dir/../checks/check-permission-surface.sh"

if [ ! -x "$checker" ]; then
  echo "FAIL: checker not executable: $checker" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

failures=0
assert_rc() {
  local want="$1" got="$2" label="$3"
  if [ "$got" -ne "$want" ]; then
    echo "FAIL: $label: exit $got, want $want"
    failures=$((failures + 1))
  fi
}

# --- case 1: clean tree -----------------------------------------------------
clean="$tmp/clean"
mkdir -p "$clean/pkg"
cat >"$clean/pkg/ok.go" <<'GO'
package pkg

func Grant(svc *Service, roles []Role) error {
	return svc.GrantRoles(ctx, actor, tenant, roles)
}
GO

"$checker" --path "$clean" >/dev/null
assert_rc 0 $? "clean tree"

# --- case 2: one planted violation per rule --------------------------------
bad="$tmp/bad"
mkdir -p "$bad/pkg"

cat >"$bad/pkg/ephemeral.go" <<'GO'
package pkg

func Seed(a *Authz) error {
	return a.AddGroupPolicy(GroupPolicy{Subject: "u", Role: RoleAdmin, Domain: "*"})
}
GO

cat >"$bad/pkg/untyped.go" <<'GO'
package pkg

func Probe(e Enforcer) (bool, error) {
	return e.EnforceAny("u", "*", "orders", "read")
}
GO

cat >"$bad/pkg/raw.go" <<'GO'
package pkg

func Ask(e Enforcer) (bool, error) {
	return e.Enforce("u", "*", "orders", "read")
}
GO

cat >"$bad/pkg/sentinel.go" <<'GO'
package pkg

var errPanel = errorfamily.NewRejection("adminui.forbidden", "access denied")
GO

out="$("$checker" --path "$bad" 2>&1)"
rc=$?
assert_rc 1 "$rc" "four planted violations"

for rule in ephemeral-grant untyped-enforce raw-enforce split-sentinel; do
  if ! printf '%s' "$out" | grep -q "$rule"; then
    echo "FAIL: rule $rule was not reported"
    failures=$((failures + 1))
  fi
done

# Four findings, one per planted file.
count="$(printf '%s\n' "$out" | grep -c '^  ')"

if [ "$count" -ne 4 ]; then
  echo "FAIL: expected 4 findings, got $count"
  failures=$((failures + 1))
fi

# --- case 3: inline suppression -------------------------------------------
supp="$tmp/supp"
mkdir -p "$supp/pkg"

cat >"$supp/pkg/grant.go" <<'GO'
package pkg

func Seed(a *Authz) error {
	// permission-surface-allow: throwaway enforcer in a bootstrap fixture.
	return a.AddGroupPolicy(GroupPolicy{Subject: "u", Role: RoleAdmin, Domain: "*"})
}
GO

out="$("$checker" --path "$supp" 2>&1)"
assert_rc 0 $? "suppressed violation"

if ! printf '%s' "$out" | grep -q 'suppressed: 1'; then
  echo "FAIL: suppression not counted: $out"
  failures=$((failures + 1))
fi

# --- case 4: test files are exempt ---------------------------------------
exempt="$tmp/exempt"
mkdir -p "$exempt/pkg"

cat >"$exempt/pkg/grant_test.go" <<'GO'
package pkg

func TestSeed(t *testing.T) {
	_ = a.AddGroupPolicy(GroupPolicy{Subject: "u", Role: RoleAdmin, Domain: "*"})
}
GO

"$checker" --path "$exempt" >/dev/null
assert_rc 0 $? "test files exempt"

# --- case 5: identity-model is the definition site -----------------------
own="$tmp/own"
mkdir -p "$own/identity-model"
cat >"$own/identity-model/authz.go" <<'GO'
package identitymodel

func (a *Authz) AddGroupPolicy(g GroupPolicy) error {
	return a.ApplyGroupPolicy(g)
}
GO

"$checker" --path "$own" >/dev/null
assert_rc 0 $? "identity-model definition site"

if [ "$failures" -ne 0 ]; then
  echo "check-permission-surface self-test: $failures failure(s)"
  exit 1
fi

echo "check-permission-surface self-test: 5/5 cases pass"
