# Status Report: Time-Travel Real-Data Fix + Foreign Merge-WIP Unblock

**Date:** 2026-11-24 06:09 CET
**Scope:** This session's work in cqrs-htmx (dashboardui time-travel UX, prompted by the renamer consumer's "useless" time-travel page) + the foreign `identity-model` merge-event WIP breakage this session had to unblock.
**Trigger:** `https://renamer.home.lan/cqrs/time-travel/File/StreamMarker:…?v=9` rendered only event types + timestamps with a full-page-reload slider — "useless".

## Verification Scope (what this report backs, and what it does not)

**VERIFIED this session, with commands:**
- dashboardui suite + race: `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` → green (module root + core).
- Lint: `golangci-lint run` (dashboardui) → green after one golines reformat.
- Gates: `nix run .#check-templates`, `.#check-css-bundles`, `.#check-css-bundle-classes`, `.#check-codegen` → all green.
- Coverage: `bash scripts/checks/check-coverage-threshold.sh dashboardui` → 73.4% (threshold 60).
- E2E: `playwright test dashboard-live.spec.ts --project=chromium` (with the nix `playwright-browser-libs` forest + managed chromium 1255) → **5/5 passed**, including the extended time-travel spec (post-swap step row expanded with payload, exactly one step row) and the console guard (no JS errors from the new wiring).
- Workspace build: `go build ./...` (go.work root) → green. identity-model build + tests → green.

**NOT verified (and why):**
- **Firefox engine** — chromium leg only; the firefox advisory battery was not run.
- **`nix run .#check-modules` composite** — only the legs this change touches were run individually; the full composite (incl. release-train, version drift, docs freshness, erraudit, cqrs-lint) was not.
- **erraudit / cqrs-lint deltas** — new dashboardui code was not swept against either advisory surface.
- **CI** — nothing pushed; the pre-push hook + CI verdicts are unexercised.
- **Renamer consumer** — NOT bumped; the live page at renamer.home.lan still runs the OLD dashboardui. Everything below describes library-master truth only.
- **a11y (axe) on the new markup, dark-mode visual of `step-row`, back-button history semantics after `hx-sync="replace"`** — not exercised.
- **LSP project diagnostics** — the IDE kept reporting 5 templ/gopls errors ALL session; build + test + lint prove them stale (gotcha 14), left unchased.

---

## a) FULLY DONE

1. **Diagnosis** — live-fetched the renamer time-travel page: confirmed the failure shape (types + timestamps only, no payloads; slider = `window.location.href` full reload per step; subtitle promising "state at each step" with no state anywhere).
2. **Payload column** (`dashboardui/timetravel.templ`, `handlers_timetravel.go`) — new `timeTravelRow` (event + rendered payload + `isStep`); `buildTimeTravelRows` renders every event's payload through the existing seam (`payloadBytes`: typed `RegisterPayloadType` registry → configured renderer → `DefaultPayloadRenderer` JSON/CBOR/diagnostic-fallback). Collapsed `<details>` per row; the step at the selected version is highlighted (`tr.step-row`) and expanded with a "this step" badge.
3. **htmx slider** — slider now carries `hx-get` + `name="v"` + `hx-trigger="change"` + `hx-target="#main-content"`/`hx-select`/`hx-swap="outerHTML"` (gotcha-17 self-pinning inside the boosted shell) + `hx-push-url="true"` + `hx-sync="this:replace"`. Deleted the document-level `change` → `window.location.href` listener; kept the delegated `input` display updater and the synthetic-change keydown path (both still e2e-pinned).
4. **CSS** — `step-row` / `cell-payload` / `payload-details` / `payload-pre` added to the module theme block; Tailwind bundle verified byte-identical class set (custom classes emit no utilities) — `check-css-bundles` + `check-css-bundle-classes` green, no rebuild needed.
5. **Codegen** — `nix run .#gen` (canonical templGen) + `.#check-codegen` green.
6. **Tests** — `dashboard_test.go` pins updated (hx-get/name="v"/payload/step-row/collapsed-details); NEW `timetravel_page_test.go` (golden `time_travel_detail_page.golden` + payload-column, step-row-exclusivity, htmx-slider specs); full suite + race green; coverage 73.4%.
7. **E2E** — `dashboard-live.spec.ts` extended: after ArrowLeft, the step row (`tr.step-row`) is open with `"seq": "1"` visible and exactly one step row exists post-swap. Chromium 5/5.
8. **Docs** — dashboardui README § Time-Travel Slider rewritten (partial swap, inline payloads, typed-renderer note); CHANGELOG [Unreleased] entry with the full rationale.
9. **Research answers delivered** — the "why own CSS" question (Tailwind IS used via compiled bundle; templ-components adopted at ~94/100; three documented divergences in ADR-0053 with revisit criteria; the custom block is theme layer + widgets the library lacks).
10. **Foreign-breakage unblock** — see (d)/(b): workspace build restored; identity-model build + tests green.

