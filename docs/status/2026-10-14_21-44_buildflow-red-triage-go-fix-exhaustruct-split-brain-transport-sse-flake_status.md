# Status: BuildFlow Red Triage — Transient go-fix Reads Unmasked a Real Tool Split-Brain (go fix vs exhaustruct), the SSE Fan-Out Test Flake Root-Caused and Made Deterministic, Skips Vindicated (2026-10-14 ~21:00 → 21:45)

> **Repo state at writing:** HEAD `f1bb0aa1` (daemon heuristic; tree clean except one FOREIGN in-flight diff: `M loginpage/handler.go` — authored by the concurrent session, untouched by this one, per gotcha 4). Local ref shows `origin/master..HEAD` = 0 ahead, i.e. a concurrent session PUSHED master including this session's daemon-captured fixes — remote state not independently fetched/verified this session. **Concurrent activity was heavy all session**: two daemon commits (21:05 `04407856` dep-update sweep + flake.lock, 21:10 `e247ae5a` modernize-fixer sweep + go.sum reverts) raced this session's BuildFlow run and are the root cause of one of the three pasted failures.

> **Session input:** the user pasted a BuildFlow run with 3 failures (`go-fix`, `go-fix [usermgmt]`, `test-race`), 10 config skips flagged as "very sus", a `DISCOVERY DEGRADED` nix note, and 9 unavailable tools. Scope guard honored: no research beyond this session's run and observations.

---

## Executive summary

