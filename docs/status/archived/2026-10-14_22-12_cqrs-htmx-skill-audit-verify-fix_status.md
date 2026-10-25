# Session Status: cqrs-htmx Skill Audit — Verify, Fix, Fan-Out

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-15_01-18_cqrs-lint-4.16.1-suppression-alignment_status.md`](2026-10-15_01-18_cqrs-lint-4.16.1-suppression-alignment_status.md), which noticed tails routed (00-07 security passes, 12-16 vendorHash/lychee, OQ29, S40); the skill-maintenance core is the skill-maintenance-gates TODO row (harvested 2026-10-24). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-24)).**

> **Session scope:** the prompt was "Is the cqrs-htmx skill superb and UP TO DATE?" — executed as a
> claim-by-claim audit of `.agents/skills/cqrs-htmx/` (SKILL.md + 4 reference files + evals.json)
> against the actual codebase, followed by fixes, delivery-layer repair, and verification.
> This report covers **only this session's run and what it noticed** — no unrelated research.
> Concurrent sessions were active throughout (foreign `errors.go`/`adminui/util.go` edits,
> heuristic commits `0642ed38`/`5d6fc804`/`481b0ef8`, load average 64).

**Report:** 2026-10-14 22:12 CEST · **Skill:** `.agents/skills/cqrs-htmx/` · **Branch:** master

---

## a) FULLY DONE

1. **Skill loaded and audit plan executed end-to-end.** skill-creator skill loaded first (mandatory trigger); 8-step todo plan (read → verify structure → verify claims → delivery layer → fix → gates → verdict) fully completed. Evidence: all 8 todos completed.
2. **Full skill inventory read.** SKILL.md (534 lines) read in full; bundled resources enumerated (`references/core-api.md`, `gotchas.md`, `realtime.md`, `usermgmt.md`, `evals/evals.json`). Evidence: tool transcripts this session.
3. **~130 root-module API symbols verified to exist.** Two rg sweeps (main surface + deprecated alias surface) over the root module; exactly one claimed symbol was missing (`WSOOBHTML`). Evidence: sweep output "SWEEP-DONE / SWEEP2-DONE" with zero other MISSING lines.
4. **`MapError` mapping verified at source.** `errors.go:66-110` (carrier → explicit sentinels → `errorfamily.Classify().HTTPStatus()`) + go-error-family `family.go` values: Rejection 400, Conflict 409, Transient 503, Corruption **500**, Infrastructure **503**. The skill claimed Corruption→422 / Infrastructure→500 / "NotFound→404" — all wrong, now fixed.
5. **WebSocket removal state pinned precisely.** Zero WS-family hits in the tree (`WSBroadcaster`, `WSMessage`, `DispatchWSCommand`, `WSOOBHTML`, `htmx-ext-ws`); CHANGELOG removal entry sits under `[Unreleased]` (latest tag v4.14.1); `docs/migrations/v4-to-v5.md` + ADR-0046 read. The "removed in v5" framing is correct; one stale in-body WS mention deleted.
6. **UserID alias chain proven — the skill's top gotcha was factually dead.** root `context.go:16 UserID = id.UserID`; `identity-model/id.go:18 UserID = id.UserID`; `usermgmt/id.go:13 UserID = identitymodel.UserID` → **one type everywhere**. Strict constructors verified at `identity-model/id.go:42,55,83` (`SyntheticUserID`/`ParseUserID`/`GenerateUserID`) and usermgmt re-exports them (`usermgmt/id.go:41-50`). Gotcha rewritten as strict-construction guidance.
7. **ActorID surface verified.** root `context.go:186-204` (`ActorID = id.ActorID`, `ParseActorID` re-export, `ImpersonatorID = ActorID`); `NewActorID` confirmed deleted (AGENTS gotcha 15); kind constructors (`id.NewUserActor`/`NewBotActor`/`NewSystemActor`) live. gotchas.md §ActorID rewritten.
8. **Broadcaster/SSE semantics verified.** `BroadcastOnSuccess` is a method on `*Broadcaster` (`sse_broadcaster.go:92`); go-sse `DefaultSubscriberBuffer = 64` (`fanout.go:19`); non-blocking drop semantics confirmed; `SSEEvent = sse.Event` alias with `Event`/`Data` fields matches every snippet in the skill.
9. **Sub-module claims verified.** adminui `Config` fields (Service/Title/BasePath/Mode/ModeTenantAdmin/TenantID/AccentColor/Authorizer/LogoutURL/SSEURL — `adminui/config.go:38-78`); both adminui + dashboardui `Middleware()` delegate to `RecommendedSecurityMiddleware` (`adminui/handler.go:246`, `dashboardui/handler.go:40`); usermgmt `NewService`/`Close`/`NewAuthHandler`/`NewSessionMiddleware`/`ValidateCommand`/`DefaultEventCatalog`/`NewSQLiteEventSourcedSetup`/`RebuildProjection` all present; auth route table matches `usermgmt/http.go:199-207`; setup `RequireSession`/`RequireSessionRedirect`/`CommandMiddleware` present (`setup/session_gate.go:15,32`, `setup/config.go:462`); datastar module path confirmed (`datastar/go.mod`).
10. **Embedded asset versions verified.** htmx 2.0.10, sse ext 2.2.4, idiomorph 0.7.4 (root tests + example output pin all three). Skill's claims were correct — kept.
11. **Docs/examples inventory verified.** All 8 guides the skill references exist; `docs/migrations/v3-to-v4.md` exists; ADRs 0037/0046/0055/0056/0011 exist; `transport/WithSSEFilter`/`WithSSEMaxReplay` exist (`transport/serve.go`); flake per-module test apps exist (`flake.nix:622,676`); admin-demo port is **18930** (`examples/admin-demo/main.go:5,52`), not the skill's claimed 8097.
12. **All 10 stale claims fixed across 4 files.** SKILL.md (12 edits: root row, loginpage row, +datastar row, +bridges note, v3/v4 sentence, WSOOBHTML removal, port 18930, MapError line, WS note precision, gotcha 2 rewrite, +transport/ADR-0056 SSE pointer, +2 Where-to-look entries + 3 example additions); gotchas.md (3 edits: go.work 29-module inventory, UserID section, ActorID sections); usermgmt.md (UserID bridge section rewritten); evals.json (2 × `/v3`→`/v4`).
13. **Delivery layer repaired.** `~/.config/crush/skills/cqrs-htmx` → repo checkout symlink created (was missing while go-cqrs-lite had one); read through fan-out verified; guard green: *"OK: 47 skill symlink(s) … resolve cleanly (no /nix/store targets)"*.
14. **Formatting + commit verified.** `nix run .#fmt` on the 4 files: 0 changed (edits were already format-clean). Committed content grep-verified in `481b0ef8`: corrected MapError line present, datastar row present, **zero** occurrences of `WSOOBHTML`/`:8097`/`lp-*`/`/v3` in the committed skill blobs.

