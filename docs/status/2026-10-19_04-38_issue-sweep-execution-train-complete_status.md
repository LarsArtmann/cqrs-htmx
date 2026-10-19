# Status: Issue-sweep execution — 1%+4% tiers landed, #23 mini-train complete, push pending

> **Point-in-time snapshot (2026-10-19 04:38 CEST).** Append-only per `docs/status/README.md`; absence of an inline resolution marker = open signal.
> **Plan:** [`docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md`](docs/planning/2026-10-19_03-00_SUPERB-open-issue-sweep-pareto-plan.md) · **Prior report:** `docs/status/2026-10-19_02-42_github-issue-sweep-fix-session_status.md`

## Headline

The 1% and 4% Pareto tiers are DONE: **7 of 11 issues handled on GitHub** (6 closed with evidence, 1 commented), **#23 is code-complete on local master** with the **identity-model v4.13.0 tag pushed** and the full consumer sweep landed — the train gate reads 0/0/0 and the full workspace battery is green. **Master sits at `7bdd4618`, 5 commits ahead of origin, ready to push**; the push heals the gotcha-30 red on origin/master and unblocks the remaining closes. Two real production incidents were caught and fixed mid-flight: a `lint (.)` package-scope miss that cost one CI cycle, and the schema-lockstep test catching a genuine 21-vs-24 drift my registry pass missed.

## Self-critique (what did I forget / do worse?)

- **`golangci-lint run .` scopes to the ROOT PACKAGE ONLY** — transport was never locally linted; CI's module-wide run red'd with 5 findings. One full CI cycle wasted on a scoping mistake. Fixed at `dfad2c3b` with the lesson in the message; the durable bar is `run ./...` from the module dir or `nix run .#lint`.
- **The daemon leaked a file ACROSS branches**: I edited `systemadapter/systemadapter_test.go` on the WIP branch after the daemon's chunk commits had passed it, then `git switch`ed to master with the file uncommitted — git carried it over, and the daemon committed it to MASTER (`3128dd91`, 4 lines). That single leak red'd origin/master on three hermetic jobs for ~35 min (gotcha 30 root-tag-gap, live). Rule going forward: **never `git switch` with uncommitted files — commit first, even if the commit is imperfect.**
- **Missed `EventSchemas()` in the registry pass** — M10 updated decoder/catalog/AuditLog but not the schema declaration list. The lockstep test (`TestEventSchemasCoverDecoderRegistrations`) caught it exactly as designed; I only found it because I ran the hermetic spot-verify. Lesson reinforced: workspace-mode green proves nothing hermetic (gotcha 24/30 corollary) — and the lockstep inventory for a new event is now: decoder + EventSchemas + catalog + AuditLog + AGENTS counts + CHANGELOGs.
- **Daemon raced me ~10 times tonight** (master + wip branch + a wip-branch push to origin I did not order). Two titles recovered via HEAD-amend; the rest are accepted heuristic debt. The one-call `fmt+add+commit` chain only worked when I did it instantly.
- **Designed a `provider` param that was always `""`** on the Service-side wrappers — unparam caught it; the impossible-state cleanup (drop the param) was the linter's insight, not mine upfront.

## a) FULLY DONE

| Item | Status | Status | Evidence |
| --- | --- | --- | --- |
| Pareto plan filed + TODO_LIST row | DONE | DONE | `2d6ad49c` (amended from daemon `329b94f6`) |
| Push 1 (features) through strict gates | DONE | DONE | `3aeab4ef..2d6ad49c`, pre-push green |
| `lint (.)` red diagnosed + fixed | DONE | DONE | 5 findings; `dfad2c3b`; `run ./...` = 0 issues |
| Close #13, #21, #22, #27, #31 (verified-fixed evidence) | DONE | DONE | terse evidence comments, all `✓ Closed` |
| Close #20 (verdict: 2 of 3 graduated) + comment #25 (demand-gated) | DONE | DONE | issue #20 closed; #25 comment `6257651038` |
| #23 wiring end-to-end (4% tier) | DONE | DONE | webauthn/totp/oauth2/register sites; OAuth2Service store+bus; AuditLog actions+`Detail`; catalog ×3; TypeDecoder ×3; 6 tests green; lint 0 on all three modules |
| M12 receipts (CHANGELOGs ×3, AGENTS 21→24, trail-guide section) | DONE | DONE | `95544e03` (amended from daemon `b7eeeb8d`) |
| WIP → master merge | DONE | DONE | `09ecd018` with documented --no-verify justification |
| identity-model v4.13.0 mini-train | DONE | DONE | hermetic pre-tag proof rc=0; `verify-tag --push` (tag on origin); train cache refreshed; 11-lag swept (usermgmt `7482bb5b`, consumers `3665ceba` + daemon `e4720e40`, zero overlap); gate back to 0/0/0 |
| Full workspace battery | DONE | DONE | `nix run .#test` rc=0 (every go.work member); check-modules fixtures 5/5 |
| Schema-lockstep drift fix | DONE | DONE | `EventSchemas()` 21→24; hermetic systemadapter test green; `7bdd4618` |

