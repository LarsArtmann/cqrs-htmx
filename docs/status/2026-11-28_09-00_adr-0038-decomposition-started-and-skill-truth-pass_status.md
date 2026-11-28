# Status — the ADR-0038 decomposition started, the seam's docs landed, the skill truth-passed

**Date:** 2026-11-28 09:00 CET · **Branch:** `master` · **HEAD:** `9f83de54`
(`git rev-parse --short HEAD`, run at report time; pushed — `git rev-list
--left-right --count origin/master...HEAD` reads `0 0`)

**Scope line.** VERIFIED at report time, by command, at `944defec` (06:00–06:03;
the one commit on top of it, `9f83de54`, touches `FEATURES.md` only and was
re-checked with the method guard and the API pin at 09:02): the `*Service`
method-count guard (`bash scripts/checks/check-service-methods.sh` → 75 of 85;
`BotService` 6, `TenantService` 15, `OAuth2Service` 14, each under 30) and its
fixture self-test (10 of 10); shellcheck over the two edited scripts and over all
158 (`check-shellcheck.sh` clean); the usermgmt module hermetic
(`GOWORK=off go build ./... && go vet ./...` OK; `go test ./... -count=1` ok,
20.5 s) and its API pin (`TestService_ExportedAPISurface` PASS); every usermgmt
consumer — systemadapter, adminui, loginpage, setup, gfl-usermgmt,
integration_test, e2e/server — built, vetted and tested per module (all `ok`);
usermgmt under `golangci-lint run` hermetic (0 issues); `check-skill-claims`
(252 + 109 documented symbols resolve), `check-skill-snippets` (51 fences),
`check-readme-splitbrain`, `check-docs-links` (2,155 links OK); the docs-gates
battery as one command (8 PASS, 2 FAIL — `audit-all`, `cadence`, both
pre-existing); the two docs ratchets (`check-docs-tail-budget --ratchet` within;
`check-docs-corpus-budget --ratchet` RED at +156, see b7). Earlier in the session,
at the tree as it stood then: usermgmt under `-race` (ok, 27.5 s), the whole
hermetic lint sweep (17 of 17 modules), `check-module-isolation` (every module
builds and vets standalone), `nix run .#check-cqrs-lint` (all configured modules,
strict, zero findings). CI was read through `gh run view` for `9f83de54`.

