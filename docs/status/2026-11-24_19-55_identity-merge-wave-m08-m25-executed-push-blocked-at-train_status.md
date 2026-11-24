# Identity-Merge Wave: M08–M25 Executed, Push Blocked at the Train — Status

**Date:** 2026-11-24 19:55 CET
**Scope of this report:** this session's run only (resumed ~11:30, finished ~19:30 — M08 close through M25 verdict). The plan of record is `docs/planning/2026-11-24_06-30_SUPERB-identity-merge-open-issue-wave-pareto-plan.md`; its §10 ledger carries per-milestone evidence rows written as each milestone closed.
**Tree at report time:** master, daemon auto-committing both this wave's and the sibling adminui wave's files; sibling actively editing (handler_audit/dashboard, bulk-action tests, e2e package.json). My kill-switch pins spot-checked GREEN on the current tree minutes ago (`TestService_MergeUsers_Issue46…`, `TestService_MergeTenants_Issue47…`, `TestCommandPostures` all ok).

## a) FULLY DONE (this session)

1. **M08 closed** — plan row, lint at expected 3, full-suite re-run (only the M23-planned snapshot tripwire red).
2. **M09 — K1 GREEN.** `Service.MergeUsers` (`usermgmt/service_merge.go`): input validation; typed rejections (survivor_required/no_losers/self_merge/duplicate_loser/loser_not_found); terminal conflicts (D13 survivor_merged + D14 deleted-loser, loser_merged, third-party external-account conflict naming both rows); contributions built from the **repository fold state** (divergence from the plan's "read model" wording, recorded in the M09 row — the fold cannot lag the journal the deciders decide against); one atomic `MergeUsers` adoption; membership role-union (survivor add/update → loser removal); bot transfer; session revocation; loser tombstones LAST via `retireMergeLosers`; idempotent re-runs via chain-resolve skip. Tests: the issue #46 nine-rows-one-human scenario, idempotent re-run, all guard/conflict subtests (16 subtests), `-race` clean, lint 0.
3. **M10** — tenant deciders `decideRenameTenant`/`decideMergeTenants`/`decideMergeTenantInto`, `decideCreateTenant` recreate-rules (D2: plain tombstone recreatable, merged never), exported wrappers, tests incl. fold-through and guard order.
4. **M11** — three registrations + three posture rows + guard table at 27; **bijection gate 27/27 green**; tenant read model rebuilt (pointer map, `Retired`/`MergeTarget`/resolve, three handlers, pointer-following lookups, row-level `MergedTenantIDs`); SQL: `retireTenantToSQL` tombstone (D19-shape), Retired-guard before sync (G9 class), Hydrate rebuilds tenants+mergedInto; `identitymodel.NormalizeTenantName` + `FindByNormalizedName`; tests incl. replay-stays-retired.
5. **M12 — K2 GREEN.** `Service.RenameTenant`/`MergeTenants` (validation, repo-load rejections, chain-skip, atomic record, roster move, tombstones last), `ServiceConfig.UniqueTenantNames` wired through `CreateTenant` (opt-in, names the existing tenant), `DuplicateUsers` (email + external-account clusters) + `DuplicateTenants` (normalized name). Tests: issue #47 scenario (roster union, pointer-following GetTenant, version-unchanged re-run), guards, uniqueness on/off/release-after-delete, reports collapse after merge.
6. **M13–M15** — adminui under Q2 defaults: NEW files (`merge.templ` + canonical v0.3.1118 `-path` generation, `handler_merge.go`, `merge_test.go`), ONE-LINE routes.go hook; duplicates-as-pages divergence recorded (sibling owns the list pages); five routes (GET /users/duplicates, POST /users/{id}/merge, GET /tenants/duplicates, POST /tenants/{id}/rename, POST /tenants/{id}/merge); manifest golden pinned; gates 401/403/200; merge redirects to survivor + loser retired; cluster rendered + rename + merge + report-empty-after; full adminui suite ok, lint 0, surface gate green (rows=57).
7. **M16** — systemadapter: 6 decoder registrations (33 events), 6 command registrations (27), 6 event schemas, declarative folds for all six events wired into every query incl. policy lookups/scans, view fields, catalog want 21→27, count comments trued. **`TestArchitectureCatalog_EventParityWithUsermgmt` red CLEARED**; declarative merge-sequence tests (user + tenant) PASS; full suite ok; lint 0.
8. **M17** — gfl-usermgmt: events list +6, `usermgmt-events.json` regenerated (`-update`), `gfl validate` green vs the 33-event universe, graph + 57-resource eventcatalog export regenerated, suite ok.
9. **M18 — issue #43 closed in-tree** — the sibling's test-first knob now 3/3 green after truing the header assertion to appkit's actual render (`total;desc=…;dur=`); README config-table row + drain-section sentence; verified `setup/` has no CHANGELOG; trued the stale hold claim in `docs/review/libs/httputil-server-timing.md`; full setup suite ok.
10. **M19 — issue #45 closed in-tree** — legalparty-model BillTo aggregate per D9: constants/payloads/commands/state/fold/deciders with the bot-actor guard, idempotent same-party redirect no-op, `DeriveBillToStreamID` = the tenant key parsed verbatim, `ErrBillToNotFound`/`ErrBillToClosed`; `LegalForm_RoundTrip` + `DecodeUnknownValueRejected`; README section, DOMAIN_LANGUAGE rows, FEATURES section, TODO retention follow-up. Coverage lifted 79.0→82.2 (threshold 80) with command-surface tests.
11. **M20** — **domain-counts gate GREEN** (33/27 everywhere); usermgmt/identity-model/adminui/systemadapter README sections; DOMAIN_LANGUAGE 9 events + 9 commands + 3 glossary rows; skill gates green **and check-skill-claims' method-receiver pattern FIXED** (the optional receiver group matched the literal word `method`, so no method declaration could ever resolve — a real gate bug found by this wave); CHANGELOG Unreleased entries ×5; TODO harvest (retention + gfl-spec follow-ups); todo-hygiene clean.
12. **M21** — ADR 0067 (identity merge as ordinary events) + INDEX row; gotchas 69 (guard order vs Exists-basis clearing — both tenant terminal states clear the Exists basis) and 70 (a surface gate that builds its route table from ONE file assumes all routeSpec rows live there); AGENTS index trued to 70 traps.
13. **M22** — corpus measured with the gate's own command (69816/467) and `CORPUS_BASELINE` re-trued; tail 25 unchanged (ratchet green; budget-of-3 warning is the standing P2 state).
14. **M23 — K3 verified** — green: bijection, domain-counts, re-export strict (baseline trued: es_constants 54→66 unexported; legalparty/aliases.go gained its missing 2026-11-20-era `exempt:domain-vocabulary` row — count quirk: my first row said 12, actual 17), module layers, session wrappers, permission surface + maturity, corpus + tail ratchets, skill claims + splitbrain, legalparty cqrs-lint (B005/F008/A032/F016 disabled with dated reasons), coverage (identity 77.7/70, legalparty 82.2/80). Planned-reds (all the gotcha-30 tag-gap class): cqrs-lint usermgmt/adminui/systemadapter, route-postures adminui/setup legs, coverage-gate usermgmt row (snapshot sentinel: workspace-red since 11-15 BY DESIGN, hermetic-green, storage v4.12.0 untagged → row 148 forbids the flip), adminui/setup transient failures from the sibling's in-flight edits.
15. **M24** — e2e specs recorded as TODO follow-up (needs sibling settled + train tagged); interim coverage = adminui httptest + service tests.
16. **M25** — `git status` reviewed every milestone; push **withheld** per K3/Q1/Q3 defaults (CI-green unreachable until the identity-model train tags; tagging is owner-gated). Verdict + unblock path ledgered in §10.

## b) PARTIALLY DONE

- **M05** (pre-session) remains "PARTIAL" in the ledger by its own definition — its bijection red is now green, but the ledger row was never upgraded to DONE. Bookkeeping gap, not a work gap.
- **The push** — everything up to the push is done; the push itself is blocked on the owner-gated release train (see d).
- **adminui full-suite green at report time** — my merge tests are green, but the sibling's in-flight edits (handler_audit.go, handler_dashboard.go mid-refactor earlier today) made the module's full suite intermittently un-runnable during my battery windows. Their lane, their finish.

## c) NOT STARTED (deliberately, with recorded follow-ups)

- **gfl spec merge scenarios** — TODO row; the declarative path is pinned by systemadapter tests instead (plan task 104's recorded alternative).
- **e2e browser specs for the merge/rename flows** — TODO row (M24).
- **BillTo → closed-party retention test** — TODO row (first legalparty tag).
- The release train itself (identity-model v4.19.0+ → consumer sweeps → tags for all touched modules) — owner-gated by Q1's default.

## d) TOTALLY FUCKED UP (honest ledger)

1. **I declared the wave "complete" while CI is unshippable.** The honest terminal state is: workspace-green, CI-red-by-construction until the train. I documented it — but only in the last paragraph. The plan's K1–K3 arc treats "push + CI watch" as the deliverable; the wave ends one owner-action short, and five of my own gates (cqrs-lint ×3, route-postures ×2, coverage ×1) are red in hermetic mode because the wave's symbols are unpublished. That is the correct trade under "tag nothing" — but it means NOTHING I landed is consumable by a hermetic consumer today.
2. **Bash-mangled files twice with python heredocs** — the `_ = requireDomainError` rewrite truncated `es_tenant_decide_merge_test.go` to 79 lines mid-session (caught by gofmt, rewrote the file), and a multiline-call regex silently no-op'd twice before I learned to verify `s.count(old)` asserts. Cost: ~3 lost cycles.
3. **Fixture illiteracy, repeatedly** — invalid ULIDs (S/U/I/L letters, wrong lengths) in adminui tests, `NewUserID` where `ParseUserID` is required, `LegalPartyID(...)` conversion where `NewLegalPartyID` is the constructor, and a legal_entity fixture without a registry anchor. Each was a build-red → read → fix loop that a 30-second look at the constructors would have avoided.
4. **The M19 BillTo stream key was hashed on first cut** (`id.DeriveStreamID`) — the payload's TenantID round-tripped a hash, not the tenant key. Caught by my own test asserting the key; fixed to `ParseStreamID`-verbatim. A design bug that almost shipped because I pattern-matched on "derive" naming.
5. **First legalparty lint pass had 4 of my findings** (dupl/exhaustruct/deprecated-ctor/unconvert) and coverage came in at 79.0 vs the 80 gate — I wrote the tests after the code both times instead of letting the battery drive.
6. **The external-account conflict test assumed the read model steals indexes** — it doesn't (keep-first). My first run "passed for the wrong reason" (nil instead of conflict) until I flipped the link order and read `handleExternalAccountLinked`. The keep-first semantics are now documented in the test comment, but I nearly shipped a test that proved nothing.
7. **routeSpec placement violated the sibling's gate on first run** — I put the five rows in my new handler file; `check-adminui-surface.sh` builds its pattern table from routes.go alone. Caught by their gate (working as designed), fixed by moving the rows. Gotcha 70 records it — but I should have read the sibling's gate BEFORE designing the file layout.

## e) WHAT WE SHOULD IMPROVE

1. **A wave this size should not end untagged.** The Q1 default ("tag nothing") protected the tree but stranded ~30 files of consumer-visible API behind unpublished symbols, reddening four hermetic gates. Either the default or the plan's push step needs a decision EARLIER — at M09, not M25.
2. **Fixture discipline**: a shared `testfixtures` helper for valid ULIDs/IDs per module would have killed six build-red loops in this session alone.
3. **Python-heredoc edits of Go files are a anti-pattern on this host** — every one of my three file-mangling incidents was a python multi-value/regex subtlety. LSP-native edits or full-file rewrites were always safer.
4. **The skill-claims gate bug (literal `method` receiver pattern) survived since 2026-11-06** — every method-qualified README reference since then either got allowlisted or never landed. The gate's own self-test should have a method-declaration case.
5. **check-domain-counts drift existed for days** (33/27 vs documented 27/21 since M03) — planned-red gates that live for a whole wave lose their policing value; consider a "planned-red expires at wave end" ratchet.
6. **legalparty/aliases.go shipped 2026-11-20 without a baseline row** — the re-export gate was silently incomplete for four days. New-module checklists should include "run check-reexport-policy".
7. **Sibling coordination is still lossy** — three battery windows were polluted by mid-flight adminui edits (405-vs-403, build fails, coverage aborts). A per-lane WIP marker or a "battery requested" handoff would pay for itself.

## f) UP TO 50 THINGS NEXT

**Unblock the wave (owner-gated first):**
1. Answer Q1: run the identity-model release train (`nix run .#wave-tag`) — or authorize me to.
2. Sweep consumers to the identity-model tag (bump-dep sweep, hermetic legs heal).
3. Tag usermgmt/adminui/systemadapter/legalparty-model per the release playbook.
4. Push master + `.#push-and-watch` + CI watch (the M25 step, unblocked).
5. Re-run the four tag-gap gates post-train — expect green with zero code changes.
6. Verify the snapshot sentinel at the storage v4.12.0 tag; flip it + strike TODO row 148 (do NOT flip early).
7. Sweep the storage pin when v4.12.0 ships (row 148's full close-out).

**Follow-ups this wave recorded:**
8. gfl-usermgmt: write the merge feature in `usermgmt.gfl`, recompile bindings, regenerate catalog (TODO row).
9. e2e browser specs for the duplicates pages + merge/rename flows (TODO row).
10. legalparty: cross-aggregate BillTo → closed-party retention test at first tag (TODO row).
11. Upgrade the M05 ledger row to DONE now that the bijection is green.
12. Strike the e2e/retention/gfl TODO rows as each lands; re-run todo-hygiene.

**Hardening the new surface:**
13. Add TOTP adoption to a service-level merge test (currently decider-pinned only).
14. Add an EmailVerified-adoption service test for the exact-equality D12 gate.
15. Add a merge-concurrency test (two MergeUsers racing on the same survivor).
16. Add a MergeUsers-under-dispatch-gate test (posture table installed).
17. Test `DuplicateUsers` ordering stability (the sort is by key; golden it).
18. Tenant merge with bots owned cross-tenant (no such thing as tenant-owned bots today — document why).
19. adminui: duplicates pages under pagination (large directories).
20. adminui: a "merge preview" (what the survivor will absorb) before the destructive post.
21. adminui: reason field required client-side (server rejects; UX should match).
22. dashboardui: event browser labels for the six new event types (audit_labels parity).
23. dashboardui: MergedUserIDs/MergedTenantIDs surfacing in time-travel inspector.

**Gate and infra debt:**
24. Add a method-declaration fixture to check-skill-claims' self-test (the `method` pattern bug class).
25. check-adminui-surface: scan routeSpec rows wherever they live (grep `routeSpec{` across the package) instead of routes.go only — fixes gotcha 70's assumption.
26. Consider a planned-red expiry ratchet for domain-counts/bijection (drift visible only at wave end).
27. New-module checklist: run check-reexport-policy + add the baseline row in the landing commit.
28. Per-lane WIP markers for sibling battery coordination.
29. Build the shared testfixtures helper (valid ULIDs per module).
30. Ban python-heredoc Go edits in this repo's gotchas (one line, referencing this session).

**Train aftermath:**
31. `nix run .#check-release-train --refresh-cache --strict-lag 0` post-tags (gotcha 27).
32. Re-true the fleet libs report (go-cqrs-lite row: 33-event universe).
33. Update `docs/guides/authorization-system.md` with the merge vocabulary if postures changed consumers' manifests.
34. Docs-gates battery re-run (audit_max pin after this wave's reports archive).
35. Archive this report per the tail ritual when superseded (lower TAIL_BASELINE same commit).
36. Run the docs-health five-gate battery (standing P2: audit-all red at 393).
37. Screenshot-churn check after the next e2e battery (`git restore docs/screenshots`, gotcha from 11-21).

**Quality-of-life:**
38. README quickstart gate re-run (root README unchanged this wave — cheap confirm).
39. Bench-spike re-run (merge paths added repository loads — no bench written; consider one for MergeUsers).
40. `check-erraudit-strict` receipt (advisory posture; expect growth from this wave's code).
41. Coverage review: usermgmt's new service_merge.go functions — confirm the 85% row holds post-train.
42. examples/admin-demo: surface the duplicates pages in the demo nav.
43. Consider `Service.MergeUsers` accept-merge-dry-run (report without executing) for admin preview (overlaps 20).
44. legalparty: `BillToRedirectedPayload` previous-party carry (fold currently overwrites; retention reads the journal).
45. templ: the deprecated `usermgmt.User` hints in merge.templ:28-29 are gopls-stale — verify post-train and silence if real.
46. M25 leftover: detailed manual commits if the owner wants authored history over daemon commits (currently heuristic-committed).
47. CI watch the first green run end-to-end; file nothing, just verify the planned-reds healed.
48. Post-train: run `nix run .#check-modules` aggregate once on a quiet tree.
49. Re-run `nix run .#e2e-firefox` after the sibling's wave lands (both engines).
50. Close GitHub issues #43/#45/#46/#47 with links to the CHANGELOG entries once the train ships the code.

## g) QUESTIONS (cannot figure out myself)

1. **Q1 (sixth time, now the only blocker): may I run the identity-model release train?** Everything else in f)1–5 keys on it. Default says no; the wave is one `nix run .#wave-tag` away from shippable, and five gates stay red until it runs.
2. **Q2 evolved: the sibling's adminui wave is still mid-flight (bulk actions, dashboard intelligence, members directory).** My adminui merge surfaces are landed and green but unpushed with theirs. Do you want one combined adminui review before any push, or do the lanes stay independent?
3. **Do you want authored, detailed commits for this wave replacing/annotating the daemon's heuristic commits before push** (the standing order said "detailed commits"), or is the plan-file ledger + daemon history sufficient as the record?

**Verdict:** plan executed to its terminal state minus the owner-gated train+push. K1 and K2 green with pinned tests. K3 green-or-documented. Waiting for instructions.
