#!/usr/bin/env bash
# check-dep-budgets.sh — Enforce per-module production dependency limits
# Adapted from go-cqrs-lite's CI-enforced dependency budget model.
#
# Prevents god-modules from accumulating unbounded dependencies.
# Usage: ./scripts/check-dep-budgets.sh
# Exit: 0 = all modules within budget, 1 = at least one module exceeds budget

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 1

# Dependency budgets per module.
# Key = module directory, Value = max direct production deps (excluding test-only).
# These are CURRENT counts + 20% headroom. Adjust when intentionally adding deps.
declare -A DEP_BUDGET=(
  ["."]=19                # Root: 17 current (casbin, form, nosurf, branded-id, cqrs-lite x6, httputil, ulid, ginkgo, gomega, x/time + go-codec [direct since the 2026-09-20 DecodePayload re-exports in payload.go])
  ["identity-model"]=10   # identity-model: 8 current (casbin, branded-id, cqrs-lite event/id/codec/metadata, ulid, rapid)
  ["usermgmt"]=28         # usermgmt: 25 current (casbin, cqrs-htmx, branded-id, cqrs-lite x8, sqlite, ulid, rapid, watermill, kv)
  ["usermgmt/totp"]=3     # totp: 1 current (pquerna/otp)
  ["usermgmt/webauthn"]=3 # webauthn: 1 current (go-webauthn/webauthn)
  ["usermgmt/oauth2"]=5   # oauth2: 3 current (oauth2, oidc, go-jose)
  ["adminui"]=12          # adminui: 10 current (cqrs-htmx, usermgmt, identity-model, templ, httputil, error-family + templ-components family x4 [root/htmx/icons/utils — one logical dep split into 4 Go modules since v1.8.2])
  ["loginpage"]=5         # loginpage: 3 current (cqrs-htmx, usermgmt, templ)
  ["dashboardui"]=23      # dashboardui: 21 current (cqrs-htmx, templ, go-humanize, go-codec, cqrs-lite x9 [command/event/eventtest/id/listing/projectionhost/query/snapshot/storage-memory], error-family, go-sse, httputil, templ-components family x5 [root/errorpage/htmx/icons/utils — the 2026-09-17 adoption program, docs/planning/2026-09-17_13-22_*]; +2 slack while the 12-step ladder executes)
  ["datastar"]=6          # datastar: 6 current (go-datastar, go-datastar/broadcast [deprecated alias facade: type Broadcaster = broadcast.Broadcaster + constructor shims, 2026-09-17 upstream move; drops back to 5 at v5 facade removal], go-sse, cqrs-lite event/id, testify)
  ["setup"]=23            # setup: 23 current (cqrs-htmx family x7 [root/usermgmt/identity-model/adminui/dashboardui/loginpage/datastar], cqrs-lite event/id/projectionhost/storage-memory/watermill/eventtest/storage + command/v4 [direct since ServiceConfig.CommandMiddleware threading, 2026-09-18 — consumer seam for dispatch middleware], go-sse [direct since /sse replay], go-appkit [RunWithAppkit], error-family, httputil, modernc.org/sqlite [ADR-0050 wave]; 2026-09-14 OTel sprint added middleware/v4 [production: Config.Observability accepts *middleware.OTelBundle] + otel/v4 [test-only: tracer for wiring tests]; 2026-09-17 +go-datastar/broadcast [setup.Mount serves the datastar script/feed, concurrent session's ADR-002 extraction, temp sibling replace documented in go.mod])
  ["systemadapter"]=16    # systemadapter: 13 current (usermgmt, identity-model, cqrs-lite event/id/record, system, metaengine, projectionadapter, sqliteengine, projection, projectionhost, storage/memory, error-family)
  ["health"]=6            # health: 5 current (cqrs-htmx root, go-health, go-health-dashboard, go-error-family, samber/do)
  ["auditlog"]=5          # auditlog: 4 current (go-error-family, samber-do-auditlog, samber/do, testify)
)

failed=0

echo "=== Dependency Budget Check ==="
echo ""

for mod in "${!DEP_BUDGET[@]}"; do
  mod_path="$REPO_ROOT/$mod"
  if [[ ! -f "$mod_path/go.mod" ]]; then
    continue
  fi

  budget=${DEP_BUDGET[$mod]}
  module_name=$(grep -m1 '^module ' "$mod_path/go.mod" | awk '{print $2}')

  # Count direct require entries (exclude replace and retract blocks)
  # Also exclude indirect deps (marked with // indirect)
  # And standalone comment lines (e.g. //cqrs-lint:ignore(...) suppressions) — they are not deps
  # Handles both require ( ... ) blocks and single-line require statements
  dep_count=$(
    cd "$mod_path" || exit 1
    awk '
        /^require \(/ { in_req=1; next }
        /^\)/ { in_req=0 }
        in_req && /^\t/ && !/^[[:space:]]*\/\// && !/\/\/ indirect/ { count++ }
        /^require [^(]/ && !/^[[:space:]]*\/\// && !/\/\/ indirect/ { count++ }
        END { print count+0 }
    ' go.mod
  )

  echo -n "  $module_name: $dep_count deps (budget: $budget) ... "

  if [[ $dep_count -gt $budget ]]; then
    echo "OVER BUDGET"
    echo "    Reduce deps or justify increase in scripts/check-dep-budgets.sh"
    failed=1
  else
    remaining=$((budget - dep_count))
    echo "OK ($remaining slots remaining)"
  fi
done

echo ""
if [[ $failed -eq 0 ]]; then
  echo "✓ All modules within dependency budget"
else
  echo "✗ Dependency budget exceeded"
  exit 1
fi
