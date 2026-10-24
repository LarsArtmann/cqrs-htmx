# Status Report 2026-10-17 08:55 — README Rewrite & Self-Review

> **ANNOTATED 2026-10-24** — ~~this report~~ **SUPERSEDED** by [`2026-10-17_10-41_docs-truth-wave-execution-halfway_status.md`](../2026-10-17_10-41_docs-truth-wave-execution-halfway_status.md), which harvested every §f item of this report into routed TODO/ROADMAP rows (M01) and shipped the quick-start gate it recommended (M02). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-24 (round-23 (2026-10-24)).**

**Session scope:** the "make README.md superb" task (single session, ~2026-10-17 08:30–08:55). This report covers only what that session did, verified, broke, and noticed in passing. No other repo state was researched.

**TL;DR:** README rewritten 1266 → 507 lines from a drifting reference manual into a verified sales page. Quick start compiled AND smoke-tested live. All docs gates green. Work captured by the auto-commit daemon in `554c6d0f`, `f6657873`, `0db4635f`. Three honest mistakes made and fixed mid-session (two broken links shipped to disk, a known-gotcha violation, three wasted test cycles on occupied ports).

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| 1 | Research pass: full read of old README (1266 lines), project skill, both docs gates' logic; ~40 API claims verified against source (`command.New`, `RegisterTyped`, `SyncPullHandler`, `NewJournalSSEStore`, `MustNew`, `HealthHandler`, `Stop`/`Close`, `RecommendedSecurityMiddleware`, `OOBHTML`, `CommandTyped`/`QueryTyped`, `openapi.New`/`WithOpenAPI`/`OpenAPISpecHandler`, codemod `-apply` flag from `cmd/cqrs-htmx-upgrade/main.go`, `StreamID.Get()` in go-branded-id source) | Session log; every symbol grep-verified before writing |
| 2 | New README.md written (507 lines): hero + tagline, sharpened Why, grouped feature catalog, Install, Quick start, composition model + ordering rules, decision-tree "choose your path", 14-module map, realtime/SSE/sync, error mapping, ready-made UIs + setup one-call, 13-example index, 26-guide documentation index, versioning + codemod, contributing | `git show 0db4635f` (daemon commit carries it) |
| 3 | Quick start **compile-verified**: extracted the exact fenced block from the README into a temp module (`GOWORK=off`, local replace to the repo tree, deps from proxy) — builds clean | `FINAL_BUILD_OK` in session log |
| 4 | Quick start **runtime-verified**: server ran; `POST /items` → 201, `GET /items` → `[{"id":"01M54A0BFVC2C9DFC8X8000003","name":"Buy milk"}]`, `GET /htmx.js` → 200 `text/javascript` 72513 bytes | Live smoke test, session log |
| 5 | All relative links verified | `check-docs-links.sh`: 578 links OK (was 471 pre-rewrite) |
| 6 | All 14 module import paths verified against the go.mod files (not from memory) | Session log |
| 7 | Tables normalized with prettier (only table alignment + one tab in an HTML fence changed; code blocks byte-identical — re-extracted and rebuilt after formatting) | diff + rebuild in session log |
| 8 | Gates green after all edits: `check-docs-links.sh`, `check-docs-freshness.sh` (incl. /v4 import-path rule), `check-large-files.sh` | Session log, final run 08:53 |
| 9 | AGENTS.md gotcha 10 corrected: treefmt has NO markdown formatter in this repo's flake config (the "dprint/prettier for md/json" claim was false — `.#fmt -- README.md` formatted 0 files); documented the `npx --yes prettier --write` on-demand path | `git show f6657873`/`0db4635f` |
| 10 | AGENTS.md gotcha 20 amended: README posture recorded as sales page; future sessions instructed NOT to re-grow exhaustive reference tables into it | same commits |
| 11 | Temp artifacts cleaned (`/tmp/readme-qs-*` modules, logs, pid); no stray files in repo tree | `git status` clean after daemon commits |

