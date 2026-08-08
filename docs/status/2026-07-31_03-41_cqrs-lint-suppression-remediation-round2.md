# Status Report — cqrs-lint Suppression Remediation (Round 2)

- **Date:** 2026-07-31 03:41 (CEST)
- **Session scope:** Resolve ALL remaining cqrs-lint `--strict --verbose` findings (stale suppressions, unsuppressed findings, load errors) from the prior session's paste_1/paste_2 terminal output
- **Branch:** `master` (auto-committed by the git daemon across multiple commits)
- **Verdict:** 56 of 56 findings suppressed, cqrs-lint exits 0, build+test+lint pass. **But 4 stale-suppression warnings remain** and several verification gaps exist.

> **Update 2026-08-01:** **Superseded.** The 4 stale-suppression warnings are caused by the installed
> cqrs-lint Nix binary being v0.2.2 (lacks comma-separated rule support). The go-cqrs-lite source
> already implements it. Tracked in TODO_LIST P2 ("Upgrade cqrs-lint from Nix v0.2.2"). AGENTS.md
> gotcha documents the workaround (two separate comment lines). All canonical nix gates green.

---

## a) FULLY DONE

| #  | Item                                                                                      | Evidence                                                                                                                                                                                                                                                                                                                                         |
| -- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1  | **4 stale E006 suppressions eliminated (lines 111/121/154/168 → inline on `event.New(`)** | Moved from standalone comment-line-above to trailing inline comment on the `event.New(` call. `cqrs-lint --only E006` reports 0 unsuppressed, 0 stale.                                                                                                                                                                                           |
| 2  | **E003 module-level finding suppressed via go.mod comment**                               | Added `//cqrs-lint:ignore(E003)` to `go.mod:1` (cqrs-lint supports go.mod comments). `cqrs-lint --only E003` reports 0 unsuppressed.                                                                                                                                                                                                             |
| 3  | **15 C009 (panic) suppressions relocated to `panic(` line**                               | Across 7 files: `app.go`, `context.go` (3), `event_store_sse.go` (2), `htmx_extensions.go` (1), `dashboardui/dashboard.go` (1), `identity-model/id.go` (2), `identity-model/upcaster.go` (1), `recovery.go` (2 already correct). All multi-line `panic(\n...\n)` had comments on the closing `)` — moved to the `panic(` opening line.           |
| 4  | **4 B005 (fold switch) suppressions relocated to `func Fold*(` line**                     | `identity-model/fold.go`: FoldUser, FoldMembership, FoldTenant, FoldBot. Comments were on the closing `) (Type, error) {` line — moved to the `func` line.                                                                                                                                                                                       |
| 5  | **3 B007 (catalog registrations) suppressions relocated to `catalog.Register(` line**     | `usermgmt/es_event_catalog.go`: registerUserEvents, registerMembershipEvents, registerTenantEvents. `replace_all` applied — the same pattern (comment on `EventMetadata{` instead of `catalog.Register(`) existed in all 3 functions.                                                                                                            |
| 6  | **4 A017 (snapshot strategy) suppressions relocated to `decider.NewRepository(` line**    | `usermgmt/stack_repositories.go`: user, membership, tenant, bot repos. Comments were on the closing `...)` — moved to the opening line.                                                                                                                                                                                                          |
| 7  | **1 A005 (manual projection) suppression relocated to `SubscribeAll(` line**              | `dashboardui/sse.go`. Comment was on the `if err != nil {` line — moved to `SubscribeAll(`.                                                                                                                                                                                                                                                      |
| 8  | **1 S003 (unsigned store) suppression relocated to `store.Save(` line**                   | `examples/dashboard-demo/main.go`. Comment was on closing `)` of multi-line call.                                                                                                                                                                                                                                                                |
| 9  | **4 E004 (catalog) + 4 E006 (projection) suppressions for dashboard-demo events**         | Applied the "standalone line above + inline" pattern: `//cqrs-lint:ignore(E004)` on a standalone comment line above `event.New(`, `//cqrs-lint:ignore(E006)` as trailing inline on `event.New(`. This is the ONLY working pattern given cqrs-lint v0.2.2's one-rule-per-comment limitation.                                                      |
| 10 | **Root cause analysis of suppression placement**                                          | Read the cqrs-lint source (`go-cqrs-lite/cmd/cqrs-lint/pkg/suppression/parser.go`). The suppression filter checks the finding's own line and the line above (`line` and `line - 1`). Comments on the closing `)` of multi-line constructs were on neither line relative to the finding (which fires at `panic(` / `event.New(` / `func Fold*(`). |
| 11 | **govalid-generate failure confirmed transient**                                          | Re-ran `buildflow -s govalid-generate` → 18/18 success, 0 failed. The failure in paste_2 was the known go/packages loader race documented in AGENTS.md (max_concurrency pinned to 2, retry_modifier: conservative).                                                                                                                              |
| 12 | **Build passes**                                                                          | `GOEXPERIMENT=jsonv2 go build ./...` = exit 0. All 15 modules.                                                                                                                                                                                                                                                                                   |
| 13 | **Tests pass**                                                                            | Root (4s), identity-model (1s), usermgmt (22s, `-race`), dashboardui (1.3s) — all exit 0.                                                                                                                                                                                                                                                        |
| 14 | **gofmt clean**                                                                           | All 11 changed files pass `gofmt -l`.                                                                                                                                                                                                                                                                                                            |
| 15 | **oauth2/totp GOWORK=off builds pass**                                                    | Both modules build with `GOWORK=off` (exit 0). The load errors in paste_1 were a cqrs-lint loader limitation, not a real build problem.                                                                                                                                                                                                          |

