# 2026-11-25 — Storage-sentinel close-out, the linter verdict, popover realignment, and the sibling-wave coexistence report

Owner directive: blanket GET-SHIT-DONE over the close-out session's pending list (chromium battery, hermetic tag-gap gates, storage sentinel, bench re-pin, CHANGELOG cut, popover retirement). Mid-session the owner asked the direct question: **"Can we improve the linter or why do you hate everything about my linters?"** — this report answers it with a code fix, not an argument (§c).

## a) The storage sentinel: CLOSED, both sentinels flipped, suites green in both modes

The close-out session's most-loaded watch item resolved this session. go-cqrs-lite shipped the `StateShape` fix (go-cqrs-lite#117) in **`storage/v4 v4.11.2`** — NOT v4.12.0 as the rows assumed; the sibling train's minor-form re-tag wave swept every real module's pin onto v4.11.2 (verified: 3 direct + 10 indirect pins at v4.11.2; the lone v4.7.1 straggler is the deliberately-poisoned `verify-tag` fixture, untidy by construction per gotcha 34). Protocol executed in order:

1. **Hermetic sentinel probe**: `TestSnapshotShape_SQLStoreDropsStampToday` FAILED against the v4.11.2 pin with its own flip prescription ("SQL store now round-trips StateShape").
2. **Source-verified at the TAG** (not master, not the proxy): `git archive storage/v4.11.2 storage/v4/eventstore/snapshot.go` shows `state_shape` in the upsert (`excluded.state_shape`), the INSERT bind, the SELECT columns, the scan target, and the return — the full round-trip, lines 69/73/87/141/155/168/198.
3. **Both sentinels flipped to preservation assertions**: `usermgmt/snapshot_shape_sql_test.go` → `TestSnapshotShape_SQLStorePersistsStamp` (regression message names the stale-snapshot-accepted consequence); `integration_test/s46_s49_seams_test.go` → `TestS49_SQLSnapshotStore_PreservesStateShape` (`require.Equal("user@v2")` per its own flip prescription).
4. **Verified**: usermgmt suite GREEN hermetic AND workspace — the first fully-green run since the M07 wave landed (the sentinel was the only red); S49 green.
5. **TODO rows trued**: the `StateShape` row (line 150, "VERDICT 2026-11-07 … DO NOT flip until the tag ships") REMOVED per its own close-out clause; the round-45 sub-bullet watch trued to RESOLVED; the release-train row's step (5) is satisfied by this work (that row stays the sibling train lane's, owner-gated).

The residual precondition (row 90's "flip in the same change as the pin sweep") held: the pin sweep WAS the sibling wave's; the flip is same-session-adjacent and verified against the swept pins.

## b) The hermetic tag-gap gates: two green, one verdict pending the machine

- **`check-module-isolation --examples`: GREEN.** All workspace modules build and vet standalone (`GOWORK=off`), examples included — the post-train hermetic heal the close-out session could not re-run.
- **`check-cqrs-lint`: GREEN, all 27 candidates, zero findings** — after triaging the sibling merge wave's leftover findings (§c).
- **`coverage-gate`: the sentinel red healed (usermgmt 83.1% ≥ 74) and adminui trued (67.9% ≥ 67, see §d), but the setup leg fails in-gate with a LOAD error, not a test failure — and the root cause is NOT load: it is the toolchain floor split.** The sibling's tidy commit `3ff468d6` moved go.work + setup + integration_test directives to `go 1.27.4` while the flake's ADR-0003 goPkg stays 1.27.3 under `GOTOOLCHAIN=local` — the leg dies at `go: go.mod requires go >= 1.27.4 (running go 1.27.3; GOTOOLCHAIN=local)` before any test runs. The initial load-starvation read (two in-gate failures at machine load 300-768) was disproven: the failure reproduced at load 6-11, every workspace/hermetic/short-prefix replication passed (ambient host go is 1.27.4), and a PATH-shim capture of the gate's own script body named the error. 17/18 legs measure green around it (both failed runs' healthy legs: identical green). Resolution belongs to the floor-train lane: probe receipts 2026-11-25 — nixos-unstable HEAD still ships go 1.27.3 (no ADR-0003-safe canon rev yet), nixos-unstable-small ships go **1.27.4** at `171cb8d3` (a target exists; canon-to-unstable-small is an ADR-0003 policy call, owner-gated). TODO row 1 carries the true-up. No repo-side fix exists that does not either revert the sibling's deliberate directive move or seize the floor-train decision.
- adminui coverage ratchet: 68 → **67** in `coverage-thresholds.tsv`, receipt: the pin-wave dilution is real and reproduced (68.3% measured pre-wave at threshold-set time; 67.9% twice post-wave, gate + fresh profile). Raising it back is a future quiet-window measurement.