NOT VERIFIED, and why: `nix run .#check-modules` as a composite (its new stages
were run singly; the flake was parse-checked and the parity self-test run, never
the composite itself); the chromium battery, `.#coverage-gate`,
`.#check-erraudit-strict`, `.#bench-spike` (not run); adminui under `-race`
(ran without it); `check-release-train` / `check-version-drift` (not re-run — a
sibling session bumped `httputil` v1.9.0 → v1.9.1 in every `go.mod` while this
session worked); `TestSSE_CrossModuleWireFormatContract` (red in CI, belongs to
the sibling session's dashboardui route wave — not investigated, not touched);
`references/usermgmt.md`, `references/realtime.md` and the module READMEs (not
audited for the stale-symbol class that `references/core-api.md` carried); the
rest of `FEATURES.md` beyond the eleven rows this session wrote or trued. The
skill reference was verified against the PUBLISHED `httputil` v1.9.0 by hermetic
`go doc`; v1.9.1 landed afterwards and was not re-checked.

Everything below is limited to this session's own work and what it noticed on
the way, as instructed. The session ran 02:59–09:05 CET.

---

## a) FULLY DONE

| # | Item | Receipt |
| --- | --- | --- |
| a1 | **CI's last red from the previous session is fixed at its root: the `*Service` method-count guard.** It read 96 against a ceiling of 85. The ADR forbids raising it again, so the decomposition it reserves that crossing for was started | The crossing, measured per commit: 80 on 2026-11-03, 86 at 2026-11-24 11:42, 95 by 12:06 the same day, 96 on 2026-11-27 22:58. Now 75 |
| a2 | `BotService` extracted — `RegisterBot`, `DeleteBot`, `GetBot`, `ResolveBotByToken` plus the two bot helpers that lived in other files (`deleteBotsForUserBestEffort`, `transferBotsToSurvivor`); it owns the bot read model and the token pepper | 6 methods; hermetic lint 0 |
| a3 | `TenantService` extracted — create, suspend, reactivate, delete, rename, merge, duplicate detection, roster reads, and the five unexported merge helpers; it owns the tenant read model, the tenant repository and the unique-names switch | 15 methods |
| a4 | `Service` embeds both BY VALUE; their methods are promoted, so `svc.RegisterBot(...)` is unchanged and `svc.BotService.RegisterBot(...)` names the owner. No sub-service holds a pointer back to `*Service`. The dead `botRepo` field (written at construction, never read) left with the move | `service_core.go`, `service_factory.go` |
| a5 | **The "non-breaking" claim is pinned, not asserted.** `TestService_ExportedAPISurface` renders the exported method set of `*Service` through reflection into `usermgmt/testdata/service_api_surface.golden` — captured BEFORE the move (68 signatures) and byte-identical after it | Sabotage run: with `RegisterBot` renamed on the sub-service, five test call sites fail to compile with `type *Service has no field or method RegisterBot` — proof that those call sites resolve only through promotion. File restored, sha equal before and after |
| a6 | The guard cannot be dodged by renaming the god-object: every other `*XService` type in the package gets its own ceiling (30), discovered from the receivers, never listed. Self-test 6 → 10 cases | `scripts/checks/check-service-methods.sh`, `scripts/selftests/test-check-service-methods.sh`; shellcheck clean |
| a7 | The guard and its self-test are wired where they were missing: `check-modules` gains `service-methods`, `service-methods-self-test` and `large-files`; CI gains the self-test step. The self-test had been wired into NO gate — its first case ("default threshold passes") was failing locally for four days with nobody running it | `test-check-modules-parity.sh` all cases green; `flake.nix` parses |
| a8 | ADR-0038 amended: status (EXECUTION STARTED in v4), an execution log with the three decisions made while executing, a remaining-increments table, and the ceiling-history row for the crossing | `docs/adr/0038-service-decomposition-proposed.md` |
| a9 | Receipts for the change: usermgmt `CHANGELOG.md` `[Unreleased]` (Changed), a ROADMAP paragraph (the type-level decomposition ships in v4 minors; the module split stays gated), two AGENTS.md additions (the composition rule on the usermgmt entry; the newly wired guards in the Gates row) | written |
| a10 | **AGENTS.md "Props seam" Key Patterns block** — the previous report's number-one gap: the two invariants, out-of-band on fragments only, the free refusal, response parity, loader-agnostic loaders, the two hooks, the two `App.Query` behaviours that do not apply, the tag constraint | AGENTS.md, Key Patterns |
| a11 | The skill's view-sets section now teaches what the code does: `OnLoad` in the example fence, a paragraph on the free refusal, bullets for `OnLoad`, response parity and the two non-applying `App.Query` behaviours; a Path B note on the sub-services | `.agents/skills/cqrs-htmx/SKILL.md` (section titles untouched — they are citable) |
| a12 | **The skill's core reference taught things that do not exist.** `references/core-api.md` got a truth pass against the PUBLISHED API (hermetic `go doc`, `GOWORK=off`): removed `app.DispatchWSCommand`; `/v3` → `/v4`; the error rule re-pointed from the non-existent `event.New*` constructors to `go-error-family`; the family table corrected (five families, Corruption 500, Infrastructure 503, no NotFound family — `httperr.WithHTTPStatus`); `Config.MaxBodySize` zero means NO limit (it said 10 MB); `IncludeInternalDetails` added; the CSRF / rate-limit / security-header rows moved to their `httputil` names (the root names are deprecated v5-removal aliases); template helpers take the request, not a token; extension names `cqrs-query`/`idiomorph`/`sse` (it listed `ws`); `app.QuerySet` row; the UserID sentence (one type across root, identity-model, usermgmt — it said "distinct"); the query-set section (`OnLoad`, parity, refusal, nil → zero value, cost, availability) | `check-skill-claims` 252 + 109, `check-skill-snippets` 51 |
| a13 | `references/gotchas.md`: the `/v3` import example and the error-rule fence fixed. AGENTS.md's own "No stdlib error constructors" rule fixed the same way | written |
| a14 | FEATURES.md: the two seam rows trued and marked unreleased; `Event-Sourced User`, `Tenant Lifecycle`, `Bot Management` carry current counts; `Session Rotation` marked `REMOVED` (it described the removed `UpdateRoles` command as a live feature); five new rows — email uniqueness on change, identity merge, tenant merge and duplicate detection, service composition | header updated to 2026-11-28 |
| a15 | `docs/guides/fullstack-wiring.md`: a props-seam subsection with a fence and a See Also row | 2,155 links resolve |
| a16 | Gotcha 30 amended with the example-authoring pre-flight (check the root CHANGELOG section and the example's `go.mod` BEFORE writing example code); the AGENTS.md index cell follows | `docs/gotchas.md` |
| a17 | The plan's §12 execution log carries the follow-through, the docs batch and the CI read (three rows) | `docs/planning/2026-11-27_15-24_…pareto-plan.md` |
| a18 | The seam guide's reading list no longer presents `e2e/tests/root-htmx.spec.ts` as the seam's browser proof — that file is the 2026-11-07 gate of the older response helpers; the seam's three specs do not exist yet | `docs/guides/query-set-to-templ.md` |
| a19 | Checked on the way, found healthy: no stray binary in the tree (a 29 MB `examples/system-demo/system-demo` match is a local, gitignored build output; the largest tracked file is 0.6 MB) | `git ls-tree -r -l HEAD` |
