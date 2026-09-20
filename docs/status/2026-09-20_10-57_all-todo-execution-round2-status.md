# All-TODO Execution — Round 2 Status (SSE hardening, upstream asks, adminui Phase 1)

**Date:** 2026-09-20 10:57 CEST
**Session context:** Continuation of the "execute the ENTIRE TODO_LIST" directive (round 1 = 2026-09-20 09:05 Pareto plan + Tier-1 items). This round executed: the routed-gaps code bundle (IP/User-Agent, ClientID, API exposure), the samber-linter repro, the full SSE hardening backlog, five upstream issue filings, and adminui program P0 + Phase 1. Load average 13.8→57 the whole session (bench-spike correctly refused; 5th documented refusal).

---

## a) FULLY DONE

1. **Gate re-verification (Phase 0):** templ-components upstream still at v1.18.1 (v1.19 unreleased — sharp-cards change not shipped); go-structure-linter still v0.10.0 (suppressions feature untagged). Both TODO items remain legitimately upstream-blocked. bench-spike refused at load 57.24 (limit 8).

2. **Routed gap #3 — HTTP IP/User-Agent → event metadata (root + usermgmt):** new `requestmeta.go` (typed context keys + `enrichClientMetadata` capture via `httputil.ClientIP` precedence XFF→X-Real-IP→RemoteAddr, invalid dropped at debug level); `ContextEnrichmentMiddleware` wired; `EventOptionsFromContext` now emits `event.WithIPAddress` + `event.WithUserAgent`; usermgmt `requestContextEnricher` propagates both (every dispatched identity event now carries client IP + UA). GDPR privacy note + shadow-with-zero escape hatch documented. 8 Ginkgo specs + `TestDispatch_ClientMetadataPropagatesToEventMetadata` (event metadata value-level assertions). Guide `actor-and-audit-trail.md` updated.

3. **Routed gap #10/#11 — API exposure:** `payload.go` re-exports `DecodePayload`/`DecodePayloads`/`DecodePayloadAuto` (typed wrappers; generics can't be aliased) + tests; `DecodePaginationStrict` (parses raw `page`/`page_size` strings — `httputil.ParseUintQuery` conflates missing/malformed — then `query.Pagination.Validate()`, returning errorfamily Rejections that map to 400) + DescribeTable ×5. `go-codec` promoted to direct root dep.

4. **Routed gap #9 — offline-sync ClientID propagation (JS + Go):** `sync-client.js` mints a localStorage-persisted **ULID** per browser (`crypto.randomUUID` would fail `id.ParseClientID`'s ULID-strict parse), stamps `X-Client-Id` on every mutation; envelope headers persist it across offline retries. Root: `HeaderClientID` const, `WithClientID`/`ClientIDFromContext`, capture in `enrichClientMetadata`, propagation via `event.WithClientID` → metadata `client.id`. usermgmt enricher + `TestDispatch_ClientIDPropagatesToEventMetadata`. Asset version bumped 1.3.0→1.4.0 in all three places (client, worker, `syncVersion`) — version-pinning test green.

