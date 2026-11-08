# Owner decision sheet — one sitting, every open decision (INDEX)

**Filed:** 2026-11-08 (GET-SHIT-DONE session, plan [`M00`](../planning/2026-11-08_06-35_SUPERB-owner-unlock-v419-and-auth-adoption-pareto-plan.md) §4) · **Status: INDEX — one line + recommendation + wrong-answer cost per decision; the detail packets carry the full reasoning.** A tick is one word per row (GO / the chosen option / "adopt recommendation"). Everything this sheet unlocks is already planned and probe-verified; nothing here needs new research.

## How to answer

Reply inline next to the ID, e.g. `M2M D1: GO` or `browser-history: admin-tier`. Adopting a row's recommendation verbatim can be batched as `all recommendations GO` — every recommendation below was written to be safe to adopt (all are additive or reversible; the irreversible directions are the ones NOT recommended).

## A. Sitting packets (multi-decision, pre-staged in full)

| ID | Decision | Recommendation | Cost if wrong | Detail |
|----|----------|----------------|---------------|--------|
| M2M D1 | Bot-token M2M: supported surface or deprecate? | **Supported** | LOW if supported (additive); HIGH+irreversible-ish if wrongly deprecated (deletes a published surface) | [M2M packet](2026-11-07_m2m-owner-decisions-d1-d4.md) |
| M2M D2 | Bot authority: scopes, owner delegation, or consumer-owned? | **Bot scopes, event-derived** (replay-safe, the only shape consistent with gotcha 45 + ADR-0057) | LOW-MEDIUM (derivation is additive; an overrule adds a step, deletes nothing) | [M2M packet](2026-11-07_m2m-owner-decisions-d1-d4.md) |
| M2M D3 | Can one agent act for several humans at once? | **Single-delegator in v4** (widens to a list later without a store migration; narrowing is the expensive direction) | LOW | [M2M packet](2026-11-07_m2m-owner-decisions-d1-d4.md) |
| M2M D4 | Machine access in the audit journal, at what granularity? | **Yes — per-authentication-outcome events, fingerprint-only** (never per-request; attribution already rides the actor chain) | LOW (events additive; granularity down is a projection concern) | [M2M packet](2026-11-07_m2m-owner-decisions-d1-d4.md) |
| migration D1–D5 | usermgmt→system migration shapes | **Defaults already ADOPTED** (recorded in the migration plan) — no tick needed; W1 probes are GO | — | [Migration plan](../planning/2026-11-07_10-45_SUPERB-usermgmt-system-metaengine-migration-plan.md) |

## B. Product postures (block consumer auth waves)