## b) PARTIALLY DONE

| # | Item | What's done | What remains | Effort |
| --- | --- | --- | --- | --- |
| 1 | README accuracy verification | ~40 spot-checks against source; all structural claims verified | Residual claims trusted from the OLD README (which no gate verifies): account-lockout config details, ACK protocol specifics, `JournalSSEStore` mapper nuances, "27 middleware factories" count (AGENTS.md-sourced, not recounted from `go-cqrs-lite/middleware`), "21 event payloads / 20 commands" identity-model counts (drift-prone on next domain change) | S |
| 2 | Section (f) of this report → TODO_LIST/ROADMAP harvest | Report written (this file) | HARVEST not run — user ordered WAIT FOR INSTRUCTIONS after the report. The 45 items below die in this timestamped file unless harvested | S |
| 3 | Markdown formatting hygiene | README normalized once via unpinned `npx prettier` | No formatter wired into treefmt or any gate (repo-wide gap, not README-specific) — nothing prevents future md drift | M |
| 4 | README internal anchors | Two `#ready-made-uis--the-one-call-setup` links hand-derived per GitHub slugger rules | Not machine-verified (no local GitHub-flavored slugger); renders on assumption | S |

## c) NOT STARTED

| # | Item | Why not started |
| --- | --- | --- |
| 1 | CHANGELOG receipt for the rewrite | Deliberate per policy A8.3 (docs-only change → receipt in touched doc, not CHANGELOG). Listed here for visibility, not as debt |
| 2 | pkg.go.dev re-verification | The badge/link assumed valid; module docs render on pkg.go.dev takes days post-tag; out of session scope |
| 3 | README screenshot / demo GIF | Never attempted — owner aesthetic call (question g-1) |
| 4 | `examples/README.md` index page | Examples linked via per-dir READMEs + one table in README; a dedicated index was noticed-missing, not built |
| 5 | Sub-module READMEs for `usermgmt/totp`, `usermgmt/webauthn`, `usermgmt/oauth2` | Don't exist (discovered when my first draft linked them); README now links the directories instead |
| 6 | Codespell / markdownlint pass over the new README | BuildFlow has codespell on-demand; not run this session |

## d) TOTALLY FUCKED UP!

Nothing in the final deliverable is broken — all gates green, code verified. But the SESSION had five real, honest failures:

| # | Failure | Severity | Root cause | Mitigation |
| --- | --- | --- | --- | --- |
| 1 | **Shipped two broken links to disk**: first draft linked `usermgmt/totp/README.md` and `usermgmt/oauth2/README.md` — files that do not exist. Caught ~10 min later only because I ran an existence check before the gates; the links-check gate would also have caught it, but the daemon could have committed the broken state in between | Medium (was on disk ~10 min; daemon commits within minutes — got lucky) | Wrote links from assumption ("sub-modules surely have READMEs like their parent") instead of checking first — violated the session's own verify-first discipline | Fixed to directory links; existence now pre-verified for every linked path |
| 2 | **Known-gotcha violation in first quick-start draft**: used `id.NewStreamID().String()` — output showed `"StreamMarker:01M54..."` because gotcha 25 (display vs identity form) is documented in AGENTS.md and I had READ it that morning | Low (caught in own smoke-test output review) | Applied the rule to storage/identity sites in past sessions but reflexively used `.String()` for a JSON demo field | Changed to `.Get()`; smoke test re-run clean |
| 3 | **Three wasted verification cycles on occupied ports**: first smoke test hit `:8080` (SigNoz), read foreign HTML as if it might be my server; second hit `:8099` (InboxClean, CSRF 403); third had a sequencing bug (port `sed` ran before re-extract). ~3 tool calls burned on test-harness bugs, not the deliverable | Low (time only) | (a) Interpreted client output before checking the SERVER log; (b) assumed a second common port was free; (c) mutation ordering in a compound command | Fixed by reading server log first + unique high port; lesson: on any bind-sensitive test, assert the server log BEFORE trusting client output |
| 4 | **Unpinned network tool in a hermetic-minded repo**: ran `npx --yes prettier` (downloads whatever version) to format the README — not BuildFlow's pinned prettier, not in any lockfile | Low (prettier md output is stable; BuildFlow re-formats staged files with its own version anyway) | Treefmt had no md formatter (discovered mid-session), and I reached for the fastest tool instead of stopping at "no canonical formatter exists" | Benign here; the real fix is f-2 (wire a formatter), not more ad-hoc npx |
| 5 | **AGENTS.md edit rejected once**: attempted multiedit having only seen the file via injected project context, not the View tool — the tool correctly refused; one extra round trip | Negligible | Assumed injected context counted as "read" for edit-tool purposes | Re-viewed the exact lines, then edited successfully |

