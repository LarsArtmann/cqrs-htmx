# Status Report — dashboardui Integration Modes (Autodetect + Embeddable Layout Seam)

**Date:** 2026-10-04 13:14 CEST
**Session scope:** One feature stream only, per operator instruction — "make dashboardui trivially easy to integrate (ONLY a destination) AND composable for projects wanting tighter integration." Everything below is from this session's run; no unrelated research was done.
**Deliverable shape:** `.md` written at an explicit path demand — an override of the skill's HTML-canonical default (operator instruction wins; flagged per skill contract).

---

## a) FULLY DONE

Each item is verifiably complete: committed (auto-commit daemon), tests green, gates green.

| # | Work | Evidence | Scope |
|---|------|----------|-------|
| A1 | **`Autodetect(store any) (Config, error)`** — probes a store for all 11 go-cqrs-lite introspection interfaces (EventSource, EventByIDLoader, SeekableJournal, Journal, StreamReader, `*projectionhost.Host`, DeadLetterStore, CommandJournal, QueryJournal, SnapshotStore, EventBus) and returns a ready `Config`; rejection when no read interface exists. Kills the consumer type-assertion dance (verified real: InboxClean hand-writes exactly this dance in `cmd/inboxclean/web.go:284-294`). | `dashboardui/autodetect.go` + `autodetect_test.go` (7 tests), committed `1f2daac6`; full dashboardui module suite green | dashboardui |
| A2 | **`Config.Layout` embed seam** — `LayoutFunc` + exported `PageMeta` (title, brand, base path, capability-filtered `Nav` as `NavLink` with templ-components icon names, `CSSURLs`, `ScriptURLs`, nonce, capabilities) + `pageData.shell` wiring + `layout.templ` branch (HTMX partials bypass the shell; nil = byte-identical built-in document — proven by untouched goldens/a11y/CSP tests). | `dashboardui/layoutfunc.go`, `config.go`, `dashboard.go`, `layout.templ` + regenerated `layout_templ.go`; `layout_embed_test.go` (3 test groups: shell replacement, meta/nav/asset-URL data, HTMX bypass), committed `1b307e09`; `nix run .#check-codegen` → **PASSED** | dashboardui |
| A3 | **`setup.Config.DashboardLayout`** — the seam passes through the one-call bundle (`buildDashboardConfig`), with passthrough test asserting `bundle.Dashboard.Config().Layout != nil`. | `setup/config.go`, `setup/setup.go`, `setup_defaults_test.go`, committed `1b307e09`; full setup suite green (9.4s, workspace mode) | setup |
| A4 | **Docs:** README § "Integration Modes" (3-tier table: Destination / Embedded / Headless data, per-tier examples + shell contract incl. don't-load-htmx-twice, ToastContainer, error-shell exception); CHANGELOG `[Unreleased]` Added entry (duplicate DLQ bullet avoided — entry merged into one Added block); AGENTS.md dashboardui line carries the 2026-10-04 integration-modes memory; setup/README canonical config table gains `DashboardLayout` row. | README committed `42f7e1c7`/`b595ee24`; all docs daemon-committed; `nix run .#fmt` on all 4 md files → 0 pending deltas | dashboardui, setup, AGENTS.md |
| A5 | **Verification battery (blast radius):** dashboardui full suite green (incl. golden + a11y + CSP = default path unchanged); setup full suite green; `golangci-lint run` → **0 issues** both modules; root `go build ./...` workspace OK; `e2e/server` build OK; `examples/dashboard-demo` build OK; `integration_test`: 106 test groups run, `TestFullstackUI_DashboardRenders` **PASS**, `TestSSE_CrossModuleWireFormatContract` PASS. | Command outputs in session log, 2026-10-04 ~12:45–13:05 | cross-module |
| A6 | **Hygiene fixes on sight:** stale `[FromBundle]` godoc reference in `Dashboard`'s doc comment replaced with the real constructors (`New`/`MustNew`/`Autodetect`); `LogoutURL` doc now states it is ignored under a custom Layout. | `dashboard.go` doc comment, committed | dashboardui |
| A7 | **Foreign-work discipline:** concurrent-session churn identified and quarantined, never reverted — 42+ `scripts/checks|selftests|tools` repo-root refactor files, later `usermgmt/{go.mod,go.sum,mysql,postgres,sqlite}_setup.go,http.go` + new `usermgmt/session_gate.go`. Tree preflight (`nix run .#preflight-tree-check`) passed before edits. | `git status` at 13:10: only foreign usermgmt files remain uncommitted; zero of my files | repo tree |

---

## b) PARTIALLY DONE

| # | Item | Works now | Remains open | Blocker | Effort |
|---|------|-----------|--------------|---------|--------|
| B1 | **Cross-module release state** | Workspace-mode build + tests fully green (go.work resolves my new `dashboardui.LayoutFunc` locally) | setup's **hermetic** `GOWORK=off` build resolves the *published* dashboardui and fails on `Layout`/`LayoutFunc` until the next train | Release-train discipline (gotcha 6): tag dashboardui first, then bump setup. Mid-train bump commits fail strict gates **by design** | S (at train time) |
| B2 | **Full workspace battery `nix run .#test`** | Launched; dashboardui/core/setup-level modules that mattered verified per-module instead | Battery died in the **ROOT module** setup phase: `golang.org/x/tools@v0.51.0: missing go.sum entry` — foreign usermgmt-session go.mod churn mid-flight. Never re-run clean | Foreign session still mutating usermgmt; re-run on a quiet tree (`nix run .#wait-tree-quiet`) | S |
| B3 | **Integration-test coverage of the new seams inside this repo** | `TestFullstackUI_DashboardRenders` (setup-wired dashboard, built-in shell) passes | No in-repo integration test mounts the dashboard **with a custom Layout** behind consumer auth (my coverage lives in dashboardui's own suite) | Nothing — just not written this session | M |
| B4 | **Examples discoverability** | README documents the embed contract with a code sketch | No runnable `examples/` app demonstrating `Config.Layout` against a real consumer shell | Nothing | M |
| B5 | **Markdown formatting proof** | `nix run .#fmt` on the 4 touched md files emitted 0 changes | "emitted 0" means treefmt scanned nothing — clean-by-absence, not clean-by-check | Unclear which formatter (if any) covers md in this treefmt config; low stakes | S |

---

## c) NOT STARTED

Planned/designed during this session, consciously deferred — with the reason, and whether it is still wanted.

1. **Per-panel component exports (the "panels layer")** — the DiscordSync audit's middle layer (export `overviewContent`/`projectionHealthPanel`-grade components with stable data shapes so consumers embed single panels in their own templ pages). Deferred: bigger API-stability decision (exported data shapes = permanent surface); wanted — it is the *structural* integration tier beyond shell injection. **This is the main follow-up design conversation.**
2. **Headless JSON endpoints** (`/-/api/overview`, `/-/api/projections`, `/-/api/events`) — deferred because `core/` (importable pure-data) plus root's `ProjectionStatusHandler` already cover most data-integration needs; PapDashboard-class consumers asked for fragments+data, not REST. Still wanted as a cheap tier-3 add-on.
3. **Error/404 pages through the custom Layout** — documented as deliberately built-in-shell-only this session; routing `notFoundHandler`/`renderError` through the consumer shell would finish the chrome story. Wanted.
4. **Toast auto-hosting for embed mode** — embed shells must include `feedback.ToastContainer` themselves (documented); a helper or auto-injection was not built. Nice-to-have.
5. **Consumer follow-through** — telling InboxClean (and annotating the PapDashboard/DiscordSync feedback files) that the exact pain they filed now has a seam. Not started; zero-risk, high-trust value.
6. **HARVEST of section (f) into TODO_LIST/ROADMAP** — not run: operator said "THEN WAIT FOR INSTRUCTIONS". Flagged so it is not entombed in this timestamped file.
7. **Tags/CHANGELOG dates** — no version cut (NEVER-tag discipline; uncommitted-tree rule; train ordering in B1). Intentionally not started.

---

## d) TOTALLY FUCKED UP

Radical honesty, scoped to this session. **No user-facing breakage, no data loss, no broken consumers** — every failure below was caught by my own verification loop before yielding. The genuinely broken things in the tree (root-module battery failure, signing test) are *foreign-session* states, listed for completeness with attribution.

| # | What is broken/wrong | Severity | Root cause | Mitigation |
|---|----------------------|----------|------------|------------|
| D1 | **Two first-run test failures on a small feature.** (1) `TestAutodetect_MemoryStoreWiresReadInterfaces` asserted the memory store does NOT implement `SeekableJournal` — it does; I wrote the assertion from assumption instead of inspecting go-cqrs-lite's interface set. (2) `TestEmbed_LayoutReplacesBuiltInShell` asserted the content contains "Overview" — that text lives in the built-in header the test itself replaces; structurally incoherent assertion. | Low (caught in minutes) | Assertion design from memory, not from the counterparty artifact — the exact anti-pattern the templ-components skill preaches against ("verify integrations against the counterparty artifact") | Fixed same-session; lesson recorded in (e)-1 |
| D2 | **Three lint findings shipped at write time** in one small file (`cyclop` 15>12, `nonamedreturns`, gofumpt misformat in `assetURLs`). This repo's bar is lint-clean-at-write; I used the linter as a checklist instead of predicting its rules (probe chains need an upfront `//nolint:cyclop` with reason; no named returns; gofumpt return-block indentation). | Low | Speed bias; didn't run scoped lint until after docs phase | All resolved; 0 issues final. Better: lint after each file, not after each phase |
| D3 | **Wasted two verification cycles misreading `check-codegen`.** The gate says "differ from committed versions" — I read that as source↔generated drift and re-ran `gen` twice; the gate is *commit-relative* (`git diff --exit-code`) and cannot go green in a dirty tree. I have run this gate's sibling behavior in this repo for weeks and still didn't read the 20-line flake app first. | Low (time only) | Didn't read the gate implementation before interpreting its output | Understood + documented here; the sync property itself WAS proven (gen idempotent) |
| D4 | **Test-file compile error** — wrote `templ.ComponentFunc` with an anonymous `interface{ Write([]byte) (int, error) }` param instead of `io.Writer`; conversion refused. | Trivial | Careless signature recall | One-line fix |
| D5 | **Full battery launched into a mutating tree.** I started `nix run .#test` while the foreign usermgmt session was actively editing go.mod files; it failed on root-module go.sum churn and the run was wasted. My own gotcha-4 knowledge says verify tree quiet first (`wait-tree-quiet`). | Low | I scoped the blast radius manually instead, but the failed run produced a misleading "FAIL" headline | Blast radius covered per-module; battery re-run queued in (f) |
| D6 | *(Not mine — attribution for the record)* Root-module battery failure (x/tools go.sum) and `TestSigningEncryption_AuthzProjectionSurvivesCrypto` red are the foreign usermgmt/auth-hardening session's in-flight state + the known gotcha-24 signing-upstream class (go.work replaces `signing` with a live checkout). | High for whoever's CI runs next | Foreign session mid-mutation | Do not touch; re-verify after that session lands |

---

## e) WHAT WE SHOULD IMPROVE

1. **Assertions from artifacts, not memory.** Both D1 failures came from assuming the counterparty's shape. Concrete fix: before writing capability/interface assertions, `go doc` or grep the interface set — make this the reflex, it is already the written rule in the templ-components skill.
2. **Lint at file granularity, not phase granularity.** Running scoped `golangci-lint run` right after each new file (not after docs) would have collapsed D2's three findings into zero round-trips.
3. **Read the gate before interpreting the gate.** Every check script is <100 lines in `flake.nix`/`scripts/checks/`. New rule for myself: a failing gate gets its source read before a second execution.
4. **A seam without a runnable demo repeats the repo's discoverability disease.** `Config.Layout` is documented, but the repo's own history (grep-able adoption tables, `docs/research/` first) shows docs-only features get rediscovered the hard way. The embed pattern needs an `examples/` proof + a guide page (items F6/F7).
5. **`PageMeta.Nav.Href` is relative-to-BasePath** — every consumer must join paths themselves. Design review: absolute (BasePath-prefixed) hrefs, or both fields, is friendlier; changing later is a breaking API change, so decide before the first tag.
6. **LSP diagnostics stayed noisy/stale all session** (phantom `unused`/`gci` on clean files, phantom cyclop after the nolint landed) — gotcha 14 holds; CLI-only verification remains the only truth. No action needed beyond not regressing this discipline.
7. **Daemon-commit drift inside a working session:** my Go changes were committed by the daemon mid-session while docs were still pending, splitting one logical change across 3+ heuristic commits with useless messages. If attribution matters for this feature, consider a squash or a follow-up docs-only commit with a real message before the train tags.

---

## f) TOP 50 NEXT TASKS

Brainstorm per operator request ("up to 50") — a *brainstorm, not a commitment list*; most items beyond the first ~15 are ROADMAP fuel. Impact/Effort: Critical/High/Medium/Low — S <30min, M 30min–2h, L >2h.

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Tag dashboardui via `scripts/verify-tag.sh` carrying Autodetect + Layout seam (train wave 1) | Critical | S | Release |
| 2 | Bump setup's dashboardui require to the new tag; tag setup (train wave 2) | Critical | S | Release |
| 3 | Post-train: `GOWORK=off` go build/tidy/vet per touched module (hermetic proof) | High | S | Quality |
| 4 | Re-run full `nix run .#test` battery after `nix run .#wait-tree-quiet` | High | S | Quality |
| 5 | HARVEST section (f) into TODO_LIST.md (actionable) + ROADMAP.md (ideas) | High | M | Docs |
| 6 | Build `examples/dashboard-embed-demo`: consumer shell + `Config.Layout` + auth middleware, runnable | High | M | Feature |
| 7 | Write `docs/guides/dashboard-embed-guide.md` (CSP nonce flow, auth placement, toast contract, CSSURLs pitfalls) | High | M | Documentation |
| 8 | Design + ship the **panels layer**: export per-panel content components with stable data shapes (overview card, events table, projection health, DLQ list) | High | L | Feature |
| 9 | Add integration test: embedded dashboard (custom Layout) mounted behind consumer auth middleware in `integration_test` | High | M | Quality |
| 10 | Run `nix run .#coverage-gate` — dashboardui threshold holds with the new files | High | S | Quality |
| 11 | Run `nix run .#check-cqrs-lint` + `nix run .#erraudit-inventory -- --gate` over the new code | Medium | S | Quality |
| 12 | Golden-test the embed mode (pin a sample consumer-shell document) | Medium | S | Quality |
| 13 | CSP test for embed mode: nonce propagates through a consumer shell to panel scripts | Medium | S | Quality |
| 14 | End-to-end toast test: embed + ReadOnly=false + write action → `dashboardui:toast` HX-Trigger contract | Medium | M | Quality |
| 15 | Decide `NavLink.Href` relative vs absolute before first tag (breaking-after-tag otherwise) | High | S | Feature |
| 16 | Route `notFoundHandler`/`renderError` through the custom Layout when set | Medium | S | Feature |
| 17 | Auto-host or warn on missing `ToastContainer` in embed mode (silent toast loss today) | Medium | S | Feature |
| 18 | Headless JSON endpoints (`/-/api/overview`, `/-/api/projections`, `/-/api/events`) as tier-3 | Medium | M | Feature |
| 19 | ADR for the layout seam (why document-level injection, HTMX bypass contract, error-shell exception) | Medium | S | Documentation |
| 20 | Update `FEATURES.md` — honest inventory: embed seam + Autodetect as features | Medium | S | Docs |
| 21 | Update the cqrs-htmx skill (`SKILL.md` + references) with Autodetect + Layout usage | Medium | S | Docs |
| 22 | Annotate `docs/feedback/processed/2026-08-05_dashboardui-architecture-decomposition.md`: panels-layer partially addressed, seam shipped | Medium | S | Docs |
| 23 | Cross-link PapDashboard feedback: `DashboardLayout` + Autodetect change their calculus (items #1/#2 adjacent) | Medium | S | Docs |
| 24 | Adopt Autodetect in InboxClean (replace the 3-assertion dance) + drop `sseDeadlineFree`-class friction upstream | Medium | M | Feature |
| 25 | adminui parity decision: same `Layout` seam for adminui? (split-brain risk if the two panels diverge) | Medium | M | Feature |
| 26 | loginpage parity check (does the 2026-10-04 templ-components adoption already cover embed?) | Low | S | Feature |
| 27 | `ExampleLayoutFunc` godoc example (pkg.go.dev discoverability) | Low | S | Docs |
| 28 | Autodetect: interface inventory sweep against current go-cqrs-lite — any new probes missed? | Medium | S | Quality |
| 29 | Autodetect: `slog.Debug` the detected capability set (wiring observability) | Low | S | Feature |
| 30 | PageMeta: expose current page path/query (consumer active-nav beyond the `Active` flag) | Low | S | Feature |
| 31 | PageMeta: expose CSRF form token for shells that render write-action forms | Medium | S | Feature |
| 32 | README troubleshooting block: "unstyled panels = you skipped `meta.CSSURLs`" | Low | S | Docs |
| 33 | Repo-wide grep for stale godoc `[...]` references (the `[FromBundle]` class elsewhere) | Medium | S | Cleanup |
| 34 | dashboard.js under embed: auto-connect SSE when `/-/events/stream` route exists even without indicator element (or document the no-op) | Medium | S | Feature |
| 35 | Consider exporting `BuildNav(caps)` so custom shells can rebuild nav programmatically | Low | S | Feature |
| 36 | Rename `pageData.shell` → `pageData.layout` (Config.Layout naming consistency) | Low | S | Cleanup |
| 37 | Theme deep-cut: `Config.Theme` consumer `@theme` token passthrough (visual integration beyond chrome) | Low | L | Feature |
| 38 | Verify `check-css-bundles`/`check-css-bundle-classes` still green post-change (no new classes expected — prove it) | Medium | S | Quality |
| 39 | bench-spike: pin that the `layout()` branch adds zero measurable render cost | Low | S | Quality |
| 40 | README: add the 3-tier table to the module one-pager in AGENTS-referenced docs (docs/guides cross-link) | Low | S | Docs |
| 41 | Setup docs: add a `DashboardLayout` code snippet next to the config-table row | Low | S | Docs |
| 42 | Test: `Autodetect(nil)` and Autodetect-on-typed-nil messages render the `%T` store type usefully | Low | S | Quality |
| 43 | Confirm CI gates catch a daemon split-commit of `layout.templ` vs `layout_templ.go` (check-codegen in CI covers it — verify the CI wiring, not just local) | Medium | S | Quality |
| 44 | Squash/annotate the heuristic daemon commits for this feature before tagging (attribution hygiene) | Low | S | Cleanup |
| 45 | Consider `Config.Layout` accepting a small interface instead of func (mockability/future props) — API review before tag | Medium | S | Feature |
| 46 | Add an `Integration Modes` row to the dashboardui ↔ adminui cross-reference block in both READMEs | Low | S | Docs |
| 47 | Sweep setup `doc.go` for panel descriptions needing the embed mention | Low | S | Docs |
| 48 | e2e/server: optionally switch its dashboard mount to Autodetect (dogfood the ease claim in-repo) | Medium | S | Feature |
| 49 | Write the CHANGELOG date + cut notes when the train actually goes (item 1/2) | Medium | S | Release |
| 50 | After first real consumer adoption: retro the seam API against actual usage (rename/reshape window closes at v5) | Medium | M | Feature |

---

## g) TOP 3 QUESTIONS (cannot answer myself)

1. **Train timing:** Should the dashboardui → setup wave tag now, or ride after the in-flight identity-auth-hardening session lands its usermgmt work? I tried to infer intent from `git log` (the 2026-10-04 planning commit) and the tree state, but the other session's release intention is unknowable from here — and tagging wrong order poisons the module proxy (gotcha 5/6).
2. **API end-state for "tighter integration":** Is document-level shell injection (`Config.Layout`) the long-term integration surface, with the panels layer (item 8) as sugar — or should exported per-panel components become the primary API? This decides whether I invest in the panels layer (L-effort, permanent exported surface) now or after first real adoption.
3. **Panel-family scope:** Should adminui (and loginpage) get the identical `Layout` seam in the same train, or do you want the pattern validated by a real consumer (e.g. InboxClean) first? Same-seam-now avoids split brains; wait-and-see avoids two modules carrying an unproven API shape.

---

### HARVEST note (skill contract)

Section (f) is the primary input for `docs-health` → HARVEST into `TODO_LIST.md`/`ROADMAP.md`. **Not run this session** — operator instruction was to write the report and wait. Until harvested, these 50 items live only in this timestamped file.

### Appendix — verification commands (this session)

```
go test . ./core/ -count=1                 # dashboardui: ok (full suite)
go test . -count=1                         # setup: ok (9.4s)
golangci-lint run                          # dashboardui: 0 issues; setup: 0 issues
nix run .#check-codegen                    # PASSED
nix run .#fmt -- <14 touched files>        # Go: 2 cosmetic fixes; md: 0 deltas
go build ./...                             # root workspace: OK; e2e/server: OK; dashboard-demo: OK
go test . -count=1 -v                      # integration_test: 106 groups, 1 FAIL (foreign signing class)
nix run .#preflight-tree-check             # OK before edits
nix run .#test                             # ABORTED: root go.sum churn (foreign session) — see B2
```
