# M01: First Consumers and Plan Catalog (billing lane)

**Status:** Accepted
**Date:** 2026-11-21
**Closes:** M01 of the [v3 billing execution plan](../planning/2026-11-20_20-25_SUPERB-stripe-billing-v3-execution-plan.md) and its kill criterion K1 ("No named paying consumer by the end of M01 → do not start M02").
**Related:** [ADR 0065](../adr/0065-billing-processor-and-merchant-of-record.md) (S00: Stripe as processor, owner holds tax), [v2 billing plan](../planning/2026-10-24_18-15_SUPERB-stripe-billing-v2-self-review-hardened-pareto-plan.md) (design authority), TODO_LIST row "Stripe billing bounded context".

## Context

ADR 0065 closed the S00 gate on 2026-11-20 but had to record the demand side as an explicit
owner override: no admissible consumer artifact existed, and kill criterion K1 stayed live.
On 2026-11-21 the owner named the paying consumers directly, which is the evidence class the
override explicitly awaited.

## Named consumers

| Consumer | Repo | Evidence | Standing |
| --- | --- | --- | --- |
| **Issue Shield** (GitHub issues backup & export, issue-shield.com) | `~/projects/sales-landing-page` | **Stripe Checkout is live in production since 2026-09-19**: `data/pricing.ts` carries three live USD price IDs, the Prisma `Subscription` model persists `stripeSubscriptionId`/`stripeCustomerId`, webhooks (5 event types incl. refunds) are signature-verified, and the dashboard is gated by checkout session ID | **Primary consumer.** Satisfies K1 and the K4 expressibility check on its own |
| **ssh-key-monitor** (SSH key change-detection, Rust) | `~/projects/ssh-key-monitor` | Named by the owner 2026-11-21 as the second paying consumer; productization path (hosted monitoring tier) not yet scoped | Recorded consumer. Plan-catalog rows land when its hosted tier is scoped; nothing in the design depends on them yet |

Issue Shield's live checkout is precisely the demand artifact ADR 0065 lacked: a product that
already sells subscriptions and one-time exports through Stripe, today wired by hand in its
landing-page server. The billing lane's purpose for this consumer is to replace that
hand-rolled path with the event-sourced domain (entitlement read model, dunning choreography,
reconciliation) behind the same Stripe account.

## Plan catalog (the three live plans, verified in `data/pricing.ts`)

| PlanKey (domain) | Live Stripe PriceRef | Currency | Amount | Renewal | Entitlement |
| --- | --- | --- | --- | --- | --- |
| `annual_protection` | `price_1UHL6YI1VKvghLb2Lo49swDt` | USD | 49.50 / year | recurring, annual | backup + export |
| `monthly_protection` | `price_1UHL6jI1VKvghLb2MCKaqSuZ` | USD | 7.36 / month | recurring, monthly | backup + export |
| `one_time_export` | `price_1UHL6kI1VKvghLb2Lszt21YA` | USD | 18.66 once | none (one-time) | export only |

The Stripe price IDs are quoted here as evidence only. They are public checkout identifiers,
not secrets — but per the placement rule below they never appear in domain code or events.

**K4 verdict (expressibility):** the first consumer's pricing is fixed-price, single-currency,
with a one-time variant — fully expressible by the planned `PlanCatalog`. K4 does not fire.
The one-time purchase is modeled as a plan whose renewal shape is `none`: the aggregate
reaches its terminal entitlement state without a renewal path; it is not a second aggregate.
Final type shape is pinned in [ADR 0066](../adr/0066-billing-domain-design.md).

## Limits are named, not a single seat count

Grounding on the real consumer exposed a design correction: Issue Shield's actual limit axis
is **protected targets** (GitHub repos/orgs per subscription), not operator seats. A catalog
whose only limit field is `seats` cannot express its plans honestly. ADR 0066 therefore
specifies a **named limit set** (`seats`, `protected_targets`, later `monitored_hosts` for
ssh-key-monitor) with `seats` kept as a conventional member so the identity choreography
vocabulary (`UpdateSeats`) survives unchanged.

## Plan → role mapping (consumer-owned)

The domain grants **entitlements**; it never hardcodes consumer role names. The catalog
carries a plan→roles table whose values are consumer vocabulary, applied at the consumer's
composition root through the durable path (`Service.GrantRoles` / `RevokeRoles` — never
`Authz.AddGroupPolicy`, gotcha 45). For Issue Shield the initial mapping is:
`annual_protection` / `monthly_protection` → the app's paid-user capability set (backup +
export), `one_time_export` → export-only capability. This mapping replaces the current
`/dashboard?session_id=…` checkout-session gate, which is the weakest link in the live flow.

## Price-ID placement rule

Domain vocabulary is `PlanKey` + opaque `PriceRef`; the literal `price_…` strings live only
in the `billing/stripe` adapter's configuration (consumer deployment config), never in
`billing-model` types, events, or any other package. This is guard 1 of the v3 plan and
keeps the S18 vocabulary-ban gate meaningful.

## Consequences

- K1 is cleared with evidence (this file); M02 (ADR 0066) is unblocked and executed same day.
- ssh-key-monitor's catalog rows are deferred until its hosted tier is scoped — recorded, not forgotten.
- M05's "every plan has a seat limit and a role set" verify now reads "every plan has its named limits and a role set" per the named-limits correction.