## b) PARTIALLY DONE

1. **Reference-file audit is spot-check, not exhaustive.** What works: stale-token probes (`WSBroadcaster|WSOOB|htmx-ext-ws|lp-|8097|hand-rolled` → zero hits) + the two dead sections rewritten. Open: core-api.md / realtime.md / usermgmt.md (~2000 lines) not read line-by-line against the source; other unprobed claims remain (EventCatalog FNV-1a ETag/1-yr cache, `CSRFTestToken` behavior, `ContextEnrichmentMiddleware(nil)` ULID, `HealthHandler` JSON shape, `RegisterErrorClassifications` sync.Once). Blocker: none — time/scope tradeoff. Effort: M.
2. **evals.json corrected but never executed.** What works: 3 prompts + assertions exist, `/v3` asserts fixed. Open: the skill-creator eval loop (with-skill vs baseline subagent runs, grading, viewer, benchmark) has never run — the skill's trigger/quality is unmeasured. Blocker: none (needs explicit go-ahead; subagent-heavy). Effort: M–L.
3. **Module table completeness.** datastar row added + one prose line covers health/auditlog/systemadapter as "optional bridges"; they still have no proper table rows. Effort: S.
4. **Commit message quality debt.** My detailed commit message was lost to the daemon race; content landed inside heuristic `chore: auto-commit 8 changed file(s)` (`481b0ef8`) **mixed with 4 foreign Go files** from a concurrent session. Not reworded per gotcha 4 (reword re-runs the full hook and loses the ref race again). Effort: S (accept) or M (amend after daemon quiesces).
5. **Description (frontmatter) not re-optimized.** Judged accurate post-audit ("removed in v5 — SSE-only now" matches the migration docs), but the skill-creator trigger-optimization loop was not run. Effort: M.

## c) NOT STARTED