## b) PARTIALLY DONE

1. **Foreign identity-model merge-event feature** — the other session's WIP (user/tenant merge events) is committed everywhere EXCEPT the fold.go tail, which arrived corrupted AND incomplete. I unblocked it: removed the `arg_value><b88a6f17>` corruption token and added the two missing `UserState` fields (`MergedInto`, `MergedUserIDs`) + `IsMerged()`, mirroring that diff's own `TenantState` pattern. The FEATURE itself (deciders, commands, projections, usermgmt wiring, tests for merges) is NOT mine and NOT done — identity-model passes its tests, but nothing exercises the merge events yet.
2. **Consumer rollout** — the fix exists only in library master (daemon-committed across 4 auto-commits). The renamer needs: a dashboardui (family) train → pin bump → consumer CSS rebuild → live verification. Not started.
3. **docs-health HARVEST** — section (f) below is report-only; TODO_LIST.md was NOT updated (you instructed WAIT).

## c) NOT STARTED

- Firefox e2e leg for the new assertions.
- Full `nix run .#check-modules` composite / `train-preflight` before any push.
- Release train (no version bump, no tags).
- AGENTS.md / gotchas.md updates for this session's lessons (below).
- The 50-item list below is brainstorm, not commitment.

## d) TOTALLY FUCKED UP (and what it cost)

1. **The foreign `identity-model/fold.go` state** — corrupted token + missing fields = the ENTIRE workspace did not compile; the first e2e attempt died at webserver build. Root cause UNINVESTIGATED: which process wrote `arg_value><b88a6f17>` (concurrent session mid-write? daemon snapshot-restore? gotcha 47 class) is unknown. My repair was surgical and preserves their design, but the merge feature remains half-landed across the repo.
2. **My markup assertions were wrong three times in a row** — I asserted `&quot;` escaping (templ emits `&#34;`), `open=""` (templ emits bare `open`), and a bare-ULID `hx-get` segment (`StreamID.String()` is display-form, `StreamMarker:<ulid>` — gotcha 25, which I had literally re-read that hour). Cost: one failed test cycle before the `-update` golden revealed the real bytes.
3. **Two failed e2e launches before green** — (a) bare playwright run without the nix lib forest (`libglib-2.0.so.0` missing — I skipped the flake-app env contract AGENTS documents), (b) webserver build broken by the foreign fold.go.
4. **Stale LSP errors all session** — 5 phantom templ/gopls errors persisted in diagnostics despite green build/test/lint; acknowledged but never reset (LSP restart was never attempted — cheap experiment skipped).
5. **I noticed the dirty foreign file only when e2e broke** — `git status` showed `identity-model/fold.go` from early on; a competent session reads unexpected diffs IMMEDIATELY (the AGENTS rule), not after a downstream failure surfaces them.

## e) WHAT WE SHOULD IMPROVE (incl. what I forgot)

