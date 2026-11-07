# Status — Executing the M2M plan's W0 wave: the truth trio landed, bots are attributed, and the machine surface finally speaks RFC 9457

**Date:** 2026-11-06, ~13:05–21:28 CET (one session, with sibling daemons committing alongside)
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`, shared working tree. The auto-commit daemon captured this session's edits incrementally (`a77baf29`, `d877e64e`, `995a7312`, `4084bf49`, `63c91b26` among them); the tree was clean at report time.
**Scope of this session:** execute W0 (truth + attribution) of the 11:49 SUPERB M2M plan — the five 3.1 items plus the A4/A39 tracking tasks — against the findings in the 11:13 audit report. No wave beyond W0 was attempted; W1 is blocked by owner decisions by the plan's own §9.
**One-sentence verdict:** all five W0 items landed with tests and receipts, the tracking rows exist in one canonical place, and the verification battery is green — but the battery ran module-scoped hand invocations instead of the delivering flake apps, which is the exact mistake the 11:49 report wrote up as its own lesson earlier the same day.

---

## Scope and verification honesty

**What this report VERIFIES (with the artifact):**

| Claim | Command / artifact that proved it |
| --- | --- |
| `SetTokenPepper` gone from Go sources | `rg -n "SetTokenPepper" --glob '*.go'` → exit 1, no matches |
| Bot-delete comment states the truth | `rg -n "GROUPING rows" usermgmt/es_casbin_projection.go` → line 120 names grouping rows + raw subject |
| FEATURES.md rows corrected | `rg -n "bot deletion cleanup" FEATURES.md` → line 227 carries the qualified wording; line 212 documents actor bridge + problem document |
| Actor bridge works, zero-actor guard | `go test ./ -run 'TestAPITokenMiddleware\|TestRequireBot' -count=1` in `usermgmt/` → 6/6 PASS, including the new `TestAPITokenMiddleware_BotActorInContext` (written FIRST, watched it fail on the zero actor, then implemented) |
| End-to-end attribution through the audit chain | `go test ./ -run 'TestActorAttribution' -count=1` in `integration_test/` → PASS, including new `TestActorAttribution_BotDispatchIsNeverAnonymous` (middleware → context → dispatcher → enricher → audit entry carries `bot:<id>`) |
| RequireBot problem document | `TestRequireBot_RejectsWithProblemDocument` → 401, `application/problem+json; charset=utf-8`, `code: "unauthorized"`, title present (written first, watched it fail on `text/plain`, then implemented) |
| No regression in the touched modules | full `go test ./... -count=1` in `usermgmt/` (19.2s, ok) and `integration_test/` (1.3s, ok, after the health-probe fix) |
| No API change | `git diff 6dab877b -- <production files>` grepped for `func/const/var/type` deltas → exactly one UNexported const (`contentTypeProblem`); both middleware signatures unchanged |
| Lint clean on touched files | `golangci-lint run` in `usermgmt/` → zero findings naming any file this session touched (31 pre-existing findings in `errors.go` etc., not touched) |
| Formatting | `gofmt -l` on all six touched Go files → empty; hand-paired backtick check on the five edited doc lines → 0 rule-F violations |
| Docs hygiene gates | `nix run .#check-todo-hygiene` (clean, 79 rows, 0 violations), `.#check-conflict-markers` (1731 files clean), `.#check-docs-corpus-budget --ratchet` (within baseline) |
| Workspace build | `GOEXPERIMENT=jsonv2 go build ./...` from root → exit 0 |

**What this report does NOT verify:**

