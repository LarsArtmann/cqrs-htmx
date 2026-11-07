# Status — Second pass: the M2M findings report, and a design answer I published before proving it reachable

> **ANNOTATED 2026-11-07** — ~~this report~~ **SUPERSEDED** by [`2026-11-06_21-28_m2m-w0-execution-truth-attribution-landed_status.md`](../2026-11-06_21-28_m2m-w0-execution-truth-attribution-landed_status.md), which the SUPERB design answer was adopted and executed by the M2M W0 report; design decisions recorded in the decision batch. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-07 (round-23 htmx-session (2026-11-07)).**

**Date:** 2026-11-06, 11:05–11:50 CET
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`, shared working tree with sibling sessions committing alongside this one (the daemon auto-committed this session's report as `be5dbb6f`; `d66af0c5` and `048f041b` after it are not mine).
**Scope of this session:** two deliverables on top of a single question. First, an audit report for the answer I gave about usermgmt's machine-to-machine surface (`docs/status/2026-11-06_11-13_usermgmt-m2m-ai-agent-answer-audit-and-two-policy-identity-traps_status.md`). Second, a design answer to "what would make project-to-project, data-mesh, AI-agent, and M2M auth superb and default-secure", followed by a Pareto execution plan (`docs/planning/2026-11-06_11-49_SUPERB-m2m-agent-and-data-mesh-auth-pareto-plan.md`).
**One-sentence verdict:** the findings are real and each is reproducible with one command, but the process defects of the first pass repeated in the second: I recommended a policy mechanism before proving a policy row for a machine principal was creatable at all, I pinned a corpus baseline from a measurement taken while sibling sessions were writing to the same directory, and I verified the ratchets with hand-run scripts instead of the invocations that CI and check-modules actually run.

---

## Session shape

| Step | What | Output |
| --- | --- | --- |
| 1 | Audited the previous answer against the code | 7 findings, each with file:line and a repro command |
| 2 | Wrote the first report | `docs/status/2026-11-06_11-13_…status.md` |
| 3 | Honoured the docs ratchets | tail 16 to 17, corpus 52003 to 52272, both re-run green |
| 4 | Checked the report against the docs-gates format rule | scripted backtick-pairing check, 0 violations |
| 5 | Answered the design question | 7 sections: principal model, credential lifecycle, one principal abstraction, data-mesh mapping, observability, wave plan, non-goals |
| 6 | This report and the plan | planning document with Pareto layers, scored task tables, Mermaid graph |

No production code was written, no test was run, no linter was run, no ADR was written, no issue was filed. The only executable artifacts are two budget scripts and one bespoke format checker.

---

## a) FULLY DONE

### a1. Seven findings, each reproducible in one command — tracked in the TODO_LIST.md M2M rows since 2026-11-06 (W0 fixed findings 3 and 6; the remaining five ride plan W1/W2)

All were verified by reading, and each carries a command in the report's verification table.

| # | Finding | Evidence |
| --- | --- | --- |
| 1 | Casbin bot-delete cleanup is wrong twice: subject is the raw stream ID while the dispatcher checks a kind-prefixed one, and `RemoveAllRolesForUser` removes grouping rows only, so direct policy rows survive | `usermgmt/es_casbin_projection.go:61`, `:119-128`; `authz_dispatch.go:612`; `id/v4/actor_id.go:156-161` |
| 2 | No durable grant path for a bot: `BotRegistered` is not projected, `RolesForActor` returns nil for machine actors, `GrantRoles` is membership-shaped, leaving only the documented ephemeral `AddPolicy` trap | `usermgmt/es_casbin_projection.go:48-56`; `usermgmt/service_grants.go:25`; `identity-model/authz_engine_test.go:467-480` |
| 3 | A doc comment advertises `Service.SetTokenPepper`, which no code implements | `usermgmt/service_bot.go:26-29`, one occurrence repo-wide |
| 4 | A bot cannot subscribe to any realtime feed: both SSE handlers are session-gated | `setup/sse.go:97`, `:109` |
| 5 | Machine access is absent from the audit journal: no bot auth event exists at all | repo-wide search for token/bot auth event names returns nothing |
| 6 | `RequireBot` rejects with plain text while every other rejection speaks RFC 9457 with a stable code | `usermgmt/api_token_middleware.go:66-70` |
| 7 | Token authentication survives a restart only because hydration rebuilds the token-hash index | `usermgmt/sql_hydrate.go:280-311` |

### a2. The documentation ratchets were honoured, not bypassed

The report grew the live status tail and the docs corpus, so both baselines were bumped in the same change, following the rule that deliberate growth travels with the artifact causing it. Verified with the ratchet modes:

```
tail-budget: 17 live report(s) in docs/status/
✓ tail-budget: within the ratchet (target budget still 3)
corpus-budget: 52272 line(s) across 347 markdown file(s) under docs/
✓ corpus-budget: within the ratchet (baseline 52272)
```

### a3. The format rule was checked mechanically, not by eye

The docs-gates annotate rule about characters touching a backtick was verified with a script that pairs backticks left to right per line and flags a closing backtick followed by an alphanumeric or an opening one preceded by one. Result: 0 violations. This is the only place in either pass where I mechanised a check instead of asserting compliance.

### a4. The design answer is decision-shaped, not idea-shaped

It produced, in order: the architectural hole with a four-row evidence table; the credential lifecycle with the cheap half first; the single-principal abstraction with the seven default-secure properties; a data-mesh mapping table showing that mesh is convention plus catalog rather than new machinery; the observability question stated as a query that must be answerable; a four-wave table with the reason each wave comes when it does; and an explicit non-goals section so the plan does not grow an agent framework inside a library.

### a5. Nothing was verschlimmbessert

Zero code changes means zero regression surface. The two documents added no claim to any gate's input, and the only numbers touched were the two baselines, both re-verified by running their gates.

---

## b) PARTIALLY DONE

### b1. The design answer is prose only

No ADR, no acceptance criteria, no issue. The plan document written in this pass supplies the criteria and the task decomposition, but the two decisions that must precede code (the principal model, the authority source) still have no ADR stub.

### b2. Every effort estimate is a guess, labelled as fact

"Hours", "days", "weeks" in the wave table are intuition. The only measured quantities in either pass are file and line references. An estimate that turns out wrong is cheap; an estimate presented as a plan is not, because it sets the reader's expectation of when the hole closes.

### b3. The "roughly seventy percent of the pieces exist" figure is rhetoric

I did not count. Either count (capabilities reused versus capabilities missing) or drop the number. As written it invites a reader to believe the remaining work is small, which is the opposite of the safe reading.

### b4. The real docs-gates battery was never run

Four of the five gates touch exactly what I wrote: freshness, annotate verdicts, routing, and cadence. I ran a hand-written approximation of one formatting rule and two budget scripts. The deliverable would be checked by that battery in CI or in a docs-health sweep, not by me.

### b5. The task ordering in the answer was intuitive, not scored

The waves were ordered by argument, not by a rubric. The plan document corrects this with explicit impact, effort, customer-value, and risk columns, but the correction costs a second document where one scored table would have done.

### b6. Findings live in a status report, which is not a tracker

Seven defects that a consumer can hit today exist only inside a historical document. The repo's own docs-health rules say a status report is a snapshot, not a work item. They belong in the ticket system or in the TODO list with owners.

---

## c) NOT STARTED

Ten things this pass should have done and did not:

1. Run the five-gate docs battery against the two new documents.
2. Write the two ADR stubs (principal model; machine authority source).
3. File the seven findings as issues, or move them into the TODO list with owners.
4. Verify that the sync pull and push handlers can authenticate a machine principal, which the design answer implicitly assumes when it says every surface should accept one.
5. Check whether projections other than the Casbin one have the same raw-subject versus prefixed-subject mismatch (flagged as a next step, never grepped).
6. Check whether `Bot.Deleted` is ever observable given the read model deletes the entry outright, which determines whether tombstoning is available as a design option.
7. Measure the hot-path cost of a single-principal resolution before proposing it, since the dispatch gate already had a lock-removal pass for exactly this reason.
8. Decide whether the machine credential fingerprint in an auth event should be the hash prefix, a truncated HMAC, or a separately minted identifier.
9. Re-measure the corpus after the daemon's commit to prove the pin still matches the tree.
10. Check whether the plan's waves collide with sibling sessions' in-flight lanes before declaring an order.

---

## d) TOTALLY FUCKED UP

### d1. I recommended a mechanism before proving it reachable

The first answer told a consumer to gate an agent with a policy row on the kind-prefixed subject. The audit then proved the projection cannot create that row, and that the only row-creating path left is the one the repo documents as an ephemeral bug. I published the recommendation, then discovered the constraint that makes it non-durable. The order should have been the reverse: prove the row is creatable through a sanctioned path, then recommend the gate.

The design answer does open with "fix this first", so the two documents do not contradict each other. But a reader who stopped after the first answer got the broken recipe. That is the exact failure the repo calls a split brain, committed in prose instead of in code.

### d2. The corpus baseline was pinned from a measurement taken inside a race

I measured 52272 lines, wrote the number, and re-ran the gate in the same command. Sibling sessions were committing to this tree the whole time (`d66af0c5`, `048f041b` landed after my commit). If any of their documents changed between my measurement and the daemon's commit of my baseline, the pin is wrong in one of two directions, and both are bad:

- Pinned too high: silent growth is permitted, and the ratchet reports green while the corpus inflates.
- Pinned too low: the next committer gets a red gate whose cause is invisible.

I did not re-measure after the commit, and I did not check the count at commit time against the count I wrote. The mitigation is one command, and it belongs in the same session, not the next one.

### d3. I verified with a hand-run equivalent instead of the delivering invocation

The gates were run as `bash scripts/checks/…` rather than through the flake apps that check-modules and CI invoke. The logic is identical today, and that is precisely the assumption that ages badly: an app can acquire different flags, a different working directory, or a different toolchain, and my green would not have noticed. The repo has a written lesson about verifying the delivering layer rather than the commit; I applied it to skill fan-out two days of commits ago and not to my own gate run.

### d4. The most important table in the design answer has a mislabelled row

The row that reads "Human authority" carries the evidence that bots are not projected, which is the opposite of what the label says. In a document whose whole value is a scan of one table, a mislabelled row is not a typo; it inverts the finding for anyone skimming.

### d5. The same question set now exists in two documents

The three owner decisions appear in the audit report and again in the plan. Two live copies of the same open decision is how the next session answers one and forgets the other. One list, one owner, one place.

---

## e) WHAT WE SHOULD IMPROVE

### e1. Publish only what is reachable

Before writing "do X to get Y", prove three things: X is expressible with the sanctioned mechanism, Y is observable, and the failure mode of X is stated. The first answer failed the first test.

### e2. Baseline pins need a post-commit re-measure

The safe sequence in a shared tree is: write the artifact, measure, write the pin, commit, then re-measure from the committed tree. The fourth step is the one I skipped, and it is the only one that catches a sibling edit landing inside the window.

### e3. Run the gate the way the gate ships

Flake app or script is a real difference, and the cheap habit is to run the documented invocation once per document change, not the equivalent.

### e4. Count or drop the number

Percentages of "pieces that exist" are rhetorical unless they are counts of named capabilities. Otherwise the plan inherits a claimed head start it does not have.

### e5. One decision list, with an owner

Owner questions belong in the TODO list or an ADR stub with a named decider, not repeated at the bottom of every report.

### e6. Findings need a home with a life longer than a report

A status report is a snapshot by the repo's own rules. Seven actionable defects need either issues or tracked rows, otherwise the next docs sweep archives them and the knowledge becomes archaeology.

---

## f) Up to 50 things we should get done next

Grouped by intent; the execution decomposition lives in the plan document named at the top of this report.

**Immediate truth fixes (do not wait for a wave)**

1. ~~Delete the false `SetTokenPepper` sentence at `usermgmt/service_bot.go:28`.~~ done at `d877e64e`
2. ~~Rewrite the bot-delete comment at `usermgmt/es_casbin_projection.go:119` to state what is actually removed.~~ done at `d877e64e`
3. ~~Correct the `FEATURES.md` row that claims complete bot policy cleanup.~~ done at `d877e64e`
4. ~~Fix the mislabelled row in the design answer's evidence table (`docs/status/2026-11-06_11-13_…`).~~ done — git log -S over full history finds the phrase only in the two reports own critique prose (a53f0909, 686e3cad) — the mislabelled evidence table never existed in-tree, nothing to fix (conversation-only artifact, W0 report §b1)
5. ~~Collapse the duplicated owner-question lists into one tracked place.~~ done at `995a7312`

**Reachability and correctness (the real hole)**

~~6. Add a `BotRegistered` branch to the Casbin projection deriving rows from scopes.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 1 — durable grant path (plan T8/W1, blocked on owner decisions D1/D2)
~~7. Add `BotScopesChanged` so authority itself is an event.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — BotScopesChanged is plan A9/T9 and rides the D2 authority-source answer
~~8. Make bot deletion remove direct policy rows, on the subject spelling the dispatcher checks.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 2 — subject-correct delete cleanup (plan T10/W1)
~~9. Prove the four-step invariant in one test: register, authorize, delete, re-authorize, expect deny.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the four-step invariant test is plan T11/A11, sequenced behind T8/T10
~~10. Add a restart test: register, restart with hydration, authenticate, expect success.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 7 — hydration-dependent token resolution (plan T12/W1)
11. Grep every other projection for the raw-subject versus prefixed-subject mismatch.
12. Grep every other `RemoveAll*` call for the grouping-only blind spot.
13. Decide whether deleted bots tombstone instead of vanish, then check whether `Bot.Deleted` is reachable at all.
14. ~~Add an assertion that a dispatched command from a machine principal never carries a zero actor.~~ done at `236177a6`

**Credential lifecycle**

~~15. Add a token prefix and checksum so a leak is identifiable without disclosure.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — token prefix + checksum is plan A5/T6
~~16. Require expiry at registration and enforce it in resolution.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — required expiry is plan A6/T7
~~17. Add rotation with an overlap window and its own event.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — rotation overlap is plan A26
~~18. Replace the single pepper with a keyring and a key-id in the hash.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — keyring pepper is plan A25
19. Add audience binding so a credential minted for one project is refused by another.
20. Design the short-lived exchange (long-lived credential to minutes-long token) before implementing it.
21. Verify or justify the resolution path's comparison against a constant-time alternative.
~~22. Add a test that a rotated-away token stops resolving and that the failure is a 401, not a 500.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — the rotation test rides plan A26

**Principal and surface consistency**

~~23. Resolve one principal abstraction from either a cookie session or a bearer credential.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — principal resolution is plan A17
~~24. Make the SSE and DataStar feeds accept a machine principal with a scoped filter.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 4 — SSE/DataStar session-gating (plan T15/W2)
~~25. Make the sync pull and push handlers accept the same principal.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — sync pull/push is plan A19
~~26. Make the Huma adapter and the route pipeline derive the same subject for the same caller.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — Huma/route subject parity is plan A20
27. ~~Replace `RequireBot`'s plain-text rejection with the shared problem document and a stable code.~~ done at `4084bf49`
28. ~~Bridge a machine principal into the actor chain inside the token middleware.~~ done at `4084bf49`
~~29. Measure the hot-path cost of principal resolution before adopting it, and record the number.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — hot-path benchmark is plan A27
30. Add a gate or test asserting every documented policy-subject form matches the dispatcher's output.

**Delegation (the agent differentiator)**

~~31. Draft the delegation aggregate: delegator, delegate, scope subset, purpose, expiry, revocation.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row gates W2 (D3 shapes the delegation shape) — the delegation aggregate is plan A21
~~32. Enforce that delegation never exceeds the delegator's authority at dispatch time.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row gates W2 — dispatch-time subset re-evaluation is plan A22
~~33. Record both actors on every delegated event, reusing the impersonation shape from ADR-0060.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row gates W2 — on-behalf-of audit fields are plan A23 (ADR-0060 shape)
~~34. Add a revocation path that takes effect on the next dispatch without restart.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row gates W2 — the revocation path rides plans A21/A22
~~35. Prove the demotion case: grant while admin, demote the human, expect the agent denied.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row gates W2 — the demotion proof is plan A24

**Observability**

~~36. Add a successful authentication event carrying a credential fingerprint, never the credential.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — audit-journal silence (plan A7/W1, granularity is owner decision D4)
~~37. Add a failed authentication event with a reason and no secret material.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — audit-journal silence (plan A7/W1, granularity is owner decision D4)
38. Journal denials with the decision reason instead of only returning a status.
39. Add per-principal usage counters so idle credentials are visible.
40. Write the query that must be answerable, as a test fixture, not as prose.

**Documentation and examples**

41. Write the missing bot section in the usermgmt README: register, mount, bridge, authorize, revoke.
~~42. State plainly that realtime feeds are session-gated today and what the request/response alternative is.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 4 — the README statement rides the SSE session-gating finding (plan T15/W2)
43. Add a runnable example of a machine principal calling a command endpoint.
~~44. Document the hydration requirement for token authentication.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 7 — the hydration requirement doc rides plan T12/W1
~~45. Record the principal model and the authority-source decisions as ADRs before code.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — D2 gates ADRs A15/A16 before any W1 code
46. Add the data-mesh resource naming convention to the authorization guide.
47. Publish the catalog access metadata field so a data product can declare what it requires.

**Process and proof**

~~48. File the seven findings as issues, or as owned rows in the TODO list, and delete them from the report.~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row — the owned-rows venue was taken (W0 §a6), the report stays as the findings source
~~49. Extend the authorization proof kit with a principal matrix covering user, bot, system, service, and delegation.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the principal matrix is plan A13/T13
50. Add a post-commit baseline re-measure step to the report-and-plan habit, so a shared-tree pin is never written blind.

---

## g) Three questions only you can answer — tracked canonically in the TODO_LIST.md "M2M owner decisions" row since 2026-11-06

1. **Do the seven findings become issues now, or wait for the wave that fixes them?** My preference is issues now, because a status report is archived and the finding becomes archaeology, but filing seven issues may collide with how you triage this repo's backlog, and that is your call rather than mine.

2. **Should the two ADRs be drafted in this session, before any code?** I believe they must, since the principal abstraction and the authority source are both hard to reverse and the code currently implies three mutually exclusive answers. But a session that spends its remaining budget on documents instead of on the truth fixes is a trade I should not make alone.

3. **Does one agent ever act for several humans at once?** This is the one question I cannot answer from the code and it changes the data model: a delegation record per delegator implies the agent's authority is the union of live grants evaluated per dispatch, whereas a single-delegator model can be folded into one field on the bot. Everything else in the plan is expressible either way.

---

## Verification notes

| Claim | Command |
| --- | --- |
| Neither report violated the backtick format rule | scripted pairing check over both files, 0 violations |
| Tail and corpus ratchets are green | `bash scripts/checks/check-docs-tail-budget.sh --ratchet` and the corpus equivalent |
| The report this one audits is committed | `git log --oneline -1 -- docs/status/2026-11-06_11-13_…status.md` returns `be5dbb6f` |
| Working tree state at report time | `git status --short` empty; two sibling commits landed after mine |

No build, test, lint, or docs battery was executed. The one open risk carried forward from this pass is the corpus pin written inside a shared-tree write window, which is re-measured at the end of this session rather than left for the next one.
