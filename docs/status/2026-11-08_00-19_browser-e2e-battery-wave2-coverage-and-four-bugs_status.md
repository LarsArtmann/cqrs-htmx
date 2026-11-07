# Browser-E2E Battery Wave 2 — Repo-Wide Coverage + Four Shipped Bugs Found & Fixed

> Status report · 2026-11-08 00:19 · Continues the 09-13 wave-1 report (plan of
> record: `docs/planning/2026-11-07_08-55_SUPERB-browser-e2e-battery-ci-and-datastar-coverage.md`).
> Session scope was the blanket directive: execute the SUPERB plan milestone by
> milestone, verify each, keep going.

## a) FULLY DONE (this session)

1. **Session-state re-verification** — all wave-1 files intact after daemon
   auto-commits (`data-init` fix, datastar.spec.ts, config entries, plan, reports).
2. **M7 — examples/basic as e2e surface** (`root-htmx.spec.ts`, 5/5 green):
   `BASIC_DEMO_ADDR` env override (admin-demo pattern, default `:8096`, e2e
   `:18932`), fourth webServer entry, spec covering RenderPartialOrFull
   (fragment+OOB vs document keyed on `HX-Request`), RenderIf (`HX-Target:
   #item-count` → bare "N items"), in-browser OOB swap via `htmx.ajax` (badge
   updated, OOB element consumed not dumped), the SSE-triggered refresh loop on
   `/`, and a console-error guard.
3. **BUG #2 FOUND+FIXED — examples/basic shipped dead-in-browser**: every index-page
   form (Add/Greet/Calculate) posts urlencoded (native htmx) while all three
   handlers used `cqrshtmx.DecodeJSON*` → 400 on every submission, invisible
   because `basic_test.go` only posted JSON node-side. Fixed by switching the
   three page-form handlers to `DecodeForm`/`DecodeFormTyped`/`DecodeFormQueryTyped`
   (root's browser-true path; JSON decode stays showcased by `/api/audit` and the
   QUERY-method search twins), tests + README truthed same-change. The battery's
   first root-surface spec caught a shipped-dead example on its first run —
   exactly the class it exists for.
4. **M8 — examples/setup-demo as e2e surface** (`setup-bundle.spec.ts`, 6/6 green):
   `SETUP_DEMO_ADDR` env override, fifth webServer entry (`:18933`), spec covering
   session gates (401 anonymous on `/admin/` + `/dashboard/`), `/health` readiness
   JSON (projections + sse-hub ok), dev-login round-trip, both panels under
   session, the dual-transport broadcast hub, console guard.
5. **BUG #3 FOUND+FIXED — setup-demo `/ds-demo` page was triple-dead in-browser**:
   (a) no `data-nonce` on `<html>` → DataStar's expression engine (Function
   constructor) blocked by the bundle's nonce CSP (no unsafe-eval) — fixed by
   stamping `httputil.NonceFromRequest(r)` into the page; (b) no
   `data-init="@get('/ds/events')"` → the page never subscribed to the feed at
   all; (c) `data-signals-broadcasts="0"` per-key form → switched to the verified
   object-expression form. `httputil` promoted to a direct dep of the example.
6. **BUG #4 FOUND+FIXED — htmx's injected indicator stylesheet CSP-errors under
   the setup bundle**: htmx injects a nonce-less `<style>` at DOMContentLoaded
   unless `includeIndicatorStyles:false`; under the bundle's `style-src 'self'
   'nonce-…'` every admin/dashboard page load logged a blocked-style console
   error. Both panels ship their own indicator CSS, so both layouts
   (`adminui/layout.templ` `headExtras`, `dashboardui/layout.templ` head) now emit
   `<meta name="htmx-config" content='{"includeIndicatorStyles":false}'/>`
   (htmx reads it at DOMContentLoaded, so head placement suffices). `_templ.go`
   regenerated with the canonical pinned v0.3.1100 generator; both module suites
   green; dashboard+admin specs 19/19 green.
7. **M9 — dashboard depth** (`dashboard-live.spec.ts`, 5/5 green): e2e/server now
   wires `EventBus: eventtest.NewFakeBus()` + explicit
   `StreamReader: listing.NewInMemoryStreamReader(dstore)` (the auto-derive does
   not fire for this config shape — the time-travel page rendered its
   "Configure a StreamReader" empty state) + **`POST /api/debug/emit`** (one
   distinctive `E2e.debugEmitted` event into journal AND bus). Spec: live append
   (row lands at top of `#events-tbody` without reload), journal-replay honesty
   (fresh load still lists it — the counterpoint to datastar's live-only feed),
   multi-tab fan-out, time-travel slider (input-event display update + arrow-key
   navigation to `?v=1`), console guard.
8. **BUG #5 FOUND+FIXED — dashboardui time-travel slider dead under boosted
   navigation**: the slider wiring bound listeners to the element at script
   load; the app-layout's `data-hx-boost` swaps pages without re-evaluating
   dashboard.js, so arriving at the detail via in-app click left the slider
   inert (worked only on direct load). Fixed by delegating `input`/`change` at
   document level (re-query per event, like the existing keydown handler); also
   fixed the keydown handler's synthetic `change` to bubble (a non-bubbling
   Event never reached the delegated listener).