5. **Hygiene micro-pack CLOSED — samber-linter repro:** the step now PASSES (929ms–2.7s, 29/29 modules, 0 failures) — the 2026-09-17 "88% failure rate (61/69)" class was the toolchain tug-of-war and died with the 2026-09-19 v4.11.0 alignment. NO exclusion needed. Its 4 real findings in `examples/samber-do-demo` FIXED honestly: `serviceLifecycle` + new `broadcasterLifecycle` implement `do.HealthcheckerWithContext` (delegating to the library's own `ProjectionReadinessCheck`/`HubReadinessCheck` gates); health probe + dashboard + both lifecycles constructed eagerly in `NewContainer`; 4 single-line `//samber-linter:allow hw-4` suppressions where the linter cannot trace the eager boot invokes. Verified: 29/29 modules, **0 findings**. (Learned: suppression directives must be single-line; ambient `GOTOOLCHAIN=local` poisons buildflow module loads — showed as phantom [Failed] on root+demo.)

6. **SSE hardening backlog — ALL items:**
   - `transport.WithSSEMaxReplay(n)` handler-level option (tail-cap semantics, works with any `sse.EventStore`).
   - `transport.SSEOptions` struct + `.Options()` (zero value = all defaults; equivalence test).
   - **Retry field on replayed events** (default `DefaultRetryHintMillis` stamped when unset; explicit retry preserved) — a deliberate replay-path WIRE FORMAT change, tested.
   - `serve_test.go` gaps: `HeartbeatJoinOnExit` (join-before-return, race-detector-guarded), `BroadcasterCloseMidStream`, `ConcurrentClients` (16 clients, all receive), `MaxReplayCapsBackfill`, `ReplayedEventsCarryRetry` — all `-race` green. (Replay-before-subscribe ordering pre-existed.)
   - `BenchmarkJournalSSEStore_EventsAfter` (10k events; first-connect + reconnect-midpoint sub-benches; `b.Loop`/`ReportAllocs` per repo policy).
   - `FuzzDomainEventToSSE` (6 seeds incl. multi-line/emoji/injection-adjacent payloads; ID-contract + frame-termination assertions).
   - `integration_test/sse_replay_contract_test.go`: `TestSSE_ReconnectWithReplay` (connect → capture → reconnect with `Last-Event-ID` → cursor event not replayed, later events replayed, retry hint present) and `TestSSE_CrossModuleWireFormatContract` (setup-shaped `/sse` vs dashboardui `/-/events/stream`: byte-identical `event:` frames for the same committed domain event).
   - setup + dashboardui full suites re-verified green after the replay wire change; transport lint 0 issues.

7. **Upstream asks — 5 issues filed (all verified against source first, voice-checked via github-voice, 0 FAIL/0 WARN):**
   - templ-components **#20** ListNote count-only/range variants (no-total cursor pagination is unusable today)
   - templ-components **#21** Grid `{children...}` renders empty when `Render()`'d standalone (hybrid string-builder path) — doc or data-driven items
   - templ-components **#22** CopyButton label span inherits ancestor color (near-invisible in dark consumer tables)
   - go-cqrs-lite **#35** ship a `RequestContext` enricher upstream (correlation/request ID, IP, UA, ClientID) so the cqrs-htmx local copy drops at the next train
   - go-cqrs-lite **#36** decouple stack root package from metaengine (`stack/options.go:12`, `stack/bundle.go:12`)

8. **Stale claim retired WITHOUT filing:** `cqrs-upgrade --workspace` already exists on go-cqrs-lite master (`findGoMods` walks go.mod files; flag wired in main.go) — the TODO's "needs multi-module mode / 22 manual runs" item is resolved upstream (arrives with the next tool release; existing upstream issue #28 covers the docs angle).

9. **Adminui program Phase 0 (preconditions):** P0.1 adminui suite baseline green; P0.2 buildflow binary rebuilt from HEAD (was 42fd89b, HEAD 551ee4f); P0.3 visual baseline — 14 screenshots (7 pages × light/dark) in `/tmp/adminui-baseline` (not committed, binary-blob rule).

10. **Adminui Phase 1 — "the 1% → 51%" component sweep (P1.1–P1.5 + most of the P1.6 gate):**
    - **P1.1** `Table.Flush: true` on all 5 Card-nested tables (dashboard recent, audit, user tenant-roles, external accounts, members); the global `.overflow-hidden > .overflow-x-auto` **`!important` hack deleted** from tailwind.css.
    - **P1.2** `icon()` now delegates to `icons.IconWithStrokeWidth(name, "h-[18px] w-[18px]", 1.8)`; hand-rolled `iconSVG` + `templ.Raw` deleted (−20 LOC, escaping-safe).
    - **P1.3** new `statusBadge` wrapper (Dot: true) on verified/unverified/MFA + tenant active/suspended/deleted ×2 sites; accent badges (audit actions) deliberately dotless.
    - **P1.4** `statCard` gains `Tone` + `Href`; dashboard maps blue/purple/green tones + drill-down links (users/tenants/audit; tenant-mode members); Change/Trend omitted with honest-UI reason documented in the struct comment.
    - **P1.5** compact cells: `CellPadding: TableCellPaddingCompact` ×6 + `Striped` on users/audit + all 25 hand-rolled `td px-4 py-3` → `py-2` across 5 row templates (Body-slot rows don't inherit TableProps padding).
    - **Gate:** `check-codegen` PASSED; adminui tests `-race` green; coverage **69.8%** (gate ≥66, up from 69.0); CSS bundle rebuilt via `nix run .#build-adminui-css`; visual diff vs baseline meaningful and exactly as intended — user-detail page height **1049→1011px** (compact rows), status dots present (5 on users page), toned stat tiles + linked cards verified in-DOM.

---

## b) PARTIALLY DONE

1. **Adminui Phase-1 gate (P1.6) — 2 lint findings open:** `goconst: 1` + `gofumpt: 1` on adminui (from the new `statCardClass`/statusBadge code). Everything else in the gate is green. Fix is minutes.
2. **Adminui Phase-1 visual verification — members/tenant-detail gap:** AE=0 diff on both pages because the admin-demo seed creates **no tenant memberships** — the members table (Flush + compact changes) renders as empty state on both sides. Structural changes are test-verified but not visually confirmed. Options: seed one membership in the demo, or extend an adminui render test.
3. **M021 (icon stroke-width parity) not explicitly confirmed** — icons render via the typed component (verified in DOM), but no pixel-level parity check against the old 1.8-stroke SVGs was performed.
4. **Gated-items verification:** findings-gate + templ-components gates re-verified (closed); **V007 migration maturity** and **appkit ADR-001 (b)–(f)** not yet re-assessed this session.

---

## c) NOT STARTED

1. Adminui **Phase 2** (PageHeader ×6, Breadcrumbs ×3, FilterInput search) — M035–M050.
2. Adminui **Phase 3** (Pagination on users/tenants/audit — rows 51+ unreachable today; Spinner hx-indicator; LoadingButton; errorpage across the 14 error paths; EmptyState actions) — M051–M078.
3. Adminui **Phase 4 delight lane** (Modal confirmations, Dropdown user menu, PolledRegion, dark-QA sweep, ThemeToggle spike) — M079+.
4. **Docs hygiene pass:** TODO_LIST truth updates (routed-gaps item now DONE for #3/#9/#10/#11; samber-linter resolved; cqrs-upgrade `--workspace` landed upstream; findings-gate TODO text stale — `.buildflow.yml` already at `fail_on: critical` with the full restoration history inline; OTel-posture note item should move to ROADMAP per its own "not here" wording) + CHANGELOG `[Unreleased]` entries for everything in section a).
5. **Final table-view report** (session deliverable).
6. User decisions still owed (parked by design): `/sse` authz posture, datastar-demo rebrand-or-remove, dashboardui theme-toggle.
7. Force-push history purges (v4 branch blobs, master setup-demo blob) — need explicit approval.
8. DataStar Tier 4 M11–M16 — demand-gated per ADR-0050.
9. `check-cqrs-lint` in CI — blocked on a Go-installable cqrs-lint distribution (upstream).

---

## d) TOTALLY FUCKED UP

Nothing irreversible. Honest low points:

1. **The visual-baseline harness took 4 attempts**, each failure self-inflicted:
   - First "baseline" shot against **bank-sync** — another project's service that owns :8097. I assumed the port held a stale admin-demo and only discovered otherwise because `/dev-login` 404'd. Never verified the server identity before capturing. (Did NOT kill the foreign process — correct.)
   - `networkidle` never fires on SSE pages (the feed holds the connection open) → switched to `load` + settle.
   - A sed patch rewrote the wrong path constant and **overwrote the baseline with Phase-1 shots** → recovered the true pre-phase-1 binary via a git worktree at `b0f265b6` (daemon had already absorbed Phase 1 into HEAD).
   - **The big one:** I built the demo with `GOWORK=off` (hermetic-battery habit), which silently resolved the **published** adminui v4.11.0 — Phase-1 markup was absent from the served pages. Burned ~20 min on badge-probe tests and process forensics before checking the env var. Example binaries that consume in-repo modules must build in **workspace mode**; GOWORK=off is for per-module verification only.
2. Two probe test files (`badge_probe_test.go`) were created in the repo tree for debugging and removed after — should have lived in a scratch location.
3. New dev-replace debt created: `usermgmt/go.mod` now carries `cqrs-htmx/v4 => ../` (removal-condition commented) until the next family train tags a root version with the new context symbols.
4. The `domainFrame` parser in the wire-contract test took 3 iterations (string-suffix `\n` assumption, then path mismatch) before settling on frame-splitting with `slices.Contains`.

---

## e) WHAT WE SHOULD IMPROVE

1. **Screenshot harness should be a checked-in tool, not a /tmp script** — the N5 dashboard spec exists (`e2e/tests/screenshots.spec.ts`); adminui deserves an equivalent spec (admin-demo webServer + `tests/admin-screenshots.spec.ts`) so the P2/P3/P4 visual gates stop re-deriving the harness. The `/dev-login` + `waitUntil: "load"` + workspace-mode-build learnings belong in that file's comments.
2. **admin-demo seed should include one tenant membership** so the members table (and its Flush/density classes) is visually exercised at all.
3. **Worktree hygiene:** `/tmp/cqrs-adminui-head` worktree still registered — `git worktree remove` owed (it also holds a detached checkout of `b0f265b6`).
4. The daemon absorbed Phase 1 into heuristic commits mid-verification (`72c24b2e` captured a **non-compiling** intermediate state — `components_templ.go` referencing the already-deleted `iconSVG`). This is the documented daemon race; explicit per-phase commits would have kept every committed tree buildable. (Needs user authorization — see question 1.)
5. `verify-before-filing` + `github-voice` worked exactly as designed for the 5 upstream issues (source-level verification first, voice checker 0 FAIL) — this is now the default pattern for upstream asks; the TODO_LIST upstream-asks entries can be retired as "filed, tracked upstream".
6. The bench-spike load-refusal streak is 5 sessions long — consider a cron/agentd task that fires the gate when `uptime` 1-min load < 8, instead of manual sessions discovering the window.

---

## f) Up to 50 things next

1. Fix the 2 adminui lint findings (goconst, gofumpt) — Phase-1 gate fully green
2. Remove the `/tmp/cqrs-adminui-head` git worktree
3. Extend admin-demo seed with one tenant membership (visual coverage for members table)
4. Check-in `tests/admin-screenshots.spec.ts` (Playwright admin harness; port learnings from e)
5. Adminui Phase 2: PageHeader on dashboard page (M034-adjacent)
6. Adminui Phase 2: PageHeader ×5 remaining pages (users, tenants list/new/detail, audit, members) — M035–M040
7. Adminui Phase 2: Breadcrumbs on user/tenant detail + tenant new — M041–M043
8. Adminui Phase 2: delete the 3 hand-rolled "← Back" rows — M044
9. Adminui Phase 2: FilterInput replaces the hand-rolled search form — M045–M047
10. Adminui Phase 2 gate: gen + codegen + CSS + screenshots + race + coverage — M048–M050
11. Adminui Phase 3: Pagination on users (page param + TotalPages + footer) — the highest-value functional fix
12. Adminui Phase 3: Pagination on tenants + audit
13. Adminui Phase 3: Spinner hx-indicator on swap regions
14. Adminui Phase 3: LoadingButton on mutating forms/buttons
15. Adminui Phase 3: errorpage.ErrorAlert across the 14 handler error paths
16. Adminui Phase 3: kill the `err.Error()` response leaks (handler_tenants.go:45,78,89,100)
17. Adminui Phase 3: EmptyState action buttons
18. Adminui Phase 3 gate
19. Adminui Phase 4 (delight lane — droppable): Modal confirmations, Dropdown menu, PolledRegion, dark-QA sweep, ThemeToggle spike
20. Docs: TODO_LIST truth pass (routed gaps #3/#9/#10/#11 DONE; samber-linter CLOSED; cqrs-upgrade --workspace landed; findings-gate text stale)
21. Docs: CHANGELOG `[Unreleased]` entries for round 2 (IP/UA, ClientID, payload/pagination API, SSE hardening incl. the replay wire-format note, sync v1.4.0, samber-do-demo health wiring, upstream issue links)
22. Docs: retire the OTel-posture note item to ROADMAP (per its own wording)
23. Re-verify V007 (39 findings) maturity: `cqrs-upgrade -dry-run --workspace` from the new master
24. Assess appkit ADR-001 (b)–(f) — default-flip decision inputs
25. Write the final table-view session report
26. Ask the 3 questions below (section g) and act on answers
27. Usermgmt dev-replace strip + next family train (root tag carrying IPAddress/ClientID symbols)
28. Track upstream: templ-components #20/#21/#22 (watch for v1.19 — sharp cards + CSS rebuild + screenshot pass)
29. Track upstream: go-cqrs-lite #35 (RequestContext enricher — drops the local copy)
30. Track upstream: go-cqrs-lite #36 (stack decoupling — drops usermgmt's transitive metaengine require)
31. go-structure-linter release watch (suppressions tag) → restore `fail_on: critical` permanently-verified
32. bench-spike idle-window automation (load < 8 trigger)
33. Wire `TestSSE_ReconnectWithReplay` + wire-contract test into CI docs (they run via `.#test`)
34. Consider seeding `adminui` coverage of `statCardClass` (trivial, but keeps 69.8% floor moving up)
35. `/sse` authz posture decision (rec A) — one-pager exists
36. datastar-demo rebrand-or-remove decision — owner call
37. theme-toggle decision — parked until a consumer asks
38. Force-push approvals for the two history purges (v4 branch ~27.7MB, master 27MB blob)
39. DataStar Tier 4 M11 (signal-patch spike) — only on demand evidence
40. cqrs-lint Go-installable distribution ask (unblocks the CI gate)

(40 real items; the remaining plan micro-tasks M051–M101 are covered by items 11–19.)

---

## g) Questions I cannot figure out myself

1. **Commit authorization:** this session's work is being shredded into `chore: auto-commit` heuristic commits by the daemon — one absorbed a non-compiling intermediate state, and per-phase narrative is lost (the documented daemon-race lesson says commit at phase boundaries, but my default rule is never to commit without your say-so). **Should I make explicit per-phase commits from here on** (e.g. "adminui: phase 1 — Flush/icons/dots/statcard/density"), or keep letting the daemon absorb?
2. **Adminui program depth:** your "WHOLE TODO LIST" reads as approval for the plan's P0–P3 (through the 20%→80% gate: pagination, loading states, error pages). The P4 delight lane (Modal confirmations, Dropdown user menu, PolledRegion polling, dark-QA, ThemeToggle spike) is marked "independently droppable" in the plan. **Include P4 in this execution, or stop at the P3.6 gate?**
3. **Destructive approvals:** the two history purges (rewrite `origin/v4` to strip ~27.7MB of binaries; filter-repo master to purge the pushed 27MB setup-demo blob) both require force-pushes you must approve explicitly — **approve now, defer, or drop?** (Same answer slot covers the parked product calls if you want them taken: `/sse` posture rec-A, datastar-demo remove, theme-toggle keep-as-is.)
