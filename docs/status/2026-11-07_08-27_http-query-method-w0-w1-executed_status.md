# Status — HTTP QUERY (RFC 10008) workstream: plan, re-review, recovery, W0+W1 executed

**Created:** 2026-11-07, 08:27 CET · **Session scope:** the QUERY-method workstream only (plan `docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md` + its W0/W1 execution). Everything below is what THIS session did or noticed in passing; no other workstream was researched or verified.

**Verified this session, with the artifact:** Go stdlib QUERY round-trip (live throwaway module), htmx 4 `hx-query` (four.htmx.org fetch), upstream DataStar `@query()` + our pinned bundle 1.0.3 lacking it (data-star.dev fetch + the minified action table in `go-datastar/static@v0.6.1`), OpenAPI 3.2 `query` Path Item field (spec text fetch), httputil csrf.go:789 safe-list (module cache source), `decodeJSONBody` no-method/no-Content-Type gate (`decoder.go:36-53`), `ReadSignals` QUERY-by-construction (`inbound.go:55-68`), full root suite `-race` (23.4s green), integration_test `-race` (green), examples/basic live PARITY-OK, BuildFlow `golangci-lint` step (0 findings in my files), skill-snippets 45/45, skill-claims 221, docs-links 1463, corpus ratchet (baseline 55012), todo-hygiene clean.

**NOT verified (and why):** `nix run .#coverage-gate` (not run — new root code has tests but no coverage receipt yet), `.#check-cqrs-lint` and full `.#check-modules` (not run — sibling session active in the tree, gotcha 41/51 load; the targeted gates above stand in), runtime browser behavior of htmx-2-with-QUERY (static proof only, NV5), proxy/WAF QUERY support (NV2, deliberately no folklore).

## a) FULLY DONE (this session)

1. **Assessment → verified fact base.** QUERY = the CQRS read side as an HTTP method; server side already worked (`App.Query` method-agnostic; stdlib routes `QUERY /path` — proven live). 16 facts S1–S8/C1–C6/P1–P4 with citations in plan §3.
2. **SUPERB plan** (256 lines after amendments): thesis, non-goals, 7 invariants, fact base, design (3-rung fallback ladder + middleware placement-as-security-decision + decoder-driven Accept-Query + RFC media-type ladder as opt-in), 6 waves W0–W5, 5 owner decisions D1–D5, verification receipts, honest errata.
3. **Brutal re-review** ("is it superb?"): found 4 real defects in my own plan — the OUTERMOST middleware instruction (would couple fallback auth to an upstream CSRF fix), AcceptQuery validation overpromise (no mechanism), C5 asserted-not-verified, C1 version detail second-hand. All fixed in place; recorded as plan §9.1 + erratum E5 (the W1.1 CSRF-token test promise conflated layers).
4. **Gotcha-47 recovery.** A sibling session's `Git Town WIP` stash captured the uncommitted plan + TODO row + baseline bump mid-session (strike three for gotcha 47). Recovered read-only via `git restore --source=stash@{0}` (stash untouched — it holds the sibling's work); provenance note in plan §9.1.
5. **W0 executed:** guide `docs/guides/http-query-method.md`; `examples/basic` `QUERY /api/items/search` + POST fallback twin (live byte-parity `PARITY-OK`; README curl recipes); `integration_test/query_method_test.go` (three-rung byte parity, empty-body zero-value, mux `Allow: QUERY` on 405 — published-API-only so the hermetic isolation leg stays green); TODO_LIST P1 row.
6. **W1 executed (root, additive v4 API):** `MethodOverrideMiddleware` (`method_override.go`; QUERY-only override target; clone-not-mutate; legacy `X-Http-Method` honored; `MethodQuery` constant), `RequireAnyMethod` (405 + `Allow`), `RequireContentType` (400/415 ladder) + `httperr.ErrUnsupportedMediaType` → 415, decoder-driven `Accept-Query` (JSON query decoders advertise `application/json` incl. error responses; form decoders advertise nothing; `AcceptQuery` appends). 20 new tests (7 middleware incl. the wire-POST placement proof, 8 options, 4 integration + parity).
7. **True-ups:** skill § QUERY subsection (fences parse-gated green), CHANGELOG [Unreleased] entry, plan W0/W1 rows marked DONE with receipts, NV5 resolved by static proof (`methodsThatUseUrlParams: ["get","delete"]` in the embedded 2.0.10 → QUERY rides the body path; runtime proof still owed by W2.4 e2e), corpus baseline 54613 → 55012 across three same-commit bumps.

## b) PARTIALLY DONE