1. **Render-first-then-assert** — golden-mint (`-update`) BEFORE hand-writing substring assertions; never assert templ output from memory.
2. **Read unexpected diffs at first sight** — the foreign fold.go sat in `git status` for the whole session before it bit.
3. **Use the sanctioned runner first** — the flake e2e app exists precisely to wrap browsers; replicating its env ad hoc cost a failed run (then worked, and the cached `.drv` realize trick is worth keeping).
4. **Forgot: index-page subtitle still says "see the state at each step"** — I updated the detail page hint but not the index subtitle; with payloads shown the wording is *closer* to true but "state" is still a stretch. Reword.
5. **Forgot: my `UserState.IsMerged()` addition is untested** — no test pins it; if the owning session is dead, its tests won't exist either.
6. **Forgot: no intervention record** — the fold.go repair (corruption removal + fields) has no CHANGELOG/receipt anywhere outside this report; the eventual commit diff will show edits with no narrative.
7. **Forgot: gotcha candidates not recorded** — the corruption-token class and the templ output-form trio (`&#34;`, bare bool attrs, display-form StreamID in hx-get) belong in `docs/gotchas.md` per the write-it-down rule; not written.
8. **Payload size is unbounded in the table** — bytes render into the DOM even when collapsed; a huge payload bloats the page. Length-cap + "full payload on the detail page" link is the honest shape.
9. **Per-row `payloadBytes` cost** — every render decodes every event's payload; large stores × many rows = page latency. Bench or lazily fetch.
10. **Dark-mode + mobile rendering of the new table column unverified** — `color-mix` tint and the 4-column table on narrow screens need one visual pass.
11. **The aggregate timeline (`/aggregates/{type}/{id}`) didn't get the payload column** — consistent-by-omission, but the asymmetry is undocumented.
12. **Dead JS cruft** — `slider.dataset.sliderDisplay ||` fallback references an attribute I removed; harmless but should be cleaned.
13. **LSP hygiene** — an `lsp_restart` would have cleared the phantom errors; cheap and worth doing next time instead of narrating staleness.

## f) Up to 50 things to get done next (brainstorm — ROUTE through docs-health HARVEST; most are ROADMAP fuel, not commitments)