---

## b) PARTIALLY DONE

| # | Item                                                       | Gap                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| - | ---------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **4 stale E004 warnings remain**                           | The standalone `//cqrs-lint:ignore(E004)` comment lines above `event.New(` trigger "stale suppression" warnings because cqrs-lint's stale detector checks if the rule fires on that exact line — E004 fires on line+1 (the `event.New(` line), not the comment line. The suppression WORKS (E004 is suppressed), but the stale warning is cosmetic noise. Root cause: cqrs-lint v0.2.2 doesn't support comma-separated rules (`ignore(E004,E006)`) or multiple directives per comment line. The go-cqrs-lite SOURCE already has this fix (parser.go supports `SplitSeq(rawIDs, ",")`), but the installed Nix binary (v0.2.2) predates it. Upgrading cqrs-lint would eliminate all 4 stale warnings. |
| 2 | **golangci-lint: root has 1 pre-existing unparam finding** | `decoder.go:22:84: readBodyForDecode - result 0 (T) is always nil (unparam)`. Pre-existing — not in a file I touched. Not my responsibility but noted for completeness. dashboardui has pre-existing findings (nestif:4, nonamedreturns:1, varnamelen:7, goconst, cyclop, gocognit, contextcheck) — all in files I didn't touch. My changed files (`sse.go`) have 0 lint issues.                                                                                                                                                                                                                                                                                                                    |

---

## c) NOT STARTED

