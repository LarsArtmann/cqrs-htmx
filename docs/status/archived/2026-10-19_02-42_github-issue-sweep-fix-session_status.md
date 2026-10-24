# Status: GitHub issue sweep — review all open issues and fix them

> **ANNOTATED 2026-10-24** — ~~this report~~ **SUPERSEDED** by [`2026-10-19_08-47_issue-sweep-execution-complete_status.md`](2026-10-19_08-47_issue-sweep-execution-complete_status.md) (same series, later in the
> session). The successor re-covers every section with the outcome, so nothing here is a
> unique open item. Strike depth: supersession verdict (round-18 precedent for
> series-internal reports). **ARCHIVED 2026-10-24 (round 22).**


> **Point-in-time snapshot (2026-10-19 02:42 CEST).** Append-only per `docs/status/README.md`; absence of an inline resolution marker = open signal.

## Session intent

User: "Review all open github issues and fix them!" — with READ/UNDERSTAND/RESEARCH/REFLECT + step-by-step execute/verify discipline. 11 open issues: #13, #18, #19, #20, #21, #22, #23, #25, #27, #28, #29, #31.

## Headline

Two issues fixed end-to-end with commits (**#28** request-scoped SSE filter, **#29** TailReader seam), **five issues verified already fixed in tree** (#13, #21, #22, #27, #31 — none closed on GitHub yet), **one issue mid-implementation** (#23 auth audit events — identity-model + usermgmt core written, wiring/registries/train missing), two not started (#18, #19), two awaiting triage comments (#20, #25). **4 files sit UNCOMMITTED in the tree right now** (the #23 work) — daemon-capture risk if the session dies.

## Self-critique (what did I forget / do worse?)

- **Forgot to compile-test the #23 work at each write.** The first `auth_audit.go` draft used APIs I had not verified (`slog` hack import, nonexistent `id.NewUserActorFromUserID`, assumed-error `NewStreamID`). Caught by signature checks before compiling, but the failure mode (write-first, verify-later) is exactly what the repo's read-before-write rule forbids.
- **Forgot the daemon.** #23's identity-model + usermgmt edits have sat uncommitted across the whole tail of the session. One daemon poll = heuristic capture of a HALF-BUILT feature. The micro-ritual (verify → commit at every phase boundary) was applied for #28/#29 and then dropped.
- **Forgot a doc my own comment references.** `auth_audit.go` points readers at `docs/guides/actor-and-audit-trail.md` "see docs/…" for stream placement rationale — that section does not exist yet. Lying comment until written.
- **Bench honesty gap:** the new tail-reader bench showed ~equal allocs vs the scan path on the in-memory store (in-memory events are structs — no decode to save). The O(journal)-vs-O(maxReplay) win materializes only on SQL-backed journals; I documented that in the bench comment but the numbers alone don't demonstrate the win.
- **Design churn mid-flight:** OAuth2Service turned out to be a separate struct without store/bus, forcing a package-level refactor of the audit helper + constructor change — I designed the Service-side helper before mapping ALL emission sites and their receivers.

## a) FULLY DONE

| Item | Status | Status | Evidence |
| --- | --- | --- | --- |
| Skill load (mandatory) + full issue inventory (11 bodies read) | DONE | DONE | session start |
| Triage pass: verify current code state per issue | DONE | DONE | see table below |
| #28 `WithRequestFilter` (request-scoped SSE filter) | DONE | DONE | commit `1909bda8`; transport/serve.go:130-156, 366-384; 4 new tests (per-request scoping, live-during-replay exactly-once, 400 on derivation error, nil→construction fallback); full transport suite green `-race` |
| #29 `TailReader` seam (O(maxReplay) fresh connects) | DONE | DONE | commit `4b679996`; transport/journalsse.go (interface + detection + `eventsAfterTail` + defensive truncation); 4 tests (serve/ignore-on-cursor/error-surface/over-delivery-truncate); bench sub-bench `first-connect-tail-reader` + honest `tailJournal` model; vet + suite green |
| Verified-ALREADY-FIXED: #13 (panic log attrs) | DONE (verify) | DONE (verify) | recovery.go:66-88 cites cqrs-htmx#13 inline |
| Verified-ALREADY-FIXED: #21 (rate-limit keys) | DONE (verify) | DONE (verify) | usermgmt/http.go:123 `KeyExtractorFromClientIP` + usermgmt/CHANGELOG entry |
| Verified-ALREADY-FIXED: #22 (enrollment gating) | DONE (verify) | DONE (verify) | ADR-0055; `requireSessionUserTarget` at webauthn_http.go:47,64 |
| Verified-ALREADY-FIXED: #27 (replay tail semantics) | DONE (verify) | DONE (verify) | commit `0e3f33a0`; paged rolling-tail read journalsse.go:148-187; full-scan tail journalsse.go:225-235; tail tests journalsse_test.go:345,372 |
| Verified-ALREADY-FIXED: #31 (detailed health) | DONE (verify) | DONE (verify) | health/probe.go:54 `NewWithDetailedCheck`; `DetailedHealthRecorder` + `Duration` on the wire probe.go:146-168 |

## b) PARTIALLY DONE

| Item | Status | Status | Remaining |
| --- | --- | --- | --- |
| #23 auth audit journal events | PARTIAL | PARTIAL | identity-model: `AggregateTypeAuth` + `EventLoginSucceeded/LoginFailed/RegistrationClosed` + 3 payload structs + `AuthMethod`/`AuthFailureReason` vocabularies written (constants.go, events.go — UNCOMMITTED). usermgmt: aliases + `allAuthEventTypes` (es_constants.go) + `auth_audit.go` helper (appendAuthAuditEvent/journalLoginSuccess/journalLoginFailure/journalRegistrationClosed — UNCOMMITTED, NOT COMPILED). Missing: call-site wiring (webauthn/totp/oauth2/register), OAuth2Service store+bus fields, AuditLog actions+cases+EventTypes(+Detail), event catalog ×3, systemadapter decoder ×3, tests, CHANGELOGs, AGENTS 21→24, actor-and-audit-trail.md section, mini-train (identity-model tag → usermgmt+systemadapter require bumps) before any push |
| GitHub issue lifecycle | PARTIAL | PARTIAL | zero comments/closes posted yet — all evidence is local; closes should ride a green push |
| Triage #20 (setup demand signal) | PARTIAL | PARTIAL | code evidence gathered (`Config.HealthChecks []cqrshtmx.NamedCheck` = consumer-check aggregation shipped; `DisableLogin` documented at setup/config.go:498-514; health module graduated with NewProbe/Recorder/RecorderChain); middleware-interleave seam + fullstack-wiring.md § DisableLogin docs NOT verified → comment/close verdict unwritten |

## c) NOT STARTED

| Item | Status | Status |
| --- | --- | --- |
| #18 typed PayloadRenderer (`RenderTyped[P]` + per-type registry) | NOT STARTED | NOT STARTED |
| #19 dashboardui `Config.EmptyStateCopy` | NOT STARTED | NOT STARTED |
| #25 comment (demand-gated session lifecycle — leave open) | NOT STARTED | NOT STARTED |
| CHANGELOG entries for this session's transport features | NOT STARTED | NOT STARTED |
| Docs for #28/#29 (frontend-sync/sse guides + README mentions) | NOT STARTED | NOT STARTED |
| `ReadLastN` on usermgmt SQL event store (real-world seam consumer) | NOT STARTED | NOT STARTED |

## d) TOTALLY FUCKED UP

| Item | Status | Status | Detail |
| --- | --- | --- | --- |
| Nothing catastrophic | NONE | NONE | No reverts, no data loss, no broken gates, no daemon capture this session. The one live risk: 4 uncommitted #23 files (identity-model ×2, es_constants.go, auth_audit.go) — a half-built feature one daemon poll away from a heuristic commit |

## e) WHAT WE SHOULD IMPROVE

1. **Phase-boundary discipline for multi-file features:** #28/#29 each ended in an immediate verify+fmt+commit; #23 sprawled across 4 files with zero verification checkpoints. Rule going forward: compile+vet after EVERY file, commit after every coherent sub-phase (identity-model payloads = one commit).
2. **Verify APIs before writing:** signature-check `id`/`event` constructors BEFORE drafting (the NewUserActor/NewStreamID misses cost a full rewrite of auth_audit.go).
3. **Map every emission site + receiver BEFORE designing the helper:** OAuth2Service's separate-struct discovery mid-implementation forced a refactor; a 10-minute call-site survey would have produced the package-level signature first time.
4. **Never ship a comment referencing a doc that doesn't exist:** either write the doc section in the same change or drop the reference.
5. **State bench limitations in the result, not just the comment:** the tail-reader bench does not show the win on in-memory stores; say so in the report/PR, not only in code.
6. **Issue closes should batch with the green push:** verified-fixed evidence (#13/#21/#22/#27/#31) references commits already on origin — those five can close NOW; #28/#29 should close after CI green on their commits.

## f) Next tasks (priority-ordered)

1. Compile + vet the 4 uncommitted #23 files RIGHT NOW (daemon race).
2. Commit identity-model changes as their own commit (phase boundary).
3. Refactor `appendAuthAuditEvent` to package-level `(logger, store, bus, …)` signature.
4. Add `store`/`bus` fields to `OAuth2Service` + `NewOAuth2Service` params (+ caller).
5. Wire webauthn `FinishLogin` success/failure (ceremony_failed, no_credentials at BeginLogin).
6. Wire totp `VerifyTOTP` success/code_invalid.
7. Wire oauth2 `FinishLogin` success + failure reasons (state_invalid, provider_mismatch, token_exchange, no_email) — only where a user exists.
8. Wire `provisionLinkedUser` registration_closed (oauth2) + service_register.go:75 maxUsers site.
9. AuditLog: 3 action constants + `auditActionFor` cases + combined EventTypes (user+auth).
10. AuditEntry: add `Detail` populated from LoginFailedPayload.Reason.
11. es_event_catalog.go: register 3 auth events (Aggregate "Auth").
12. systemadapter/type_decoder.go: 3 `Register` calls; comment 21→24.
13. Tests: journal-on-success, journal-on-failure, registration-closed probe, AuditLog folding, decoder roundtrip.
14. Assert user read model does NOT subscribe auth event types (guard test).
15. identity-model payload roundtrip tests.
16. `nix run .#fmt -- <files>` + `golangci-lint` on identity-model/usermgmt/systemadapter.
17. CHANGELOG: identity-model (3 events + aggregate), usermgmt (journaling + AuditLog + catalog), systemadapter (decoder 21→24).
18. AGENTS.md: "21 event payload structs" → 24; "21-event TypeDecoder" → 24; identity-model description counts.
19. Write docs/guides/actor-and-audit-trail.md § auth audit streams (the comment reference).
20. Decide + implement opt-out knob (ServiceConfig) for auth audit journaling (or document why always-on).
21. Mini-train: CHANGELOG → commit → tag `identity-model/v4.13.0` via verify-tag --push → `--refresh-cache`.
22. bump-dep identity-model in usermgmt + systemadapter go.mods.
23. Hermetic verify: `GOWORK=off go build/test` per changed module (gotcha 30 root-tag-gap).
24. `nix run .#test` + `.#check-modules` full battery.
25. Push; `.#push-and-watch` CI to green.
26. Close #28, #29 with commit evidence.
27. Close #13, #21, #22, #27, #31 with verified-fixed evidence.
28. Comment #23 with design (Auth-stream placement, known-user exclusion rationale, best-effort append).
29. Comment #20 (what graduated: HealthChecks + health module; what remains: middleware seam, DisableLogin docs) — then close or split.
30. Comment #25 (demand-gated; leave open; restate trigger condition).
31. #18: implement `TypedPayloadRenderer[P]` + `RegisterPayloadType` registry in dashboardui/core.
32. #19: implement `Config.EmptyStateCopy` (typed panel keys, upstream defaults).
33. CHANGELOG + docs for #18/#19; close both.
34. Docs: extend docs/guides/frontend-sync.md or sse guide with `WithRequestFilter` + `TailReader` recipes.
35. README feature catalog mentions (auth audit events; request filter).
36. Implement `ReadLastN` on usermgmt SQL event store (`ORDER BY … DESC LIMIT n` reversed) — makes #29 real for SQL consumers.
37. Add `SSEOptions.RequestFilter` conversion test.
38. Verify `event.New` payload-stamp behavior for auth events (codec/encoding parity with es_decide events).
39. Consider rate-limit interplay doc for registration-probe journaling (bounded-write claim).
40. Check dashboardui/adminui event-type displays for graceful unknown-type rendering of Auth events.
41. Roadmap entry: opt-in unknown-user attempt journaling (deliberately excluded; document trigger).
42. Session hygiene: re-check `git log` for daemon heuristic captures during this session.
43. Consider `train-preflight` before the mini-train push.
44. Post-push `verify-remote-tags` (peeled-ref audit).
45. TODO_LIST: record deferred items (SQL ReadLastN if deferred, unknown-attempt opt-in, middleware seam from #20).
46. Evaluate `AuthFailureReason` exhaustiveness vs all actual failure paths (lockout? session-expired?).
47. Check `check-command-bijection`-style gates unaffected (events, not commands — confirm).
48. Update docs/agents-notes.md with the OAuth2Service-separate-struct lesson if it generalizes.
49. Consider golden/fixture impact: event catalog JSON golden tests (if any) need the 3 new entries.
50. Post-train: re-run `nix run .#erraudit-inventory` (new warn sites in auth_audit.go must carry context).

## g) Questions for the user

1. **#23 semantics:** Is the deliberate exclusion set right — login failures journalled ONLY for known users (unknown-user attempts left to request logs/rate-limiters as a DoS-write defense), and audit appends BEST-EFFORT (an audit-store outage never fails a login)? Or do you want fail-closed auth and/or a `ServiceConfig` opt-out (default on vs off)?
2. **Mini-train authority:** Fixing #23 properly REQUIRES tagging identity-model (new symbols → hermetic CI is red until consumer requires bump, gotcha 30). Cut + push `identity-model/v4.13.0` as part of THIS session, or park #23 as a clean local branch/commit and run the train when you say so?
3. **Issue closes:** Close the five verified-already-fixed issues (#13/#21/#22/#27/#31) now with evidence comments (their commits are already on origin/master), or batch all closes until this session's remaining work is pushed and CI is green?