1. **The workstream:** W0+W1 of 6 waves. W2 (htmx 4 embed + `cqrs-query` auto-downgrade extension) blocked on D3/NV4; W3 (DataStar fork bundle bump) on D4/NV1; W4 (`openapi.Query()` + 3.2.0 auto-bump, `ETagResults`, huma verdict) and W5 (httputil CSRF safe-list ask, verified proxy table) unstarted but unblocked.
2. **Verification battery:** targeted gates green (list above); the heavyweight legs (coverage-gate, cqrs-lint, full check-modules) not run this session.
3. **Root README:** the module README's feature list does not yet mention the QUERY surface (skill + guide do). Noticed while writing this report — not fixed.

## c) NOT STARTED

W2–W5 tasks verbatim in the plan; ADR-0064 (the ladder as library contract); the release train for the new root API (consumers cannot `go get` `MethodOverrideMiddleware` until root tags v4.18+); the examples-wide QUERY recipes (datastar-demo, middleware-showcase); the deprecated-`HTMXScriptHandler` SA1019 sweep (4 modules, pre-existing, noticed in the lint findings).

## d) TOTALLY FUCKED UP (honest ledger)

1. **I let the daemon be my only commit safety while a sibling ran Git Town.** The plan vanished mid-edit; 15 minutes of recovery work + one failed multiedit. Lesson recorded: under sibling-session load, verify files persist on disk after each phase (gotcha 4/47 interaction).
2. **First-draft test quality was churn, not craft:** one test file shipped with five defects in a row (missing `testing` import; invented `http.MethodQuery` — probed AFTER writing instead of before; `res.Status` vs `StatusCode` type error; a wrong `io` helper call; dead `doQuery` assignment), then a package-boundary mistake (referencing an unexported constant from `cqrshtmx_test`) and a `errors.New` format-args bug. Build/tests caught all of them, but 4 lint iterations followed (wsl/errcheck/forcetypeassert/canonicalheader). One careful pass checking signatures first would have been one iteration.
3. **The plan's original W1.1 done-when promised a CSRF-token refusal test that belongs to httputil, not to us** — conflation caught in re-review, recorded as erratum E5, replaced by the wire-POST placement proof.

## e) WHAT WE SHOULD IMPROVE

1. Signature-check-before-write for every new API touch (grep the definition first; I did this for `DecodeJSONQuery` and it paid off — skipped it for net/http constants and it cost a cycle).
2. Write wsl-clean Go the first time (blank line above an assignment after a non-assign statement) — the repo's linter is strict and the fixer's rewrites then touch pre-existing files (it reformatted `examples/admin-demo/main.go` and reordered an import in `examples/basic` — verified fine, but a fixer touching files outside my change is risk I invited).
3. Run the coverage/cqrs-lint legs for new root code in the same session, not "later".
4. Root README row for new public surface at landing time, not at next-touch time.
5. The lint-finding triage habit that worked: separate MY findings from pre-existing ones before fixing (the 4 SA1019s were correctly left alone).

## f) Next (up to 50, impact-ordered; 1–8 are the QUERY workstream's own tail)

