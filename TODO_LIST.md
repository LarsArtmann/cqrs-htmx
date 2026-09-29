# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision, v5 plans, and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-09-26 (round-12 full-list execution) | **Round 12 (2026-09-26):** the whole open list was executed in one pass — bench-spike closed on an idle window (P1 done; the deferred post-bump verification pass is COMPLETE), the buildflow CSS-bundle producer killed at source and the exact-class-set drift gate shipped atomically (first catch: the Grid adoption's one-utility delta), dashboardui adopted PolledRegion/Grid/RelativeTime with page-level Ginkgo goldens and dark-mode axe sweeps (which found and fixed three real WCAG failures: dark links 2.70:1, sidebar text 4.08:1, and the gray-200/700 token remaps breaking library dark: variants), httputil v1.4.0 released plus a go.work stub-replace to end a 3-day workspace-wide ambiguous-import outage, upstream asks recorded in templ-components (4 asks) and BuildFlow (3 asks), and every outstanding verification sweep executed — verdicts in CHANGELOG | **Version:** v4.12.0 family train (next train carries httputil v1.4.0 + the templ-components v1.19.4 alignment, M18) | **Modules:** 28 in `go.work` | **Coverage:** 15/15 gates green (re-run 2026-09-29 post PapDashboard-feedback work: setup 87.9%, systemadapter 91.9%, health 100%, auditlog 100%, dashboardui/core 88.7%) | **Lint:** 0 issues / 15 modules (re-run 2026-09-29 — includes the two pre-existing findings cleared in `1eb7be83`)

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High impact (next train follow-through)