9. **dashboard.spec journal-tolerance**: exact-count assertions (`#stat-total-events`
   = "4", events rows = 4) would redden once dashboard-live's emits land in the
   shared journal (dashboard-live sorts first alphabetically); lowered to
   >= 4 with a comment. Doc comment updated to point at dashboard-live.spec.
10. **Full battery: 80 passed / 0 failed / 2.7 min** (98 tests total; 18
    screenshots tests skip under battery load — pre-existing probe shape; the
    09-13 sync-tail failures did NOT reproduce this run).

## b) PARTIALLY DONE

1. **M10 — bridges smoke (health + auditlog)**: `e2e/server/bridges.go` written
   and VERIFIED LIVE — `hostStatusProvider` adapter (Host →
   ProjectionStatusProvider), `/bridges/health/ui` (health.NewDashboard over
   cqrshealth.NewProbe, 200 + "Health Dashboard" title), `/bridges/auditlog/`
   (cqrsauditlog.WithAuditLog + do.NewWithOpts + one provided service, 200 +
   viewer title), `POST /api/debug/invoke` (202; invoking the service makes the
   plugin record → viewer streams). Server builds + boots; **the
   `bridges.spec.ts` smoke spec is NOT written yet**.
2. **Flake measurement**: one clean full-battery data point collected; no second
   run, no root-cause work on the sync-tail/under-load-skip class.

## c) NOT STARTED