~~1. W4.1 `openapi.Query()` — 4 touchpoints (`PathItem.Query`, `hasMethod`, `operations()`, constructor) + 3.2.0 auto-bump (test-pinned).~~ h	35e02bd3 — openapi/builder.go:161 func Query; 3.2.0 auto-bump pinned by openapi/query_test.go (CHANGELOG W4.1 entry)
~~2. Release-train decision: tag root v4.18 so consumers get the fallback ladder (or hold for W4 in the same tag).~~ h	6b804a01 — D18 train-GO; root v4.18.0 tagged with the CHANGELOG section cut
~~3. W2.3 embed htmx 4.x alongside 2.0.10 (`htmx/htmx4.min.js` + `ScriptHandler4()` + `Version4()`; default unchanged; NV6 re-verify the exact pin).~~ r	TODO_LIST.md HTTP QUERY row W2 lane (D1 GO per D18 — embed alongside, opt-in; default flip stays v5)
~~4. W2.4 `cqrs-query` extension (upgrade + auto-downgrade + sessionStorage memory, ~1 KB, CSP-safe) + browser e2e — gated on D3.~~ r	TODO_LIST.md HTTP QUERY row W2 lane (D3 GO per D18 — ships as a JS asset)
~~5. W2.2 NV4: htmx 4 extension compat for embedded SSE/idiomorph (verdict before W2.3 ships).~~ r	TODO_LIST.md HTTP QUERY row W2 lane (NV4 verdict owed before W2.3 ships)
~~6. W3.1 NV1: identify the upstream DataStar release that added `@query()`.~~ r	TODO_LIST.md HTTP QUERY row W3 lane (plan NV1 — pinned upstream 1.0.3 lacks @query())
~~7. W3.2 fork `go-datastar/static` bundle bump + wire-format goldens + treadmill sweep — gated on D4.~~ r	TODO_LIST.md HTTP QUERY row W3 lane (D4 per D18 — ride the releases adding documented actions)
~~8. W5.1 httputil ask + patch: QUERY joins the CSRF safe list (csrf.go:789); flip the CSRF-interaction doc when merged.~~ r	TODO_LIST.md HTTP QUERY row W5 lane (csrf.go:789 safe-list; flip the CSRF-interaction doc when merged)
~~9. W4.2 `ETagResults()` opt-in (FNV over rendered bytes; If-None-Match → 304; machinery exists in internal/hx).~~ h	036e59fe — options_render.go:136 ETagResults(); pinned by options_etag_test.go (CHANGELOG W4.2 entry)
~~10. W4.3 huma QUERY verdict (NV3) + README gap-note or recipe.~~ h	8fc6d6e4 — huma/query_method_test.go pins the NV3 verdict NO (huma v2.40.0 lacks QUERY)
~~11. W4.4 ADR-0064: the fallback ladder as library contract.~~ h	8fc6d6e4 — docs/adr/0064-http-query-fallback-ladder.md accepted (the ladder as library contract)
~~12. Root README row for the QUERY surface (guide link).~~ v	README.md:87 — the HTTP QUERY row with the guide link (:110 carries the openapi 3.2 auto-bump)
13. `nix run .#coverage-gate` receipt for the new root code.
14. `nix run .#check-cqrs-lint` + full `.#check-modules` once the sibling session's load clears.
~~15. W5.2 verified-only proxy table (Cloudflare/nginx/Envoy/Caddy/ALB; cite per row; unknown stays unknown).~~ r	TODO_LIST.md HTTP QUERY row W5 lane (the verified-only proxy table)
~~16. W5.3 v5 flags into ROADMAP (htmx 4 default flip, 422 validation status, extension retirement).~~ r	TODO_LIST.md HTTP QUERY row W5 lane (plan §W5.3 — ROADMAP v5-section rows)
~~17. examples sweep: datastar-demo QUERY recipe (`@post`+override now, `@query()` post-bump).~~ r	TODO_LIST.md HTTP QUERY row W3 lane (plan W3.3 — README recipes: @post+override now, @query() post-bump)
~~18. `nix run .#check-erraudit-strict` re-receipt (TOTAL was 319; new code adds).~~ v	re-run this session: TOTAL=326 (was 319 — the QUERY wave's +7 recorded)
~~19. docs-gates battery re-run (this report adds §f items; audit-all pin policy respected).~~ v	09-00 report §a14: battery 4/5 PASS on the final tree — audit-all owner-gated per the pin policy
~~20. SA1019 deprecated-`HTMXScriptHandler` sweep across examples + e2e/server (pre-existing, 4 modules).~~ v	zero HTMXScriptHandler uses remain under examples/ + e2e/ (verified this session; surviving refs are root's own deprecation examples + upgrade-tool tests)
21. `MethodQuery`-naming check against a future stdlib `http.MethodQuery` (alias plan if Go adds it).
22. Consider offering `MethodOverrideMiddleware` upstream to httputil after it proves general (D5 default says root-first).
23. Conditional-QUERY documentation deepening (Vary, equivalent resources, Content-Location) once `ETagResults` lands.
~~24. Getcha 47 hardening: a "verify my files still exist" habit (or hook) for sessions overlapping Git Town siblings.~~ v	habit recorded in three living places — gotchas.md #47 defense addendum, this report §d1's per-phase file check, 09-13 §e3's HEAD:<file> ritual

## g) Questions I cannot answer myself

~~1. **(D3)** Ship the htmx-2 `cqrs-query` auto-downgrade extension as a real JS asset in the `htmx/` package, or keep that package JS-version-pure and document the override recipe until htmx 4 is the default? Shipping JS is the showpiece; staying asset-minimal is the module's current shape.~~ v	D18 (TODO_LIST decision table): D3 GO — the extension ships as a real JS asset in the htmx/ package
~~2. **(D1/W2.3)** Is embedding htmx 4.x ALONGSIDE 2.0.10 in v4 acceptable now, or does the `htmx/` package stay single-version until the v5 window flips the default?~~ v	D18: D1 GO — htmx 4 embeds alongside 2.0.10 in v4 (opt-in); the default flip stays the v5 window
~~3. **(Release train)** Tag root v4.18 immediately so consumers can use `MethodOverrideMiddleware`, or hold the tag until W4 (`openapi.Query()`) lands in the same release?~~ h	6b804a01 — D18: train GO — root v4.18.0 tagged now; W4 rides v4.19