- [ ] **templ-components v1.19.4 bump train (scheduled 2026-09-25, M18 sweep).** All 12 module go.mods sit on v1.19.2 (direct: adminui, dashboardui; indirect: e2e/server, examples/*, health, integration_test, setup). v1.19.4 content that matters: nix `result*` symlink hygiene (unblocks prepared-source consumers), `htmx.PolledRegion` hour-interval normalization (1h → 3600s — audit for h-suffix `Every:` usages first; NOTE 2026-09-26: dashboardui's adopted region uses an explicit `Trigger`, unaffected), and the v1.18.1→v1.19.x eager-contract rework consumers must adapt to (no `load` in `hx-trigger` for eager PolledRegions — the dnsblockd migration is the worked example; dashboardui's region is non-eager, unaffected). Bump order: direct modules first, then `go work sync` + indirect ride-along; re-golden after. RIDES WITH: httputil v1.4.0 (already required in all 22 module go.mods since 2026-09-26).

---

## P2 — Medium impact (tooling & quality)

- [ ] **Rebuild the system cqrs-lint binary, then verify C040 stays silent.** The 2026-09-29 C040 phantom class (21 "dead fold case" warnings on identity-model folds) was FIXED AT SOURCE on 2026-09-30 in `~/projects/go-cqrs-lite` `cmd/cqrs-lint` (committed by its daemon): the collector now resolves event-type arguments through const AND var-alias chains (`var eventUserRegistered = identitymodel.EventUserRegistered` — the gotcha-15 pattern) and recognizes `EventCatalog.Register(EventMetadata{Type: ...})` declarations; regression tests added, full module suite green, verified 0 findings with C040 ENABLED on the full root walk and all 13 per-module runs (fixed binary at `/tmp/cqrs-lint-fixed`). NO config exemption exists or is needed — `.cqrs-lint.json` carries the story. Remaining: run the system rebuild so `cqrs-lint` on PATH picks up the fix, then confirm `cqrs-lint --strict --verbose .` shows zero C040 on a root walk. Pre-existing, non-gating warnings under the newer linter source (identity-model 16× E005 + usermgmt 41× V007 + 1× A016 + 4× V006 — verified identical on the pre-fix source) are separate triage, not this item.
- [ ] **SidebarNav revisit criteria (dashboardui).** Custom dark sidebar stays while ALL of: (1) the library ships a dark-token shell matching `--sidebar-bg`/`--tc-sidebar-*` variables, (2) mobile drawer keeps zero-JS, (3) adminui's SidebarNav adoption has proven the theming tokens in production. RE-VERIFIED 2026-09-26 against v1.19.2: criterion (1) is still unmet — the library's AppShell uses only `--tc-sidebar-w` (width); there is no dark sidebar-surface token family. Stays open until the library ships one.
- [~] **Wire remaining `check-*` apps into CI.** All gates except `check-cqrs-lint` run in CI. Round-12 (2026-09-26) closed two parity gaps the sweep found: dashboardui joined the codegen drift lane (was adminui+loginpage only) and `check-docs-freshness.sh` itself now runs in CI (was self-test-only). Remaining: `check-cqrs-lint` (blocked: Nix-only binary; needs a Go-installable distribution). `bench-spike` stays LOCAL-only by decision (machine-pinned baseline).
- [ ] **Re-enable BuildFlow `go-version-auto-configure` only after upstream fixes.** ASKS RECORDED 2026-09-26 in the BuildFlow repo's TODO_LIST (BF1–BF3, D1 precedent): (BF1) prune `testdata`/fixture trees from `surface.Discover`'s walk (the 27057e5c tree corruption — tidy deleted every fixture require line and the guard tests vacuously passed); (BF2) a policy knob for the go-directive rule (major.minor vs this repo's deliberate 1.27.1 patch floor); (BF3) `exclude:` should MERGE with `DefaultExcludePatterns` instead of replacing them (verified 2026-09-20; the repo mirrors the defaults manually today). Re-enable condition: BF1+BF2 shipped AND `buildflow --build-mode pre-commit --staged-only --dry-run` shows no fixture/directive churn.
- [ ] **DataStar Tier 4 — demand-gated panel variants (ADR-0050; Tiers 1-3 SHIPPED).** Each milestone (M11–M16) is gated on consumer demand evidence per ADR-0050 §Decision 1 — plan: `docs/planning/2026-09-07_17-18_datastar-dual-frontend-rollout.html`. Nothing actionable without the demand signal.

---

## P3 — Technical debt & future

- [~] **Migrate usermgmt off go-cqrs-lite v5-removed APIs (V007, cluster 1 of 3).** Remaining: the 68 `storage.SQLViewStore`-family findings, gated on metaengine layout-planning covering secondary-index semantics + declarative hydration (ADR-0051 go-criterion). Clusters 2+3 were deprecation-marking only (done). Companion: go-cqrs-lite removes `stack/v4` entirely in v5 (owner decision, recorded in that repo's ROADMAP).
- [~] **Adopt appkit as the `setup` server layer (ADR-0052).** Nothing actionable before the v5-window revisit. NEW REMOVAL CONDITION (2026-09-26): the `go.work` go-etag stub-replace (`go-etag => v0.6.0`) drops when go-appkit > v0.5.1 publishes its split-module go.mod (the fix already sits on appkit master — 82 unpushed-tag commits and red master CI on unrelated own gates, so round-12 did NOT release it).
- [ ] **Add cqrs-lint strict CI gate.** Blocked: cqrs-lint is a Nix-only binary; needs a Go-installable distribution or a Nix CI runner (same block as the P2 CI item). Run ONLY via the flake app.
- [ ] **Hardware watch: `/mnt/buildcache`.** Healthy as of 2026-09-26 (158G free of 220G). The fill-drain cycle recurs — `df -h` first when gates fail en masse. Needs a human decision on the reclaim candidates (rust/ 155G + sccache/ 20G — not this repo's to delete). VCS-cache origin-loss class stays gated (`check-vcs-cache`, 33 entries healthy).
- [ ] **Remove `ProjectionLayer` in the v5 removal bundle (all prep DONE).** Accepted limitation recorded in `docs/guides/v5-removal-inventory.md`; removal proceeds with the limitation documented unless an upstream `system.New` checkpoint/DLQ option lands first.
- [ ] **Decide `examples/datastar-demo` rebrand-or-remove — OWNER CALL.** Evidence recorded 2026-09-22: healthy, accurately named, the only Datastar-native example, ADR-0050's route-split story points at it. Recommendation: KEEP AS-IS. Decision remains the owner's.
- [ ] **loginpage templ-components adoption — OWNER CALL (routed to ROADMAP OQ21).** Survey complete 2026-09-23; full adoption trades the module's zero-heavy-dep differentiator for design-system consistency. Criteria in OQ21.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision, v5 plans, and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