- **M11** — loginpage-surface logout/`/auth/me` posture spec (e2e/server extension).
- **M2** — the CI `e2e` job (bun + Playwright chromium + sibling-clone + SHA pins).
- **M4** — firefox project (E2E_BROWSER_PATH scoped to chromium, webauthn testIgnore).
- **M5** — plan §Execution-log receipts, e2e/README matrix, corpus/tail rituals
  for the plan edits (this report's own rituals ARE done).
- **M6** — logical commits + push + `.#push-and-watch`. **The SUPERB paste
  explicitly authorized detailed commits AND push for this train — unexecuted.**

## d) TOTALLY FUCKED UP (self-attributed failures)

1. **The stale-server trap — twice.** A leaked pre-change server held `:18923`
   and `reuseExistingServer: !CI` silently reused it, so I debugged a phantom
   (chased "slider still broken" and "bridges 404" against MY OWN old binary).
   Burned the better part of an hour across M9/M10. Mitigation now ritual:
   `pkill -x server; pkill -f '/tmp/e2e-server'; ss -tlnp | grep 18923` before
   every spec run — must land in e2e/README (M5).
2. **Overwrought code first draft**: `emitDebugHandler`'s response write was
   nonsense (`jsontext.WriteTo(osFile(w), …)` — no such helper); simplified to
   `fmt.Fprintf` before build.
3. **`toHaveHeader` on an APIResponse** — not a function in this Playwright
   version; fixed to a direct header map read.
4. **Wrong URL shape in dashboard-live.spec** (`/dashboard/events/` trailing
   slash → 404 page → "no rows"); existing specs use no trailing slash.
5. **Arrow-key sequence bug in my own spec** (ArrowRight from the probe-left
   value navigated to v2); fixed with an explicit value reset.

## e) WHAT WE SHOULD IMPROVE

1. **The e2e local loop needs a guard, not a habit**: a `pretest` hook or
   config-level port-collision assert so a stale `:18923` FAILS LOUDLY instead
   of silently testing yesterday's binary.
2. **Silent `DecodeForm` sharp edge**: a JSON body posted to a `DecodeForm`
   endpoint decodes as an EMPTY form (ParseForm ignores non-urlencoded bodies)
   → 201 with zeroed fields, no error. Hit it live in the OOB spec. Deserves a
   gotcha entry + possibly a content-type sniff upstream.
3. **Screenshot spec's availability probe skips under load** — 18 of 98 tests
   vanish from a full run without a red. Make the probe retry, or make skips
   loud (summary-level "18 SKIPPED" is easy to miss in `tail -12`).
4. **Bridge/example pages should get CSP-console CI coverage earlier**: bug #4
   lived because no spec asserted console cleanliness under the bundle's real
   CSP until M8.
5. **Dead-in-browser is a PATTERN, not bad luck** (datastar-demo, basic-forms,
   ds-demo — three surfaces, one genus: node-side tests can't see it). The CI
   e2e job (M2) is the systemic fix; prioritize it next session.

## f) NEXT (priority order, up to 50)

1. M10 finish: write `bridges.spec.ts` (health page renders demo-projection
   check; auditlog viewer renders; POST /api/debug/invoke → connected viewer's
   event surface gains an entry; console guard) + run green.
2. M11: loginpage logout flow + `/auth/me` posture spec on the e2e/server
   loginpage surface (401 after logout, 200 + identity before).
3. M2: CI `e2e` job — sibling-clone step, setup-go (go.work), setup-bun SHA
   resolved live via `gh api` with `# vX.Y.Z` comment, `bun install
   --frozen-lockfile`, `bunx playwright install --with-deps chromium`, suite
   run (config carries GOWORK/GOTOOLCHAIN/GOEXPERIMENT).
4. Flake: second full-battery run for a rate; then either raise the two sync
   tail specs' poll ceilings or split projects (sync serial / rest parallel);
   root-cause candidate: worker startup race under load (`getEvents()` → 0).
5. M4: firefox project; cache `/mnt/buildcache/playwright/firefox-1543`;
   testIgnore loginpage-webauthn (Playwright WebAuthn is Chromium-only);
   30-min kill point; promote to CI only if locally green.
6. M5: plan §Execution-log receipts (M1/M7/M8/M9 + bugs #2–#5); e2e/README
   matrix (spec × surface × server) + stale-server kill ritual + sanctioned-run
   warning; corpus/tail rituals for any .md growth.
7. M6: logical commits (e2e battery incl. example fixes / CI job / docs+plan+baselines;
   stage ONLY session files — siblings' `method_override_test.go`,
   `options_query_method_test.go`, `http-query-method.md` excluded), push via
   gotcha-50 protocol if the pre-push hook reds on the documented API-vs-tag
   class, then `nix run .#push-and-watch` for the CI job's first run.
8. `gofmt`/`nix fmt` scoped pass over session-owned files before commit (gotcha 53).
9. e2e/server: add the stale-port guard (config-level or README ritual first).
10. Root-htmx spec: assert the OOB span's swap strategy attribute shape
    (`hx-swap-oob="true"`) exactly (currently substring).
11. setup-bundle.spec: add anonymous `/ds/events` → 401 assertion (the session
    gate on the FEED itself, currently only implied).
12. dashboard-live: pin the SSE indicator state transitions (open → event →
    count badge) via `[data-sse-status]`.
13. axe sweep over the two new examples (:18932/:18933) — extend screenshots +
    axe specs to the new webServers.
14. Basic example: consider `POST /api/greet` returning its greeting into the
    DOM (today the greet form has no visible response target — half-dead UX).
15. Gotcha entry: DecodeForm silently decodes JSON bodies as empty forms.
16. Gotcha/README entry: the stale-webServer trap + kill ritual.
17. dashboardui: consider the same `data-nonce` page pattern for any future
    DataStar surface (the setup-demo fix is the reference implementation).
18. Plan: mark the parked pixel-regression baseline decision (needs pinned
    runner image) explicit in the next train.
19. loginpage-golden spec: re-baseline against the nonce-CSP-era output if the
    family bumps ever change inline script emission.
20. e2e: extract the shared "console guard" helper into a helper module
    (duplicated in 4 specs now — fine at 4, extract at 5).
21. e2e/server: emitDebugHandler could take a stream-type parameter for future
    specs (currently fixed `E2e` vocabulary — fine).
22. Consider a `check-e2e-ports` gate: fail if any expected webServer port is
    occupied at suite start (mechanizes item 9).
23. When CI e2e lands: set `retries: 1` behavior review (CI already retries 1;
    with the flake root-caused, consider 0).
24. M7 follow-up: assert `/api/items/paginated` + QUERY search twin from the
    browser spec (currently node-only coverage via Go tests).
25. Track the auditlog viewer's `servicesOverflow`/`eventsOverflow` signals in
    the M10 spec (they appear in the SSE patch stream).
26. `bridges.spec`: also assert the health dashboard's JSON API shape if it
    exposes one (verify surface during spec writing).
27. Consider examples/catalog-demo e2e once the upstream templ-era docserver
    release ships (TODO row 51b dependency).
28. Full battery third run after M10/M11 land; update the flake rate.
29. Docs: `docs/guides/datastar-integration.md` could reference the
    `data-nonce` requirement (nonce CSP section) — small truthing.
30. `docs/guides/authorization-system.md`: nothing observed this session; skip.
31. After CI green: pin the e2e job's Chromium version in CI for determinism.
32. Consider `--with-deps` fallback path for non-Ubuntu runners (nix runners
    would need E2E_BROWSER_PATH wiring in CI too).
33. Battery timing budget: 2.7 min local; measure CI delta when M2 lands.
34. The 18 screenshot-skips: after flake work, confirm they stop skipping.
35. e2e/README: document that `nix run .#e2e` is the sanctioned one-command
    path (config header says it; README must repeat it).
36. Plan receipt should name bug #2–#5 explicitly (the battery's ROI evidence).
37. TODO_LIST row for the DecodeForm content-type sniff question (upstream
    behavior decision — owner call).
38. TODO_LIST row: promote `check-e2e-ports` idea or drop it after item 9.
39. Consider a11y (axe) on the auditlog viewer + health dashboard (bridge
    surfaces are new to the battery).
40. When v5 planning resumes: the ds-demo fix pattern (`data-nonce` +
    `@get` subscribe) belongs in the DataStar guide's CSP section.
41. `examples/basic` README: the run section still says 8096 default — true,
    but add the e2e override env documentation (one line).
42. Verify `nix run .#e2e` still passes end-to-end after M10/M11 (the flake
    app wraps the same suite; cheap confidence).
43. After push: watch for the release-train gate's fresh-tag gotcha 27 on any
    new tags (none expected this train — no tags planned).
44. If the pre-push hook reds on API-vs-tag: gotcha-50 protocol, name it in
    the commit message.
45. `docs/gotchas.md`: the boosted-navigation JS-wiring trap (bug #5 class)
    deserves entry #56 — element-bound listeners die under hx-boost swaps.
46. Skill (`cqrs-htmx` SKILL.md): after M2 lands, add the CI e2e job to the
    verification-path section.
47. Consider `testIgnore` for firefox on `sync.spec.ts` if IndexedDB/SW shapes
    diverge (measure first).
48. The two carried owner questions below block M2's final shape and M4's
    promotion — answering them early next session saves a rebuild.
49. After this report's rituals: run `check-docs-links` (new file = new links).
50. Next status report after M2+M6: include CI's first e2e run verdict.

## g) QUESTIONS FOR THE OWNER (cannot be resolved from the tree)

1. **Sync-tail flake strategy** (carried from 09-13, now with one clean data
   point): retry-mask the two tail specs' poll ceilings, or root-cause the
   worker-startup race first?
2. **CI e2e budget** (carried, now blocking M2's exact shape): ~4–6 min e2e job
   per PR, or train-lane-only (nightly/manual)?
3. **Firefox promotion** (carried, blocks M4's CI line): promote to CI once
   locally green, or keep local/nightly-only?

## Ritual receipts

- `docs/status/TAIL_BASELINE` 13 → 14 (this report, fourteenth live).
- `docs/CORPUS_BASELINE` re-trued to measured same-change (this report + its
  manifest row; ledger sentence appended inline in `docs/status/README.md`).
- The SUPERB plan's §Execution log receipts are still PENDING (M5 — this report
  is not that receipt).
