# Status Report — PapDashboard Feedback Processing (setup non-adoption)

**Date:** 2026-09-29 11:46 (session started ~10:15)
**Session task:** Process `docs/feedback/new/2026-09-29_papdashboard-setup-non-adoption.md` — PapDashboard's ranked explanation of why it does NOT use the `setup` module (7 improvement asks), following the repo's feedback convention (new/ → act → annotate → processed/).
**Scope of this report:** this session's run only, as instructed.

---

## Session summary

The feedback file was read, cross-verified against source, and triaged. Of the 7 asks: **2 code seams + 1 upstream-mandated migration are implemented and tested**, 2 are doc deliverables, 2 are architectural items destined for ROADMAP. The session is mid-Phase-2-commit: the `nix fmt` verification tail is still running and the auto-commit daemon raced the phase-2 commit (content committed under a heuristic message; the two new test files are still untracked).

**Every feedback claim was verified against the tree before acting** — no claim was taken on faith:

| # | Feedback claim | Verification result |
|---|---|---|
| 1 | `setup.New` builds usermgmt unconditionally (setup.go:123) | ✅ Confirmed — `buildService` runs whenever `Config.Service` is nil; Disable* flags skip panels only |
| 2 | No consumer middleware seam | ✅ Confirmed — `Bundle.Middleware()` was security + logging only |
| 3 | No way to feed checks into `/health` | ✅ Confirmed — `healthHandler()` built only ProjectionReadinessCheck + HubReadinessCheck |
| 4 | SSE wire contract frozen both sides | ✅ Confirmed — transport envelope golden-pinned here; no encoder seam exists |
| 5 | Capability floor undocumented | ✅ Confirmed — facts exist in code (`transport/journalsse.go`, `dashboardui/config.go`) but no table in docs |
| 6 | codec/v4 deleted upstream, requires linger | ✅ Confirmed — go-cqrs-lite ADR-0128 deleted the shim; usermgmt imported it in 11 files |
| 7 | No setup-vs-hand-wiring decision doc | ✅ Confirmed — no such guide in `docs/guides/` |

One reference could NOT be resolved: the feedback cites "upstream #67/#68" for the codec cleanup — no issues #67/#68 exist in LarsArtmann/go-cqrs-lite (GraphQL lookup failed for both). Likely PapDashboard-internal issue numbers or a different tracker. The underlying work (codec/v4 removal) is real regardless (ADR-0128).

---

## a) FULLY DONE

1. **Feedback triage + full claim verification** (table above). All 7 items classified: #2, #3, #6 = implement now; #5, #7 = docs now; #1, #4 = ROADMAP with acceptance criteria.
2. **Feedback item 6 SHIPPED — usermgmt migrated off the deleted `codec/v4` shim onto `go-codec`** (commit `d8684d5c`):
   - 11 files import-swapped (`codec` package name identical, zero code changes beyond imports).
   - `usermgmt/go.mod`: `go-cqrs-lite/codec/v4 v4.4.0` require dropped; `go-codec v0.3.0` (already published, already a direct dep of identity-model) promoted to direct.
   - Verified: `GOWORK=off` tidy + build + vet clean; full usermgmt race suite PASS (19.7s); workspace build green for root, setup, adminui, systemadapter.
   - `setup/go.mod` still carries codec/v4 as `// indirect` — expected: it resolves through the PUBLISHED usermgmt tag; drops at the next usermgmt re-tag (release-train mechanics, not a defect).
3. **Feedback item 2 IMPLEMENTED + TESTED — consumer-owned middleware chain** (in tree, commit pending):
   - `Config.ExtraMiddleware []func(http.Handler) http.Handler` — composes INSIDE the built-in security stack, innermost before routes; listed order = first outermost (cqrshtmx.Chain convention); nil = byte-identical legacy chain.
   - `Config.DisableSecurityMiddleware bool` — removes the built-in security layer; consumer rebuilds the full chain in ExtraMiddleware (including `RecommendedSecurityMiddleware()` itself if only order changes).
   - Applies to every serve path automatically (`Handler`, `Run`, `RunHandler`, `RunWithAppkit` all compose through `Bundle.Middleware()`).
4. **Feedback item 3 IMPLEMENTED + TESTED — injectable health checks** (in tree, commit pending):
   - `Config.HealthChecks []cqrshtmx.NamedCheck` appended after built-ins in the mounted `/health`; consumer checks report in the same body, same 503 semantics, `NamedCheck.Timeout` honored.
   - `validateHealthChecks()` fail-fast: rejects duplicate names, collisions with built-ins (`projections`, `sse-hub`), empty names, nil Check funcs — the readiness body keys by name, and a nil Check would panic at probe time.
