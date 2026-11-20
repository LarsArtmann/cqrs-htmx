# ADR 0065: Billing Processor and Merchant of Record

**Status:** Accepted
**Date:** 2026-11-20
**Driver (owner):** "For now I own tax" — the answer to the S00b question of the billing plan.
**Related:** [v2 billing plan](../planning/2026-10-24_18-15_SUPERB-stripe-billing-v2-self-review-hardened-pareto-plan.md) (plan of record, S00), [v1 billing plan](../planning/2026-10-24_17-32_SUPERB-stripe-billing-bounded-context-pareto-plan.md) (decision evidence), [S18 portability gate](../planning/2026-10-24_18-15_SUPERB-stripe-billing-v2-self-review-hardened-pareto-plan.md) (vocabulary ban), TODO_LIST row "Stripe billing bounded context", [owner decision sheet](../proposals/2026-11-08_owner-decision-sheet.md).

## Context

The billing lane has been parked since 2026-10-24 because its S00 gate demanded two
artifacts before any code: a consumer-demand artifact (or an explicit park decision) and a
merchant-of-record decision. The 2026-10-25 verdict recorded both as missing.

The merchant-of-record question is not a vendor preference. It decides who is legally the
seller, and therefore who carries VAT and sales-tax registration, invoice requirements,
dunning, chargebacks, and refund liability. Two shapes exist:

| Shape | Who sells | Tax liability | Code consequence |
| --- | --- | --- | --- |
| Processor (Stripe Checkout / Billing) | The owner | The owner | Dunning and suspend choreography stay ours to model; Stripe only reports what happened |
| MoR provider (Paddle-class) | The provider | The provider | Dunning and reconciliation re-scope (plan kill criterion K3), adapter set shrinks |

Demand research performed for this ADR (2026-11-20): zero admissible third-party artifacts.
An all-state issue search for billing, stripe, subscription, and payment terms returned two
unrelated issues (a legal-party type request and a closed SSE filter request); the repository
tree contains no billing package, and every billing hit under docs is a plan, a proposal, or
a TODO row. No fleet product has asked for billing in writing.

## Decision

1. **Stripe is the processor; the owner is the merchant of record and holds tax liability**
   ("for now" — explicitly revisitable). Stripe supplies payment capture, subscription
   billing, invoices, and dunning mechanics; it does not become the seller.

2. **The S00a demand gate is recorded as an owner override, not as demand evidence.** The
   admissible class (issue, external ask, concrete consumer repo need) remains empty. The lane
   proceeds because the Product Owner directed it, and this ADR says so out loud instead of
   relabelling the directive as a consumer artifact. Kill criterion K1 (no consumer artifact
   within two planning cycles) stays a standing risk, and the owner may re-park at any time.

3. **K3 does not fire.** Dunning-to-suspension (`PaymentFailed` × grace → `SuspendTenant`
   with the `billing` reason) and reconciliation stay in the plan's scope, unchanged.

4. **The domain stays processor-agnostic.** Plan §2.1 is untouched: `billing-model` speaks
   `ProcessorRef` and `ProcessorGateway`, the words Stripe, `evt_`, `cus_`, and `sub_` exist
   only inside the `billing/stripe` adapter, and the S18 boundary gate enforces it. A later
   switch to a MoR provider is an adapter swap plus the K3 re-scope — not an aggregate rewrite.

5. **Tax is owner operations, not library code.** The library ships no tax engine and no rate
   table: registration thresholds, filing cadence, reverse-charge handling for business
   customers, and tax-inclusive versus tax-exclusive pricing are product and accounting
   decisions. A consumer may enable Stripe Tax behind the adapter, which changes no domain type.

## Consequences

- **PCI scope stays minimal by construction:** card data never touches this library or a
  consumer's server — hosted Checkout or Stripe Elements only. A raw primary account number in
  a request body is a defect, not a configuration.
- **Webhook verification is mandatory, not optional:** as the merchant we are liable for forged
  or replayed events, so signature verification plus the plan's raw-body ordering guard apply
  to every processor callback. The existing fold-level dedup (bounded seen-refs) covers replay
  after verification, not instead of it.
- **Dunning is modelled, not delegated:** grace periods, retry cadence, and the moment a tenant
  is suspended remain decider state, because the library's job is to make billing state
  observable in the same event stream as identity.
- **Refunds, chargebacks, and disputes** arrive as processor facts and become domain events;
  the policy for what a refund does to entitlements is a product decision this ADR does not make.
- **Not decided here:** pricing model, plan tiers, currencies, trial length, refund policy, and
  the S25 metrics formulas. Those are product decisions with no bearing on the domain shape.
- **Unblocked next milestone:** S01 (the billing ADR, currently reserved as ADR 0057 in the plan
  and needing renumbering because that number is taken by the authorization contract) is now
  the first executable step. Nothing is built yet; this ADR closes the gate, it does not start
  the implementation.

## Status of the S00 gate

| Part | Verdict 2026-11-20 |
| --- | --- |
| S00a demand | No admissible artifact; owner override recorded (explicit, dated, reversible) |
| S00b merchant of record | Decided: Stripe as processor, owner carries tax liability |
| Implementation | Not started — S01 through S25 remain open at the owner's discretion |
