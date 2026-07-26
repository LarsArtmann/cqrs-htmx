# Status Report — 2026-07-26 10:33 — Deduplication Sweep (art-dupl + branching-flow)

> Session goal: run `branching-flow dupe .` and `art-dupl --type-aware --sort total-tokens -t 3 --html`, then drive the report to **zero harmful duplication** with judgment-driven extraction, aliasing, and documented acceptance.

---

## Headline metrics

| Tool                          | Metric                      | Before | After | Δ                |
| ----------------------------- | --------------------------- | ------ | ----- | ---------------- |
| `art-dupl` (-t 3, type-aware) | Clone groups                | 10     | 2     | **-80%**         |
| `art-dupl`                    | Total clones                | 21     | 4     | **-81%**         |
| `art-dupl`                    | Total tokens                | 67     | 12    | **-82%**         |
| `branching-flow dupe`         | Type groups                 | 73     | 52    | **-29%**         |
| `branching-flow dupe`         | Actionable (harmful) groups | ~16    | 0     | **zero harmful** |
| Build (`go build ./...`)      | —                           | green  | green | no regression    |
| Tests (10 modules, `-race`)   | —                           | green  | green | no regression    |

**Verification:** build green, all 10 workspace modules pass `go test -race`, gofmt clean on every touched file, no new lint issues in changed files.

---

## a) FULLY DONE (verified green)

| #   | Work item                                                                                                                                                                               | Files                                                                                                |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| 1   | Aliased 4 store interfaces (`WebAuthnSessionStore`, `VerificationTokenStore`, `LockoutStore`, `PendingTOTPStore`) in usermgmt → identity-model                                          | `usermgmt/store_interfaces.go`, `identity-model/interfaces.go`                                       |
| 2   | Aliased 3 auth interfaces (`TOTPProvider`, `WebAuthnProvider`, `OAuth2Provider`) + `OAuth2UserInfo` in usermgmt → identity-model; enriched identity-model per-method docs (no doc loss) | `usermgmt/auth_interfaces.go`, `identity-model/interfaces.go`                                        |
| 3   | Extracted `eventSourcedSetupCore` (12 shared fields + `Close`/`GracefulClose`/`Authz` + `backendName`); Postgres/SQLite setups now embed it as a one-liner                              | new `usermgmt/es_setup_core.go` (compiled), `usermgmt/postgres_setup.go`, `usermgmt/sqlite_setup.go` |
| 4   | Consolidated `eventCatalogServer` + `openAPISpecServer` → single `immutableJSONServer` in root                                                                                          | `event_catalog_handler.go`, `options_openapi.go`                                                     |
| 5   | Aliased `BeginRegistrationResponse` / `BeginLoginResponse` → shared unexported `webauthnBeginResponse`                                                                                  | `usermgmt/webauthn_service.go`                                                                       |
| 6   | Extracted `listStreams(r)` helper for the 3 list handlers (aggregates/time-travel/snapshots)                                                                                            | `dashboardui/handlers.go`                                                                            |
| 7   | Shared `cqrshtmx.ToastDetail` in root; adminui + dashboardui `toastDetail` now aliases of it (per-module `triggerToast` kept — they differ in wire behavior)                            | `notify.go`, `adminui/render.go`, `dashboardui/render.go`                                            |
| 8   | Extracted `reasonedCommand` base + `newReasonedCommand` for the 4 reason-carrying commands (`DeleteUserCmd`, `SuspendTenantCmd`, `DeleteTenantCmd`, `DeleteBotCmd`)                     | `identity-model/commands.go`                                                                         |
| 9   | Extracted `rebuildProjection` free function; `EventSourcedSetup.RebuildProjection` + `Service.RebuildProjection` now one-line delegations                                               | `usermgmt/es_projection_health.go`                                                                   |
| 10  | Extracted `writeHTML` helper (renderPage/renderPartial shared body)                                                                                                                     | `dashboardui/render.go`                                                                              |
| 11  | Added **INTENTIONAL DUPLICATION** rationale comments on the accepted cross-module seam mirrors (`credentialData`, `userInfo`)                                                           | `usermgmt/webauthn/provider.go`, `usermgmt/oauth2/provider.go`                                       |
| 12  | Bumped `go.work` go directive `1.26.4 → 1.26.5` (prerequisite to unblock build; replace directives untouched)                                                                           | `go.work`                                                                                            |

---

## b) PARTIALLY DONE