| ID | Decision | Recommendation | Cost if wrong | Detail |
|----|----------|----------------|---------------|--------|
| browser-history | Domain-keyed commands (ClassifyURL/TagDomain/UntagDomain/OverrideDomain/RemoveOverride — streams keyed `classify/<domain>`): any-authenticated or admin-tier? | **Any-authenticated** — single-operator tool class (same as InboxClean's uniform shape); admin-tier would add role infrastructure a one-user product has no boundary for | LOW — a posture is a declaration + one policy row; switching later is cheap | TODO row 26 C28 pre-stage; plan M06 |
| Swetty | Repo unlocatable (no checkout, not in `gh repo list`, not archived) — does it exist under another name? | **Name the repo or strike the row** (C27 leg cannot be verified against a repo that cannot be found) | LOW (a stale row costs drift; striking costs nothing if it truly is gone) | TODO row 24 C27 verdict |
| go-datastar v0.8.0 | May a session cut the fork's four-module lockstep release (bundle-only, DataStar 1.0.4, per their checklist)? | **GO** — the 1.0.4 bundle is verified in their master (static tests green); the cut is mechanical; it unblocks the W3.2 static sweep + the v4.19 wave | LOW — a tag is retractable; the alternative stalls v4.19 on a lane with no owning session | plan M09; TODO row 34 |

## C. Small engineering calls (each closes a TODO/D-table row)

| ID | Decision | Recommendation | Cost if wrong | Detail |
|----|----------|----------------|---------------|--------|
| D3 | rawIDToken suppress confirm | **Confirm-and-close** (executed per recorded decision) | none | D-table |
| D4 | Fleet cqrs-lint pin swap | **Bump pin to the 4.20.0 tag** (steps: round14-tail §8; B024 suppressions re-add) | LOW (documented FPs re-suppressed same-change) | [Round-14 packet](2026-10-01_owner-decision-packet-round14-tail.md) §8 |
| D6 | cqrs-lint distribution tag | **Tag the nested `cmd/cqrs-lint` module** in go-cqrs-lite (unblocks the CI gate chain) | LOW (a tag; retractable) | [T23 packet](2026-10-01_t23-t18-owner-decision-packet.md) |
| D7 | feedback-inbox epoch + SEC move | **Epoch 2026-08-01, move ratified** (as pre-staged) | LOW | [T18g packet](2026-10-01_t23-t18-owner-decision-packet.md) |
| D8 | Toolchain policy (OQ26) | **KEEP the patch pin — close as decided** | LOW | D-table |
| D11 | Scorecard evidence policy | **Keep module-root min** (deterministic, canonical) | LOW | D-table |
| D13 | ParseUserID prefix-strip: generic vs whitelist | **KEEP generic strip** (fuzz pins + memo done; whitelist adds churn without a demonstrated attack) | LOW either way (pins exist) | [D13 memo](../drafts/2026-10-07-d13-parseuserid-prefix-policy-memo.md) |
| D14 | Cut `httputil/server_timing/v1.0.2` (committed LICENSE via proxy) | **Cut** — mechanical verify-tag release, no code delta | LOW | D-table |
| D15 | BuildFlow pre-commit budgets, remaining halves | **Config tuning** (gomod-freshness exclusion for `scripts/testdata/` + skip tuning) over standing `--no-verify` | LOW | D-table |
| D16 | GitHub Releases per module tag | **Per ROOT tag only, default DEFER** (28-module 1:1 releases is noise; rotate first) | LOW | D-table |
| D17 | gosec CI pin floats `@master` | **Pin to a tagged release** (plan M22 executes on tick) | LOW | D-table |
| D9 | `examples/datastar-demo` rebrand-or-remove | **KEEP as-is** (2026-09-22 survey evidence, re-confirmed) | LOW | D-table |

## D. Security / infra posture (each unlocks a planned milestone)

| ID | Decision | Recommendation | Cost if wrong | Detail |
|----|----------|----------------|---------------|--------|
| architecture-gate | Configure `SSH_PRIVATE_KEY` + drop `continue-on-error` (enforce the EventCatalog breaking-change gate), or keep the honest skip? | **Enforce** — the probe skips honestly today; the catalog gate deserves teeth. Tick = "secret added" and M22 drops the skip + adds the fixture self-test | MEDIUM if enforced without the secret (a permanently red gate wedges nothing — no branch protection — but adds noise) | TODO row 101; plan M22 |
| repo protection | Require the currently-passing checks on master? Is `DontMergeMeYet` the intended merge gate? | **Require test/lint/build/checks; DontMergeMeYet is NOT a merge gate** (scratch tool) | MEDIUM — requiring checks makes skipped jobs advisory→blocking, so CI flakes block merges until green | TODO row 103; plan M22 |
| ExtraNav (a) | Reject dangerous URL schemes (`javascript:`, `data:`, `vbscript:`) in `NavEntry.Href` at Validate? | **Adopt** — templ's attribute escaping prevents attribute breakout but NOT a working `javascript:` link on click; validation is the real boundary (the `AccentColor` precedent) | LOW (additive validation; dangerous schemes have no legitimate nav use) | TODO row 43 |
| ExtraNav (b) | `aria-current="page"` on active nav links (a11y gap; changes every sidebar golden in both UI modules)? | **Adopt** — real gap; the golden churn is one mechanical regen pass, diffed line-by-line | LOW | TODO row 43 |
| RequireSession home | Re-export a session gate in usermgmt, or document setup as the home? | **Document setup as the home** in the usermgmt README (avoids the ADR-0062 alias class; Path B consumers get a copy-paste gate snippet) | LOW | TODO row 126 |

## E. Channel / parked (owner-only actions)

| ID | Decision | Recommendation | Cost if wrong | Detail |
|----|----------|----------------|---------------|--------|
| PapDashboard reply | Send the drafted reply (OQ23 GO-conditional / OQ24 NO-GO)? | **Send as drafted** — prerequisites verified live on the proxy | LOW | TODO row 122 |
| Stripe S00 | Demand artifact + MoR decision before any billing code | **Stays PARKED** until real demand exists; tick = name the MoR + artifact | HIGH if wrongly started (31h plan on an unvalidated demand) | TODO row 109 |
| /mnt/buildcache reclaim | One-time prune at a maintenance window | **Prune at the next quiet window** (urgency DOWN at 72% free) | LOW | D-table D5; plan M25 |

_This sheet adds no new decisions — every row already lived in a TODO row, D-table row, or packet; it only indexes them. Answering all of it converts the gated milestones in the 06-35 plan (M04–M07, M09, M21, M22) to GO without further sitting._
