# Fleet-Plan Wave 5 Status — A8/A13/A16/A18/A19/A20/A21 landed; usermgmt SA1019 sweep done; A22 mid-flight

**Date:** 2026-11-06 22:03 CET
**Session:** fleet-plan wave 5 (resumption 3), plan of record `docs/planning/2026-11-06_00-37_SUPERB-fleet-libs-audit-follow-through-pareto-plan.md`
**Directive:** "NOW GET SHIT DONE! The WHOLE TODO LIST!" — executed A8 onward; report demanded mid-A22.
**Predecessors:** 13-24 report (A7), 05-24 report (waves 1-2 state), 02-44 report (waves 1-2 execution, archived by THIS report's tail ritual).

## §a What got done (all verified green unless noted)

1. **A8 examples/basic partial-render showcase — COMPLETE.** `/demo/items` serves three shapes from two endpoints: direct visit = full templ page, HTMX request = list fragment + `OOBHTML("item-count", …)` out-of-band badge (`RenderPartialOrFull`, typed mappers); `/demo/items/count` answers the bare count when `HX-Target` is `#item-count` (`RenderIf` custom predicate via `htmx.Target`, the direct-import spelling) and the full page otherwise; history-restore renders full (pinned). First templ usage in the example: `demo.templ` + generated `_templ.go` (pinned v0.3.1100 via `nix build .#templGen` — the ambient templ is the gotcha-10 trap, dodged cleanly this time, zero stamp churn). Result-type guard uses `errorfamily.NewRejection` (banned-stdlib-errors rule holds even in examples). Receipt: `TestSmoke_PartialRenderShowcase` (5 subtests) green under `-race`; hermetic `GOWORK=off` build OK (examples are NOT in CI's lint matrix — verified). README + package doc + root CHANGELOG + skill-doc examples bullet updated.

2. **A13 L1 tail — CLOSED (pre-landed work verified, receipts + TODO truth written).** Found the sibling sessions had already landed ALL the code: `TestAuditLog_ConcurrentAppendsUnderRace` (8 writers × 64 events vs 4 readers over a 64-entry window that WRAPS, exact Count/Dropped, distinct-AggregateID lost-slot detection — verified green under `-race`) and both A013 pointer-embeds (`e2e/server/sync_types.go`, `examples/sync-demo/main.go`, branching-flow verdicts frozen). My delta: receipt runs (sync-demo `-race` suite green; e2e/server build+vet clean), the missing root-CHANGELOG L1-completion receipt, and the TODO_LIST row trued (L1 tail + F1 struck done).

3. **A16 systemadapter engine presets — COMPLETE.** Five breadth presets (`RecommendedPebbleDeployment(dir)`, `RecommendedBBoltDeployment(path)`, `RecommendedPostgresDeployment(dsn)`, `RecommendedDuckDBDeployment(path)`, `RecommendedTursoDeployment(dsn)`) sharing a `singleEngineDeployment` core (one engine holds every role — the right topology for embedded/single-server). Dependency discipline is the design decision: the four heavy engines are CONFIG-ONLY (their drivers self-register in the CONSUMER's binary via a documented one-line blank import), and exactly one light leaf dep enters the module — bboltengine v4.3.3 (pure Go) as the family's construction proof: `TestRecommendedBBoltDeployment_EndToEnd` drives system.New → RegisterUser → drain → FindByEmail on a real bbolt file. Dep budget 17→18 justified in the gate. `examples/system-demo` gained `-engine memory|sqlite|bbolt` + `-dsn` flags selecting presets. Fixed in passing: a PRE-EXISTING wrapcheck on `architecture_catalog.go` (sibling session's file, would have reddened CI's lint matrix) plus gci/prealloc follow-ups; systemadapter lint **0 issues**, suite green, dep-budget gate green. README table + guide §breadth + CHANGELOG receipt.

4. **A18 codec AutoDetect stamps decision — COMPLETE (N/A verdict).** ADR-0056 Amendment 2 records why the lying-stamp path does NOT adopt `codec.AutoDetect`: the heuristic's own contract excludes the use ("diagnostics and tooling — NOT a security boundary"), `opaque` is the honest no-promise shape by design, and client obligations stay minimal/stable. Revisit trigger recorded (a VERIFIED framing-detection API upstream).

5. **A19 codec Diagnose in the payload inspector — COMPLETE.** `DefaultPayloadRenderer` gained the degraded path: when the stamped encoding fails to decode (corrupt CBOR OR the lying-JSON-stamp trap), the payload renders as CBOR extended diagnostic notation via `codec.Diagnose` instead of surfacing binary bytes in the UI; undiagnosable bytes keep the corruption error and `RenderPayload`'s raw-bytes last resort. Happy paths unchanged (valid CBOR still pretty-prints JSON — pinned). Receipt: `core/payload_diagnose_test.go` ×3. dashboardui lint 0, suite green, CHANGELOG receipt.

6. **A20 codec EncodePooled bench spike — COMPLETE (DROP verdict + upstream ask).** `usermgmt/codec_pool_bench_test.go` measures the exact seam upstream `event.New` plain-Encodes (`event_new.go:94`): pooled wins the median 245.8 vs 259.3 ns/op (5×2s runs) with 497→208 B/op, and the real path costs ~589 ns / 1459 B / 10 allocs per `CredentialAdded`. cqrs-htmx CANNOT adopt (zero cqrs-htmx-owned CBOR encode call sites — every encode rides upstream `event.New`), so the win is only reachable upstream: recorded as upstream ask (g) in TODO_LIST row 51 (owner-gated filing), bench stays as receipt (not in the machine-pinned gate). Also wrote usermgmt's MISSING A1/R2 `ArchitectureCatalog` CHANGELOG receipt while in the file (CHANGELOG-debt item partially cleared).

7. **usermgmt SA1019 sweep — COMPLETE (final-battery item pulled forward).** 47 deprecated-alias sites repointed across 6 files to direct subpackage imports (`httperr.WithHTTPStatus/MapError/ErrorCode`, `eventcatalog.New/Handler/Catalog/EventMetadata/PayloadField`); unused `cqrshtmx` root imports dropped where the file no longer needs them; two table globals (`eventCatalogSpecs`, `everyDispatchedCommand`) carry the repo-convention `//nolint:gochecknoglobals` with reasons. usermgmt lint **0 issues**, full suite green.

8. **A21 templ charts panel (event throughput) — COMPLETE, the module's first charts consumer.** `/throughput`: journal events bucketed into fixed windows (default last 24h / 12 two-hour buckets ending at the NEWEST event; `?buckets=` clamped 4–48) rendered as a pure-SVG `display.LineChart` — zero client JavaScript, strict CSP holds by construction. `core.LoadEventThroughput` reuses the journal-tail rolling paging (10k cap) and reports an HONEST window (bucket width, in-window count, off-chart older count, scan count — subtitle carries all of it). Route gated `HasEventRead` like Command Fates; nav row `icons.Chart`; `EmptyStateThroughput` copy; manifest pins trued (`wantPanels` +Throughput:1, journal-only 11→12). Receipts: core bucket-math tests (incl. off-chart accounting + arg validation) and panel tests (render incl. `<svg` + subtitle, empty state, route gating, buckets parsing). CSS bundle rebuilt same-change (byte-identical — classes already in the set; both bundle gates green). Coverage 73.4% (threshold 60), lint 0, README (capability table + section + adoption-table row) + CHANGELOG receipt.

9. **A22 appkit lifecycle-hook forwarding — MID-FLIGHT (interrupted by this report demand).** Landed-but-UNVERIFIED: `setup.Config` gained `AppkitStartHooks`/`AppkitDrainHooks`/`AppkitShutdownHooks []appkit.Hook` (full doc comments; appkit v0.10.0's three-phase hook contract) and `run_appkit.go` forwards all three into `appkit.ServiceConfig`. NOT done: setup build/test/vet, the B93 shutdown-order test, B94 docs note, CHANGELOG receipt. First next action: `cd setup && GOEXPERIMENT=jsonv2 go build ./... && go test`.

## §b Deviations from the plan

- **A13 executed as verification + receipts, not new code** — the sibling sessions (2026-11-05/06) had already landed every micro-task; the plan table's "20%" status was stale. Same for the usermgmt SA1019 class (pre-existing findings, not mine — fixed anyway to keep CI green).
- **A16 scope decision:** the plan said "pebble/bbolt/postgres/duckdb/turso behind options"; I shipped all five as config-only presets + ONE real-engine proof (bbolt) rather than five construction tests — pebble (~30-module tree), pgx, duckdb (prebuilt C bindings), and turso/purego do not belong in a library module's graph. Documented per-preset; the audit's actual ask ("one preset function away from demoable per engine") is fully satisfied.
- **A20 verdict is DROP, not adopt**, despite the median winning — adoption requires a call site and cqrs-htmx owns none (the seam is upstream `event.New`). The plan's "adopt only if median wins" bar is met upstream only; recorded as upstream ask instead.
- **Final-battery SA1019 sweep was partially pulled forward** into per-module work (usermgmt this session; systemadapter inside A16; dashboardui last session) because each module's lint had to be green for its own task's verification anyway.

## §c What could blow up in our face (risks)

- **A22 edits are committed-unverified** (daemon sweeps every few minutes): if `appkit.Hook` or the config fields miscompile, setup's CI lint/test reddens until B92-B94 complete. Low risk (3-field forwarding, appkit already a direct dep), but it IS an open edge.
- **Remaining lint-matrix modules are UNMEASURED** for the SA1019 class: root `.`, adminui, identity-model, loginpage, datastar, huma, setup, health, auditlog. dashboardui/usermgmt/systemadapter are proven clean; the other nine may still carry findings that redden CI (the exact dashboardui-11-findings class from wave 4).
- **templ v0.3.1100 stamp discipline:** two new `_templ.go` files this session (examples/basic, dashboardui throughput) — both generated with the pinned flake generator, stamps verified; but ANY future `templ generate` by an ambient v0.3.1114 churns 13+ files now.
- **Dep budget 18/18:** systemadapter has ZERO slots again; the next dep there needs a fresh justification line.
- **Tail/corpus rituals this report:** archiving the 02-44 report (below) keeps the tail at 18; corpus baseline rises with this file (ledger row written same-change).

## §d What I totally fucked up (honest mistakes)

- **Two edit-tool misfires** (A8: a helper-insert edit designed twice, one inverted; a demo.templ import edit sent to main.go) — caught by verify-first habits, zero damage, but they cost a round trip each.
- **`architecture_catalog.go` prealloc fix broke the composite literal** (swapped the literal opener for `make()` without rewriting the body) — caught by immediate re-read; repaired before any build.
- **Throughput test expectations wrong twice:** first fixture derived ages from `base` instead of the NEWEST event (bucketing is newest-relative); the same failing test then exposed a REAL off-by-one-window label bug in my loader (labels were bucket-END times). Both fixed; the test earned its keep.
- **`rg -rn` struck again** (the hook's third-strike class) — known trap, re-ran with `rg -n`.
- **templ API guesswork:** wrote `p.base()` for BaseProps before reading how sibling templ files do it (`utils.BaseProps{...}` literals) — one iteration lost to a compile error I could have avoided by reading first.

## §e What could be improved (implementation/quality)

- Read sibling test fakes BEFORE writing test fixtures (the fakeCommandJournal-is-for-commands stumble in A21's panel test — the event journal fake is `fakeSeekableJournal`).
- Derive expected values from the code's frame of reference (newest-relative ages) before writing assertions — two of my three test-math errors were frame-of-reference mistakes.
- Batch `.golangci.yml` exhaustruct ignores only after trying a fully-constructed literal; A21 followed the A7 precedent without re-litigating it (fine, but the literal was 4 fields away).
- The Diagnose fallback (A19) and the throughput loader (A21) both introduce display-time CPU on the request path (trial decode / bucket scan) — bounded (10k scan cap, 1 MiB diagnose cap upstream) but worth remembering if panels get hot.

## §f Next tasks (up to 50, in execution order)

1. A22 finish: setup build + vet + suite; B93 shutdown-order test (drain-hook → shutdown-hook → bundle-Close ordering, hooks forwarded proof); B94 docs note; setup CHANGELOG receipt.
2. A23 systemd docs note (production-readiness guide §: RunWithAppkit + StartHook `sd_notify READY=1` + DrainHook `STOPPING=1` — rides A22's hooks).
3. A24 atomic-write `cmd/cqrs-htmx-upgrade/rewrite.go` + fingerprint-failure test.
4. CHANGELOG debt: dashboardui A2 SSE-hub leg receipt (verify sibling wrote it), health A3 receipt, A27-check (upgrade-CLI ExitCode + ErrorContext receipts exist?).
5. A2 adminui SSEHubStats leg (unparked): re-verify adminui lane quiet, `Config.SSEHubStats` additive card, suite + CHANGELOG, rebase care around config.go/handler_dashboard.go.
6. Remaining SA1019 sweep: run per-module lint on root/adminui/identity-model/loginpage/datastar/huma/setup/health/auditlog; repoint findings; each module lint 0.
7. Final battery: full `nix run .#check-modules` (incl. --examples leg, tidy-sweep, dep budgets, reexport policy, route postures); `.#check-erraudit-strict` advisory; `.#check-readme-badges`; `docs/review/libs/index.html` go-etag row correction (erratum 5); plan Table A trues (A8/A13/A16/A18/A19/A20/A21 → done); `rm -rf /tmp/etag-probe /tmp/archive-spec.tsv` (this ritual re-created the spec file — removed after apply); TODO_LIST sweep-6/row-34 style trues for A16/A21 landings.
8. Corpus baseline + ledger row for THIS report (done same-change, see §h); TAIL_BASELINE unchanged (archive 1 + add 1 = 18, gate green).
9. Wave-6 status report at the next natural boundary (after A2-adminui + final battery) — archive ritual candidate: the 05-24 report.

## §g Questions (max 3)

1. **identity-model v4.16.0 tag (carried):** cut when the concurrent lane is quiet, or leave to you? (No push performed — standing rule.)
2. **A17 chromium screenshots (carried):** sanction nix-profile chromium for the render lane, or waive to structural verification?
3. **A22 hook-ordering contract (NEW):** currently `RunWithAppkit` runs consumer ShutdownHooks BEFORE the bundle's own `Close()` (telemetry flushes while bundle stores/projections/SSE hub still live; appkit's own drain flips readiness first). I documented this order; confirm it is the desired contract or say the word and I'll flip Close first (behavior change, needs its own test + receipt).

## §h Ritual receipts (this report)

- **Archived:** `2026-11-06_02-44_fleet-plan-execution-waves-1-2-status.md` → `docs/status/archived/` (SUPERSEDED by this report; A26 unit 3 of 3 — the archive round is COMPLETE). Tail stays 18 (add 1 + archive 1); `--ratchet` green.
- **Corpus:** baseline updated same-change with the ledger row in `docs/status/README.md`.
- **Survival check:** performed post-write (daemon race guard).