**Structural fuck-up worth naming (design-level, not caught by any gate):** the rewrite silently breaks every EXTERNAL deep link to old README anchors (`#user-management-usermgmt`, `#sse-server-sent-events`, …). Internal consumers are unaffected (docs index rebuilt), but anything on the internet pointing at `README.md#section` now 404s. GitHub offers no anchor-redirect mechanism; the honest options are HTML `<a id>` stubs or accepting the break. Decision deferred to owner (f-10).

## e) WHAT WE SHOULD IMPROVE!

| # | Pattern | Impact | Concrete fix |
| --- | --- | --- | --- |
| 1 | **README drift is invisible to CI.** The old README had silently-wrong architecture claims (file tree listing `csrf_middleware.go` as root implementation when CSRF is a deprecated httputil re-export) and under-sold major features (OpenAPI builder absent entirely). Nothing failed | Consumers read lies; features rot undiscovered | The rewrite fixes content once; f-1 (compile-the-quick-start gate) + f-9 (feature-claim sweep) mechanize the class |
| 2 | **No markdown formatter is wired anywhere** (AGENTS.md claimed one existed — corrected this session, but the gap itself remains) | Table/style drift accumulates; every session hand-rolls formatting | f-2: add a prettier/dprint program to treefmt or a `check-md-format` gate |
| 3 | **Verification-by-smoke-test has no reusable harness.** The extract-from-README → temp module → build → run → probe loop was ~30 lines of throwaway bash, rebuilt 4 times in one session (see d-3) | Every future README/example change re-pays this cost | f-1: make it a CI step; then the harness exists exactly once |
| 4 | **Sub-module README coverage is uneven** (3 auth sub-modules lack one; 4 of 13 examples lack one) — the root README now papers over it with directory links | Consumers land on bare directories | f-4/f-5: one-paragraph READMEs, cheap |
| 5 | **AGENTS.md contains at least one claim falsifiable in 118ms** (`.#fmt` formatting markdown) that survived because nobody ran the counter-example | Sessions repeat wrong commands | Fixed this instance; systemic fix is f-25 (verify-external-claims pass over AGENTS.md tooling claims) |
| 6 | **Port hygiene on this machine**: 8080 and 8099 are permanently occupied by sibling services (SigNoz, InboxClean) | Any local server test silently tests the WRONG app if it doesn't check the server log first | Session-local lesson recorded in d-3; candidate for crush-config lessons if it recurs |

## f) Next tasks (45, ranked by impact)

Routing per docs-health HARVEST: **T** = TODO_LIST (actionable now), **R** = ROADMAP (idea/long-term), **D** = decision needed first. Impact: C critical / H high / M medium / L low. Effort: S <30min / M 30min-2h / L >2h.