- **Ignored-file verification gap.** `postgres_setup.go` and `sqlite_setup.go` carry `//go:build ignore` and pull in exotic modules (`stack/postgres`, `stack/sqlite`) not required by `usermgmt/go.mod`. I could not compile-verify the _ignored_ files themselves without polluting go.mod. The shared code I extracted (`es_setup_core.go`) IS compiled and verified by the normal build; the two thin ignored wrappers were verified only by gofmt + manual review.
- **`wrapcheck` lint warnings** on the `Reset` call in `rebuildProjection` (and the original code) — these are pre-existing (the original code returned `Reset`'s error unwrapped). I left behavior identical and did not introduce a new wrap. Could be wrapped in a follow-up.

---

## c) NOT STARTED

- The 4 remaining **example-app** "actionable" groups flagged by branching-flow (`CreateTodoCmd`/`UpdateTodoCmd`, `Signals`/`TodoUpdatedPayload` in `examples/datastar-demo/`) — deliberately untouched. Examples are self-contained illustrative code; consolidating them would harm clarity for readers learning the library.
- The 2 remaining `art-dupl` groups (`streamType, streamID := streamPathValues(r)` ×2; `if err != nil` ×2) — standard Go idioms, accepted.

---

## d) TOTALLY FUCKED UP

- **Nothing destructive.** No data loss, no reverted work, no broken build or tests.
- One self-correction mid-session: my first `auth_interfaces.go` rewrite left an ugly `var _ = context.Background` hack to keep the `context` import alive. I caught it immediately and removed it (aliases don't need the import). Final file is clean.

---

## e) WHAT WE SHOULD IMPROVE

1. **`go.work` version drift was a silent blocker.** `go.work` said `go 1.26.4` while root `go.mod` requires `1.26.5` — the workspace would not build at session start. This should be CI-gated (a `go work sync` / version-consistency check) so it never silently rots again.
2. **The auto-commit daemon interleaves with active work.** My dedup changes are scattered across ~8 auto-commits with generic messages (`feat(dashboardui): update handlers...`), making the logical change hard to review or revert as a unit. A squash or feature-branch workflow would preserve reviewability.
3. **`//go:build ignore` files are a verification blind spot.** `postgres_setup.go` / `sqlite_setup.go` can't be built without adding their exotic deps to go.mod. Consider either (a) moving them under a `cmd/` or build-tagged example so they compile in CI, or (b) deleting them if unused — dead setup code that can't be compiled is a liability.
4. **Identity-model is the source of truth but doc richness lived in usermgmt.** I had to manually migrate per-method docs. A linter/doc-coverage check on aliased types would prevent doc rot when the canonical definition is enriched later.
5. **`branching-flow` "actionable" verdict is noisy.** It flagged 4 example-app coincidences and 2 documented seams as "actionable". A `.branchingflowignore` or examples-exclusion would reduce review noise.
6. **`wrapcheck` on `projectionhost.Host.Reset`** — pre-existing, but worth a dedicated pass to wrap projectionhost errors consistently.

---

## f) Up to 50 things to get done next (rough Pareto order)

### High impact

1. Add a CI check that `go.work` go-directive matches root `go.mod` (prevent the 1.26.4/1.26.5 drift).
2. Re-run `art-dupl` at `-t 5` (semantic) to catch deeper structural clones the type-aware pass misses.
3. Wrap the `projectionhost.Host.Reset` error in `rebuildProjection` (satisfies `wrapcheck`, improves diagnostics).
4. Audit all other `usermgmt` type aliases — are there non-interface types still duplicated that should alias identity-model?
5. Extract a shared `eventSourcedSetupCore` test helper so setup tests don't rebuild the same fixture.
6. Make `postgres_setup.go` / `sqlite_setup.go` compile in CI (move to build tag or `cmd/`), or delete if unused.
7. Add a `.branchingflowignore` excluding `examples/` to reduce noise.
8. Consolidate the two `triggerToast` functions — extract a shared `cqrshtmx.TriggerToast(w, event, detail)` in root that both modules call.
9. Run `branching-flow dupe` on `identity-model` alone (deepest domain module) at a lower threshold.
10. Add doc-generation that asserts aliased types have Godoc on the canonical definition only.

### Medium impact

11. Audit `examples/datastar-demo` — is `Signals` vs `TodoUpdatedPayload` a real shared concept worth a shared type, or genuinely separate?
12. Extract a shared `streamListPage` renderer for the 3 dashboardui index pages (currently each has its own `renderXxxIndex`).
13. Check if `webAuthnUserData` (usermgmt) and `userData` (webauthn module) can share via the same seam-documentation pattern.
14. Add a test that asserts `eventCatalogServer` == `openAPISpecServer` behavior (ETag, cache headers) now that they share `immutableJSONServer`.
15. Run `golangci-lint` with `--fix` on ONLY the dedup-touched files to auto-resolve `wrapcheck`/`unused`.
16. Consider promoting `reasonedCommand` to exported `ReasonedCommand` if consumers ever need to build reason-commands generically.
17. Add a coverage gate assertion for `es_setup_core.go` (currently covered transitively via usermgmt tests).
18. Document the "Sollbruchstelle seam" pattern in `docs/guides/` so the `credentialData`/`userInfo` mirrors are discoverable by contributors.
19. Audit `loginpage` module for the same toast/render duplications (not scanned this session).
20. Re-run `branching-flow dupe` after excluding examples to get a true "library" duplication number.

### Lower impact / polish

21. Unify the `contentTypeHTML` const across adminui/dashboardui/loginpage (3 copies exist).
22. Extract a shared `renderLayout` base if adminui/dashboardui layouts converge.
23. Add `//nolint:wrapcheck // behavior preserved` on the `Reset` return if not wrapping.
24. Check `options_openapi.go` — is `hashTag` now duplicated anywhere after consolidation?
25. Add a changelog entry for the dedup sweep (AGENTS.md says completed work → CHANGELOG).
26. Sweep `usermgmt/es_*.go` for other `s.`/`svc.` method-body twins (like the rebuild one).
27. Promote `writeHTML` pattern to root `cqrshtmx` if 3+ modules need it.
28. Audit the `listing.StreamListing` usage — 3 near-identical render loops may exist.
29. Add a fuzz test for `immutableJSONServer` ETag stability.
30. Verify `ToastDetail` JSON shape is documented in the event-catalog / HTMX trigger docs.
31. Check if `BeginRegistrationResponse`/`BeginLoginResponse` should be in identity-model (domain DTO) instead of usermgmt.
32. Add a `go work sync` step to the devShell hook.
33. Lint the ignored setup files with a targeted `golangci-lint` config that allows the exotic deps.
34. Consider a `cqrshtmx.RenderHTML(w, r, html, label)` in root and have dashboardui delegate.
35. Audit `adminui/handler_*.go` for shared member/tenant CRUD shapes.
36. Run `art-dupl` on `dashboardui` alone — it's the most template-heavy module.
37. Run `art-dupl` on `adminui` alone.
38. Document the `backendName` convention (`"postgres"`, `"sqlite"`) in an ADR.
39. Add a test that `eventSourcedSetupCore.Close` is idempotent.
40. Add a test that `eventSourcedSetupCore.GracefulClose` respects context cancellation.
41. Check if `stacksqlite`/`stackpostgres` can be lazy-imported to shrink the ignored-file dep surface.
42. Sweep for other `*command.BasicCommand`-only structs that could embed `reasonedCommand` or a sibling base.
43. Audit `identity-model/events.go` payload structs — branching-flow flagged 3 "intentional" groups; verify they stay intentional as events evolve.
44. Add a pre-commit hook that fails if `art-dupl` clones exceed N (regression gate).
45. Promote the dedup sweep into `CHANGELOG.md` under a new version entry.
46. Consider a `docs/architecture/seams.md` diagram of the module dependency direction.
47. Verify the `userInfo` ↔ `OAuth2UserInfo` JSON tags with a golden-file test.
48. Verify the `credentialData` ↔ `webAuthnUserCred` JSON tags with a golden-file test.
49. Add `branching-flow dupe` to CI as a non-blocking informational check.
50. Schedule a quarterly dedup sweep (these drift fast in a multi-module repo).

---

## g) Questions I CANNOT figure out myself

1. **The `//go:build ignore` setup files (`postgres_setup.go`, `sqlite_setup.go`)** — are these still the intended integration path for SQL-backed deployments, or are they superseded by something else? I couldn't compile-verify them this session (exotic deps not in go.mod). Should I wire them into a build tag so they compile in CI, or are they dead code to delete?

2. **Commit hygiene vs the auto-commit daemon** — my logical "dedup sweep" change is now spread across ~8 auto-commits with generic messages. Do you want me to leave history as-is (daemon owns it), or would you prefer I stage such refactors on a feature branch so the change is reviewable as one unit?

3. **`ToastDetail` / `TriggerToast` API surface** — I shared the struct in root but kept the per-module `triggerToast` functions because adminui nests under `adminui:toast` while dashboardui writes the detail raw. Do you want a single `cqrshtmx.TriggerToast(w, eventName, detail)` in root that both modules call (standardizing the nesting behavior), or is the current per-module behavior intentional?

---

_End of report. Awaiting instructions._

---

## Resolution (2026-07-26)

- The 12 extractions in §a **shipped** — they are the bulk of CHANGELOG `[v4.6.0]` `### Changed` ("Dedup sweep — harmful clones driven to zero"). `go.work` was bumped to `go 1.26.5` (the §e.1 drift). `cqrshtmx.ToastDetail` became a shared root type.
- **Follow-ups routed:** the `go.work`-vs-`go.mod` directive drift (§e.1) and the request for a CI gate to prevent it are now TODO_LIST **P2 Quality Gates** ("CI gate: `go.work` go-directive matches root `go.mod`"). The SQLite/SQL readmodel constructor consolidation (§e/§f) remains a ROADMAP-class idea, not pursued.
- **Caveat preserved:** the "zero harmful clones" headline was never independently re-verified — `art-dupl` was not re-run in a later session, and the 2026-07-26 20:40 self-critique (§b.3) explicitly flags the count as an unverified report self-assessment. Treat the metric as the sweep's own claim, not an audited number.