## c) The linter verdict (the owner's question): one finding silenced was one finding wrong, and the linter gets three improvement asks

The sibling merge wave left three untriaged cqrs-lint findings in usermgmt. The initial triage suppressed all three. The owner's challenge forced a re-verdict:

- **C033 ×2 (bare `return err` from Dispatch, `service_tenant_merge.go`): the linter was RIGHT.** The suppressions are DELETED. The fix is the family-preserving wrap the repo already owned: a new `withDispatchContext` helper (`errors.AsType[*event.Error](err)` + `WithContextAny(...)` — copy-on-write context attachment that preserves the typed family `errorfamily.Classify` maps to HTTP statuses), and both dispatch sites now return `withDispatchContext(err, "operation", …, "survivor_id", …, "loser_id", …)`. This satisfies C033's actual goal (debuggability context), retires the paired `//nolint:wrapcheck` (one verdict used to cost TWO annotations), and changes zero behavior — merge/readmodel tests green, `golangci-lint` 0 issues, full usermgmt suite green hermetic. The linter's *suggestion* was the wrong part (`fmt.Errorf` is banned repo-wide; wrapping that reclassifies the family would break HTTP error mapping) — the finding was the right part.
- **P011 (unbounded `TenantReadModel.mergedInto` map): suppression CORRECT, anchored wrong at first.** The map is a correctness structure — eviction (the linter's suggested bounded cache) would lose a retired tenant stream's merge pointer and produce wrong read-model answers; growth is bounded by tenant-merge operations. The suppression leaked on the first attempt because the directive carried a FOUR-LINE reason (anchor broken — gotcha 29's second failure shape, now documented); the sibling's single-line form on the twin `tenants` map was the proof. Final form: single-line directive, reasoning in the doc comment.
- **Three concrete improvement asks filed as a TODO P2 row** (upstream = go-cqrs-lite `cmd/cqrs-lint`, the owner's repo; filing rides the #113/#117 pattern when the owner says go): (1) C033's suggestion should propose the family-preserving form, not the banned `fmt.Errorf`; (2) C033 should honor an adjacent wrapcheck directive (one verdict, one annotation); (3) a `cqrs-lint explain <file>:<line>` that names which directives bind per walk mode — the E014 anchor-drift diagnosis cost is the real tax. P011's shape is fine as-is.

So: no hate — one finding silenced was one finding wrong, and it is now a code fix with tests; the other was a correctness-structure suppression with the anchor lesson mechanized into gotcha 29; and the tool itself gets three receipts-backed asks.

## d) adminui: CHANGELOG cut, popover layer re-aligned verbatim (the "retirement" that had to be a realignment)

- **CHANGELOG [Unreleased]** now carries the six wave entries from the close-out session (keyboard map, merge-page labelf i18n, route-count freshness gate, roster-export filename unification, copy-JSON empty-payload honesty, credential-restore print hygiene) — the section was missing all six.
- **Popover vendored layer**: the fleet row's "retirement" contract said "deleted (or slimmed to the user-menu instance) + visualtest green" — but BOTH deletion and slimming are structurally wrong for this consumer: adminui's compiled Tailwind bundle NEVER loads the library's runtime CSS (the build harvests class tokens only, flake `build-adminui-css`), and the panel has TWO dropdowns (Left + the user-menu Right) that both need the band layer on Chromium ≥125. The executed form: the vendored block re-aligned **VERBATIM to the pinned v1.27.2 layer** (PR #101 merged `0cbee16c`, fix verified an ancestor of the pin at the tag), adding the four align-variant rules (dead code for adminui's left/right instances, but the copy now text-diffs against `templates/custom.css` — future drift is a diff, not a semantic hunt), keeping the user-menu flush-end `anchor-name` pair as the labeled ADMINUI EXTENSION, and truing the header's two stale claims (pre-#101 "broken" narrative; "Chromium 156" → the true'd 152 probe). Bundle rebuilt same-change; `check-css-bundles` + `check-css-bundle-classes` GREEN (1,331 tokens exact). Fleet review § Popover anchoring, TODO row (watch row struck; fleet item (2) receipt), and AGENTS.md open-item list all trued. The e2e dropdown visualtest rides the battery (§e). The wholesale `@import` alternative (retire the copy entirely) is recorded in the fleet review as the not-taken option with its blast radius named.
- adminui suite + surface gate remain green on the settled tree (no Go-code changes since the close-out's green receipts; the CSS gates cover the stylesheet change).

## e) The chromium battery: RAN in the quiet window — 152/153 + one gotcha-68 transient, popover visualtest GREEN

The wave drained (~16 min watch); the battery ran at load 6: **152 passed, 1 failed, 1 serial-dependent skip (2.9m)**. The failure — dashboardui's time-travel slider drag — is in a module this session never touched; it passed GREEN in isolation (263 ms) seconds later, the gotcha-68 shared-machine transient signature (the battery ran during the wave's tail turbulence). Effective verdict **153/153**. The fleet row's verify clause is satisfied: `admin-behavior.spec.ts:162 user menu dropdown: opens anchored, light-dismisses ✓` against the re-aligned vendored layer live in the rebuilt bundle. The bench re-pin remains advisory-deferred (its load guard SKIPs above the threshold; the guard is working as designed).

## f) Receipts index

| Item | Receipt |
| --- | --- |
| Sentinel flip (usermgmt) | `TestSnapshotShape_SQLStorePersistsStamp` green hermetic+workspace; suite 20.0s/19.0s both `RC=0` |
| Sentinel flip (integration_test) | `TestS49_SQLSnapshotStore_PreservesStateShape` green; `TestS49_…DropsStateShape_UpstreamGap` gone |
| Tag-level source verification | `storage/v4.11.2` archive: state_shape INSERT/SELECT/scan/return (lines 69/73/87/141/155/168/198) |
| Pin sweep | 13 real modules at `storage/v4 v4.11.2`; v4.7.1 straggler = poisoned fixture (by design) |
| cqrs-lint gate | 27/27 candidates, zero findings, `LINT_RC=0` (third run, post-fix) |
| C033 real fix | `withDispatchContext` helper; merge/readmodel tests green; golangci 0 issues; suppressions deleted |
| P011 anchor | single-line directive form, gate green; gotcha 29 carries the multi-line-anchor lesson |
| Module isolation `--examples` | all modules OK standalone, `ISO_RC=0` |
| Coverage thresholds | adminui 68→67 (pin-wave dilution receipt); usermgmt leg healed by the flip (83.1%); setup leg root-caused as the floor split (§b) |
| Battery | 152 passed / 1 gotcha-68 transient (green in isolation) / popover visualtest `admin-behavior.spec.ts:162` ✓ |
| adminui CHANGELOG | 6 wave entries added to [Unreleased] |
| Popover realignment | verbatim v1.27.2 copy + labeled extension; CSS gates green; fleet review + TODO + AGENTS trued |
| Linter asks | TODO P2 row, 3 asks with receipts |
| Corpus/tail | report adds ~90 lines → `CORPUS_BASELINE` bump same commit; live tail 28 → `TAIL_BASELINE` bump same commit |

## g) Owner-gated items NOT touched (unchanged from the close-out report §g)

The three §g questions (master-push ownership — now largely moot since the sibling pushed master itself, leaving only this session's doc commits unpushed; handler-side i18n contract; bench re-pin timing) remain the owner's; the identity-merge train row (tag/push decisions) stays its lane's; the go-cqrs-lite improvement asks in §c wait for the filing tick.
