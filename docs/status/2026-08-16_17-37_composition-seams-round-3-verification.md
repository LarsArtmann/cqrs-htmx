# Round-3 Composability Work — Verification Session Status

**Date:** 2026-08-16 17:37
**Scope:** This session only — resumed after round-3 self-review, re-synced stale todos, applied P1 fixes, ran verification gates, diagnosed + fixed a pre-existing systemadapter lint breakage.
**Tree state:** clean at `66195d5f` (see §d — auto-git daemon absorbed this session's work).

---

## a) FULLY DONE

1. **Todo re-sync** — the inherited list was stale (3 items marked pending were already done); corrected against actual tree state first.
2. **Tree/commit verification** — round-3 work was committed as `62544247`; concurrent sessions landed `3a57cc40` (usermgmt checkpoint-hydrate) and `342465c3` (httputil v0.12.0 bump) on top. No re-verification races this time.
3. **gci import-order "issue" in setup/sse.go** — was **stale LSP diagnostics**, not a real finding. `golangci-lint fmt --diff` clean, `golangci-lint run` 0 issues. Closed with zero changes. (Third time stale LSP has cost a phantom work item.)
4. **dashboardui deprecated-call migration (P1)** — `dashboard.go:64` now calls `transport.NewJournalSSEStore` instead of the deprecated root re-export. Confirmed `cqrshtmx` import remains legitimately used (Broadcaster, lines 27/58). Build+vet+test green.
5. **Root doc.go updated** — JournalSSEStore example now shows the canonical `transport.` form; documents `transport.DomainEventToSSE` as the canonical envelope and when to pass a custom mapper.
6. **fakeBus → eventtest.NewFakeBus (P1)** — setup/sse_internal_test.go now uses the shared upstream fake (same as dashboardui), hand-rolled 13-line fake deleted.
7. **Golden byte-level wire test (P1)** — `TestDomainEventToSSE_WireFormatGolden` pins the exact serialized envelope bytes (field order + key spelling). The envelope is published language; drift is now a compile-time-visible failure.
8. **SSE heartbeat parity (P1)** — `setup.Config.SSEHeartbeatInterval` (default 15s, non-positive disables), wired into `sseHandler` with dashboardui's cancellable-context + deferred-join pattern (no ResponseWriter-after-return race), covered by `TestBundle_SSEHeartbeatEmission`. setup now matches dashboardui's SSE feature set.
9. **maxReplay "gap" resolved as non-gap** — dashboardui has no maxReplay either; go-sse `Replay` takes no cap option. Downgraded to a shared P2 (unbounded first-connect backfill), not a setup defect.
10. **systemadapter lint breakage root-caused + fixed (pre-existing)** — see §d/§e for the saga. Fix: TEMPORARY `replace metaengine/v4 => ../../go-cqrs-lite/metaengine` in systemadapter/go.mod with removal condition (metaengine/v4 v4.12.0+).
11. **Lint gate GREEN** — all 15 modules, 0 issues. Fixed en route: `goconst` (4x `"event"` → `sseEventType` const) and `cyclop` (test complexity 13>12 → extracted `assertPayloadFields` helper).
12. **Build gate GREEN** — all 26 modules hermetic (GOWORK=off).

## b) PARTIALLY DONE

- **Verification gates** — lint ✓, build ✓; `#check-modules`, `#check-cqrs-lint`, `#coverage-gate`, `#test`, race/fuzz/flake NOT yet run this session (interrupted by this status request). Previous session's evidence is stale given new changes.

## c) NOT STARTED

- `integration_test` + `examples/setup-demo` + `examples/dashboard-demo` explicit test runs (they built in the build gate; tests not executed).
- Docs: `setup/README.md` + `setup/doc.go` still describe the old `/sse` behavior (no replay/backfill/heartbeat mention).
- CHANGELOG entries (heartbeat, golden test, deprecated-call migration, systemadapter replace, fakeBus swap, doc.go).
- AGENTS.md updates (transport canonical envelope, systemadapter metaengine/v4 replace + removal condition).
- TODO_LIST harvest of round-3 report P1/P2 items + this session's findings.

## d) TOTALLY FUCKED UP

1. **Auto-git daemon absorbed the entire session's work into a mislabeled commit.** All 10 changed files landed as `66195d5f "feat(setup): wire configurable SSE heartbeat into setup/Bundle"` — with `Assisted-by: Crush:MiniMax-M3` attribution. The message describes only 1 of 10 files; the golden test, systemadapter replace, deprecated-call migration, and doc.go rewrite are invisible in history. No damage (tree converges, code is correct), but it's the documented AGENTS.md trap firing AGAIN because I **did not commit immediately after lint+build went green**. Lesson violated, not learned.
2. **Wrong first hypothesis on the systemadapter lint failure.** I chased `golangci-lint cache clean` (twice: system binary + `nix develop`) before reading `flake.nix`'s `goEnv` — which exports `GOWORK=off`, meaning the nix lint never used go.work replaces. Two wasted round trips; the resolution-mode question (`go list -m -json` in GOWORK=off) should have been step 1.
3. **Minor:** one edit failed on a stale anchor (`newFakeBus` comment already removed by my own earlier edit in the same session) — recovered, but I edited against a mental file image instead of the current one.

## e) WHAT WE SHOULD IMPROVE

1. **Commit after EVERY green gate.** The daemon does not wait. This is now twice-documented, twice-violated.
2. **check-modules blind spot: `go build` doesn't compile `_test.go`.** The systemadapter breakage (blank-import of local sqliteengine in a test file, resolving against published metaengine v4.11.0 which lacks `StreamLogEntry`) passed `#build` and `#check-modules` — only the lint gate (which typechecks tests) caught it. The AGENTS.md gotcha hit the gate infrastructure itself. **Recommend: add `go vet ./...` to the check-modules isolation script** (vet compiles tests).
3. **The dead-replace guard checks existence, not API drift.** All replace targets had go.mod files; the breakage was upstream-master-vs-published-tag API drift. A GOWORK=off vet per module (item 2) covers this class.
4. **Trust `golangci-lint run`/`fmt --diff` over IDE LSP diagnostics.** Session opened with 4 warnings on setup/sse.go — all stale. The LSP needs a restart after heavy multi-module edits; better: stop creating todo items from LSP output.
5. **Upstream toolchain drift signal:** `go-cqrs-lite/go.work` now requires `go >= 1.26.6` while this repo pins 1.26.5 (`GOTOOLCHAIN=local`). Standalone builds of upstream submodules already fail on it. Needs a conscious decision, not silent drift.
6. **setup and dashboardui now share ~40 lines of near-identical SSE handler shape** (subscribe → connected → replay → heartbeat-join → pump). A `transport.ServeDomainEvents(...)` helper would eliminate the copy; deliberately deferred (behavioral parity was the priority this round).

## f) NEXT — up to 50 items

**Gates (finish the sweep):**

1. `nix run .#check-modules` — must bless: setup's 2 dev replaces, dashboardui's 1, systemadapter's NEW metaengine/v4 replace, storage/v4 v4.7.1 everywhere
2. `nix run .#check-cqrs-lint`
3. `nix run .#coverage-gate` — root ≥90 with new transport lines; setup ≥80 with heartbeat paths
4. `nix run .#test` — full 17-suite run
5. `nix run .#test-race` (or `GOEXPERIMENT=jsonv2 go test -race` on setup + dashboardui + transport at minimum — new goroutine/heartbeat code)
6. `nix run .#test-fuzz` + `.#test-flake` (periodic re-validation)
7. `nix flake check --no-build`
8. `nix run .#check-codegen` (templ drift)
9. `nix run .#check-templates` (cheap, untouched but verifies)
10. integration_test: `GOWORK=off go test ./...`
11. examples/setup-demo: `GOWORK=off go test ./...` (e2e asserts 401 gates + authed flow — consumes new setup code transitively)
12. examples/dashboard-demo: build + test (dashboardui changed)

**Improvements (from §e):**
13. Add `go vet ./...` to check-modules isolation script (test-compiling gate)
14. Commit discipline: commit immediately after each green gate
15. Decide toolchain: bump to go 1.26.6 (nix pin + go.mod directives) or pin upstream expectation — see question 3

**Refactors:**
16. Extract shared SSE handler shape into `transport` (subscribe→replay→heartbeat→pump) — kills the setup/dashboardui copy
17. Shared P2: cap first-connect backfill (both endpoints stream unbounded replay; `sse.Replay` has no max — consider `ReplayFiltered` or a wrapped EventStore)
18. Golden-test the marshal-failure fallback path (Event with no Data)
19. Consider `sse.ReplayFiltered` as basis for a stream-type filter on `/sse` (pairs with open question 2)

**Docs:**
20. `setup/README.md`: `/sse` endpoint now replays (Last-Event-ID), backfills on first connect, heartbeats every 15s (configurable); document `SSEHeartbeatInterval` in the config table
21. `setup/doc.go`: same corrections
22. `CHANGELOG.md`: heartbeat, golden wire test, deprecated-call migration, systemadapter metaengine/v4 replace, fakeBus swap, doc.go rewrite (note the 66195d5f absorption so history is discoverable)
23. `AGENTS.md`: transport canonical envelope (DomainEventToSSE + EventPayload + golden test); systemadapter metaengine/v4 replace entry + removal condition (v4.12.0+); "lint gate typechecks tests, build doesn't" reinforcement
24. `TODO_LIST.md`: harvest round-3 report P1/P2s (this session closed several: golden test ✓, fakeBus ✓, heartbeat ✓, deprecated call ✓) + add items 16/17
25. Note in CHANGELOG/AGENTS that `66195d5f` contains more than its message says (attribution + inventory)

**Upstream (go-cqrs-lite) asks — do not act locally:**
26. Tag metaengine/v4 v4.12.0+ (StreamLogEntry/SeqSeekableStreamLog) → strip systemadapter replace
27. Tag sqliteengine v4.0.2 → strip replace
28. Tag projectionadapter v4.5.0 → strip replace (round-3 §3 item 1, still open)
29. `event.Bus` Unsubscribe API (P3, tracked — the sseDone channel is the only teardown)
30. Toolchain alignment (1.26.6) — coordinate with item 15

**Release train (blocked on user answers):**
31. Family tag v4.8.1/v4.9.0: publishes `transport.DomainEventToSSE`/`EventPayload`, setup replay+heartbeat, dashboardui migration → then strip 3 family dev replaces (setup×2, dashboardui×1)
32. After tag: verify hermetic builds of all consumers from proxy (proven recipe: `GOWORK=off go mod tidy && go build && go vet` per module)

**Carried from round-3 report (still open, P2+):**
33. Byte-golden wire test ✓ DONE this session (listed for closure tracking)
34. Heartbeat option ✓ DONE this session
35. fakeBus swap ✓ DONE this session
36. `cqrshtmx` import check in dashboardui ✓ DONE this session (still needed)
37. Bridge-unsubscribe: impossible without upstream API — P3, do not act
38. `/sse` stream-type filter — awaiting answer (question 2)
39. Concurrent-session authority protocol — awaiting answer (question 1)

**Housekeeping:**
40. Verify CI workflow needs no module-list updates (no new modules this session)
41. Verify `check-replace-directives.sh` accepts the new systemadapter replace (target has go.mod — build passing implies yes, but run the gate)
42. Consider an AGENTS.md line: "LSP diagnostics after multi-module edits are unreliable — verify with golangci-lint before creating work items"
43. e2e/server: cheap build check (untouched; build gate covers it)

## g) QUESTIONS (cannot resolve myself)

1. **(Carried from round-3) Family tag timing:** cut v4.8.1/v4.9.0 now to publish `transport.DomainEventToSSE` + setup replay/heartbeat + dashboardui migration and strip the 3 dev replaces — or hold for more changes? The replaces are documented and gates can bless them either way.
2. **(Carried from round-3) `/sse` authorization:** is session-gating sufficient for replaying historical event _metadata_ (stream IDs, types, timestamps — no payloads), or should a stream-type filter land before we publish the endpoint shape?
3. **Toolchain drift:** upstream `go-cqrs-lite/go.work` now requires **go ≥ 1.26.6**; this repo pins **1.26.5** with `GOTOOLCHAIN=local` (nix). Standalone builds inside the upstream repo already fail. Bump this repo to 1.26.6, or is 1.26.5 a deliberate freeze I should respect?

---

_Session inventory: 10 files changed, all committed (by daemon) in `66195d5f`. Lint 0/15 modules. Build 26/26. Remaining gates unrun this session. Waiting for instructions._