| # | Item                                                                                                                                                                                                                                                                                                                                                                                         |
| - | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **Coverage gate** (`nix run .#coverage-gate`) — not run. Changes were comment-only relocations; coverage can't have shifted, but AGENTS.md requires verification.                                                                                                                                                                                                                            |
| 2 | **`nix run .#lint`** — not run (used `golangci-lint run` directly instead).                                                                                                                                                                                                                                                                                                                  |
| 3 | **`nix flake check`** — not run.                                                                                                                                                                                                                                                                                                                                                             |
| 4 | **CHANGELOG.md entry** — no entry added for the suppression relocation work.                                                                                                                                                                                                                                                                                                                 |
| 5 | **AGENTS.md update** — cqrs-lint suppression syntax not documented in Gotchas (the prior session's status report recommended this at item e.6 and f.10). Key learnings: (a) suppression checks line + line-above only; (b) v0.2.2 doesn't support comma-separated rules despite source supporting it; (c) go.mod comments work; (d) stale detector flags standalone-line-above suppressions. |
| 6 | **Upgrade cqrs-lint** — the go-cqrs-lite source already supports comma-separated rule IDs in `ignore(A,B)`, but the installed Nix binary is v0.2.2. Upgrading would eliminate all 4 stale warnings by allowing `event.New( //cqrs-lint:ignore(E004,E006) demo data` on one line.                                                                                                             |
| 7 | **Pre-commit hook verification** — did not verify that `buildflow --build-mode pre-commit` passes with the changes. The hook runs buildflow which includes govalid-generate (known transient flake).                                                                                                                                                                                         |
| 8 | **Update prior status report** (`docs/status/2026-07-30_23-21_cqrs-lint-79-finding-remediation.md`) — its items b.1 (stale E006), b.2 (E003), b.3 (golangci-lint not run), b.4 (coverage not run) are now partially or fully resolved. The old report should be annotated.                                                                                                                   |

---

## d) TOTALLY FUCKED UP

| # | What                                                         | Impact                                                                                                                                                                                                                                                                                                                                              | Root Cause                                                                                                                                                                                                                                                                                                                                                                              |
| - | ------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1 | **dashboard-demo data loss (TWICE)**                         | Deleted `"customerId"` field from order.placed map and ALL fields from order.shipped map when removing `//cqrs-lint:ignore(E004)` comments from `map[string]any{` lines. Required fix-up edits to restore the data. Build would have produced broken demo data (empty payloads).                                                                    | **Same failure mode as the prior session (d.3).** My `old_string` in multiedit included the data line after the comment as part of the match, consuming it in the replacement. I treated multiedit as search-and-replace instead of exact-string-match. The prior session's status report EXPLICITLY warned about this at e.1: "Stop using multiedit for function-body-spanning edits." |
| 2 | **Wasted 3 round-trips testing comma-separated rule syntax** | Tested `//cqrs-lint:ignore(E004,E006)`, `//cqrs-lint:ignore(E004) //cqrs-lint:ignore(E006)`, `//cqrs-lint:ignore(E004) cqrs-lint:ignore(E006)` — all failed. Should have read the source FIRST (which I eventually did via the agent tool), then checked the installed binary version against the source to determine if the feature was available. | Didn't verify binary version vs source version before empirical testing. The source clearly showed comma-separated IS implemented, but the installed binary predates it.                                                                                                                                                                                                                |

---

## e) WHAT WE SHOULD IMPROVE

1. **Stop consuming adjacent lines in multiedit replacements** — This is the #1 recurring failure mode across TWO sessions now. When removing a trailing comment from a line like `map[string]any{ //cqrs-lint:ignore(E004) demo data`, the `old_string` must be EXACTLY `map[string]any{ //cqrs-lint:ignore(E004) demo data` and the `new_string` must be `map[string]any{`. Including the next line's data in the match is what causes data loss. **Rule: when editing a comment off a line, match ONLY that line.**
2. **Read the source before empirical testing** — The agent tool found the suppression parser source in under 10 seconds. I spent 3 round-trips guessing at syntax. The source would have told me immediately: (a) comma-separated IS supported in source, (b) checks line + line-above, (c) one directive per line is parsed.
3. **Check installed binary version vs source version** — The go-cqrs-lite source has features the installed Nix binary doesn't. `cqrs-lint --version` (v0.2.2) vs the source commit (`5ee3832e`) would have explained why comma-separated didn't work.
4. **Run the canonical verification commands** — AGENTS.md says `nix run .#test`, `nix run .#lint`, `nix run .#coverage`. I used `go test` and `golangci-lint run` directly. These bypass the flake's hermetic environment and may miss issues.
5. **Document the cqrs-lint suppression syntax in AGENTS.md** — Two sessions have now struggled with this. The syntax rules (line+above matching, one-rule-per-comment in v0.2.2, go.mod support, stale detector behavior) should be in Gotchas.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (block clean cqrs-lint output)

1. **Upgrade cqrs-lint** from Nix v0.2.2 to the latest go-cqrs-lite build (supports comma-separated rules) — eliminates all 4 stale warnings
2. **After upgrade: collapse E004+E006 suppressions** to single `//cqrs-lint:ignore(E004,E006) demo data` inline comments in dashboard-demo
3. **After upgrade: remove the 4 standalone `//cqrs-lint:ignore(E004)` lines** in dashboard-demo
4. **Verify `cqrs-lint --strict --verbose` shows 0 warnings** (no stale, no unsuppressed)

### Verification (close gaps from this session)

5. Run `nix run .#coverage-gate` — verify all coverage gates still pass
6. Run `nix run .#lint` — verify the canonical lint command passes
7. Run `nix flake check` — full flake validation
8. Run `buildflow --build-mode pre-commit --staged-only` to verify the pre-commit hook passes
9. Run `GOWORK=off go test ./...` per changed module (root, identity-model, usermgmt, dashboardui) — hermetic test verification
10. Run `GOWORK=off go build ./...` for all 15 modules — hermetic build verification

### Documentation

11. **Add cqrs-lint suppression syntax to AGENTS.md Gotchas** — line+above matching, v0.2.2 one-rule-per-comment limitation, go.mod comment support, stale detector behavior, the "standalone line above + inline" workaround for dual-rule findings
12. **Add CHANGELOG.md entry** for the suppression relocation (56 suppressions fixed across 11 files)
13. **Update prior status report** (`2026-07-30_23-21_cqrs-lint-79-finding-remediation.md`) — annotate items b.1, b.2 as resolved, b.3/b.4 as still open
14. **Document the cqrs-lint version vs source gap** — installed v0.2.2 lacks comma-separated support that source has; needs a Nix rebuild

### Architecture / Tooling

15. **Add cqrs-lint to buildflow CI** (`.buildflow.yml`) — prevent suppression drift from being merged
16. **File a bug report or PR on cqrs-lint** — the stale-suppression detector flags working standalone-line-above suppressions; it should recognize that the rule fires on line+1 relative to the comment
17. **Consider a `.cqrs-lint.json` config** — for module-level exclusions (E003) instead of go.mod comments, IF the config parser bug (exclude array vs string) is fixed in a newer version
18. **Audit all suppression comments for consistency** — currently mixed: some on struct declarations, some on function lines, some on call sites, some standalone above

### Code Quality (pre-existing, not from this session)

19. Fix `decoder.go:22` unparam finding (root module) — `readBodyForDecode` return value always nil
20. Fix dashboardui lint findings (nestif:4, nonamedreturns:1, varnamelen:7, goconst, cyclop, gocognit, contextcheck) — all pre-existing
21. Investigate the `gopls stdversion` warnings (json.Marshal/jsontext.Value "requires go1.27" in dashboard-demo) — likely false positive from GOEXPERIMENT=jsonv2 on go1.26

### Process

22. **Create a suppression-placement test** — a Go test or script that verifies every `//cqrs-lint:ignore(RULE)` in the repo actually suppresses a finding (catches stale/misplaced suppressions automatically)
23. **Consider a `make lint-cqrs` target** in flake.nix that runs `cqrs-lint --strict --strict-load` and fails on any warning (including stale)
24. **Review whether E004/E006 should be real fixes instead of suppressions in dashboard-demo** — the demo could register events in a catalog and use a dummy projection

---

## g) Questions I Cannot Answer Myself

1. **Should I upgrade cqrs-lint from the Nix-installed v0.2.2 to a fresh build from the go-cqrs-lite source?** The source (`5ee3832e`) supports comma-separated rule IDs which would eliminate all 4 stale warnings cleanly. But the Nix package is system-managed (`/run/current-system/sw/bin/cqrs-lint`) — I don't know if rebuilding from source and replacing it is the right approach, or if there's a Nix channel update pending, or if you prefer to wait for a tagged release.

2. **Should the 4 remaining stale E004 warnings block declaring this done, or are they acceptable cosmetic noise?** cqrs-lint exits 0 with `--strict`. The suppressions WORK (E004 is suppressed). The warnings are only about the comment placement being "stale" (the rule doesn't fire on the exact comment line). I left them as-is because the alternative (not suppressing E004 at all) is worse.

3. **Should I fix the pre-existing `decoder.go:22` unparam lint finding I noticed during verification?** It's in a file I didn't touch this session (`decoder.go`), and the AGENTS.md says "Don't fix unrelated bugs or test failures (not your responsibility)." But it's a 1-line fix and it breaks the "0 golangci-lint issues" claim in AGENTS.md.
