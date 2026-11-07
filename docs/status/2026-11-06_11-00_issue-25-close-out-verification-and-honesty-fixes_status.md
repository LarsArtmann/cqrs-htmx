# Status — Issue #25 close-out: the work was already there, so I verified it, fixed two lies, and shipped the missing docs

**Date:** 2026-11-06, 10:20–11:05 CET
**Repo:** `/home/lars/projects/cqrs-htmx`, branch `master`, shared working tree (a sibling docs-health/huma session was committing alongside this one).
**Scope of this session:** [cqrs-htmx#25](https://github.com/LarsArtmann/cqrs-htmx/issues/25) — "Event-sourced session lifecycle (demand-gated) — impersonation forensics gap". I was asked to read it, break it down, execute and verify until done.
**One-sentence verdict:** the XS scope from the issue was already implemented on `master` (ADR 0060, `usermgmt/session_audit.go`) but had never been verified end-to-end, carried two honesty defects in the emission code, was undocumented in the places a consumer looks, and the issue was still open — all four are now closed; what remains red is the owner-gated release train and three siblings' in-flight lanes, none of them session-audit code.

---

## a) FULLY DONE

### a1. Established what already existed instead of re-implementing it

The issue's two checkboxes look like a fresh task; `git log` + `docs/status/2026-11-05_23-57_both-issues-implemented-extranav-and-session-lifecycle-audit_status.md` + `docs/adr/0060-session-lifecycle-audit-events.md` proved the XS half had landed on 2026-11-06. I then audited it against the issue text, not the status report's claims:

| Issue requirement                                        | Where it lives                                                   | Verdict                                       |
| -------------------------------------------------------- | ---------------------------------------------------------------- | --------------------------------------------- |
| Emit session create/revoke audit events                  | `usermgmt/session_audit.go` (`auditingSessionStore`)             | present, one chokepoint                       |
| Impersonation start/end with `By`/`Reason`/`At`          | `usermgmt/auth_audit.go` `sessionStarted`/`sessionEnded`         | present                                       |
| Keep `SessionStore` as-is                                | interface unchanged; decorator wraps in `service_factory.go:183` | present, no consumer break                    |
| Full `sessionlifecycle` module stays demand-gated        | ADR 0060 §Demand gate                                            | correctly NOT built                           |
| "Who impersonated user X between T1 and T2" after expiry | `TestSessionAudit_ImpersonationForensicsSurviveSessionDeletion`  | passes, plus new end-to-end login-path proofs |

Wiring verified rather than assumed: exactly one wrap site (`service_factory.go:183`, after the struct literal, before `NewOAuth2Service`, so `Service` and `OAuth2Service` share the decorated store); no exported session-store accessor exists (`rg 'func \(s \*Service\) (Sessions|SessionStore)'` → nothing), so a consumer cannot bypass the audit path through the public API; the only create/delete sites are `service_register.go`, `service_login.go`, `service_impersonation.go`, `service_misc.go` and they all go through `svc.sessions`.

### a2. Two honesty defects in the emission code, fixed

Both were "the name/comment says one thing, the code does another" defects — not crashes.

1. **`SessionEnded` attributed the wrong human.** For an impersonation session the event actor was the TARGET (`session.ActorID`), while `SessionStarted` already used the impersonator (`imp.By`) as its actor. One query for "everything admin Y did" therefore returned half the window, and `AuditLog`/adminui showed the victim as the actor of the end event. `usermgmt/auth_audit.go:168-195` now mirrors the start path (`actor = imp.By` for impersonation sessions); the payload still carries both (`ActorID` = target, `Impersonator` = admin).
2. **A branch that cannot be reached, documented as if it can.** `sessionEnded(ctx, session, token, cause)` tolerated `session == nil` "because the caller could not snapshot it" — but the decorator returns early on a failed snapshot, so the nil path was dead and the token parameter duplicated `session.Token`. Signature is now `sessionEnded(ctx, session, cause)`; `usermgmt/session_audit.go:95` passes the snapshot it already has. The comment now states the real invariant (the decorator only journals ends it could attribute, which is also the noise guard for unknown-token logouts).

### a3. The two login paths the issue's chokepoint claim covered were never asserted

The status report claimed "register, WebAuthn/OAuth2 login, impersonation begin/end, logout, lazy expiry, user-deletion revocation" are all journalled. Only register, impersonation, logout, expiry and user-deletion had tests; the OAuth2 and WebAuthn login paths were covered "structurally, by the shared helper". Structural coverage is not proof, so `usermgmt/session_audit_test.go` gained:

- `TestSessionAudit_OAuth2LoginEmitsSessionStarted` — full begin→callback HTTP flow through `AuthHandler`, asserts exactly one `SessionStarted`, no impersonation fields, non-empty token hash.
- `TestSessionAudit_WebAuthnLoginEmitsSessionStarted` — full passkey ceremony (`BeginLogin` + `FinishLogin`), asserts the login session carries a token hash distinct from the registration session's.
- Event-actor assertions in `TestSessionAudit_ImpersonationForensicsSurviveSessionDeletion` for BOTH the start and the end event (these fail on the old attribution — a11 mutation-check by construction).

Session-audit test count: 9 → 11.

### a4. Documentation a consumer actually needs, added or trued

- **`docs/guides/actor-and-audit-trail.md`** — new "Answering the question" subsection: the Go journal read (`event.Journal.ReadAll`, with the trap that a user stream never contains these events), the SQL query (correct `events` columns, `event_type = 'SessionStarted'`, `payload_encoding = 'json'` guard, Postgres spelling), and the live-view pointers (AuditLog/adminui, dashboardui events panel). Also states the actor rule fixed in a2.
- **`FEATURES.md`** — three stale rows trued: event catalog 25 → 27 types (5 auth-audit incl. `SessionStarted`/`SessionEnded`), event payloads 25 → 27 structs (Auth 3 → 5), audit log 14 → 16 action types (counted in `usermgmt/audit_log.go`). The Session Management row now names the audit decorator, the token-hash rule and ADR 0060.
- **root `README.md`** — the actor/audit feature bullet now names the session lifecycle pair and links ADR 0060 alongside the guide.
- **`usermgmt/CHANGELOG.md` + `identity-model/CHANGELOG.md`** — neither module had an entry for the events they shipped. Both now carry an `## [Unreleased] / Added` entry with mechanism, the token-safety rule, the verification test name, and the sweeper limitation.
- **`ROADMAP.md`** — the full `sessionlifecycle` module now has a written demand gate in the session/expiry decision list (trigger: live-session projection or per-user concurrency limits), pointing at ADR 0060 §Demand gate and `usermgmt/session_audit.go`.

### a5. Issue #25 closed with evidence

- Comment posted (`#issuecomment-6682159520`) — XS scope, the chokepoint, the token-hash rule, the criterion test, the ADR link, and the honest "unreleased until identity-model is tagged".
- Body's first checkbox ticked in place with the ADR link; the `sessionlifecycle` checkbox deliberately left open (it is the demand gate, not work in flight).
- Closed as completed. Draft kept at `docs/drafts/2026-11-06_issue-25-closing-comment.md` (github-voice rule: never `/tmp`), voice checker `--kind comment` = 0 FAIL / 0 WARN.

### a6. Verification battery actually run (not claimed)

| Leg                                         | Command                                                           | Result                                                           |
| ------------------------------------------- | ----------------------------------------------------------------- | ---------------------------------------------------------------- |
| identity-model                              | `go test ./identity-model/` + `-race`                             | ok 0.014s / ok 1.178s                                            |
| usermgmt full suite                         | `go test ./usermgmt/` + `-race`                                   | ok 19.3s / ok 22.8s                                              |
| session-audit subset under `-race`          | `go test -race -run TestSessionAudit ./usermgmt/`                 | ok 1.09s (11 tests)                                              |
| systemadapter                               | `go test ./systemadapter/`                                        | ok 0.84s / ok 2.19s (`-race`)                                    |
| adminui (downstream consumer)               | `go test ./adminui/`                                              | ok 7.35s                                                         |
| identity-model lint                         | `golangci-lint run`                                               | 0 issues                                                         |
| usermgmt lint, my files only                | `golangci-lint run` filtered                                       | 0 findings in `session_audit*.go` / `auth_audit.go` (module: 31, all pre-existing — see b2) |
| docs links                                  | `check-docs-links.sh`                                             | 1422 links OK                                                    |
| corpus ratchet                              | `check-docs-corpus-budget.sh --ratchet`                           | within (51,552 / 344 files)                                      |
| TODO hygiene                                | `check-todo-hygiene.sh`                                           | clean (76 rows, 0 dead, preamble 495)                            |
| skill/README claims                         | `check-readme-splitbrain.sh` (incl. skill-claims)                 | 104 symbols resolve                                              |
| formatting                                  | `nix run .#fmt -- <my 10 paths>`                                  | 8 files, 1 changed (FEATURES table reflow)                       |
| hermetic usermgmt build                     | `GOWORK=off go build ./...` in `usermgmt/`                        | expected RED — see b1 (tag lag, not code)                        |

---

## b) PARTIALLY DONE

### b1. The releases are still unpublished, so the hermetic gates stay red

Verified honestly rather than hand-waved:

- `identity-model`'s newest tag is **v4.15.0**, which predates `SessionStarted`/`SessionEnded`; there is no `identity-model/v4.16.0`.
- `GOWORK=off go build ./...` in `usermgmt/` fails on `undefined: identitymodel.EventSessionStarted`, `SessionEndCause`, `SessionStartedPayload`, `SessionEndedPayload` **and** `cqrshtmx.DispatchPosture` — two independent tag lags (session events + ADR-0063's dispatch-boundary API), the gotcha-30 root-tag-gap class.
- The published artifacts are NOT broken: `usermgmt/v4.19.0` does not contain `session_audit.go` (`git cat-file -e usermgmt/v4.19.0:usermgmt/session_audit.go` → absent), so only HEAD is un-publishable. That distinction matters for anybody triaging CI: this is a train gap, not a poisoned tag.
- The train itself is owner-gated (TODO_LIST P1 row, which already lists the session events). I did not tag or push: pushing tags is explicitly out of scope for me.

### b2. usermgmt's module lint is red for reasons that are not this work

`golangci-lint run` in `usermgmt/` reports 31 issues: 27 `staticcheck` SA1019 (consumption of the now-`// Deprecated:` root aliases — `cqrshtmx.PayloadField`, `EventCatalog`, `MapError`, …), 2 `gci` (`command_postures.go`, `es_event_catalog.go`), 2 `gochecknoglobals`. All of them are W0/ADR-0063 fallout owned by the direct-imports lane; zero sit in the files this session touched. I did not "fix" them: W1 will rewrite those call sites, and touching them here would collide with the sibling session.

### b3. Corpus baseline bumped without a ledger sentence

`docs/CORPUS_BASELINE` now holds the measured truth (51,552 lines / 344 files), which includes ~900 lines of sibling growth that landed while I was working (their 09-38 and 10-32 reports, the `huma/` module docs). The repo's own ritual wants a ledger sentence in `docs/status/README.md`; that file's ledger is one giant line the sibling docs-health pass was actively rewriting, so I deliberately left it alone rather than risk clobbering a concurrent human-quality edit (gotcha 47/53). This report's own ledger sentence is the debt.

### b4. The skill was not touched

`.agents/skills/cqrs-htmx/SKILL.md` still does not mention session lifecycle events. The guide is the canonical source and the sibling session was editing that skill for W0, so I left it. It is a real (small) discoverability gap for consumers who only read the skill.

---

## c) NOT STARTED

1. **The release wave** (identity-model tag → root tag → usermgmt/systemadapter/consumers sweep). Owner-gated push; prepared only as the TODO row's ordered plan.
2. **The full `sessionlifecycle` module** — correctly not built (no demand signal; ADR 0060 and the new ROADMAP row both record the trigger).
3. **Exact end events for swept sessions.** ADR 0060 §Limitations: both the SQL `EvictExpired` sweeper and the in-memory one bulk-delete outside the decorated `SessionStore` interface, so a session that expires unused and is swept before any lazy check gets `SessionStarted` but no `SessionEnded`. Closing it means widening the store interface (or moving the sweeper behind the service) — deferred, still documented in three places.
4. **Auth-method on `SessionStarted`** (webauthn/oauth2/totp). `LoginSucceeded` carries it; the session event deliberately does not. Not asked for.
5. **A benchmark of the decorator's extra `Find` per `Delete`** — logout now reads the session before deleting it (needed to attribute the end). Unmeasured; the bench-spike gate does not cover the logout path.
6. **SQL-dialect proof of the decorator's unknown-token behaviour.** The decorator's find-first guard is dialect-independent in theory; only the in-memory semantics are pinned by test today.
7. **Session-audit events in adminui's audit *labels* for the two new actions** — verified present (`adminui/audit_labels.go:40,56`), but no rendered-page golden asserts them.
8. **Tail-tail drainage** (15 live reports vs budget 3) and the `docs-gates audit-all` red — the docs-health lane's, not this one's.

---

## d) TOTALLY FUCKED UP

1. **I read two Go files with `bash` (`cat`/`sed`) and then tried `edit` on them** — rejected twice with "you must read the file before editing it". The rule is in my instructions and the file was small; this was sloppiness, not bad luck.
2. **One no-op edit** on the draft (old string and new string differed only by a trailing newline). Pure waste.
3. **Shipped a duplicated phrase into the draft** ("are emitted are emitted") — I "remembered" writing it clean instead of re-reading. Caught only because I re-read the file before posting; the voice checker does not catch grammar.
4. **Ignored a documented trap: `rg -rn`** was blocked by the host hook (the third-strike trap AGENTS/gotcha 3 spells out verbatim). Cost one round trip and a scolding from my own tooling.
5. **Ran `go test ./integration_test/` blind** and burned a run on a panic in `go-health v0.10.0`'s `RegisterRoutes` (`health_probe_test.go:109`) — a dependency bump from the sibling's commit `ffa50a92`, not my change. I should have checked the module's known-red status first; the repo has a status report saying which legs are red.
6. **Chased a moving ratchet.** I bumped `docs/CORPUS_BASELINE` to 51,552 while a sibling was actively adding reports, then twice reported a tail-ratchet red (14, then 15) that the sibling moved under me. Correct end state, noise in the middle.
7. **Started from "this issue is unimplemented"** and spent the first several tool calls exploring before checking `docs/adr/` + `docs/status/` for the issue number. The check is one `rg -n "#25" docs/`; doing it first would have skipped the whole reconnaissance-by-grep phase.
8. **Nearly edited `docs/status/README.md`'s ledger line** while another session was rewriting it. I caught myself on the diff (the line had already changed under me once), but the intent was wrong for a shared tree.

---

## e) WHAT WE SHOULD IMPROVE

- **Session start, first move: "is this issue already implemented?"** `rg -n "#${ISSUE}" docs/adr docs/status AGENTS.md CHANGELOG.md` before any design. Two prior reports existed; both said "code green, wave blocked". Reading them first is cheaper than reconstructing the state from source.
- **Audit-event actor semantics deserve a table, not prose.** There are now six event families where "who is the actor" is a judgement call (command events, `LoginSucceeded`, `SessionStarted`, `SessionEnded`, revocation, impersonation end). One table in `docs/guides/actor-and-audit-trail.md` would have made a2 a 30-second review instead of a discovery.
- **Count-pinned doc claims keep rotting** (this session trued three rows in `FEATURES.md` that said 25/3/14 against a code truth of 27/5/16). Either derive them in the doc from a single constant, or gate them: a `check-doc-counts` that asserts `len(eventTypeRegistry)` / `len(auditActions)` against the numbers quoted in `FEATURES.md` and `usermgmt/README.md`.
- **Deprecation noise is drowning the lint signal.** 27 of usermgmt's 31 findings are SA1019 on the ADR-0062 aliases; until W1 lands, module lint cannot tell me whether MY change introduced anything. Landing W1 (or a temporary `SA1019` exclusion scoped to the alias consumers) restores the gate's usefulness.
- **A shared worktree needs a written protocol, not two sessions being careful.** Both of us bumped `CORPUS_BASELINE`/`TAIL_BASELINE` in the same hour; the ledger's own text now records "sibling growth measured after the freeze" repeatedly. The repo already says "use `git worktree add`" (gotcha 47) — this session is the third data point for why.
- **Status-report growth vs. the ratchet.** Every report costs a corpus bump and a tail bump; the tail is at 15 against a budget of 3. The archive pass is overdue and is the only way this ritual stops costing a bump per session.
- **Test the chokepoint at every entry, not once.** The "one chokepoint" claim is what let three login paths share zero direct tests. A table-driven test over the entry points (register/OAuth2/WebAuthn/impersonation) would have made the two new tests one row each.

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

**Close this session's edges (blocking the "done" claim):**

~~1. **Cut the release wave, in dependency order**: `identity-model v4.16.0` (session events) → root (`DispatchPosture`, `DropStats` — already pending) → `usermgmt` → `systemadapter`/`adminui`/`dashboardui`/`setup` → consumer sweep. Verify: `nix run .#wave-tag -- identity-model v4.16.0` (dry-run first), then `nix run .#check-release-train --refresh-cache --strict-lag 0`, then `GOWORK=off go build ./...` in `usermgmt/` and `systemadapter/`.~~ verified — wave cut 2026-11-07: all tags verified on origin (identity-model v4.17.0, root v4.17.0 then v4.18.0, usermgmt v4.20.0, systemadapter v4.13.0, adminui/dashboardui/setup v4.17.0; TODO_LIST header)
~~2. Write the corpus ledger sentence for this report in `docs/status/README.md` (the sibling's ledger line is stable now). Verify: `bash scripts/checks/check-docs-corpus-budget.sh --ratchet`.~~ verified — the ledger sentence exists: the 10-54 pass ledgered this report (+209 lines, docs/status/README.md corpus ledger, baselines pinned)
~~3. Raise `TAIL_BASELINE` to the live count in the same change as this report, or archive two settled 2026-11-05 reports instead. Verify: `bash scripts/checks/check-docs-tail-budget.sh --ratchet`.~~ verified — TAIL_BASELINE was raised 15 to 16 same-change with this report (ledger); it stands at 15 after the round-24/25 archive passes archived ten reports

**Session-audit follow-on (same code, small):**

4. Add the two events to `.agents/skills/cqrs-htmx/SKILL.md` (one sentence + a pointer to the guide's "Answering the question"). Verify: `bash scripts/checks/check-readme-splitbrain.sh`.
5. Add an `actor-attribution` table to `docs/guides/actor-and-audit-trail.md` covering the six event families (see e). Verify: docs links gate.
6. Extend `integration_test/actor_attribution_test.go` with a session-lifecycle case: begin → end impersonation, assert the AuditLog entry actor is the admin for BOTH entries. Verify: `go test ./integration_test/ -run TestActor`.
7. Adminui rendered-page golden asserting `session_started`/`session_ended` labels and the cause as detail (`adminui/audit_labels.go:40,56`). Verify: `go test ./adminui/`.
8. Decide the sweeper gap: either widen `SessionStore` with a cause-carrying bulk-evict, or move `StartCleanupSweeper` behind the service so `EvictExpired` crosses the decorator. ADR 0060 §Limitations names both; pick one and write the ADR amendment. Verify: a test that a swept-unused session produces a `SessionEnded{cause: expired}`.
9. In-memory `EvictExpired` parity with the SQL one (it is currently unreachable from the service, so the gap is symmetrical — say so in the ADR if 8 is deferred).
10. Benchmark logout: `BenchmarkLogout` with/without the decorator's extra `Find`, `b.Loop()` + `ReportAllocs`, then decide whether the read is worth a snapshot-free path (e.g. `FindForDeletion`). Verify: `nix run .#bench-spike`.
11. SQL-dialect contract test: decorator `Delete` on a missing token journals nothing under sqlite + postgres dialects. Verify: extend `usermgmt/session_store_contract_test.go`.
12. Add `AuthMethod` to `SessionStartedPayload` only if a consumer asks — record the omission as a decision, not a TODO.

**Doc-count rot (class fix):**

13. `check-doc-counts` gate: assert the numbers quoted in `FEATURES.md` + `usermgmt/README.md` (27 events / 21 commands / 16 audit actions) against the code. Verify: fixture self-test with one planted drift.
14. Replace the hand-written counts with code-derived ones at doc generation time (or a single `docs/generated/counts.md` referenced by the others).
15. ~~Grep the living docs for other `N event` / `N action` / `N command` claims and true them in the same pass (start with `identity-model/README.md`, `docs/guides/event-catalog-guide.md`, `docs/PERMISSIONS.md`).~~ verified — counts trued: the event-catalog guide reads Registered Events (27 total) and the ledger records the 25-to-27 trues across AGENTS, the root README, the usermgmt README and the guide table

**Verification hygiene:**

16. Add a `session-audit` entry-point matrix test (one row per login path) replacing the two ad-hoc tests — table-driven, same coverage, less prose.
17. Pin the "one chokepoint" invariant mechanically: a grep gate that `sessions.Create`/`sessions.Delete` appear only in the audited call sites (the pattern `check-session-route-wrappers` uses).
18. Add the decorator to the coverage gate's per-file list if usermgmt's threshold was carried by unrelated files.

**Release/train glue (owner-gated, but prep is free):**

19. Put the tag order for session events into `docs/guides/release-playbook.md` as the worked example of a two-module API dependency (identity-model → usermgmt).
20. After the wave: `nix run .#verify-remote-tags` and `nix run .#push-and-watch`; close the TODO P1 row with the CI run id.
~~21. Check `usermgmt/CHANGELOG.md`'s new entry lands under the version that actually ships (currently `## [Unreleased]` — move it to the tag's section at wave time; same for `identity-model`).~~ verified — both entries landed under shipped sections: usermgmt v4.20.0 (2026-11-07) and identity-model v4.17.0 (2026-11-07) carry the session-lifecycle entries

**Sibling-lane items this session noticed but must not touch:**

~~22. `integration_test` panic: `go-health v0.10.0` `Probe.RegisterRoutes` (`health_probe_test.go:109`) — the health bridge needs updating to the new API (owner: appkit/go-health sweep session).~~ verified — green this session: the Health probe tests pass, including TestHealthProbe_StartAndRoutesE2E which mounts Probe.RegisterRoutes (health_probe_test.go:110); no panic at go-health v0.10.0
~~23. usermgmt's 31 lint findings (27 SA1019 + 2 gci + 2 gochecknoglobals) — W1 of the direct-imports plan.~~ resolved — routed to TODO_LIST.md:22 direct-imports row (W1 rides the v5 window)
24. `setup` lint `nestif` in `bundle.go:226` (DropStats shutdown block) — already in the TODO P1 row.
~~25. `docs-gates audit-all` red (393 unharvested items vs pinned 322) — needs the harvest-or-archive decision the sibling's 09-38 report lays out.~~ resolved — routed to TODO_LIST.md:46 audit-all row (the harvest-or-re-pin decision)
~~26. Tail at 15 vs budget 3 — archive the settled 2026-11-05/06 set (`nix run .#archive-status`).~~ resolved — routed to TODO_LIST.md:21 plan-of-record row (the archive pass to a tail of 3)
~~27. The two rule-F inline-code findings the sibling reported fixed — re-run `docs-gates annotate .` to confirm nothing regressed.~~ verified — annotate clean: the 09:30 pass fixed the five rule-F findings (146 files, 555 struck, 0 findings) and the round-24 battery stayed clean (TODO_LIST row 46)

**Broader follow-ups worth scheduling:**

28. `examples/` snippet showing the forensics query (a consumer-facing proof that the journal answers the question) — pairs with the new guide section.
29. `docs/guides/session-forensics.md` only if the guide section grows past ~60 lines; right now consolidation was the right call.
30. Audit the other "structural coverage" claims in status reports: a test named after a chokepoint claim should exercise every entry, or the claim is prose.
31. Review `Service.EndImpersonation`'s authorization: today anyone holding the token can end it; the audit trail now records who *was* impersonated but not who *ended* it. Decide whether the end event needs its own actor (the caller) rather than the impersonator — this is a real design question the unified actor choice surfaced.
32. Consider a `SessionEndCause` for "admin revoked one session" (today it journals `revoked` only for the whole-user sweep; a single admin-initiated kill has no distinct class).
33. Document the `payload_encoding = 'json'` guard's scope: CBOR-payload events (ADR-0059) will not be readable by the doc's SQL without a decoder — say so or add a `json_valid(payload)` variant.
34. Check whether `AuditLog`'s 10k window can evict a session event before a compliance query runs; if yes, the guide should say "journal, not AuditLog" more loudly (it does, once).
35. Add a `dashboardui` filter preset for the auth stream so the events panel can show only session lifecycle (today: manual type filter).
36. Re-verify the issue body's remaining unchecked box reads as "demand gate", not "open work" for the next reader (it does now, but a future scan will read it as a TODO).
37. Keep `docs/drafts/2026-11-06_issue-25-closing-comment.md` as frozen history or delete it after the issue closes — pick a convention (the drafts dir has both shapes).
38. Capture this session's "read the status report before the code" lesson in `docs/runbooks/session-start.md` (one bullet, with the `rg -n "#<issue>" docs/` command).
39. If the decorator gains a third responsibility (e.g. sweeper emission), split `auditingSessionStore` into `sessionAuditEmitter` + store decorator before it becomes a god object.
40. Re-check the `SessionStarted` payload's `StartedAt` source (`session.CreatedAt`) against clock skew in SQL: `created_at` is stored by the caller, not by the DB — no issue today, worth a line in the ADR.

---

## g) QUESTIONS (cannot figure out myself)

1. **Release cadence:** do you want the wave cut now (`identity-model v4.16.0` → root → `usermgmt`/`systemadapter`/consumers), or do these ride the next family train? Pushing tags is yours; the hermetic gates stay red either way until it lands, and the TODO P1 row already carries the order.
2. **`SessionEnded` actor semantics for expiry:** I unified impersonation ends to the impersonator (matching `SessionStarted`). Should an *expired* impersonation session also be attributed to the admin (my current choice), or to the system/service actor, since nobody acted at that moment?
3. **Shared-tree protocol:** while a sibling session is mid docs-health pass, should I keep bumping `docs/CORPUS_BASELINE`/`TAIL_BASELINE` (batch-of-one truth vs. their frozen numbers), or leave those files to the owning session and only report the delta? I chose "bump to measured truth, skip the ledger line" and want the rule written down either way.