1. **Skill-creator eval loop** (3 test prompts, baseline snapshot runs, grading, benchmark viewer) — waiting on go-ahead.
2. **Description trigger-optimization** (20 eval queries → optimization loop → best_description).
3. **HARVEST of this report's §f into `TODO_LIST.md`/`ROADMAP.md`** — deliberately not executed: the instruction was "write report, then WAIT". The 50 items below are brainstorm, not commitments.
4. **Mechanized skill-drift gate** — nothing exists that catches skill-vs-source drift mechanically (this audit found 10 drifts entirely by hand).
5. **`health`/`auditlog`/`systemadapter` module table rows** in SKILL.md.
6. **Deep re-audit of the three untouched reference files** (core-api.md, realtime.md, usermgmt.md remaining sections).
7. **Split-brain scan for the dead UserID gotcha in module READMEs** (adminui/dashboardui/loginpage/usermgmt READMEs may still teach `.Get()` bridging — unverified).
8. **Verification of remaining unprobed skill claims** (list in b.1).

## d) TOTALLY FUCKED UP

1. **I lost the commit race I was warned about — and the warning was in my own context.** Severity: message/attribution debt, zero data loss (content grep-verified in `481b0ef8`). Root cause: I chained `fmt && git add && git commit` with the full 200s BuildFlow hook while `uptime` (printed in the *same* chain, i.e. consulted too late) showed **load 64.46** and a concurrent session was mid-flight. The hook ran >3 minutes; the daemon moved HEAD; my commit died with `fatal: cannot lock ref 'HEAD': is at 481b0ef8… but expected 5d6fc804…` (exit 128). Gotcha 4's micro-ritual says check uptime BEFORE choosing full-hook vs `--no-verify` — I had the data in the wrong order of operations. Mitigation: content is committed and verified; the heuristic message is accepted debt per gotcha 4 (never reword while the daemon lives). The durable fix is in §e.1.
2. **Machine-load etiquette failure.** My hook run executed golangci-lint across ~17 modules (+ cqrs-lint, test-compile) at load 64 while another session was actively editing `errors.go`/`identity-model/errors.go`. Nothing corrupted (BuildFlow passed 356/0), but I added minutes of contention to someone else's critical path. Root cause: same as d.1. Mitigation: §e.1/§e.2.
3. **A tool-flag mistake produced garbled evidence mid-audit** (self-inflicted, caught before it could mislead): `rg -rn "UserID ="` used the *replace* flag, printing `n identitymodel.UserID`-style garbage; I re-ran cleanly instead of concluding from it. Severity: none shipped; noted because it was one step from becoming a wrong "verified" claim.
4. **Nothing else is fucked.** The skill itself is in good shape post-fix: all 4 committed files grep-verified stale-token-free, fan-out guard green, fmt clean.

## e) WHAT WE SHOULD IMPROVE

1. **Order-of-operations for hook commits under load.** Impact: high (this exact failure cost a clean commit message and ~200s of shared-machine contention). Fix: make the pre-commit chain `nix run .#wait-tree-quiet && <uptime/load check> && fmt && git add && git commit` — the load check must *gate* hook selection (full hook vs documented `--no-verify`+justification), not decorate the log.
2. **Skill-vs-source drift should be mechanized, not audited by hand.** Impact: 10 drifts survived ~3 months of heavy doc-gating (check-modules, docs-freshness, CSS class-set gates — none look at the skill). Fix: `scripts/checks/check-skill-claims.sh` with a forbidden-token list (deleted symbols like `WSOOBHTML`/`NewActorID`, moved ports like `:8097`, dead claims like `/v3`), shipped atomically per gotcha 19 (checker + fixture self-test + flake app + check-modules stage + CI step).
3. **Skill snippets deserve a compile gate.** SKILL.md carries ~25 Go snippets that nothing compiles. Impact: the highest-probability future drift class. Fix: extract snippets into an `examples/skill-snippets/` module (or `docs/` go:build-ignore fixtures like `check-templates` does) so doc-rot breaks CI.
4. **evals.json exists but the skill has never been measured.** Impact: trigger description and body quality are unproven assumptions. Fix: one skill-creator pass (evals + description optimization) — then re-run after the v5 train.
5. **Fan-out completeness is convention-enforced only.** The missing symlink was found by analogy (go-cqrs-lite had one), not by a guard. Fix: extend crush-config's fan-out guard to assert every repo with `.agents/skills/*/` has a global link.
6. **Cross-session concurrency posture.** Three heuristic commits + 4 foreign dirty Go files + load 64 were all visible signals before I started the heavy phase. Impact: races and contention. Fix: treat `preflight-tree-check` + a load check as a mandatory pre-phase for any multi-minute verification, not just for tree-mutating batch steps.
7. **Version-pinned claims need a flip checklist.** "removed in v5" phrasing (description + body + evals) will all need past-tense when the v5 train cuts. Fix: one tracked item (§f.9) rather than re-discovery.