5. **9 new tests, all green** (`setup/setup_middleware_seams_test.go`, `setup/setup_health_checks_test.go`): position pin (extras inside security), listed-order pin, disable-flag pin, legacy zero-value pins, health append/503/validation/zero-value. Full setup race suite PASS (13.6s).

---

## b) PARTIALLY DONE

1. **Phase-2 commit** — the code is complete and tested but NOT properly committed:
   - Daemon committed the 3 source files as heuristic `chore: auto-commit 3 changed file(s)` (`243fd944`) while `nix fmt` was still running.
   - The 2 new test files are still **untracked** (`??`), so the heuristic commit contains seams WITHOUT their tests.
   - Needed: finish format, lint, then a fixup commit of the tests + README/doc.go updates + proper message (or amend `243fd944` if still the tip with no foreign mixing — verify first per the daemon-races runbook).
2. **`nix fmt`** — started ~20 min ago as a background job, still "running" with no output. Normal treefmt runtime here is ~1-2 min; this smells stuck (eval-cache lock or queue). Must be checked/killed and re-run **scoped to the 5 changed files** (`treefmt <paths>`) instead of whole-tree.
3. **Docs for the new seams** — setup/README.md config table + setup/doc.go customization-ladder mention for `ExtraMiddleware`/`DisableSecurityMiddleware`/`HealthChecks` not yet written (planned as part of the phase-2 commit).

---

## c) NOT STARTED

