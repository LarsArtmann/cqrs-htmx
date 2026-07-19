# Module Modularization Assessment — cqrs-htmx

**Date:** 2026-07-19 | **Verdict:** Module boundaries are correct; internal package decomposition is the next lever.

## Current state (12 modules)

| Module                                                   | Purpose                                         | Files                               | Verdict                                                                                                                                                             |
| -------------------------------------------------------- | ----------------------------------------------- | ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| root (`cqrs-htmx/v4`)                                    | HTTP/CQRS/HTMX glue                             | 138 .go                             | Right-sized. Building blocks (SSE/WS/ratelimit) are candidates for internal/ packages but not separate go.mods.                                                     |
| `usermgmt/v4`                                            | Event-sourced user/membership/tenant/bot domain | 185 .go (55 `es_*`, 21 `service_*`) | **Right go.mod boundary, wrong package boundary.** Should stay one module but split into `internal/user`, `internal/membership`, `internal/tenant`, `internal/bot`. |
| `usermgmt/{totp,webauthn,oauth2}/v4`                     | Auth strategies                                 | 1-3 .go each                        | Excellent. Each strategy is independent, structural-typed, independently versioned. **Do not merge.**                                                               |
| `adminui/v4`                                             | Ready-made admin dashboard                      | ~25 .go                             | Right-sized consumer module.                                                                                                                                        |
| `loginpage/v4`                                           | Passwordless login page                         | ~10 .go                             | Right-sized consumer module.                                                                                                                                        |
| `integration_test`                                       | Cross-module bridge tests                       | ~13 .go                             | Correct as a separate module — verifies interfaces compose.                                                                                                         |
| `examples/{basic,admin-demo,catalog-demo,datastar-demo}` | Demos                                           | 1-5 .go each                        | Each is its own go.mod — appropriate for independently runnable demos.                                                                                              |

## Direction-neutral analysis

**Question:** Split further, merge, or accept?

**Answer:** Accept the current go.mod layout. The boundaries pass the Unix-philosophy litmus test:

1. **Each module does one thing well** ✓ — auth strategy vs UI panel vs HTTP glue are clearly distinct concerns.
2. **Composition via interfaces** ✓ — `Enforcer`, `TemplComponent`, `TOTPProvider`, `WebAuthnProvider`, `OAuth2Provider` are the "pipes."
3. **Mechanism, not policy** ✓ — usermgmt core has no WebAuthn/TOTP/OAuth2 imports. Strategies plug in.
4. **Opaque internals** ~ — usermgmt leaks internals via flat layout (see below).

## The one real problem: usermgmt internal decomposition

`usermgmt/` has 185 .go files in a single package. The `es_` prefix sprawl reveals sub-domains hiding inside one namespace:

```
es_bot_*         (7 files)    → internal/bot
es_membership_*  (8 files)    → internal/membership
es_tenant_*      (7 files)    → internal/tenant
es_decide_*      (7 files)    → internal/user/decide (or internal/decision)
es_scenario_*    (4 files)    → internal/testing (or _test package)
es_state.go etc               → internal/user
service_*         (21 files)  → stays at top level (public Service API)
```

**Why split internal packages (not modules):**

- No go.mod overhead
- Internal packages can be tested independently
- Public `Service` type stays as the composition facade
- Compile-time enforcement of sub-domain boundaries (e.g. `internal/bot` can't import `internal/tenant`)
- Faster incremental compiles

**Why NOT split into separate go.mods:**

- The 4 aggregates (User/Membership/Tenant/Bot) share event store, projection host, and read-model infrastructure. Splitting forces those into yet another module, multiplying go.mod count for no consumer benefit (consumer imports `usermgmt` once).
- Same go.mod = same dep tree = zero consumer benefit (documented in TODO_LIST.md, deferred for this reason).

## Recommendations (Pareto order)

1. **[Low effort, High value]** Split usermgmt into `internal/user`, `internal/membership`, `internal/tenant`, `internal/bot`. Public API unchanged. ~1-2 days.
2. **[No effort]** Keep auth strategies as independent modules. Already correct.
3. **[No effort]** Keep integration_test separate. Already correct.
4. **[Blocked upstream]** Drop `go.work` replace directives once go-cqrs-lite publishes clean v4.0.3+.
5. **[Decision needed]** Decide what to do with `examples/datastar-demo` — rebrand or remove (see architecture review).

## Module isolation verification

CI script `scripts/check-module-isolation.sh` confirms all 8 main modules build standalone under `GOWORK=off`. Dep budgets (`scripts/check-dep-budgets.sh`) and replace-directive checks (`scripts/check-replace-directives.sh`) are enforced. **This is a well-disciplined multi-module workspace.**