All three failures are understood and fixed; both failed steps re-ran green (`go-fix --fix` 30/30, `test-race` 1/1). The go-fix pair was a **transient mid-write read** (BuildFlow at 21:07 raced the concurrent sweep that rewrote `identity-model/fold.go` at ~21:10) — but chasing it exposed a REAL, durable conflict: **BuildFlow's go-fix shells toolchain `go fix ./...`, which is `//nolint:modernize`-UNAWARE and actively rewrites the explicit nested embedded-core literals that exhaustruct demands** (febb8ef5's 2026-10-11 fix, re-broken same-day by the sweep). Resolved by adopting promoted-key literals as canonical for embedded-core types and moving the burden to `exhaustruct.exclude` in identity-model + usermgmt; idempotence proven with `go fix -diff` (empty in root, identity-model, usermgmt) and a full go-fix pass that rewrote nothing. The test-race failure was a **genuine flake**: `TestServeDomainEvents_ConcurrentClients` broadcast after a blind 50ms sleep, racing 16 handler goroutines' subscription under `-race` + machine load; rewritten deterministically (SubscriberCount wait + per-client delivery-signal channels). The scary-looking `user.reg\xffistered` marshal error is **expected test noise** from `TestDomainEventToSSE_MarshalFailureFallback`, which deliberately feeds an invalid-UTF-8 event type. The skips are **documented, not sus** (`.buildflow.yml:47-101`).

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **go-fix ×2 failure root-caused as transient**: `fold.go` was read mid-write (`expected 'package', found 'EOF'`) by BuildFlow at 21:07:26 while the concurrent sweep rewrote it; final state landed 21:10:57 (`e247ae5a`). File verified clean afterward: gofmt rc=0, `go build` rc=0 | timeline: test log 21:07:26 vs `e247ae5a` 21:10:57 |
| 2 | **Tool split-brain root-caused**: BuildFlow go-fix = `go fix ./...` per module (BuildFlow checkout `tools/providers/formatter_args.go:173`, `NewGoFixProvider`, hardcoded args, no per-analyzer knob; `-embedlit=false` exists in `go tool fix` but is unwired). `go fix` (Go 1.27 modernize driver, `embedlit` analyzer confirmed via `go tool fix help embedlit`) does NOT honor golangci `//nolint:modernize` — live-proven when the fixer re-broke the restored nested literals WITH the directives in place | BuildFlow checkout 7ccd8094e; `go tool fix help` analyzer list |
| 3 | **Split-brain resolved durably**: promoted-key literals adopted as CANONICAL for embedded-core types; `go fix` applied to the 6 conflict sites (2 identity-model + 4 usermgmt); `exhaustruct.exclude` extended in `identity-model/.golangci.yml` (ExternalAccount) and `usermgmt/.golangci.yml` (ExternalAccount + BotReadModel + MembershipReadModel + TenantReadModel) with a rationale comment. Both modules lint **0 issues** hermetically (module-dir, GOWORK=off) | `identity-model/.golangci.yml:95-99`, `usermgmt/.golangci.yml:99-104` |
| 4 | **Fixer idempotence proven**: `go fix -diff ./...` in root, identity-model, usermgmt → empty diff, rc=0 each; then full `buildflow -s go-fix --fix` → 30/30 with ZERO source rewrites (post-run `git status` showed only this session's intentional doc/config files) | run at 21:34:10, `30 success, 0 failed` |
| 5 | **Transport SSE flake root-caused + fixed**: `TestServeDomainEvents_ConcurrentClients` used `time.Sleep(50ms)` before broadcasting; handler subscribes BEFORE sending "connected" (`transport/serve.go:285-288`), so a slow goroutine under `-race`/load legitimately missed the fan-out (client 8's body = retry+connected only — exactly the observed failure). Rewritten deterministically: `waitForSubscribers` polls the RLock-protected `SubscriberCount()`; per-client `deliverySignalWriter` closes a channel when the marker streams through (sniff buffer is handler-goroutine-local; channel close is the only cross-goroutine handoff) | `transport/serve_test.go:507-625`; 5× `-race -count=5` green; `buildflow -s test-race` 1/1 (27.3s) |
| 6 | **`\xff` mystery dismissed as expected noise**: `jsontext: invalid UTF-8 within "/type"` + `eventType="user.reg\xffistered"` is the DELIBERATE input of `TestDomainEventToSSE_MarshalFailureFallback` (`transport/event_sse_test.go:168`), pinning json/v2's invalid-UTF-8 fallback (doc comment at :154-158 says exactly this). No memory corruption, no transport bug | `transport/event_sse_test.go:153-191` |
| 7 | **Skips vindicated**: templ-generate (bundled templ emits directory-prefixed `FileName:` fields), tailwind-build (only output = gitignored orphan styles.css, 2026-09-23 incident), deadnix (not installed) — all three carry written reasons in `.buildflow.yml`; skip_steps block documented LOAD-BEARING (2026-10-07 deletion incident) | `.buildflow.yml:47-102` |
| 8 | **AGENTS.md gotcha 7 updated in-session** (aggressive update protocol): the 2026-10-14 addendum records that `//nolint:modernize` cannot stop go-fix, promoted-key is canonical for embedded-core types, the exclude-based equilibrium, and that the nested-BaseProps convention survives only until go-fix touches those literals | `AGENTS.md` gotcha 7 (daemon commit `5ba9751d`) |
| 9 | **Usermgmt `-race` suite green at session end** (after one observed transient FAIL — see b2) and identity-model `-race` green; touched files treefmt-clean (`formatted 3 files (0 changed)`) | rc captures in session log |
| 10 | **Foreign-diff hygiene held**: `loginpage/handler.go` (concurrent session, not mine) identified and left untouched; daemon-captured commits not reworded (gotcha 4) | `git status` at 21:44 |

## b) PARTIALLY DONE

| # | Item | Gap |
|---|------|-----|
| 1 | **Verification breadth**: affected modules + the two failed BuildFlow steps verified green. NOT run this session: `nix run .#test` (full 28-module workspace battery), `.#test-all` (incl. e2e/examples), `.#check-modules` (full gate battery), `nix flake check`. The concurrent dep sweep (go.sum churn across all 28 modules) makes collateral drift plausible elsewhere | CI may surface what local didn't check |
| 2 | **usermgmt `-race` FAIL observed ONCE, then vanished**: first `-race` run FAILED (29.7s), the re-run passed — and the second was **(cached)**, so the "green" evidence is weaker than it looks, and the failing test's NAME was never captured (grep ran after the passing rerun). "Machine load flake" is an ATTRIBUTION, not a diagnosis | Needs `-count=1 -race` reruns capturing test names |
| 3 | **vendorHash.nix staleness WARN** (buildflow preflight, every run): go.sum churn from the concurrent sweep may have genuinely drifted the FOD hash. NOT verified — and per gotcha 31 the checker's finding alone must never be trusted; a real `nix build` (its `got:` sha) is the only truth | Flagged, not adjudicated |
| 4 | **Blast-radius of remaining nested embedded-core literals NOT inventoried**: gotcha 7's nested-BaseProps convention (adminui/dashboardui/loginpage + templ-components consumers) is the same landmine — the next go-fix pass over those modules will rewrite those literals and their exhaustruct configs (adminui, loginpage confirmed in the exhaustruct-enabling list) will flag them. Documented in gotcha 7; sites NOT enumerated | Pre-emptive excludes not prepared |
| 5 | **`check-branching-flow` not re-run** after `serve_test.go` grew new branches/loops (waitForSubscribers loop, Write sniff). The SARIF ratchet fails on NEW findings; whether the additions crossed a threshold is unverified | One gate run would settle it |
| 6 | **docs/agents-notes.md long-form entry not written** — the split-brain narrative (timeline, dead ends, decision) lives compressed in gotcha 7 only; repo convention wants dated histories in agents-notes | Doc gap |
| 7 | **"9 tools unavailable" attributed to the documented bare-shell class but never enumerated** (`buildflow doctor` would list them); "DISCOVERY DEGRADED" nix note not investigated at all (out of session scope, honestly skipped) | Advisory-only items |
| 8 | **CHANGELOG/gotcha-20 adjudication for the published-module rewrites** (identity-model/usermgmt source changed, behavior-identical): reasoned "not consumer-visible → no receipt" but did not double-check the policy's edge for behavior-identical source rewrites riding the next train | Decision not recorded anywhere durable |

## c) NOT STARTED

| # | Item | Why |
|---|------|-----|
| 1 | **HARVEST of section (f) into TODO_LIST/ROADMAP** (docs-health HARVEST mode) | User ordered report-then-wait; skill says section (f) must not stay entombed in a timestamped file |
| 2 | **Push + CI watch** (`nix run .#push-and-watch`) — moot if the concurrent session already pushed, but the push state is the OTHER session's claim, not verified | Pre-push strict gates not exercised by this session |
| 3 | **BuildFlow upstream improvement**: expose per-analyzer flags (or file excludes) for go-fix via tool_options — today's case is the motivating repro | Own-repo feature work |
| 4 | **exhaustruct/exhaustruct_v5 upstream filing**: old exhaustruct's embedded-core false positive + v5's panic on promoted-key literals (verify-before-filing gate first) | Not started |
| 5 | **`go tool fix help inline` investigation**: the `//go:fix inline` directive system exists — check whether an IGNORE-class directive exists upstream that survives `go fix` (would be a code-side guard that actually works) | Not started |
| 6 | **Per-module CI isolation for the `test` job** (gotcha 40c open question): today's flake class hides/shows depending on module batching | Pre-existing roadmap item, untouched |

## d) TOTALLY FUCKED UP

Nothing SHIPPED broken — but three process failures are worth owning bluntly:

1. **I ran a MUTATING fixer before fixing the conflict it participates in.** After restoring the nested literals, I ran `buildflow -s go-fix --fix` to "verify" — which predictably re-broke them and cost a full diagnostic cycle. The non-mutating probe (`go fix -diff`) existed one command away and would have proven the nolint-guard hypothesis false WITHOUT touching the tree. Restore-then-run-the-thing-that-destroys-the-restore is backwards; probe with -diff FIRST, always.
2. **I repeated documented shell traps I already know.** `cmd | tail; echo $?` (reports tail's rc — gotcha 3 verbatim) at least twice; `PIPESTATUS` under mvdan/sh (prints `n/a`); and the **false green**: `golangci-lint run ./identity-model/...` from repo ROOT under GOWORK=off — a module pattern that matches nothing, 0 findings, gotcha-2 class. Caught only because usermgmt's parallel run showed findings identity-model's run had to mirror. A fresh session hit three documented gotchas; that is not a tooling problem, it is an attention problem.
3. **The first "verification" of the flake fix rode the result cache**: the usermgmt `-race` re-run that "passed" printed `(cached)` — I briefly held a cached green as evidence about a failure I never saw. In a repo whose own AGENTS.md warns about stale result caches (`BUILDFLOW_NO_RESULT_CACHE=1`), that is exactly the laziness the repo documents against.

## e) WHAT WE SHOULD IMPROVE

1. **Probe dry before mutating**: for EVERY fixer/rewriter (go fix, templ-generate, tailwind, buildflow repair steps), the -diff/check equivalent runs BEFORE the mutating form — codify in gotcha 8's fallback list.
2. **Never trust a root-pattern run under GOWORK=off**: module verification is `cd <module> && GOWORK=off <tool> ./...`, full stop (mechanize the gotcha-2 rule into the habit layer).
3. **Capture rc and failing-test names in the SAME breath as the run** — a failure that vanishes on rerun without a captured name is undiagnosed, not fixed.
4. **Attribution discipline**: every transient failure gets a written timeline (file mtimes, commit times, log timestamps) BEFORE any fix — today that timeline dissolved the go-fix mystery in one look; it should be step zero, not step three.
5. **Fixer-ownership check precedes pattern restoration**: before hand-restoring any lint pattern, ask "does an automatic rewriter own this form?" — if yes, the durable fix is config-side (excludes), never code-side repetition.
6. **Load-aware test triage**: the machine ran a dep sweep + fixer sweep + BuildFlow + this session concurrently; timing flakes under that load are near-certain — but their ABSENCE of evidence must still be recorded (which test, which seed) for later pattern-mining.

## f) Up to 50 things we should get done next

Sorted by impact (session-observed only; HARVEST fuel — items marked [T] → TODO_LIST, [R] → ROADMAP, [B] → backlog/advisory):

| # | Task | Tag |
|---|------|-----|
| 1 | Re-run usermgmt `-race -count=1` (no cache) until the transient FAIL either reproduces with a test name or survives 10 clean runs; record verdict in TODO_LIST | [T] |
| 2 | Verify vendorHash with a real `nix build` (gotcha 31/32: `got:` sha is the only truth); fix vendorHash.nix or annotate the warning as false positive | [T] |
| 3 | Inventory ALL nested embedded-core literal sites repo-wide (`rg 'BaseProps:|Core\{' across adminui/dashboardui/loginpage/examples), check each module's exhaustruct config, pre-add excludes BEFORE the next go-fix pass breaks them | [T] |
| 4 | Run `nix run .#test` (full workspace battery) — post-sweep insurance for the 28-module go.sum churn this session never fully verified | [T] |
| 5 | Run `nix run .#check-modules` (full gate battery + self-tests) | [T] |
| 6 | Run `nix run .#test-all` (includes e2e/examples that `#test` excludes) | [T] |
| 7 | Re-run `nix run .#check-branching-flow` — confirm serve_test.go's new branches didn't cross the SARIF baseline; re-pin with triage-decisions amendment if needed | [T] |
| 8 | Sweep SSE-adjacent tests (dashboardui live feed, setup, datastar EventBridge) for the fixed-sleep synchronization class; convert the flaky ones to the SubscriberCount/signal pattern | [T] |
| 9 | Extract `waitForSubscribers`/`deliverySignalWriter` into a shared test helper IF item 8 finds ≥2 more sites (don't extract for one) | [T] |
| 10 | Push-state reconciliation: fetch, confirm what the concurrent session pushed, run `nix run .#verify-remote-tags` if tags moved; then `push-and-watch` for the next CI cycle | [T] |
| 11 | Write the docs/agents-notes.md entry: "The go fix / exhaustruct split-brain" (timeline 21:05→21:35, the nolint non-guard proof, the exclude equilibrium) | [T] |
| 12 | File BuildFlow upstream (own repo): go-fix tool_options for per-analyzer flags (`-embedlit=false` exists but unwired) or repair-step file excludes; this session is the repro | [T] |
| 13 | Investigate `go tool fix help inline` — does an ignore-class `//go:fix` directive exist that survives `go fix`? If yes, it beats excludes for future conflicts | [T] |
| 14 | File exhaustruct_v5 upstream (after verify-before-filing): v5 panics on promoted-key literals (gotcha 7) + old exhaustruct's embedded-core demand is a false positive against Go 1.27 promoted keys | [T] |
| 15 | Migrate `exhaustruct` → `exhaustruct_v5` in the module .golangci.ymls (deprecation warning on every run; gotcha 7 says directives must name v5) — verify v5 accepts the exclude regexes unchanged | [T] |
| 16 | Run `buildflow doctor` and enumerate the 9 unavailable tools; decide which belong in the devShell vs stay advisory | [B] |
| 17 | Investigate BuildFlow's "DISCOVERY DEGRADED: only packages.default was built" nix note — is the flake missing packages/checks exposure BuildFlow expects? | [B] |
| 18 | Review `buildflow timings --regressions` (advisory showed golangci-lint [setup] 8s→20s, [loginpage] 7.5s→17.9s — load artifacts or real regressions worth chasing?) | [B] |
| 19 | Examine the `e2e/package.json` change inside `04407856` (concurrent dep sweep; only code-adjacent file in that commit this session never opened) | [B] |
| 20 | Adjudicate + record the gotcha-20 CHANGELOG decision for behavior-identical published-module rewrites (identity-model/usermgmt ride the next train with or without a receipt?) | [T] |
| 21 | Verify hermetic (`GOWORK=off`) builds for identity-model/usermgmt CONSUMERS (setup, adminui, systemadapter, loginpage) — the literal rewrites are source-compatible, but the go.sum reverts in `e247ae5a` are unproven hermetically outside the two fixed modules | [T] |
| 22 | Confirm no OTHER identity-model types embed cores beyond ExternalAccountCore (rg for `Core ` embedded fields) so the exclude list is provably complete | [T] |
| 23 | Decide tag/train handling: identity-model + usermgmt source changed (behavior-identical). Does the next train bundle them, or is a standalone patch tag cleaner? (gotcha 5: verify-tag flow only) | [T] |
| 24 | Run `go fix -diff ./...` across ALL go.work members (candidate count printed, testdata excluded) to certify the whole tree is fix-clean, not just the three touched modules | [T] |
| 25 | Re-check `git log` for concurrent commits before ANY push (attribution hygiene; loginpage/handler.go is still in flight) | [T] |
| 26 | Force-refresh the golangci result cache for identity-model/usermgmt (`BUILDFLOW_NO_RESULT_CACHE=1`-equivalent) to re-certify 0 issues without cache | [B] |
| 27 | Amend HEAD daemon commit title for train readability (gotcha 27a: HEAD-only amend; c83c0e0f and deeper stay as accepted debt) | [B] |
| 28 | Run `nix flake check` (the vendorHash warning implies possible red; settles item 2's urgency) | [T] |
| 29 | `df -h /mnt/buildcache` ritual (gotcha 12): today ran heavy BuildFlow + race batteries during a concurrent sweep — confirm no fill-drain cycle starting | [B] |
| 30 | Grep repo docs for stale "add //nolint:modernize" guidance beyond gotcha 7 (split-brain references the new doctrine supersedes); update in the same pass as item 11 | [T] |
| 31 | lsp_restart to clear the two stale exhaustruct diagnostics that never faded all session (gotcha 14 hygiene; zero work-item value, pure noise cleanup) | [B] |
| 32 | Audit OTHER BuildFlow repair steps for the nolint-unaware class (which repair tools mutate code without reading golangci directives?) — inventory before the next surprise | [R] |
| 33 | Confirm root-module golangci explicitly on the rewritten serve_test.go (test-race compiled it; the go-fix DAG's lint legs covered it, but an explicit `cd . && golangci-lint run` printout closes the loop) | [B] |
| 34 | Add "probe with -diff first" to gotcha 8's documented fallback list (mechanical record of today's lesson, pairs with item 11) | [T] |
| 35 | Cross-project lesson → `references/lessons.md` in crush-config (committed, not in-session): "nolint directives do not guard toolchain fixers; config-side excludes are the only durable equilibrium" — templ-components fleet needs this too | [T] |
| 36 | Notify templ-components consumers (or its repo) that gotcha-7's nested-BaseProps+nolint convention has a known failure mode under BuildFlow go-fix; propose the same exclude treatment upstream | [R] |
| 37 | Consider per-module CI isolation design note for the `test` job (gotcha 40c) with today's flake as a second motivating case | [R] |
| 38 | Update `docs/status/README.md` live-tail count ("27 live" → stale by the last 4 reports incl. this one) | [B] |
| 39 | After the concurrent session's loginpage/handler.go lands: re-run loginpage lint/tests (its go-fix leg passed at 21:34 but the file has changed since) | [T] |
| 40 | Sweep `.buildflow.yml` comments for guidance that today's finding invalidates (the "BuildFlow's --fix can re-break intentional patterns" class now has a nolint-immune member; gotcha 8's list could name it) | [T] |
| 41 | Re-verify the two excludable types' excludes don't over-suppress: confirm only the 6 known literal sites exist for the excluded types (rg per type), so no NEW unchecked literal can silently regress | [T] |
| 42 | Add a short "fixer equilibrium" note to `docs/guides/` IF the team-facing doc layer exists for lint doctrine (else skip — AGENTS.md may be the right and only home) | [B] |
| 43 | Timing: capture one clean `buildflow -s go-fix --fix` runtime as the post-fix baseline (10.1s today) for future regression spotting | [B] |
| 44 | Check whether `transport` golden tests (WireFormatGolden) need the marker const (`fanoutBroadcastMarker`) kept out of golden capture paths — it's test-local, confirm no export | [B] |
| 45 | Run the `.#preflight-tree-check` before the next tree-mutating batch in this repo (daemon-race hygiene; today's session raced twice) | [B] |

## g) Questions I can NOT figure out myself

1. **Who drove the 21:05–21:10 concurrent sweep** (dep-update `04407856` + modernize fixer `e247ae5a`)? Daemon commits hide the driver. If it was your deliberate BuildFlow/fleet run, more waves may land while I work; if it was another agent session, we should coordinate before I touch shared files again.
2. **Push gating preference**: a concurrent session appears to have already pushed master (local ref shows 0 ahead). For MY follow-ups — do you want full-battery-first (`nix run .#test` + `check-modules`) before any push, or is module-scoped green + the two re-run steps your acceptable bar?
3. **Strategy call on the templ-components BaseProps convention** (gotcha 7): migrate the nested-literal convention fleet-wide NOW (proactive exhaustruct excludes + let `go fix` win everywhere), or leave untouched until a go-fix pass actually touches those literals? First is proactive but touches many files for zero behavior; second accepts one future surprise red per module.

---

*Point-in-time snapshot, 2026-10-14 ~21:45 CEST. Written per the status-report skill with an explicit user `.md` format override. Section (f) is docs-health HARVEST fuel — pending the user's instruction to run it.*