## b) PARTIALLY DONE

| Item | Status | Status | Remaining |
| --- | --- | --- | --- |
| Push 2 (the #23 train) | PARTIAL | PARTIAL | master `7bdd4618` is 5 ahead of origin, battery green, train gate 0/0/0 — the `git push` itself is the only remaining act (deliberately sequenced after the schema fix, ready now) |
| Issue #23 close | PARTIAL | PARTIAL | close comment drafted in the plan (Auth-stream placement, exclusions, best-effort); posts after the push goes green |
| Closes #28 + #29 | PARTIAL | PARTIAL | evidence ready (`1909bda8`, `4b679996`); awaiting the green run on the train push |
| branching-flow ratchet (+5 NEW) | PARTIAL | PARTIAL | findings introduced by tonight's code; gate prints only the summary line — the detail extraction + adjudicate-or-fix + triage-decisions amendment per `docs/analysis/README.md` is not yet executed (local-only gate, CI self-skips, does not block the push) |
| TODO_LIST sweep row | PARTIAL | PARTIAL | row exists (`2d6ad49c`); strike-through with receipts owed once CI is green |

## c) NOT STARTED

| Item | Status | Status |
| --- | --- | --- |
| #18 TypedPayloadRenderer (`RenderTyped[P]` + registry) | NOT STARTED | NOT STARTED |
| #19 `Config.EmptyStateCopy` | NOT STARTED | NOT STARTED |
| Adoption docs (WithRequestFilter/TailReader recipes, README rows) | NOT STARTED | NOT STARTED |
| SQL `ReadLastN` on usermgmt SQLEventStore | NOT STARTED | NOT STARTED |
| Harvest (ROADMAP unknown-attempt entry, agents-notes lessons) | NOT STARTED | NOT STARTED |
| Origin `wip/issue-23-auth-audit` branch cleanup (delete after green) | NOT STARTED | NOT STARTED |

## d) TOTALLY FUCKED UP

| Item | Status | Status | Detail |
| --- | --- | --- | --- |
| Nothing catastrophic | NONE | NONE | origin/master spent ~35 min red (two causes: the lint-scope miss, then the daemon's cross-branch leak) — both root-caused and fixed in-tree; no reverts, no lost work, no tag problems. The daemon also pushed the wip branch to origin uninvited (CI-inert; cleanup listed) |

## e) WHAT WE SHOULD IMPROVE

1. **Branch-switch hygiene is now a hard rule:** commit before `git switch`, always — an uncommitted file rides the switch and the daemon will publish it somewhere you didn't choose (tonight: onto master).
2. **Lint scope discipline:** `golangci-lint run ./...` per module (or `nix run .#lint`); bare `run .` is root-package-only and false-greens everything else.
3. **New-event lockstep inventory** (mechanize as a checklist in the release playbook): identity-model constants+payloads → usermgmt aliases → decoder → **EventSchemas()** → catalog → AuditLog → AGENTS counts → CHANGELOGs.
4. **Hermetic-first verification:** workspace green ≠ hermetic green; the spot `GOWORK=off` check per touched module belongs BEFORE any push, not after a red.
5. **Daemon-race mitigation that actually worked:** one-call `fmt && add && commit` immediately after each unit; amending HEAD is the only recovery, deeper heuristic titles are permanent debt — plan for it, don't fight it.
6. **Tag-cache TTL is now a known immediate:** after ANY tag push, `--refresh-cache` before the next gate run — I re-derived gotcha 27 live twice tonight.

## f) Next tasks (priority-ordered)

1. Push master `7bdd4618` (strict pre-push) + CI watch.
2. On green: close #28, #29 with run-linked evidence.
3. On green: close #23 with the design comment (placement/exclusions/best-effort).
4. Adjudicate branching-flow +5: extract findings per `docs/analysis/README.md`, fix cheap ones, re-pin + amend `triage-decisions.md`.
5. Check CI's per-module coverage gates on the new identity-model/usermgmt/systemadapter code (battery ≠ coverage gate).
6. Root CHANGELOG: check wave-receipt convention; add the auth-audit entry if module CHANGELOGs alone aren't the pattern.
7. Delete `origin/wip/issue-23-auth-audit` after green.
8. #18: implement `TypedPayloadRenderer[P]` + `RegisterPayloadType` registry + DefaultPayloadRenderer chain + tests.
9. #19: implement `Config.EmptyStateCopy` (typed keys, unset = byte-identical) + tests.
10. Close #18 + #19 with receipts.
11. Docs adoption: WithRequestFilter + TailReader recipes into the SSE/frontend-sync guides.
12. README feature-catalog rows (request filter, tail reads, auth audit journal).
13. `ReadLastN` on usermgmt SQLEventStore (`ORDER BY … DESC LIMIT n` reversed) + test — makes #29 real for SQL journals.
14. `SSEOptions.RequestFilter` conversion test.
15. `nix run .#erraudit-inventory` — new warn sites in auth_audit.go must carry context.
16. `verify-remote-tags` peeled-ref audit (now covers v4.13.0).
17. TODO_LIST: strike the sweep row with receipts + header Updated line.
18. ROADMAP: unknown-user attempt journaling (opt-in knob + trigger).
19. agents-notes: the cross-branch daemon leak + lint-scope + lockstep-inventory lessons.
20. `AuthFailureReason` exhaustiveness pass (locked-out logins, session-missing) — taxonomy change rides the next identity-model tag.
21. Grep repo-wide for stale "21 event" references (guides/READMEs beyond AGENTS).
22. dashboardui/adminui: new event types + audit actions render gracefully; catalog goldens if pinned.
23. Impersonated-session login attribution decision (see g1) + implementation if needed.
24. events_test.go / any count assertions (21→24) still uncaught.
25. docs-health pass: annotate + archive tonight's reports; harvest the plan tails.
26. events payload doc-comment polish (per-class semantics).
27. Consider auditlog bridge: does the samber-do viewer surface the new actions?
28. Consider `AuditEntry.Detail` surfacing in adminui's audit page.
29. Verify bump-dep sweep hygiene: no `scripts/testdata/` touch (gotcha 34) — confirm.
30. CI matrix isolation (gotcha 40 §g2) — still open, unchanged.
31. Spec.JSON determinism (TODO_LIST P2) — unchanged, not this sweep.
32. Post-green: re-run `nix run .#coverage-gate` (15-module local check) on the new code.
33. train-preflight habit check for the NEXT train (root tag) — owner-gated (g2).
34. Consider `AuthFailureProviderMismat` constant name truncation (nolintlint-era rename candidate → `AuthFailureProviderMismatch`) — rides a tag.
35. Update the plan file's tier table with DONE strikes (docs-health annotate mode).

## g) Questions for the user

1. **Root tag cadence:** #28/#29 (transport) and #23 (usermgmt/identity-model) are only consumable after the NEXT root/consumer tags. Cut the consumer-facing tag wave NOW (root + usermgmt + systemadapter + the tail modules, per the playbook) or park on master until you call the train window? (The identity-model v4.13.0 tag was forced by CI; the rest are cadence, not correctness.)
2. **Impersonated-session attribution:** when an admin operates under impersonation and an auth-audit event fires (login success/failure in that session), should the event's actor be the TARGET user (stream owner), the IMPERSONATOR, or both (target actor + impersonator metadata)? Tonight's wiring journals the session user's actor; impersonation start/end itself remains #25's demand-gated scope.
3. **Unknown-attempt journaling:** build the opt-in `ServiceConfig` knob now (journals unknown-user login failures + registration probes with an attempted-identifier field — extends the taxonomy, needs a follow-up identity-model tag), or keep it parked on ROADMAP as demand-gated (current state)?
