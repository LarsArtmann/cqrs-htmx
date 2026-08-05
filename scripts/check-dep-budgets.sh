#!/usr/bin/env bash
# check-dep-budgets.sh — Enforce per-module production dependency limits
# Adapted from go-cqrs-lite's CI-enforced dependency budget model.
#
# Prevents god-modules from accumulating unbounded dependencies.
# Usage: ./scripts/check-dep-budgets.sh
# Exit: 0 = all modules within budget, 1 = at least one module exceeds budget

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# Dependency budgets per module.
# Key = module directory, Value = max direct production deps (excluding test-only).
# These are CURRENT counts + 20% headroom. Adjust when intentionally adding deps.
declare -A DEP_BUDGET=(
	["."]=18                # Root: 16 current (casbin, form, nosurf, branded-id, cqrs-lite x6, httputil, ulid, ginkgo, gomega, x/time)
	["identity-model"]=10   # identity-model: 8 current (casbin, branded-id, cqrs-lite event/id/codec/metadata, ulid, rapid)
	["usermgmt"]=28         # usermgmt: 25 current (casbin, cqrs-htmx, branded-id, cqrs-lite x8, sqlite, ulid, rapid, watermill, kv)
	["usermgmt/totp"]=3     # totp: 1 current (pquerna/otp)
	["usermgmt/webauthn"]=3 # webauthn: 1 current (go-webauthn/webauthn)
	["usermgmt/oauth2"]=5   # oauth2: 3 current (oauth2, oidc, go-jose)
	["adminui"]=7           # adminui: 5 current (cqrs-htmx, usermgmt, templ, ulid, casbin)
	["loginpage"]=5         # loginpage: 3 current (cqrs-htmx, usermgmt, templ)
	["dashboardui"]=16      # dashboardui: 13 current (cqrs-htmx, usermgmt, templ, casbin, branded-id, cqrs-lite event/id/codec/metadata, ulid, rapid, watermill)
	["datastar"]=5          # datastar: 4 current (datastar-go SDK, cqrs-lite event/id, templ)
)

# Modules that don't need budget checks
SKIP_MODULES=("integration_test" "examples/basic" "examples/datastar-demo" "examples/catalog-demo" "examples/admin-demo")

failed=0

echo "=== Dependency Budget Check ==="
echo ""

for mod in "${!DEP_BUDGET[@]}"; do
	mod_path="$REPO_ROOT/$mod"
	if [[ ! -f "$mod_path/go.mod" ]]; then
		continue
	fi

	budget=${DEP_BUDGET[$mod]}
	module_name=$(head -1 "$mod_path/go.mod" | awk '{print $2}')

	# Count direct require entries (exclude replace and retract blocks)
	# Also exclude indirect deps (marked with // indirect)
	# Handles both require ( ... ) blocks and single-line require statements
	dep_count=$(cd "$mod_path" && awk '
        /^require \(/ { in_req=1; next }
        /^\)/ { in_req=0 }
        in_req && /^\t/ && !/\/\/ indirect/ { count++ }
        /^require [^(]/ && !/\/\/ indirect/ { count++ }
        END { print count+0 }
    ' go.mod)

	echo -n "  $module_name: $dep_count deps (budget: $budget) ... "

	if [[ "$dep_count" -gt "$budget" ]]; then
		echo "OVER BUDGET"
		echo "    Reduce deps or justify increase in scripts/check-dep-budgets.sh"
		failed=1
	else
		remaining=$((budget - dep_count))
		echo "OK ($remaining slots remaining)"
	fi
done

echo ""
if [[ "$failed" -eq 0 ]]; then
	echo "✓ All modules within dependency budget"
else
	echo "✗ Dependency budget exceeded"
	exit 1
fi