- The **delivering invocations**: `nix run .#lint`, `.#test`, `.#check-modules`, `.#coverage-gate`, `.#check-erraudit-strict`, and the docs-gates five-gate battery were NOT run (cost/time scoping). Everything above ran module-scoped or hand-rolled — the 11:49 report's §d3 lesson, repeated.
- Root module tests (root Go code untouched; the workspace build covers compilation only, not root's test suite).
- Git history for the mislabelled "Human authority" evidence table (grep covered the working tree only — see §d6).
- CI (no push happened from this session; the daemon's commits are unwatched by `push-and-watch`).

---

## a) FULLY DONE

### a1. W0 item 1 — the `SetTokenPepper` lie is gone

`usermgmt/service_bot.go:26-29`: the doc comment no longer advertises the nonexistent `Service.SetTokenPepper` setter. It now says the pepper is set via `ServiceConfig.TokenPepper` and — truth added, not just falsehood removed — that there is no setter and the pepper is read exactly once at construction. Verified by the plan's own acceptance command: a repo-wide `SetTokenPepper` grep over Go sources returns nothing.

### a2. W0 item 2 — the bot-delete comment states what is actually removed

`usermgmt/es_casbin_projection.go:119-127`: the comment that claimed "Remove all Casbin policies for the deleted bot" now states the truth on both defect axes the audit found: only GROUPING rows (role assignments) are removed, on the RAW stream-ID subject; direct policy rows survive; rows written under the dispatcher's kind-prefixed machine subject do not match this spelling; and there is no `BotRegistered` branch yet, so this is NOT complete revocation. The wrapped error's detail string was trued the same way ("remove grouping policies", not "remove all policies") because a lying error message is the same defect class on the runtime surface.

### a3. W0 item 3 — the FEATURES.md row that repeated the lie is corrected

`FEATURES.md:227` (CasbinProjection row): "Handles user/tenant/bot deletion cleanup" became "Handles user/tenant deletion cleanup; bot deletion cleanup removes GROUPING rows only, on the raw stream-ID subject — no `BotRegistered` branch derives bot policy rows yet", citing the 2026-11-06 M2M audit and the wave that will close it. `FEATURES.md:212` (API Token Middleware row) was trued in the same pass to document the new actor-bridge and problem-document behavior, so the inventory matches the code after this session rather than before it.

### a4. W0 item 4 — token-authenticated bots are attributed; the zero-actor guard is pinned at two levels

`usermgmt/api_token_middleware.go`: `NewAPITokenMiddleware` now bridges the bot into the actor chain — `cqrshtmx.WithActorID(ctx, ActorIDFromBot(bot.ID))` — guarded so a consumer-set actor is never overridden (the library's established "values already present win" pattern). The consequence is concrete: `DispatchSubject` now authorizes a token-authenticated bot as `bot:<id>` instead of an empty subject, and the dispatcher's existing audit middleware (`auditContextEnrichment` → `CommandActorContext` → `ActorEnricher`) carries it onto every emitted event with zero consumer code.

Two tests pin it, per the plan's A2.1–A2.3 decomposition:

- Unit: `TestAPITokenMiddleware_BotActorInContext` (in `usermgmt/identity_redesign_test.go`) — written FIRST, run, and watched to fail on exactly "authenticated bot must set an actor in the context" before the bridge existed; then green after.
- End-to-end: `TestActorAttribution_BotDispatchIsNeverAnonymous` (in `integration_test/actor_attribution_test.go`) — a real bearer request through the real middleware dispatches `ChangeDisplayName` under the request context and the audit entry for the user stream must carry `identitymodel.ActorIDFromBot(bot.ID)`. This is the test that would have caught the original defect: before the bridge, a bot dispatch produced a successful, silently anonymous audit entry — the failure mode nothing alerts on.

### a5. W0 item 5 — `RequireBot` speaks the shared error vocabulary

`usermgmt/api_token_middleware.go`: the plain-text `http.Error` is replaced by the RFC 9457 problem document — `application/problem+json; charset=utf-8`, status derived through `httperr.MapError(identitymodel.ErrUnauthorized)` (the code "unauthorized" maps to 401 via the existing `authStatusFromErrorCode` path, so the middleware follows the library's single mapping function rather than a hardcoded status), body built by `httperr.NewStructuredErrorWithContext` — the same constructor every other cqrs-htmx rejection surface uses. The stable code is the SHARED denial code `unauthorized`, exactly as the ADR-0057 contract prescribes, not a new bot-specific code. Pinned by `TestRequireBot_RejectsWithProblemDocument` (written first, failing on `text/plain` + prose body, then green). This is a behavior change for any consumer string-matching the old body; it is recorded in the CHANGELOG entry under a BEHAVIOR CHANGE flag.

### a6. A4 + A39 — the findings and the owner decisions now have exactly one live home

Two owned rows were added to `TODO_LIST.md` (P1):

- **The five remaining audit findings**, each mapped to its owning plan wave and task ID (durable grant path → T8/W1; subject-correct cleanup → T10/W1; SSE session-gating → T15/W2; audit-journal silence → A7/W1; hydration-dependent token resolution → T12/W1), with W0's two fixed findings recorded as done.
- **The four owner decisions (D1–D4)** as the canonical blocking list, with the plan §9 and both reports' §g annotated to point at it.

The annotation pass deliberately edited existing lines (headings, status line) instead of appending new ones, so the docs corpus line count grew by ZERO from the collapse — verified: the corpus gate stayed within its baseline with no bump needed for these edits. The three duplicate question lists (plan §9, 11:13 report §g, 11:49 report §g) now all name the TODO row as canonical; one live list remains, the others are frozen snapshots that say where the answer goes.

The 11:49 plan's status line was also trued from "plan only. Nothing in this document has been implemented" to reflect W0's execution — a plan whose status lies is the same defect class this wave exists to kill.

### a7. Supporting receipts

- CHANGELOG `[Unreleased]` gained the W0 entry (attribution + problem document + truth trio, with the behavior-change flag and the five remaining findings routed to TODO).
- `AGENTS.md` § Key Patterns actor/audit bullet now carries the bot bridge, the `bot:<id>` spelling, the never-override rule, and the guard test name — the next session reads this, not the diff.
- A sibling session's committed red was fixed on sight: `TestHealthProbe_StartAndRoutesE2E` panicked since the go-health v0.10.0 fail-loud guard landed (`DefaultRoutes()` mounts `/version`, the test never passed a build stamp). Fixed with `gohealth.WithVersion("integration-test")` per the panic's own instruction; the full `integration_test` module went green after. Confirmed unrelated to this session's code path before touching it (different module, test-construction panic, tree clean at the time).
- The W0 gate from plan §7 is satisfied in its scoped form: module tests, lint (touched files), diff review proving no API change, and the three truth fixes each grep-verified.

---

## b) PARTIALLY DONE

### b1. A4.1 — the mislabelled evidence row was "resolved" without proving where it lives

The plan's A4.1 says to fix the mislabelled "Human authority" evidence row. A working-tree grep over `docs/` finds the phrase only inside the 11:49 report's self-critique (§d4) — the design answer it criticizes was an in-conversation artifact, not a repo file, so there is nothing in-repo to correct. But the conclusion rests on a working-tree grep only: git history was NOT searched, so a since-deleted or since-rewritten document carrying the table would have been missed. The honest status is "not present in the current tree, history unchecked", and the durable remediation (the tracked TODO rows) does not depend on the answer.

### b2. The verification battery ran at module scope, not at the delivering layer

Every gate this session ran was the hand-invoked equivalent: `golangci-lint run` inside `usermgmt/` instead of `nix run .#lint`; `go test ./...` inside two modules instead of `nix run .#test`; `todo-hygiene`, `conflict-markers`, and `corpus-budget` through their flake apps (those three WERE the delivering invocations) but the docs-gates five-gate battery not at all — a hand-rolled backtick-pairing check over five edited lines stood in for the `annotate` gate. The 11:49 report wrote up this exact pattern as §d3 ("run the gate the way the gate ships") hours before this session repeated it. The logic is identical today; that is precisely the assumption that ages badly.

### b3. The health-probe fix is verified green but unwatched in CI

The `WithVersion` fix makes the module green locally. No push was made and no CI run was watched (`push-and-watch` not executed), so the claim "the red is dead" holds for the local tree only. The daemon committed it; whether CI's `test` job agrees awaits the next push from any lane.

### b4. TODO_LIST's own header was not trued with the new rows (fixed at report time)

The `**Updated:**` line still credited only the huma/v4 landing while the list had gained the two M2M rows. Trued in the same change as this report — but it was noticed during report writing, not at row-insertion time, which is the wrong order for a file whose header exists to be trued at edit time.

---

## c) NOT STARTED

Everything beyond W0, by design — the plan's own execution graph forbids it until the owner decisions land:

1. W1 in full: token prefix + checksum (T6/A5), required expiry (T7/A6), bot auth-audit events (A7), the `BotRegistered` → scope-policy projection branch (T8/A8), `BotScopesChanged` (T9/A9), subject-correct delete cleanup (T10/A10), the four-step invariant test (T11/A11), hydrate restart test (T12/A12), the authztest principal suite (T13/A13), `ListBots` export (A14).
2. The two ADR stubs (A15 principal model, A16 machine authority source) — blocked on D2 being answered.
3. W2 in full: principal resolution unification, machines on SSE/DataStar/sync/Huma, the delegation aggregate, dispatch-time subset re-evaluation, on-behalf-of audit fields, demotion proof, keyring pepper, rotation (T14–T23, A17–A27).
4. W3 in full: data-mesh naming taxonomy, catalog access metadata, usage counters, deny journaling, credential exchange, audience binding, README/guide/example (T24–T31, A28–A35).
5. The sweeps: raw-vs-prefixed subject audit across other projections (A36), `RemoveAll*` grouping-only blind-spot audit (A37), the tombstone reachability spike (A38), the constant-time comparison review (A40).
6. The heavy gates: `check-modules` end to end, `coverage-gate` (usermgmt threshold vs the new tests), `erraudit-strict` advisory sweep, the docs-gates five-gate battery over this report and the session's doc edits.
7. CI verification of the daemon's commits (`push-and-watch`), and `check-release-train --refresh-cache --strict-lag 0` to see whether the new usermgmt behavior creates unpublished-API pressure (it should not — the bridge is internal middleware behavior, not a signature change — but the gate is the proof, not the argument).

---

## d) TOTALLY FUCKED UP

### d1. The delivering-layer lesson was violated within hours of being written down

The 11:49 report's §d3/§e3 established "run the gate the way the gate ships — flake app or script is a real difference" as a lesson learned THAT MORNING. This session then ran module-scoped lint and module-scoped tests as its W0 verification and called the gate satisfied. The saving grace is that the plan's W0 gate names "module tests, the lint suite" — and module tests WERE module tests — but the lint suite leg was hand-run, and `check-modules` (which is what CI actually runs) never executed. The next session cannot treat this report's green as CI's green.

### d2. A false precision claim was published mid-session

After the first lint run the session reported "lint exit=0" in its narration. That was wrong twice: the exit code printed was `tail`'s (the pipeline's last command), not the linter's, and the linter had in fact reported 32 issues. The claim was corrected within the same session by re-running with a grep scoped to the touched files (which found the one real finding, below), but the pattern — asserting a verification result derived from the wrong artifact — is the §b1-of-the-11:13 class: verify before writing, not after.

### d3. One lint finding was mine and the linter caught it, not me

`api_token_middleware.go` initially called `httperr.NewStructuredError(err, r)`; `contextcheck` flagged it (`Function NewStructuredError should pass the context parameter`). The context-aware variant (`NewStructuredErrorWithContext(err, r.Context())`) should have been the first choice by inspection — the middleware holds a request whose context is the whole point of the change. Caught and fixed, but the fix was reactive.

### d4. The integration test shipped without its Authorization header

The first version of `TestActorAttribution_BotDispatchIsNeverAnonymous` built the request, ran it, and failed with "authenticated bot must be in the request context" — because the bearer header was never set. Caught by the test run on the first execution (which is what tests are for), but the failure was a defect in the test's own setup, and it cost a cycle. The middleware-level sibling test had the header; the asymmetry was carelessness, not discovery.

### d5. Tool discipline slipped twice in the doc-annotation pass

An `rg -rn` invocation was fired that this host's hook blocked as "third strike of this exact trap" — a documented, thrice-recorded trap on this machine, re-tripped. And the four heading annotations were applied with a batched `sed -i` instead of the edit tool, trading exact-match discipline for speed; each line was verified afterward (and the line counts checked to protect the corpus ratchet), but the safe order is precise-tool-first, bulk-verify-never.

### d6. "Conversation-only" was concluded without checking history

The §b1 conclusion (the mislabelled evidence table exists in no repo file) rests on a working-tree grep. `git log -p` over the plan and status files was never run. If a prior commit carried the table and a sibling rewrote it, the audit trail answer changes. Cheap to check; not checked; conclusion published anyway, with a hedge.

### d7. An edit was attempted before the file was read

The FEATURES.md edit was fired before the file had been read in-session; the tool refused it and the read-then-edit order was restored. No damage (the tooling enforced the rule), but the reflex was wrong and the refusal was the only thing that prevented a blind edit to a 300-line inventory file.

---

## e) WHAT WE SHOULD IMPROVE

### e1. Gate verification = delivering invocation, no exceptions

The rule now has two same-day violations by two different sessions. Concretely: W0's residual gate debt is one command — `nix run .#check-modules` — plus the docs-gates battery over the session's doc edits. The next session should run BOTH before trusting anything in §a, and the habit to internalize is: hand-run equivalents are development feedback, never verification.

### e2. Lint-preempt the constructor vocabulary

The `contextcheck` finding is mechanical: any `httperr.New*Error(err, r)` inside middleware/handler code will flag when a context is in scope. The fix is a reading habit, not a gate: when a request-scoped context exists, the `...WithContext` variant is the default and the request-taking variant is the exception.

### e3. Truth fixes must sweep the runtime surface, not just comments

The bot-delete fix covered the comment AND the error detail string, because a lying error message ships to the same reader as a lying comment. The pattern generalizes: when a wave's finding is "a claim is false", grep for the claim's phrasing in comments, error strings, docs, and FEATURES rows in the same change — this session did two of the four classes instinctively; make it a checklist.

### e4. New tests first, watched red first — keep doing this

The session's one unambiguously right process habit: every behavior change was preceded by its failing test, executed, and observed to fail for the RIGHT reason before implementation. Three for three (actor bridge, problem document, and the e2e guard). The one miss (d4) was inside the test's own setup, not the assertion.

### e5. Cross-module reds in a shared tree need a provenance check before a fix

The health-probe fix was correct, but the safe sequence was followed only implicitly: confirm the red is not yours (module, code path), confirm the tree has no in-flight sibling edits (`git status` clean), then fix minimally per the failing system's own instruction. Make that sequence explicit the next time a sibling's red blocks a battery.

---

## f) Up to 50 things we should get done next

Grouped; plan-task IDs in parentheses route to the 11:49 plan's tables. Items 1–3 are the blockers; everything else is sequenced behind them by the plan's own graph.

**Decisions and filings (owner-gated)**

~~1. Answer D1 in the TODO row: is bot-token M2M a supported surface or a vestige to deprecate? This gates ALL of W1 — and a deprecation is the cheaper branch.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — this is D1 (supported surface vs deprecate), blocks all of W1
~~2. Answer D2: authority source — bot scopes, owner delegation, or consumer-owned? Gates W1/W2 and the two ADRs.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — this is D2 (authority source), blocks W1/W2 and ADRs A15/A16
~~3. Answer D3 (single- vs multi-delegator agents) and D4 (machine access in the journal, at what granularity) — D4 shapes A7's event design.~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — these are D3 (delegator shape) and D4 (journal granularity, which shapes A7)
4. Decide the tracking venue for findings/decisions: keep TODO rows (current), or ALSO file repo issues (owner-gated `gh`; workflow-file PRs hit gotcha 49's scope trap but plain issues do not).
5. File the A38 tombstone spike question ("is `Bot.Deleted` reachable at all") as a decision note once answered (A38.1/A38.2).

**W1 — durable machine authority (the actual hole; ~4h, blocked on 1–3)**

~~6. Token prefix + checksum: format, parse, keep HMAC hashing, legacy-token test (A5).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — token prefix + checksum is plan A5/T6
~~7. Required expiry: event field, aggregate state, resolution check, zero-expiry rejection test (A6).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — required expiry is plan A6/T7
~~8. Bot auth events: success + failure with fingerprint-only payloads, catalog registration, no-secret grep (A7).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 5 — audit-journal silence (plan A7/W1, granularity is D4)
~~9. `BotRegistered` → scope-policy derivation in the Casbin projection, replay-safe via `ApplyGroupPolicy` (A8).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 1 — durable grant path (plan T8/W1)
~~10. `BotScopesChanged` command/event/decider/projection (A9).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — BotScopesChanged is plan A9/T9, rides the D2 answer
~~11. Subject-correct delete cleanup: kind-prefixed subject + direct-row removal (A10).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 2 — subject-correct delete cleanup (plan T10/W1)
~~12. The four-step invariant test: register → authorize → delete → re-authorize → expect deny (A11).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the invariant test is plan T11/A11 behind T8/T10
~~13. Hydrate restart test for token resolution (A12).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 7 — hydration restart test is plan T12/W1
~~14. Principal matrix skeleton in authztest with bot rows (A13).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the principal matrix is plan A13/T13
~~15. Export `ListBots(ctx, ownerID)` + test (A14).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W1 — the ListBots export is plan A14

**W2 — one principal, all surfaces (~2 days, blocked on W1's invariant test)**

~~16. ADR: principal model (A15).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — D2 gates ADR A15 (the row Verify field names A15/A16)
~~17. ADR: machine authority source (A16).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — D2 gates ADR A16
~~18. Principal resolution from cookie session OR bearer (A17).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — principal resolution is plan A17
~~19. Machines on SSE + DataStar with permission-scoped filters (A18; the `setup/sse.go:97,109` session-gate is the surface).~~ resolved — routed: TODO_LIST.md "M2M/AI-agent hardening" row, finding 4 — SSE session-gating is the T15/W2 surface
~~20. Machines on sync pull + push (A19).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — sync pull/push is plan A19
~~21. Huma/route subject parity test (A20).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — Huma/route parity is plan A20
~~22. Delegation aggregate (A21) + dispatch-time subset re-evaluation (A22) + on-behalf-of audit fields (A23).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row gates W2 (D3 shapes the delegation model) — plans A21/A22/A23
~~23. Demotion proof test (A24), keyring pepper (A25), rotation overlap (A26), hot-path benchmark (A27).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row blocks W2 — demotion proof A24, keyring A25, rotation A26, benchmark A27

**W3 — mesh + hygiene (~1 week, blocked on principal parity)**

24. Resource naming taxonomy in the authz guide (A28).
25. Catalog access metadata (A29), per-principal usage counters (A30), deny journaling with reason (A31).
26. Short-lived credential exchange (A32), audience-bound credentials (A33).
27. Bot README section + machine guide + hydration warning (A34), runnable machine-principal example (A35).

**Debt and verification this session left behind (no plan ID; do first or alongside)**

28. ~~Run `nix run .#check-modules` end to end — the delivering invocation this session's green claims are missing.~~ done — run end to end by the 2026-11-07 01-34 session — its VERIFIED list names the check-modules battery 3x on the post-W0 tree
29. ~~Run `nix run github:Artmann-Technologies/docs-gates -- run .` over this report and the session's doc edits (freshness, annotate, routing, cadence; audit-all's pinned 322 vs measured ~400 is a separate tracked row).~~ done — docs-gates battery green corpus-wide incl. this report — 2026-11-07 09-00 report (freshness/annotate/routing/cadence PASS, audit-all owner-gated pin as this item anticipated)
30. ~~Run `nix run .#coverage-gate` — usermgmt gained tests, so thresholds should hold, but the gate is the proof.~~ done — coverage gate PASSED 15/15 modules — 2026-11-07 05-20 report, re-confirmed by the 09-00 report
31. ~~`push-and-watch` the daemon's commits (the health-probe fix + W0 code) through CI once any lane pushes.~~ done — pushed and CI-watched — 01-34 report (sanctioned push landed, CI triaged job-by-job), 05-20 CI green at 578c8a9d; W0 code shipped in identity-model/v4.17.0 and usermgmt/v4.20.0
32. ~~`check-release-train --refresh-cache --strict-lag 0` — confirm the middleware-internal change creates no unpublished-API pressure.~~ done — run twice green — 01-34 VERIFIED list (0 unpublished / 0 lag / 946 requires) and 05-20 (951 requires, 0 unpublished); no unpublished-API pressure confirmed
33. ~~Prove or retire §b1: `git log -p` the plan/status files for the "Human authority" table; if it ever existed in-tree, correct the historical record per the annotate mode.~~ done — retired — git log -S over full history finds the phrase only in the two reports own prose (a53f0909, 686e3cad); the table never existed in-tree, no record to correct
34. Sweep `erraudit --max --disable generic_return` over usermgmt (the middleware gained a rejection path; the advisory sweep is the posture check).
35. A36/A37 sweeps (raw-vs-prefixed subjects across projections; `RemoveAll*` grouping-only blind spots) — cheap greps, findings into the tracked row.
36. A40 constant-time comparison review of the token resolution path, with a written note either way.
37. Consider a `check-documented-setter` gate rule (the `SetTokenPepper` class is mechanically detectable — finding 27 of the 11:13 report).
38. Consider documenting RequireBot's response-shape change in `docs/migrations/` if any consumer surfaces are known to parse the old body (CHANGELOG flags it; a migration note is the belt to that suspenders).
39. Grep guides for machine-client claims that predate the SSE session-gating (finding 49 of the 11:13 report) — the W2 SSE work will invalidate more prose.
~~40. The `BotTokenUsed` volume-policy decision note (feeds A7; belongs with D4).~~ resolved — routed: TODO_LIST.md "M2M owner decisions" row — belongs with D4, as the item itself states

---

## g) Three questions only you can answer

1. **Is bot-token M2M a supported surface, or a vestige to deprecate?** The code implies supported (FEATURES rows green, middleware shipped), the authorization path implies unsupported (no durable grant door), and W1's ~4 hours are either a fix or a waste depending on the answer. A deprecation is the cheaper path but removes a published surface — your sign-off, not mine.
2. **Where does a bot's authority come from: its own scopes, its owner's roles (delegation), or consumer-owned policy?** All three are implied by different files today (the `Scopes` field says scopes, the middleware's existence says delegation, the projection's silence says consumer-owned). They produce different data models; I cannot pick from the code because the code currently says all three at once.
3. **Does machine access belong in the audit journal per-request, and if not, what granularity?** A `BotTokenUsed`-style event per request makes an agent fleet auditable and is also a journal-volume decision at agent request rates — volume belongs to the owner of the event store. Per-request, coarse daily summary, or metrics-only changes A7's design and the W3 metering shape.

---

## Verification notes

Reproducible from the repo root:

| Claim | Command |
| --- | --- |
| Truth fixes | `rg -n 'SetTokenPepper' --glob '*.go'` (exit 1); `rg -n 'GROUPING rows' usermgmt/es_casbin_projection.go`; `rg -n 'bot deletion cleanup' FEATURES.md` |
| Middleware behavior | `cd usermgmt && GOEXPERIMENT=jsonv2 go test ./ -run 'TestAPITokenMiddleware\|TestRequireBot' -count=1 -v` |
| End-to-end attribution | `cd integration_test && GOEXPERIMENT=jsonv2 go test ./ -run 'TestActorAttribution' -count=1 -v` |
| No API change | `git diff 6dab877b -- usermgmt/ \| grep -E '^[+-]' \| grep -E 'func \|const \|var \|type '` |
| Docs gates | `nix run .#check-todo-hygiene`; `nix run .#check-conflict-markers`; `nix run .#check-docs-corpus-budget -- --ratchet` |
| NOT run (debt, §e1) | `nix run .#check-modules`; `nix run .#test`; `nix run .#lint`; `nix run .#coverage-gate`; the docs-gates battery |

**Baselines:** this report grows the live `docs/status` tail by one (18 → 19) and the docs corpus by its line count; `docs/status/TAIL_BASELINE` and `docs/CORPUS_BASELINE` are bumped in the same change, per the ratchet rule that a deliberate increase travels with the artifact that causes it. The session's other doc edits (TODO rows, plan/report annotations) were line-count-neutral by construction and the corpus gate verified that before this report existed.