| #  | Item                                                                 | Tier |
|----|----------------------------------------------------------------------|------|
| 1  | HARVEST this list into TODO_LIST.md / ROADMAP.md (docs-health)        | now  |
| 2  | Resolve the foreign merge-WIP: is its session alive? finish-or-hand-back (deciders/commands/projections/tests) | now  |
| 3  | Pin `UserState.IsMerged` (and the fold merge cases) with tests        | now  |
| 4  | Investigate the `arg_value><b88a6f17>` corruption source (gotcha 47 class?) | now |
| 5  | Write the two new gotcha entries (corruption token; templ output-form trio) | now |
| 6  | Record the fold.go intervention receipt (CHANGELOG or agents-notes)   | now |
| 7  | Reword the time-travel INDEX subtitle ("state at each step")          | now  |
| 8  | Clean the dead `data-slider-display` JS fallback                      | now  |
| 9  | Train dashboardui (+ family as needed) so the fix is consumable       | soon |
| 10 | Bump renamer consumer + rebuild its Tailwind CSS + verify the live page | soon |
| 11 | Payload length cap in the table + link to full payload detail         | soon |
| 12 | Bench: timeTravelRows render cost with large payloads / many rows     | soon |
| 13 | Firefox e2e leg for the extended dashboard-live assertions            | soon |
| 14 | Full `nix run .#check-modules` composite before the next push         | soon |
| 15 | erraudit strict delta check on the new dashboardui code               | soon |
| 16 | cqrs-lint gate re-run on dashboardui (new code, per-module)           | soon |
| 17 | Dark-mode visual pass on `step-row` tint                              | soon |
| 18 | Mobile/narrow rendering pass on the 4-column table                    | soon |
| 19 | axe/a11y scan of the new payload-details markup                       | soon |
| 20 | e2e: back-button returns to the previous version (history contract)   | mid  |
| 21 | htmx loading indicator on the slider swap (slow stores)               | mid  |
| 22 | Payload column on the aggregate timeline (parity) or document the exclusion | mid |
| 23 | Screenshot of the payload-column page for the README                  | mid  |
| 24 | Re-check ADR-0053 revisit criteria (zero-JS mobile drawer) — this session WAS a dashboardui UI change, the re-check rides it | mid |
| 25 | Re-evaluate the `forms.Slider` rejection row (slider + htmx attrs now exist; does the library slider fit?) | mid |
| 26 | Build/test `buildTimeTravelRows` edge paths (raw encoding, empty payload, CBOR degrade) | mid |
| 27 | Update AGENTS.md (dashboardui section) with the inline-payload pattern | mid |
| 28 | Consider documenting the pattern in the cqrs-htmx SKILL.md (additive section) | mid |
| 29 | Sweep dashboardui JS for other full-reload navigation patterns        | mid |
| 30 | aria-live on `#slider-version-display` for screen readers             | mid |
| 31 | History-push + `hx-sync replace` interplay review (aborted requests must not push) | mid |
| 32 | TenantRenamed: verify a command/decider exists for the folded event (foreign-WIP completeness scan) | mid |
| 33 | Check whether merge events need upcasters/registry entries (upcaster policy) | mid |
| 34 | Run the docs-gates battery after README/CHANGELOG edits               | mid |
| 35 | Verify `check-readme-splitbrain`/`check-skill-claims` stay green post-edit | mid |
| 36 | Coverage ratchet opportunity: dashboardui 73.4% → raise threshold row | mid |
| 37 | Assert no `tr.step-row` when `?v=` exceeds max (clamped step row sanity) | mid |
| 38 | Time-travel for streams with >20 versions: slider-only path visual check | low |
| 39 | Consider `hx-boost`-independent fallback if htmx fails to load (progressive enhancement audit) | low |
| 40 | Payload column i18n/label consistency ("Payload" vs library vocabulary) | low |
| 41 | Export the payload-seam recipe for consumers (guide section)          | low |
| 42 | dashboardui README adoption table: add a row for the payload-details pattern | low |
| 43 | Consider collapsible METADATA in the table (currently detail-page only) | low |
| 44 | Delta view ("what changed v-1 → v") as a first-class panel (diff of consecutive payloads) | low |
| 45 | The metaengine→dashboardui envelope-bridge ROADMAP row: re-check relevance after this change | low |
| 46 | JournalSSE live-update interplay: an SSE event arriving while viewing v<N> should suggest "newer version available" | low |
| 47 | Copy-payload button per table row (detail page has one; rows don't)   | low |
| 48 | `check-e2e-ran-count` expectations after spec growth (loud-skip gate math) | low |
| 49 | Push-and-watch after the train; verify-remote-tags ritual             | low |
| 50 | Fleet-libs report refresh trigger: dashboardui changed — does the templ-components row need a touch? | low |

## g) Questions I cannot answer myself

1. **Is the identity-model merge-event session still ACTIVE?** If yes, I stay off those files and it owes the deciders/commands/projections/tests (plus my two fields are its contract now). If dead, should I finish the feature — and is `EventTenantRenamed` (folded but with no visible command anywhere I looked this session) part of its plan or itself a lost fragment?
2. **Should the payload column also ship on the aggregate timeline page**, or is time-travel-only the deliberate scope?
3. **Do you want the dashboardui train cut NOW** (tag + consumer sweep + renamer bump + CSS rebuild, so the fix lands on renamer.home.lan today), or do you want to eyeball the UX first (e.g. via a local `setup-demo` against your real store)?

---

*Point-in-time snapshot; auto-commit daemon holds the work as of writing. Report file itself was NOT harvested into TODO_LIST.md per your WAIT instruction.*