| # | Route | Task | Impact | Effort | Category |
| --- | --- | --- | --- | --- | --- |
| 1 | T | HARVEST this report's (f) into TODO_LIST/ROADMAP (docs-health) | C | S | Documentation |
| 2 | T | Wire a markdown formatter into the repo (treefmt program or `check-md-format` gate + flake app + self-test, atomically per gotcha 19) | H | M | Quality |
| 3 | T | CI gate: extract README quick start → `GOWORK=off go build` in a temp module (kills the README-drift class at the root) | H | M | Quality |
| 4 | T | Re-verify residual README claims from the old text: lockout config surface, ACK protocol details, `JournalSSEStore` mapper contract | M | S | Documentation |
| 5 | T | One-paragraph READMEs for `usermgmt/totp`, `usermgmt/webauthn`, `usermgmt/oauth2` (currently bare dirs) | M | S | Documentation |
| 6 | T | One-paragraph READMEs for `examples/admin-demo`, `examples/dashboard-demo`, `examples/middleware-showcase`, `examples/system-demo` | M | S | Documentation |
| 7 | T | `examples/README.md` index (one line per demo + port) | M | S | Documentation |
| 8 | T | Link `CHANGELOG.md` from README (repo has one; README never mentions it — noticed post-hoc) | M | S | Documentation |
| 9 | T | Feature-claim sweep: diff README "What you get" vs FEATURES.md quarterly (or gate-able: cross-link every claim to a guide) | M | M | Documentation |
| 10 | D | Old-anchor deep links: add HTML `<a id="...">` stubs for the ~10 most-likely externally-linked old headings, or accept the break | M | S | Documentation |
| 11 | D | README demo asset (screenshot or short GIF from `examples/setup-demo`) — see question g-1 | M | M | Documentation |
| 12 | T | Machine-verify the two internal README anchors (GFM slugger rules) — trivial script or eyeball on next push | L | S | Quality |
| 13 | T | Run `buildflow -s codespell` over README + docs (on-demand step, never run on the new text) | L | S | Quality |
| 14 | T | Verify "27 go-cqrs-lite middleware factories" count against `go-cqrs-lite/middleware/v4` source; make it non-numeric ("every factory") if it drifts | M | S | Documentation |
| 15 | T | Same for identity-model counts ("21 event payloads, 20 commands") — these drift on every domain change; consider removing numbers from README | M | S | Documentation |
| 16 | R | `docs/reference/handler-options.md` — single canonical HandlerOption catalogue (old README tables were the only place it existed complete; now nowhere) — see question g-3 | H | M | Documentation |
| 17 | R | Security policy: `SECURITY.md` (repo has none — noticed while writing Contributing section) | M | S | Documentation |
| 18 | R | GitHub social preview image for the repo | L | S | Documentation |
| 19 | T | Verify `docs/guides/v4-to-v5.md` mentions the `cqrs-htmx-upgrade` codemod (README now features it; guide unverified this session) | M | S | Documentation |
| 20 | T | Verify `CONTRIBUTING.md` links `docs/guides/release-playbook.md` (maintainer onboarding; unverified) | L | S | Documentation |
| 21 | R | README "Add persistence in 3 lines" teaser after quick start (SQL store one-liner) | M | S | Documentation |
| 22 | R | Trim duplication: README setup snippet vs `setup/README.md` quick start are near-copies (two copies = drift risk) — link one, keep the other short | M | S | Documentation |
| 23 | R | Error-handler variant table (`DefaultErrorHandlerWithRequestID` family) — dropped from README, only lives in pkg.go.dev now; fold into f-16 reference doc if built | L | S | Documentation |
| 24 | R | usermgmt `EventHandler` callback + domain-events docs — dropped from README; verify `usermgmt/README.md` covers it, else add | M | S | Documentation |
| 25 | R | verify-external-claims-style audit of AGENTS.md "Quick Reference" command claims (the md-formatter falsity proves the class) | H | M | Quality |
| 26 | T | Confirm daemon heuristic commits `554c6d0f`/`f6657873`/`0db4635f` carry no unintended extra files (spot-check `git show --stat`) | L | S | Quality |
| 27 | R | README TOC decision: 507 lines, GitHub auto-TOC exists; a manual TOC adds drift (decide NO if unchanged in a month) | L | S | Documentation |
| 28 | R | Badge for test coverage / module count (needs CI publishing endpoint — likely not worth it) | L | M | Documentation |
| 29 | T | Add explicit link for ADR 0046 (WS removal) in README Realtime section (currently plain text while ADR-0055/0056 are links — inconsistent) | L | S | Documentation |
| 30 | R | Review `usermgmt/README.md` + `setup/README.md` for the same reference-manual bloat disease the root README had (both are long; same treatment could apply) | M | L | Documentation |
| 31 | R | `docs/README.md` landing page for the docs tree (guides index exists only inside README) | L | S | Documentation |
| 32 | T | Confirm the `/htmx.js` route name choice in README (`GET /htmx.js`) matches the majority of examples (consistency for copy-pasters) | L | S | Documentation |
| 33 | R | README FAQ from the skill's gotcha list (validation-after-decode, module suffixes, htmx double-load) — or explicitly keep gotchas in docs only | L | M | Documentation |
| 34 | R | Show example ports inline in the examples table (currently "documented port" per dir) | L | S | Documentation |
| 35 | T | Check whether pkg.go.dev rendered v4.14.1 docs (proxy lag can be days; if stale, nothing to do but wait) | L | S | Documentation |
| 36 | R | Investigate `identity-model/README.md` completeness (pure-domain module; consumers rarely need it — verify it explains WHEN to import directly) | L | S | Documentation |
| 37 | R | Consider English copy review of module READMEs by a native tone pass (consistency of voice across the 10 module READMEs) | L | M | Documentation |
| 38 | T | Note in `docs/status/README.md` live-tail count if this report changes it (27 → 28 live reports; the table's count is hand-maintained) | L | S | Documentation |
| 39 | R | Automate f-9 as a script: grep README feature bullets for guide links, assert each named symbol exists in source (lightweight claim-lint) | M | M | Quality |
| 40 | R | Add "GOTH stack" explainer doc and point the badge + callout at it (currently badge → templ-components repo only) | L | S | Documentation |
| 41 | T | Re-run `check-docs-tail-budget` (advisory) — not run this session; README grew the docs tail | L | S | Quality |
| 42 | R | Decide canonical home for "Config field" documentation (old README table dropped; Config docs now only in pkg.go.dev + guides) — folds into f-16 | M | S | Documentation |
| 43 | T | Spot-check rendered README on GitHub after next push (anchors, decision-tree box characters on mobile) | L | S | Documentation |
| 44 | R | Extract the smoke-test harness (server + Go HTTP client + unique port + log-first assertion) into `scripts/selftests/` for reuse | M | S | Quality |
| 45 | R | Record the occupied-ports lesson (8080 SigNoz, 8099 InboxClean) in crush-config `references/lessons.md` if it recurs cross-project | L | S | Process |

## g) Questions (3 — cannot be answered from code, docs, or policy)

1. **Demo asset in the README?** Do you want a screenshot or short GIF (from `examples/setup-demo`, the admin panel, or the sync demo's offline walkthrough) as the visual anchor, or keep the README code-only? Both are defensible; it's your shop window. (If yes: f-11, effort M.)
2. **Docs voice:** the rewrite leans slightly marketing ("stop writing glue", "killer demo" energy restrained but present). Keep this register for consumer-facing docs, or tighten to pure engineering terse throughout (module READMEs currently vary)?
3. **Reference-tables revival:** the exhaustive tables (every HandlerOption, CSRF/rate-limiter config fields, Service methods, response-builder methods) previously lived in the README and now live nowhere consolidated — only pkg.go.dev + scattered guides. Should I build `docs/reference/handler-options.md` as the canonical catalogue (f-16), or is pkg.go.dev the deliberately-sufficient home?

---

*Point-in-time snapshot. Per convention: annotations on this file, when it is later verified, go inline with a dated `> ANNOTATED YYYY-MM-DD` blockquote; unmarked items in (f) are the open signal. Section (f) routes: T → TODO_LIST, R → ROADMAP, D → decision pending.*