1. **Feedback item 5 — capability-floor table** (which features degrade gracefully vs. stay dark per interface: `event.Store` vs `Journal` vs `SeekableJournal` vs `StreamReader` vs `ProjectionHost`; SSE replay floor; dashboard panels per-interface). Target: setup/README.md section, cross-linked from fullstack-wiring guide. All facts already gathered and verified this session.
2. **Feedback item 7 — `docs/guides/setup-vs-hand-wiring.md`** decision doc: the honest decision tree (Paths 0/A/B/C vs setup), including "if you use one symbol, vendor the embed and skip us" and the identity-service-is-unconditional truth + flip triggers PapDashboard recorded.
3. **Feedback item 1 — setup/core + identity split** → ROADMAP entry with PapDashboard's acceptance criteria (go.mod gains setup/core but NOT usermgmt/identity-model; zero identity goroutines). Related to open OQ14 (`setup.NewFromSystem` bridge) — the core split generalizes it.
4. **Feedback item 4 — pluggable SSE envelope/encoder seam** → ROADMAP entry (touches golden-pinned transport contract; needs design, not a quick seam).
5. **Feedback file processing** — annotate with outcome blockquote + `git mv` to `docs/feedback/processed/`.
6. **CHANGELOG entries** for the codec migration + the two seams (Unreleased → Added).
7. **AGENTS.md gotcha 20** — add the feedback-processing convention (new/ → processed/ with annotation) to the docs-ownership list.
8. **Full verification battery** — `nix run .#lint` on changed modules, `nix run .#test`, `nix run .#check-modules` (docs-freshness gates will check the new guide's links).
9. **Release follow-through** — usermgmt (and later setup) re-tag on the next family train; drops setup's codec/v4 indirect and publishes the seams.

---

## d) TOTALLY FUCKED UP!

Nothing content-wise is broken — every test that ran is green and nothing was reverted. But two process mistakes:

1. **I violated gotcha 4 twice by running long verification BEFORE committing.** First `git commit` raced the daemon (hook failure delayed it; daemon landed `chore: auto-commit 13 files` — recovered by immediate local-tip amend with the proper message). Then, for phase 2, I started a 20-minute (stuck?) `nix fmt` BEFORE committing — the daemon shredded the seams commit (`243fd944`, no tests included). The correct sequence was: scoped `treefmt` on 5 files → lint → commit → THEN any long tails.
2. **Whole-tree `nix fmt` instead of scoped formatting** — 5 changed files did not need a full-tree run; it both delayed the commit (opening the race window) and is now apparently wedged. Lesson: `treefmt` accepts explicit paths; use them.

Also noted, not mine but observed: BuildFlow pre-commit fails deterministically outside the devShell (tsc/go-licenses/govulncheck + bare-shell golangci-lint fan-out, 10 steps) — the documented env-class from AGENTS gotcha 8; `--no-verify` with justification was the correct fallback, already used once this session.

---

## e) WHAT WE SHOULD IMPROVE!

1. **Commit discipline vs the daemon:** every phase boundary needs: verify-quickly → commit IMMEDIATELY → long verification. The preflight helpers (`.#preflight-tree-check`, `.#wait-tree-quiet`) exist for exactly this; I didn't use them.
2. **Scoped formatting:** add to muscle memory — `treefmt` on changed paths only; whole-tree `nix fmt` is for the rare full-tree changes.
3. **Feedback intake is under-tooled:** the new/ → processed/ flow works by convention only. A tiny `scripts/check-feedback-inbox.sh` (new/ must be empty at train time; processed files must carry an outcome annotation) would mechanize it the way status-report annotations are mechanized. (Candidate, not shipped.)
4. **The #67/#68 reference gap:** when consumers cite issue numbers in feedback, resolve them at intake time; unresolved references make triage archaeology later.
5. **README-as-canonical-config-table risk:** the new seams currently exist only in godoc; until README is updated (c3 above), the config table is silently incomplete — docs-freshness gates won't catch a MISSING row, only broken links.

---

## f) Next things to get done (session-derived, ordered)

1. Check/kill the stuck `nix fmt` background job (shell 02C); re-run `treefmt` scoped to the 5 changed setup files.
2. Clear the standing `gci` warning on `setup_middleware_seams_test.go` (import order; should resolve with scoped treefmt).
3. Commit phase 2 properly: the 2 untracked test files + README/doc.go rows (fixup or amend `243fd944` after verifying tip isolation per the daemon-races runbook).
4. Add the 3 new fields to setup/README.md config table with positions/defaults/zero-value semantics.
5. Mention the seams in setup/doc.go's customization ladder.
6. Run `nix run .#lint` for setup + usermgmt modules.
7. Write the capability-floor table (feedback #5) into setup/README.md.
8. Cross-link the capability floor from docs/guides/fullstack-wiring.md.
9. Write docs/guides/setup-vs-hand-wiring.md (feedback #7) — include the honest one-symbol vendor path, Paths 0/A/B/C tree, identity-unconditional truth, flip triggers.
10. Link the decision doc from setup/doc.go and setup/README.md.
11. ROADMAP: add setup/core + identity split entry with acceptance criteria + OQ14 relation (feedback #1).
12. ROADMAP: add pluggable SSE envelope/encoder seam entry with wire-contract context (feedback #4).
13. Annotate the feedback file with the outcome blockquote (items 2,3,6 shipped; 5,7 docs; 1,4 ROADMAP; root cause = target-audience mismatch).
14. `git mv docs/feedback/new/…​_papdashboard-setup-non-adoption.md docs/feedback/processed/`.
15. CHANGELOG Unreleased entries: codec migration (usermgmt), ExtraMiddleware/DisableSecurityMiddleware, HealthChecks.
16. AGENTS.md gotcha 20: one line for the feedback new/→processed/ convention.
17. Run `nix run .#test` (full workspace) after all edits land.
18. Run `nix run .#check-modules` (docs-freshness + release-train advisory + status gates will validate the new files).
19. Verify `docs/status/README.md` indexes this report if that's the convention.
20. Decide + execute the usermgmt re-tag path for the codec migration (next family train; `scripts/verify-tag.sh`).
21. After the re-tag: tidy setup/adminui/etc. to drop the lingering codec/v4 indirect requires (absence sweep `rg 'codec/v4' --include=go.mod` expecting zero direct).
22. Add a pin/example asserting `ExtraMiddleware` composes under `RunWithAppkit` too (currently implied via Middleware(); one e2e assert would mechanize it).
23. Consider the feedback-inbox checker script (e5.3) — gate checklist applies if built (checker + fixture self-test + flake app + check-modules stage + CI step + README).
24. Propose the deferred items as upstream asks if relevant (none identified this session beyond #67/#68 ambiguity).
25. Ask PapDashboard (via the resolved channel) to confirm the codec/v4 migration satisfies their recorded adoption prerequisite.
26. Re-check TODO_LIST P1 (templ-components v1.19.4 train) — the codec change touches the same train mechanics; coordinate the next family train contents (codec + seams + docs in ONE train vs two).

---

## g) Questions I can NOT figure out myself

1. **Train policy for this work:** should the usermgmt codec migration + the two setup seams ride the NEXT family train immediately (they only benefit consumers at published tags — PapDashboard's indirect-require drop happens only after the re-tag), or batch with the already-scheduled templ-components v1.19.4 + httputil v1.4.0 train work?
2. **The big two:** feedback #1 (setup/core + identity split) and #4 (SSE envelope seam) are architectural bets on a consumer class that today has exactly one member who explicitly says "not your fault, we're not the target app yet." Build them now, or hold as ROADMAP items pending a second consumer-shaped-like-that? (Owner strategy call — I can execute either way.)
3. **Where do "#67/#68" live?** They resolve in neither go-cqrs-lite nor cqrs-htmx issue trackers. If they're PapDashboard-internal gate references, is there anything beyond the shipped codec migration they demand for the recorded "adoption prerequisite" to count as closed?

---

*Point-in-time snapshot; annotate, never rewrite. Session continues on instruction.*
