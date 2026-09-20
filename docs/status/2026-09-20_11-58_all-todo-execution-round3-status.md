# All-TODO Execution — Round 3 Status (2026-09-20, ~11:00–11:58 CEST)

Continuation of the "WHOLE TODO LIST" execution. Round 1 shipped Retry-After + doc triage; round 2 shipped IP/UA metadata, DecodePayload re-exports, sync ClientID v1.4.0, samber-linter closure, SSE hardening, 5 upstream issues, and adminui Phase 0+1. This round: **adminui Phase 1 lint closure, Phase 2 COMPLETE + verified, Phase 3 code complete (gate mid-flight at interruption), plus two real bugs found and fixed in the demo/harness.**

Scope of this report: only this session's work. Plan: `docs/planning/2026-09-20_09-05_all-todos-pareto-comprehensive-execution-plan.html`; adminui micro-task plan: `docs/planning/2026-09-17_13-43_adminui-prettier-gap-remediation-plan.md`.

---

## a) FULLY DONE (verified this session)

1. **Adminui Phase-1 lint closure** — the 2 open findings fixed: `classNoUnderline` const extracted (goconst; used in models.go + both tenants.templ call sites), statCard literals re-formatted one-field-per-line (gofumpt). Gate re-verified: lint 0, race green, coverage 69.8% (gate ≥66).
2. **Cleanup** — worktree `/tmp/cqrs-adminui-head` verified clean and removed; no stale admin-demo processes.
3. **Adminui Phase 2 (M035–M050) — COMPLETE, all gates green:**
   - `display.PageHeader` on users list, tenants list (action slot: "New tenant" button), tenants new, tenants detail (status badge in action slot, machine name as subtitle), audit, members page (`tenantHeader` → PageHeader pattern).
   - `navigation.Breadcrumbs` on user detail (Users / email), tenant detail (Tenants / name), tenant new (Tenants / New). All 3 hand-rolled "← Back" rows deleted (M044).
   - User-detail identity preserved: avatar rides in the PageHeader action slot; display name is the subtitle.
   - **M045–M047 FilterInput adoption**: `forms.FilterInput` with `wire.Action{URL, Target:"#users-table"}`, DebounceMS 350, AriaLabel, form-level `hx-get`/`hx-trigger=submit`/`hx-target`/`hx-swap=innerHTML`/`hx-push-url`/`autocomplete=off` (htmx inheritance + form-level autocomplete cover what wire.Action can't render). No-JS fallback: real GET form to `/users`. `#users-table` restructured to a stable wrapper with the partial carrying inner content only (killed the old outerHTML+hx-select nesting pattern). **M046 decision recorded in-source**: library look accepted, no search icon (component reserves no start padding; an overlay would collide with typed text).
   - Gate evidence: lint 0, build+vet 0, tests+race green, coverage 69.6% (≥66), `check-codegen` PASSED, admin CSS rebuilt, **28 live-browser behavior checks ALL PASS** (search landmark, debounce trigger, GET fallback, aria-label, push-url, single #users-table after swap, no nesting; breadcrumbs; tenant-detail member rows; tenant-members page 200 with identity subtitle), visual diffs vs p1 baseline all accounted for as intended changes.
4. **admin-demo hardening + membership coverage (todo #7 core)**:
   - Seed now creates memberships (alice→owner@acme, bob→user@acme, carol→viewer@globex) — members tables have real rows; audit log gains member events.
   - **Second panel wired**: `ModeTenantAdmin` panel at `/tenant-admin` (BasePath `/tenant-admin`, TenantID acme, default role authorizer admits alice as owner — no custom wiring). `/dev-login-tenant` sets alice's session.
   - **REAL BUG FOUND + FIXED**: `GET /favicon.ico` fell through the mux `/` catch-all → redirected to `/dev-login` → **silently re-issued the super-admin session cookie, clobbering a just-switched tenant-admin session** (manifested as flaky 403s on /tenant-admin/members). Root-caused via CDP `Network.responseReceivedExtraInfo` set-cookie tracing after Playwright's `headers()` proved blind. Fix: explicit `/favicon.ico` → 204 handler, with the forensics documented in-source.
5. **Adminui Phase 3 (M051–M076) — CODE COMPLETE, unit-tested, lint 0:**
   - **Pagination (the rows-51+-unreachable fix)**: `pageBounds` (clamping window math), `parsePageQuery`, `listPageURL` (search-preserving base URL), `listPageSize = 50`. Wired into users (window over filtered list), tenants, and audit (`Recent(offset+limit)[offset:]` latest-first window). Shared `listPage` struct embedded in all three view models. `listPager` renders `navigation.Pagination` only when TotalPages > 1. **M058 decision**: full-navigation links (browser history honest, search preserved in URL); recorded in the pager comment.
   - **Error mapping (kills every `err.Error()` leak)**: `writeActionError` (toast + themed error page with family-derived HTTP status: Rejection 400 / Conflict 409 / Transient+Infrastructure 503 / Corruption 500 / Orchestration 202) and `actionErrorMessage` for the redirect-back flows. 16-entry code→message table (tenant lifecycle, membership, email/user exists, validation, oauth_not_configured) + per-family fallback copy. Raw errors now only logged server-side with family+code. Replaced 4 writeErrorPage + 7 toast leak sites across handler_tenants/users/members. `errorFamilyFor` upgraded from 2-way to full family mapping.
   - **Pending-state affordances**: `feedback.Spinner` + `hx-indicator` on the users search (CSS rules confirmed pre-existing in tailwind.css); `htmx.LoadingButton` on member "Add member" ("Adding…") and per-row "Set" ("Saving…") submits — hand-rolled `<button>`s carrying the library's exact published class strings (ButtonPrimary/Secondary+SM) because `display.Button` has no children slot; `hx-disabled-elt="this"` on ALL destructive/mutating buttons (user delete, unlink, tenant suspend/reactivate/delete, member remove) for double-submit protection.
   - **EmptyState actions (M075/M076)**: tenants empty → "New tenant" button; users empty with active search → search-aware copy + "Clear search" link. `empty()` wrapper extended with actionText/actionHref.
   - **Tests**: new `pagination_test.go` — pageBounds table (9 cases: empty, exact fit, partial last, zero/negative/beyond-range clamp, pageSize fallback, window-never-exceeds-total invariant), parsePageQuery (7 cases), listPageURL, and a handler-level test seeding 60 users asserting footer presence, beyond-range clamp → 200, and search+page combination. Two stale tests updated with justification (unlink: 400→503 since oauth_not_configured is transient-family; empty-state: new search-aware copy + Clear search action asserted).
   - Lint: 0 issues (after fixing exhaustive×2 by mapping `errorfamily.Orchestration`, goconst via `tenantGoneMessage` const, gochecknoglobals nolint on the lookup table, wastedassign restructure in handler_audit, gci/golines autofix).

---

## b) PARTIALLY DONE

1. **Phase-3 gate (M077) — interrupted mid-flight.** Lint reached 0; tests were green BEFORE the gci/golines autofix (which modified files) — **re-run required**. Still owed: CSS rebuild (Pagination/Spinner/LoadingButton/EmptyState-action class families are NOT in the bundle yet), `check-codegen`, race + coverage, admin-demo rebuild + screenshot diff vs the p2 baseline.
2. **Todo #7 (demo seed + e2e)** — demo side done (memberships, tenant panel, favicon fix, fixed harness selectors/session ordering). The **e2e spec check-in is NOT started**: `e2e/tests/admin-screenshots.spec.ts` does not exist; all harness scripts (shot-p2.mjs, verify-p2.mjs, geo.mjs) live only in `/tmp/adminui-baseline/` (not durable).
3. **Phase 2/3 verification assets** — new p2 baseline PNGs (18 shots incl. tenant panel) live in /tmp only; the mislabeled-"tenant-detail" discovery means the OLD p1 baseline's "tenant-detail" is actually tenants/new (only relevant when diffing; noted).

## c) NOT STARTED

- **Adminui Phase 4 (M078+)**: modal spike notes, Modal confirmations (replace window.confirm path), Dropdown user menu, PolledRegion stats, dark-QA sweep, ThemeToggle docs-only spike.
- **Docs hygiene**: TODO_LIST truth pass (retire routed-gaps #3/#9/#10/#11, samber-linter CLOSED, cqrs-upgrade --workspace landed, findings-gate text STALE), CHANGELOG `[Unreleased]` entries for everything shipped in rounds 2+3, upstream issue references (#20/#21/#22/#35/#36).
- **Gated verification**: V007 (`cqrs-upgrade -dry-run --workspace` from master), appkit ADR-001 (b)–(f) assessment.
- **Final table-view report** (the plan's closing artifact; this document is the round-3 interim).
- Full nix battery (test/lint/coverage across all modules), check-modules, bench-spike (still load-refused), integration_test run.

## d) TOTALLY FUCKED UP (own mistakes, all recovered)

1. **Edit inversion** — one multiedit had old/new swapped (userDetailAvatar insert); caught on the "1 edit failed" result, corrected immediately.
2. **Wrong type guess** — `identitymodel.Tenant` doesn't exist (it's `usermgmt.Tenant` alias); one build failure, fixed.
3. **Buggy test data generator** — my first email-padding loop in pagination_test.go produced duplicate emails for i and i+10 (would have failed Register); caught by inspection before running, replaced with `fmt.Sprintf("pager%02d")`.
4. **The 403 rabbit hole** — ~10 debug iterations on the tenant-members 403 chasing wrong theories (HTTP cache, CSRF cookie, raw-fetch vs browser headers, session rotation) before going to CDP `responseReceivedExtraInfo`, which showed the favicon→dev-login→Set-Cookie clobber in one shot. **Lesson: when the cookie jar changes without an apparent Set-Cookie, playwright's `headers()` is blind — go straight to CDP.**
5. **Trusted the harness twice** — (a) the "tenant-detail" selector matched the "New tenant" button, so that shot had been the tenants/new page since the original baseline; (b) my first shot-p2 script performed the tenant login BEFORE shooting the admin pages, producing 7 identical 403 screenshots — noticed via md5-uniqueness check, script rewritten with explicit session ordering. **Lesson: hash-uniqueness assertion belongs IN every screenshot harness.**
6. **Banned/unsupported commands** — tried `curl` once (blocked, remembered after); `kill` builtin unsupported in mvdan/sh → `pkill`.
7. **Daemon absorbed everything again** — no manual commits were made (rule 6; your Q1 authorization still pending), so this session's narrative lives only in heuristic daemon commits (`0f3384ac`, `6bafa82b`, `76688ea2`, `0cedfbef`, `c49bfd7d`). Working tree is clean at report time.

## e) WHAT WE SHOULD IMPROVE

1. **Upstream: `display.Button` needs a children/content slot** — today LoadingButton content forces hand-rolled buttons duplicating the library's exact class strings (drift risk on the next train; v1.19 "sharp cards" is already flagged). This is upstream-ask material (companion to the filed #20–#22).
2. **Upstream: `wire.Action` cannot render `hx-swap`/`hx-push-url`/`hx-select`** — adminui works around it via form-attribute inheritance (htmx inherits swap/push-url/indicator from ancestors). Documented in-source; worth an upstream issue so the workaround can retire.
3. **Screenshot harness durability** — shot/verify/geo scripts are throwaway /tmp artifacts; the md5-uniqueness + session-ordering + `:not([href$="/new"])` learnings must land in the checked-in e2e spec, or the next person rediscovers them the hard way.
4. **CDP-first debugging** for cookie/auth flakiness (see d.4).
5. **Judgment call to review**: I mapped `errorfamily.Orchestration` → HTTP 202 + "background process settling" copy (exhaustive linter forced the case). If you disagree with the semantics, it's a two-line change in `errorpage.go`.
6. **Deliberate deviation recorded**: M061–M063 wanted spinners on tenants/audit lists too — I did NOT add them: those lists issue no htmx requests (pagination is full navigation), so a spinner there would be dead UI (violates the plan's own Verschlimmbesserung guardrails). Spinners live only where requests actually fire.

## f) NEXT — up to 50 things, in execution order

1. Re-run adminui tests after the lint autofix (files changed post-green).
2. `templ generate` + `nix run .#check-codegen` (Phase-3 regen state).
3. Rebuild admin CSS (`nix run .#build-adminui-css`) — new class families from Pagination/Spinner/LoadingButton/EmptyState actions.
4. Adminui race + coverage gates (≥66).
5. Rebuild admin-demo (WORKSPACE mode), re-shoot, diff vs p2 baseline with md5-uniqueness.
6. Update `verify-p2.mjs` with Phase-3 behavior checks (pagination footer, spinner span, LoadingButton labels, hx-disabled-elt presence).
7. Mark M077 gate green → Phase 3 closed.
8. M078: read `display/modal.templ` props, write usage note.
9. Phase 4: Modal confirmations replacing window.confirm on destructive actions (keep CSP-safe data-confirm path as fallback).
10. Phase 4: Dropdown user menu (navigation package).
11. Phase 4: PolledRegion on dashboard stats (needs SSE-compatible polling; verify CSP).
12. Phase 4: dark-QA sweep across all pages.
13. Phase 4: ThemeToggle spike — docs-only per plan.
14. Check in `e2e/tests/admin-screenshots.spec.ts` (port shot-p2 learnings: load+settle, dev-login per session, selector exclusions, uniqueness assert, workspace-build note).
15. Move/record harness scripts durably (e2e/ or scripts/ with a README note).
16. Optional demo env flag to seed >50 users for visual pagination coverage.
17. e2e tenant-admin flow spec (login-tenant, members page, add/remove member).
18. TODO_LIST truth pass (retire: routed-gaps #3/#9/#10/#11, samber-linter, cqrs-upgrade --workspace, findings-gate-stale text).
19. CHANGELOG `[Unreleased]`: IP/UA/ClientID metadata; DecodePayload re-exports + DecodePaginationStrict; sync assets v1.4.0; SSE replay `retry: 5000` wire change + WithSSEMaxReplay; samber-do-demo fixes; adminui Phases 1–3 (PageHeader/Breadcrumbs/FilterInput/pagination/error mapping/spinners/LoadingButton/EmptyState actions); demo memberships + tenant panel + favicon fix.
20. Reference upstream issues #20/#21/#22 (templ-components) + #35/#36 (go-cqrs-lite) in CHANGELOG/TODO.
21. V007: run `cqrs-upgrade -dry-run --workspace` from go-cqrs-lite master; document.
22. appkit ADR-001 (b)–(f) fold-in assessment; document.
23. Full nix battery: `nix run .#test`, `.#lint`, `.#coverage-gate`.
24. `nix run .#check-modules` (isolation/drift/release-train).
25. integration_test suite (fullstack UI renders may assert adminui markup).
26. Bench-spike re-attempt (check load first; 6th documented refusal if still high).
27. Strip usermgmt's temporary `../` root dev-replace at the next family train.
28. AGENTS.md: adminui adoption table rows for PageHeader/Breadcrumbs/FilterInput/Pagination/Spinner/LoadingButton/EmptyState-actions + favicon/dev-login lesson + CDP cookie-debug lesson.
29. Annotate the adminui plan HTML: M035–M077 done-state.
30. tenants-list pagination unit test (mirror users test).
31. audit pagination test (seed >50 audit events).
32. Members card: document the not-paginated decision (bounded by tenant membership count).
33. Verify `capList` still has callers (dashboard recent?) — remove or recomment if orphaned.
34. Verify CSP tests still pass with new components (no inline JS introduced).
35. Check the autofix diff scope (`git show` on the last heuristic commits) — confirm gci/golines only touched intended files.
36. tenantSubtitle mono styling decision (old font-mono name is now plain subtitle text).
37. Consider aria-current/page-announce checks on pagination (library default verification).
38. Final table-view report (docs/status) covering ALL TODO items with done/partial/gated/user-decision.
39. Answer-pending integration: fold your Q1–Q3 answers (below) into the execution policy.
40. Load check + cleanup of /tmp servers and background processes at session end.
41. Upstream issue draft: templ-components Button children slot.
42. Upstream issue draft: wire.Action swap/push-url/select coverage.
43. Root/usermgmt modules: run their focused suites again before any train (round-2 code untouched today, but the train checklist demands it).
44. `gofmt -l` sweep over adminui (autofix aftermath).
45. Kill the /tmp/adminui-baseline demo server when Phase 3/4 visual work concludes.
46. Optional: pagination `MaxVisible` tuning for mobile (container-aware flag).
47. Optional: audit page tenant-scope filter (noted in handler comment as future work).
48. Verify LoadingButton spinner color on secondary variant in dark mode (screenshot).
49. Confirm `usermgmt.oauth_not_configured` 503 doesn't break admin-demo unlink UX (toast copy is friendly).
50. STOP and wait for instructions (per order).

## g) QUESTIONS (cannot be answered from the repo)

1. **Commit authorization (Q1, carried from round 2):** the daemon has now shredded rounds 2+3 into ~9 heuristic commits (including this session's non-compiling intermediates). Do you authorize explicit per-cluster commits at the remaining boundaries (Phase-3 gate close, Phase-4 close, docs pass, final report) — `git commit` with proper messages — or should I keep hands off and let the daemon absorb?
2. **Phase-4 scope (Q2, carried):** does "WHOLE TODO LIST" include the P4 delight lane (Modals, Dropdown menu, PolledRegion, dark-QA, ThemeToggle doc), which the plan marks droppable? Default reading says yes; confirm or cut.
3. **Force-push (Q3, carried):** the two history purges (27MB setup-demo blob in pushed history; any other rewrites) still await your explicit approval — I will not force-push without it.

---

**Bottom line:** Phase 1 closed, Phase 2 shipped and verified end-to-end in a real browser, Phase 3 is code-complete with tests+lint green but its visual/CSS/codegen gate was interrupted mid-flight (resume list: f.1–f.7). One genuine demo bug (favicon-induced session clobber) found, root-caused, fixed, and documented. Working tree clean; everything daemon-committed; waiting for instructions.
