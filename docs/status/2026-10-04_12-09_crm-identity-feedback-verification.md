# Status Report — CRM Identity-Adapter Feedback Verification Session

**Date:** 2026-10-04 12:09 (Sunday)
**Scope:** Single-task session — evaluation of `docs/feedback/new/2026-10-04_crm-identity-adapter-gaps.md`. This report covers ONLY what this session did and noticed. No code was changed this session.
**Inputs:** the feedback file, cqrs-htmx skill, usermgmt/setup/httputil source, TODO_LIST/ROADMAP greps, one post-hoc blast-radius re-check.

---

## Session summary

The user asked for an opinion on the Ledger CRM feedback file. I loaded the cqrs-htmx skill, read the feedback, and verified all five claims against source (no claim on faith). I delivered an opinion + triage order. A second pass (this report's prep) then found **two blast-radius facts my first review missed** — both recorded below in (d)/(b).

**Verification results (all five claims checked against source, 2026-10-04):**

| # | Feedback claim | Verdict | Key evidence |
|---|----------------|---------|--------------|
| 1 | WebAuthn register ceremonies bind to arbitrary `user_id` (body/query), no session check — account takeover by known ULID | **CONFIRMED — library-wide vulnerability** | `usermgmt/webauthn_http.go:43` (body), `:52` + `:14-21` (query), `usermgmt/http.go:201-202` (bare mount), `usermgmt/webauthn_service.go:80-136` (existence check only, zero authz), `setup/mount.go:45-47` (bundle mounts bare), `setup/bundle.go:159-181` (`Middleware()` adds no session). Bootstrap constraint verified: `http.go:247` sets the session cookie on `/auth/register` before any ceremony. |
| 2 | `/auth/credentials*` dead-on-arrival (context-only handlers, never session-wrapped) | **CONFIRMED — and UNDER-REPORTED by the feedback AND by my first pass** (see d-1) | `usermgmt/verification_totp_http.go:47-54` (`currentUser` = context-only), `usermgmt/http.go:207-208` (bare registration), nothing in `RegisterRoutes` or `setup.Mount`/`Middleware` populates context on that path. |
| 3 | No `RequireSessionRedirect` / `RequireSession` helper for HTML consumers | **CONFIRMED** | No matches repo-wide; setup already has the 401 flavor UNexported at `setup/mount.go:135-145`. |
| 4 | No `DisplayName` resolver over the user read model; prefixed/bare id duality | **CONFIRMED as absent** | Data exists (`usermgmt/sql_readmodel.go:41`); duality real (`PrefixedString` at `usermgmt/middleware.go:128` vs bare ids in views). |
| 5 | Rate limiter keyed on `RemoteAddr` incl. port = per-connection budget | **CONFIRMED but mis-attributed** | `usermgmt/http.go:118` pins `KeyExtractorFromRemoteAddr()`; httputil returns `r.RemoteAddr` verbatim (`httputil@v1.x ratelimit_keyed.go:59`). NOT upstream-blocked: httputil already ships `KeyExtractorFromClientIP` — one-liner in THIS repo. |

---

## a) FULLY DONE

1. Loaded the cqrs-htmx skill before touching the task (per skill-activation rule).
2. All five feedback claims verified against source with file:line evidence — including the service layer (`BeginRegistration`/`FinishRegistration` do no authorization) and the composition root (`setup.Mount` + `Bundle.Middleware` — the one-call path ships the hole).
3. Claim 1 correctly escalated from "CRM workaround" to "library vulnerability every consumer ships"; bootstrap-safety of the fix verified from source.
4. Claim 5 mis-attribution identified (fix is usermgmt-local, not an httputil blocker).
5. No-duplication check: TODO_LIST/ROADMAP greps found no existing work items for any of the five.
6. Triage order recommended: 1+2 together (deletes the CRM's `CredentialsGate`), then 5 (one-liner), then 3, then 4.
7. This self-review pass: generalized the item-2 invariant and checked first-party flow consumers — caught two gaps in my own first review (see d).

## b) PARTIALLY DONE

1. **Feedback lifecycle (gotcha 20) half-done.** Verify step: done (above). Act / annotate / `git mv` to `processed/`: NOT done — deliberately awaiting the user's call (my closing question offered process-vs-implement; user then requested this report instead). The verification evidence existed only in chat until this report captured it.
2. **Blast-radius verification incomplete on first pass, completed only during this report's prep:**
   - The dead-route class extends BEYOND `/auth/credentials*`: `/auth/me` (`http.go:270-276`), TOTP routes (`verification_totp_http.go:62,211,236`), OAuth2 unlink (`oauth2_http.go:73`) — **the entire authenticated subset of `RegisterRoutes` is dead-on-arrival when mounted via the setup bundle** (everything that calls `currentUser`).
   - The library's own `loginpage/assets/login.js:220-237` drives `register → registerBegin → registerFinish` — a first-party consumer of the flagged unauthenticated flow. It does NOT invalidate the fix (register sets the session cookie first, so a "session user must match target" gate keeps the flow working — same-origin fetch sends cookies), but it was a mandatory design constraint my first pass never surfaced.
3. AGENTS.md memory update not done — the verified vulnerability + the generalized dead-route class are not yet recorded anywhere durable (this report is the first capture).

## c) NOT STARTED

1. Code fixes for any of the five items (zero implementation this session).
2. Regression tests: session-gate tests for the ceremonies; dead-route tests (`/auth/me` with valid cookie → must be 200).
3. Migration of existing tests that currently post the ceremonies bare and would break under a gate: `handler_webauthn_test.go:24,117`, `coverage_schema_test.go:144`, `fuzz_test.go:69`, `webauthn_ratelimit_test.go:28`.
4. loginpage end-to-end verification under the gate (cookies actually riding its `postJSON`).
5. Docs: setup README security section, root README route table, skill `references/usermgmt.md` (authenticated-subset marking).
6. CHANGELOG `[Unreleased]` entries; train bundling + `scripts/verify-tag.sh`.
7. Feedback file annotation + move to `docs/feedback/processed/`.
8. ADR for the "credential ceremonies require owner session" decision (pending check whether an ADR already covers the public-route posture).
9. CRM-side acceptance run ("delete the `CredentialsGate`, nothing reddens") — needs CRM repo access.

## d) TOTALLY FUCKED UP!

1. **My first review verified the feedback's claims as written but did not generalize or look for first-party consumers — two misses, found only in this report's prep:**
   - *Miss A:* I confirmed `/auth/credentials*` dead-routes without checking whether sibling handlers share the invariant "context-only handler + bare mount = dead route." They do — `/auth/me`, TOTP, OAuth2 unlink. The feedback under-reported the class and I parroted its scope instead of testing the invariant. A consumer reading my turn-1 answer would fix two routes and ship four still-dead ones.
   - *Miss B:* I declared item 1 a vulnerability every consumer ships without inventorying the library's OWN consumers of the flow. `loginpage` (first-party, ready-made) depends on the unauthenticated register flow working — it survives the correct fix, but only by the accident of `http.go:247` ordering + same-origin cookie defaults. Had the fix been designed from my turn-1 evidence alone (plain 401 gate, no user-match rule), it could have broken the library's own login page.
2. I did not read the skill's `references/usermgmt.md` even though the skill instructs reading the matching reference before improvising judgments about auth wiring/mounting posture.
3. I did not check `git log`/ADRs for prior intent behind the bare mount (deliberate-but-undocumented vs oversight) — still unverified.
4. I took the feedback's PapDashboard-precedent sub-claim on the doc's word (the processed file exists at `docs/feedback/processed/2026-09-29_papdashboard-setup-non-adoption.md` but I never opened it) — a small violation of the "no claim on faith" rule I was otherwise enforcing.
5. Turn-1 conclusions all SURVIVE the new facts (the vulnerability is real, the triage order stands), but the evidence base I presented was incomplete — the opinion was right, the diligence was not.

## e) WHAT WE SHOULD IMPROVE!

1. **Make security-class feedback verification systematic.** Checklist for any future "handler X is broken/insecure" claim: (i) generalize the invariant across ALL sibling handlers, (ii) inventory first-party consumers of the flagged flow (loginpage/adminui/dashboardui/examples/e2e), (iii) `git log`/ADR intent check, (iv) read the relevant skill reference file. This session added (i)+(ii) only as an afterthought.
2. **Fix the class, not the instance:** `RegisterRoutes` mixes public and session-dependent handlers with no wrapper. Self-wrapping the session-dependent subset inside `RegisterRoutes` (usermgmt owns service + cookieName) kills the entire dead-route class at the root — no per-route host knowledge needed.
3. **Capture verification evidence durably at end of turn**, not only in chat — had the session ended after turn 1, the file:line proof would have evaporated.
4. **Mechanize the invariant:** a test (or script) asserting every handler that calls `currentUser` is registered behind session middleware — the dead-route class is grep-detectable; it should never have shipped.
5. Feedback processing should complete the lifecycle in the same session when the user's intent is actionable — I stopped at "want me to?" which is correct in exploration mode, but the annotate-verify-results step could have happened unconditionally.

## f) Things we should get done next (session-scoped; HARVEST candidates — NOT yet in TODO_LIST)

*Sorted by impact; grounded strictly in this session's findings.*

1. **Implement item 1:** session-gate `register/begin`+`finish` — 401 without session; session user must match target `user_id`; keep service-level API open (`BeginRegistration(ctx, userID)` stays the adminui seam). File: `usermgmt/webauthn_http.go`.
2. **Implement item 2, generalized:** self-wrap the session-dependent subset in `RegisterRoutes` (`/auth/me`, `/auth/credentials*`, TOTP, OAuth2 unlink) with the handler's own session middleware — setup inherits the fix for free.
3. **Implement item 5:** `newLimiterFromConfig` → `KeyExtractorFromClientIP()` (`usermgmt/http.go:118`) + document proxy-trust caveat in `HandlerConfig` docs.
4. Regression test: bare `RegisterRoutes` mount + valid session cookie → `GET /auth/me` = 200 (today: 401 forever).
5. Regression tests for the gate: no session → 401; session vs mismatched target user → 401/403; session vs self → 200.
6. Verify loginpage end-to-end under the gate (its `postJSON` cookie posture; Playwright or httptest through `setup.Handler`).
7. Migrate the bare-mounting tests: `handler_webauthn_test.go:24,117`, `coverage_schema_test.go:144`, `fuzz_test.go:69`, `webauthn_ratelimit_test.go:28`.
8. Integration test: full `setup.Bundle.Handler` mount → authenticated-subset routes all reachable (the e2e for the dead-route class).
9. Audit for OTHER arbitrary-user parameters in auth routes (anything reading user ids from body/query, the item-1 class beyond webauthn).
10. Export `setup.RequireSession` (reuse `mount.go:135-145`) + add `RequireSessionRedirect("/login")` — feedback item 3.
11. `Service.DisplayName(ctx, id)` with explicit id-shape contract (bare ULID + `user:`-prefixed) — feedback item 4.
12. Consider canonical prefixed-id helper in identity-model (sibling of `ParseActorID`) for the fleet-wide duality.
13. Check `docs/adr/` + `git log` for existing public-auth-route-posture decision; write the ADR for owner-session-gated ceremonies if none covers it.
14. Update `setup/README.md` security section (currently documents only the no-CSRF rationale; must document the session gate).
15. Update root README route table + skill `references/usermgmt.md`: mark the authenticated subset.
16. CHANGELOG `[Unreleased]` Security entries (items 1, 2) + Fixed (5) + Added (3, 4).
17. Read `docs/feedback/processed/2026-09-29_papdashboard-setup-non-adoption.md` — verify the cookie-gate precedent claim.
18. Check adminui for links to `/auth/credentials` (feedback speculated; unverified).
19. Annotate the feedback file (`> **PROCESSED** (date): …` blockquote) + `git mv` to `processed/` once fixes land.
20. AGENTS.md: record the dead-route class + the loginpage flow's session dependency (or record the fix, making it moot).
21. Revisit loginpage TODO item (test-depth debt): `login.js` now also carries the gate dependency — its zero-test status is riskier.
22. Re-check CSRF posture for `register/finish` once session-gated (it then rides cookie authority; SameSite=Strict currently guards — document the reasoning next to the existing no-CSRF note).
23. Add the mechanical invariant check from (e)-4 (test or script wired into check-modules).
24. Consider upstream httputil doc-comment improvement: `KeyExtractorFromRemoteAddr` should warn it keys per TCP connection.
25. Rate-limit docs: document the key-semantics change (per-connection → per-IP) wherever `WebAuthnRateLimit` etc. are documented.
26. Bundle the train: wave-ordered cuts, `scripts/verify-tag.sh` per module, pre-push release-train gate.
27. Run CRM acceptance ("delete the `CredentialsGate`, nothing reddens") — requires g-3 answer.
28. Post-fix sweep: `rg "RegisterRoutes"` across examples/e2e/integration_test to confirm no consumer depended on the dead routes' 401s.
29. TODO_LIST HARVEST from this report's section (f) — pending user instruction (see below).
30. Record in AGENTS.md: same-origin cookie default is now load-bearing for loginpage's register flow (design constraint for future CSP/cookie changes).

## g) Questions I can NOT figure out myself

1. **Fleet knowledge:** Besides Ledger CRM and PapDashboard, are there any consumers (private repos, scripts, CI harnesses) that rely on the CURRENT unauthenticated WebAuthn-register semantics — e.g., headless enrollment for another user? This decides whether item 1 lands as a behavior-tightening minor or must wait for v5.
2. **Release posture:** OK to ship items 1+2+5 as minors on the next train (401 where previously 200 on the ceremonies; previously-dead routes start working) with CHANGELOG Security notes — or hold for the v5 major?
3. **Access:** Is the Ledger CRM repo reachable from this machine (path?), so the "delete the gate, nothing reddens" acceptance suite can run as part of fix verification?

---

*Report written 2026-10-04 12:09. Point-in-time snapshot; append-only. No code changed this session. Waiting for instructions.*
