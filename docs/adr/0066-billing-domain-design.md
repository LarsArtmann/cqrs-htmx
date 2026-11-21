# ADR 0066: Billing Domain Design

**Status:** Accepted
**Date:** 2026-11-21
**Driver:** M02 of the [v3 billing execution plan](../planning/2026-11-20_20-25_SUPERB-stripe-billing-v3-execution-plan.md) — the design record required before the `billing-model` scaffold (M03).
**Related:** [ADR 0065](0065-billing-processor-and-merchant-of-record.md) (S00: processor + merchant of record), [M01 decision record](../decisions/2026-11-21_02-00_m01-first-consumer-and-plan-catalog.md) (named consumers + plan catalog), [v1 billing plan](../planning/2026-10-24_17-32_SUPERB-stripe-billing-bounded-context-pareto-plan.md) §2 (D1–D13 decision evidence), [v2 billing plan](../planning/2026-10-24_18-15_SUPERB-stripe-billing-v2-self-review-hardened-pareto-plan.md) §2 (processor-agnostic vocabulary, bounded seen-refs, bidirectional choreography, metrics doctrine).

## Context

The billing lane's gates are closed: S00 by ADR 0065 (Stripe as processor, owner holds tax
liability), K1 by the M01 record (Issue Shield with live production checkout; ssh-key-monitor
named as second consumer). Before any code exists, the domain design must be pinned in one
place so M03–M08 (types, decider, fold, scenario suite) implement a recorded contract rather
than a conversation.

This ADR does not restate the inherited decisions — it adopts them by reference and records
the deltas the named consumers force.

## Decision

### Inherited by reference

- **D1–D13** from v1 §2: external-actor doctrine, fold-is-contract idempotency,
  unknown-reference rejection, no synchronous gateway calls on read or identity paths, time
  ownership by the decider/fold inputs, choreography over orchestration, journal-not-archive,
  tenant-as-billing-target, one-way module boundary (`billing-model` imports identity-model
  only), composition-root wrap.
- **v2 §2.1–2.3**: processor-agnostic vocabulary (`ProcessorRef`, `ProcessorGateway`; the
  words stripe/`evt_`/`cus_`/`sub_` exist only in `billing/stripe`, enforced by the S18/M23
  gate), bounded seen-refs (current + previous billing period, compaction as fold invariant),
  bidirectional choreography (billing→identity suspend/reactivate/roles;
  identity→billing `UpdateSeats`/`CancelSubscription`).
- **v2 §2.5 metrics doctrine**: money events carry settled facts; CLV/MRR/churn are disposable
  projections, never aggregate state; FX rates never enter the journal.

### New deltas (this ADR)

| # | Delta | Decision |
| --- | --- | --- |
| D14 | **PlanKey / PriceRef split** | The domain speaks `PlanKey` (our catalog key, e.g. `monthly_protection`) and an opaque branded `PriceRef`. Literal processor price IDs (`price_…`) live only in the `billing/stripe` adapter's configuration, which maps PlanKey → PriceRef at the consumer's composition root. Breaking this reintroduces the portability hole v2 fixed. |
| D15 | **Entitlement → roles mapping** | The `PlanCatalog` carries a plan→roles table whose values are consumer vocabulary (strings the consumer defines), never library-hardcoded role names. Grants apply through the durable path — `Service.GrantRoles` / `RevokeRoles` (gotcha 45: enforcer-resident grants vanish on restart and the customer loses what they paid for). Entitlement status (`active`/`grace`/`suspended`/`canceled`) mirrors the subscription state the fold derives. |
| D16 | **Tax facts, not tax logic** | Money events (`PaymentReceived`, `Renewed`) carry `amount_minor` + ISO 4217 currency as settled processor facts (v2 §2.5). Tax determination, registration, and invoicing stay owner operations per ADR 0065 — the domain computes no tax, stores no rates, and records no tax amounts beyond what the processor reports. |
| D17 | **Chain of custody** | Every settled fact is traceable end-to-end: processor event → webhook ACL (signature-verified translation) → command (idempotent, dedup-keyed on the processor event ID) → domain event (carrying `ProcessorRef`) → projection. The bounded seen-refs window gives exactly-once application; the journal is the audit trail. No money fact exists without a `ProcessorRef` provenance. |
| D18 | **Named limit set** | The catalog's limits are a named set (`seats`, `protected_targets`, later `monitored_hosts`), not a single seat integer. Grounding: the first consumer's real axis is protected GitHub targets (M01 record); `seats` stays a conventional member so the identity choreography vocabulary (`UpdateSeats`, seat clamping) survives unchanged. Seat-clamp logic reads the plan's `seats` member; unknown limit names are rejected by the catalog validator, not silently ignored. |
| D19 | **One-time purchases are plans with renewal=none** | A one-time purchase (Issue Shield's `one_time_export`) is a plan whose renewal shape is `none`: the aggregate reaches its terminal entitlement state without a renewal path. Not a second aggregate, not a special command path. |

### Kill-criteria assessment

K4 ("the plan catalog cannot express the first consumer's pricing") was assessed against the
verified catalog in the M01 record: fixed-price, single-currency, one-time variant — K4 does
not fire. K1 is cleared by evidence; K2 and K3 remain live during M06+/M10+ respectively.

## Consequences

- M03–M08 may start: the type set (`ProcessorRef`, `PlanKey`/`PriceRef`, named limits, 9
  events, 7 commands, rejections) now has a recorded contract.
- The S18/M23 vocabulary gate keeps its scope; D14 gives it teeth (a price literal in
  `billing-model` is a gate red, not a style nit).
- D15 makes entitlement grants replayable: roles re-derive from events through the durable
  grant path, so a rebuilt identity projection restores paid access with no migration step.
- D18 is the one deliberate divergence from the v3 plan's "every plan has a seat limit"
  verify wording — corrected in the M01 record and here; the M05 test asserts named limits
  per plan instead.
- Implementation status: not started. Next milestone: M03 (`billing-model` scaffold).