## f) 50 things we should get done next

*(Brainstorm for HARVEST — most are ROADMAP fuel; ranked roughly by impact. Impact/Effort/Category per item.)*

| # | Task | Impact | Effort | Category |
|---|------|--------|--------|----------|
| 1 | Add `scripts/checks/check-skill-claims.sh` forbidden-token gate (WSOOBHTML, NewActorID, :8097, lp-*, /v3 in skill) — atomic per gotcha 19 | High | M | Quality |
| 2 | Compile skill Go snippets via an examples/skill-snippets module wired into CI | High | L | Quality |
| 3 | Run the skill-creator eval loop (3 evals, baselines, viewer, benchmark) | High | M | Quality |
| 4 | Run description trigger-optimization (20 queries, 5-iteration loop) | Medium | M | Quality |
| 5 | HARVEST this report's items into TODO_LIST.md/ROADMAP.md (docs-health) | High | S | Documentation |
| 6 | Deep re-audit `references/core-api.md` symbol-by-symbol against root exports | Medium | M | Documentation |
| 7 | Deep re-audit `references/realtime.md` post-SSE-only rewrite | Medium | M | Documentation |
| 8 | Deep re-audit `references/usermgmt.md` remaining sections (setup matrix, SQL dialects, import/export) | Medium | M | Documentation |
| 9 | Flip all "removed in v5" phrasing to past tense when the v5 train cuts (description + body + evals) | Medium | S | Documentation |
| 10 | Add health/auditlog/systemadapter rows to the SKILL.md module table | Low | S | Documentation |
| 11 | Verify remaining unprobed skill claims (EventCatalog ETag/cache, CSRFTestToken, ContextEnrichmentMiddleware(nil), HealthHandler shape, RegisterErrorClassifications) | Medium | M | Quality |
| 12 | Scan module READMEs for the dead two-UserID-types gotcha (split-brain class) | Medium | S | Documentation |
| 13 | Extend crush-config fan-out guard: assert repos with `.agents/skills/` have global links | Low | S | Tooling |
| 14 | Add evals for the new documented surfaces (frontend-sync ADR-0056, datastar adapter, setup path) to evals.json | Low | S | Quality |
| 15 | Document `setup-vs-hand-wiring.md` + `actor-and-audit-trail.md` in SKILL.md Where-to-look | Low | S | Documentation |
| 16 | List `system-demo`/`samber-do-demo`/`observability-demo` examples in SKILL.md Where-to-look | Low | S | Documentation |
| 17 | Add hub-first SSE vocabulary (gotcha 17: pass the hub; `Raw()` deprecated) as a top-gotcha in SKILL.md | Low | S | Documentation |
| 18 | Add a `transport.NewJournalSSEStore` usage snippet to the Realtime section | Low | S | Documentation |
| 19 | Verify adminui "serves htmx.js internally" end-to-end (assets.go route, not just grep) | Low | S | Quality |
| 20 | Verify loginpage consumer-compiled-CSS flow end-to-end (bundle gate covers it?) | Low | S | Quality |
| 21 | Schedule the next skill audit immediately after the v5 train | Low | S | Process |
| 22 | *(noticed, pre-existing)* BuildFlow binary stale: built at 518739c vs HEAD 7ccd809 — `nix build . && nix run .#reinstall` | Low | S | Tooling |
| 23 | *(noticed)* BuildFlow `go-work-paths` warn: 13 `use` paths lack `/v4` suffix match — adjudicate known-false-positive vs fix | Low | S | Cleanup |
| 24 | *(noticed)* BuildFlow `gomod-freshness` warn: 34 modules "need tidy" — run the per-module hermetic tidy sweep (gotcha 27d) and check what's real | Medium | M | Cleanup |
| 25 | *(noticed)* `vendorHash.nix` stale-FOD warning — build the failing derivation first; only its `got:` sha is truth (gotcha 31) | Medium | S | Bug |
| 26 | *(noticed)* dependabot.yml capped at 20 of 29 Go modules — raise the cap | Low | S | Cleanup |
| 27 | *(noticed)* cqrs-lint: 50 warnings remain (sync_pull.go branded-ID class, e2e BasicCommand-value class) — triage vs `rules.disable` (gotcha 28/29) | Medium | M | Quality |
| 28 | *(noticed)* e2e/server golangci findings (varnamelen, wrapcheck ×3) — fix or accept (e2e excluded from `nix lint`) | Low | S | Quality |
| 29 | *(noticed)* SECURITY.md missing Security Policy/Practices/Supported Versions sections (securitymd error ×3) | Medium | S | Documentation |
| 30 | *(noticed)* file-size-check: 86 files >350 lines (worst: oauth2/provider_test.go 1104) — split the worst 10 | Low | L | Cleanup |
| 31 | *(noticed)* lychee: 10 dead/403 links (dashboardui README :8098, ADR-0036 googlesource, 3 research docs) — fix or exclude | Low | S | Cleanup |
| 32 | *(noticed)* mypy: 2 type errors in scripts/checks/check-status-rows.py — fix signatures | Low | S | Bug |
| 33 | *(noticed)* pytest: "no tests ran" for scripts/ — fix discovery config or drop the step | Low | S | Tooling |
| 34 | *(noticed)* type-check: 8 `'unknown'` errors in tests/sync.spec.ts — type the Playwright fixtures | Low | S | Bug |
| 35 | *(noticed)* vulture: unused `DEFAULT_DIR_ARG_HELP` in scripts/lib/status_table.py — remove | Low | S | Cleanup |
| 36 | *(noticed)* vulnix CVE findings on bash/binutils builder drvs — adjudicate accept/document | Low | S | Security |
| 37 | *(noticed)* go-licenses "cannot find license" for larsartmann modules + mod-cache dirs — add LICENSE files or exclusion | Low | M | Tooling |
| 38 | *(noticed)* BuildFlow timing regressions (golangci +239–985% across modules) — almost certainly the load-64 artifact; re-measure idle before believing any of it | Low | S | Process |
| 39 | *(noticed)* sync-demo `AddSyncOpenAPI` require-predates-symbol gap (gotcha 38d) — ride the next root tag | Medium | S | Bug |
| 40 | *(noticed)* Per-module CI matrix isolation (gotcha 40c open question) so one module red can't mask another | High | L | Feature |
| 41 | *(noticed)* Release-train `--refresh-cache` inside the pre-push hook (gotcha 27e mechanical candidate, pending evaluation) | Medium | S | Tooling |
| 42 | *(noticed)* check-streamid-identity pin audit — new display-form sites since 2026-10-07? | Low | S | Quality |
| 43 | *(noticed)* branching-flow baseline drift watch (+29 pattern after feature bursts) — re-pin with triage amendment if drifted | Low | S | Quality |
| 44 | *(noticed)* identity-model still direct-deps go-branded-id — confirm only TenantID/BotID remain and note the consolidation boundary | Low | S | Cleanup |
| 45 | *(noticed)* /mnt/buildcache fill-drain cycle history — `df -h` first next session (gotcha 12) | Low | S | Process |
| 46 | *(noticed)* Full-tail ANNOTATE sweep owed for docs/status (README's own note, round-21) | Low | M | Documentation |
| 47 | Consider adding the skill-audit checklist itself as a reusable skill (audit → verify-claims → fix → fan-out) — this session's flow generalizes | Low | M | Tooling |
| 48 | Adopt "wait-tree-quiet + load gate before hook commits" as standing practice (see §e.1) | Medium | S | Process |
| 49 | Add CHANGELOG receipt adjudication for `.agents/skills/` changes (consumer-visible tooling or docs-only? gotcha 20) — one sentence in AGENTS.md ends the recurring ambiguity | Low | S | Documentation |
| 50 | Re-run the fan-out guard after the next crush-config rebuild (hand-made links can be clobbered by activation) | Low | S | Process |

## g) Three questions I cannot figure out myself

1. **Editorial policy for the v5 boundary:** should the skill's "WebSocket removed in v5" phrasing (frontmatter description + body + evals) flip to past tense **now** (tree is already WS-free) or **only when the v5 tag publishes**? I kept the v5 framing to match the project's own migration docs, but the tree reality diverges from the description today. (I tried: CHANGELOG + migration guide + ADR-0046 all frame it as the v5 cutover; none settles when *docs* should flip.)
2. **Is the skill consumer-visible deliverable or internal agent tooling?** This decides whether skill updates need a CHANGELOG receipt and release-train bundling (gotcha 20) — I assumed docs-only this session. Only you can set that policy.
3. **Standing rule for hook commits:** adopt `wait-tree-quiet` + a load gate before *every* hook-committing commit (adds minutes of latency even when idle), or only when foreign-session/load signals are visible? Today's failure argues for "always"; the latency cost argues for "conditional".

---

*Point-in-time snapshot — see `docs/status/README.md` for the annotation convention. §f is the HARVEST input for `TODO_LIST.md`/`ROADMAP.md`.*
