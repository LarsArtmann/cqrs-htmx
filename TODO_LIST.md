# TODO List — cqrs-htmx

> Short-term, actionable, bounded work. Open items only.
> Completed work lives in [CHANGELOG.md](CHANGELOG.md). Long-term vision, v5 plans, and rejected ideas live in [ROADMAP.md](ROADMAP.md).

**Updated:** 2026-08-15 | **Version:** v4.7.0 (released 2026-08-07) + `[Unreleased]` (async projection startup + ADR-0048, ActorID consolidation/ADR-0111, ADR-0114 tombstone migration, MySQL event-store completion, security middleware consolidation, httputil v0.11.0, Broadcaster Raw() accessor, fullstack UI integration tests, systemadapter module + system/metaengine integration, hermetic-build repair, new health/v4 + auditlog/v4 bridge modules, coordinated v4.8.0 family release train (PUSHED 2026-08-14, replaces stripped, drift gate green)) | **Modules:** 26 in `go.work` (15 production + 10 examples + e2e/server) | **`*Service` methods:** 73 (leading v5 indicator; see ROADMAP) | **Coverage (verified 2026-08-14, all gates green):** Root 93.4% (gate 90%), openapi 99.0%, usermgmt 80.8% (gate 74%), identity-model 75.5% (gate 70%), dashboardui 83.4% + core 86.1% (gates 60/80), datastar 97.5% (gate 90%), setup 89.5% (gate 80%), systemadapter 89.6% (gate 70%), health 100% (gate 90), auditlog 100% (gate 90), adminui 68.5% (gate 66), loginpage 79.9% (gate 79) | **Lint:** 0 golangci issues across 15 modules incl. systemadapter, health, auditlog (2026-08-14); cqrs-lint strict passes all module configs. | **✅ v4.8.0 family PUSHED (2026-08-14):** root, identity-model, usermgmt, webauthn, loginpage, dashboardui, adminui, setup (first), health (first), auditlog (first) — master + all 10 tags on origin; family dev replaces stripped, `check-modules` (isolation + relative-paths + strict version-drift) fully green. Remaining upstream asks: go-cqrs-lite projectionadapter/sqliteengine + go-datastar static/v0.2.0 (see `docs/runbooks/release-v4.8.0-push-plan.md` §3).

## Status Legend

- [ ] **OPEN** — actionable, not yet started.
- [~] **PARTIALLY DONE** — started but incomplete.

> No `[x]` items here. When a task finishes, it moves to [CHANGELOG.md](CHANGELOG.md) and is removed from this list. Deferred/rejected ideas move to [ROADMAP.md](ROADMAP.md) → "Not Planned".

---

## P1 — High impact (release follow-through & doc health)

- [ ] **Cut the upstream tags that unblock the remaining trains (runbook §3, other repos, user's call).** REMAINING: go-cqrs-lite `metaengine/projectionadapter/v4 v4.5.0` + `metaengine/sqliteengine/v4 v4.0.2` (strip their workspace replaces in the tag commits; the repo's 2026-08-15 ADR-0128 extraction commit `5127039da` or later is a valid base — re-validate the runbook §3 mapping against that tree first, it now requires external `go-codec`/`go-retry`) → then `systemadapter/v4.8.0` + strip its metaengine replaces (and the matching one in `examples/system-demo`). DONE 2026-08-15: go-datastar `static/v0.2.0` tagged; `datastar/v4.8.0` cut, pushed, and proxy-resolvable; the 2 datastar-local family replaces stripped (`b5663b2b`); check-templates wired into CI (`e7aeee97`).

---

## P2 — Medium impact (tooling & quality)

- [~] **Wire remaining `check-*` apps into CI.** `check-docs-links`, `check-service-methods`, `check-domain-counts`, `check-large-files`, `check-phantom-version`, `check-codegen` (templ pinned @v0.3.1020), and `check-templates` (hermetic GOWORK=off rework + CI step with clean-restore assertion, 2026-08-15) now run in CI. Remaining: `check-cqrs-lint` (blocked: Nix-only binary; needs Go-installable distribution — see P3.1). New modules health + auditlog are wired into CI build/test/coverage/lint/mod-tidy (2026-08-14).

- [ ] **Bump Go toolchain to 1.26.6 for two stdlib security fixes (GO-2026-6090, GO-2026-5972).** govulncheck on the root module (2026-08-15, symbol-level): crypto/tls post-handshake message flood (GO-2026-6090) and encoding/asn1 unbounded recursion (GO-2026-5972), both `Found in go1.26.5`, both `Fixed in go1.26.6`; plus 4+2 informational findings in imported packages / required modules our code never calls. Triage: patch-class DoS, no RCE/data exposure; consumer binaries compiled with their own ≥1.26.6 toolchain are unaffected — exposure is limited to our dev/e2e/example binaries. BLOCKED on nixpkgs: both the locked flake input and current nixos-unstable ship `go_1_26 = 1.26.5`, and builds run `GOTOOLCHAIN=local`, so raising the `go` directive floor alone breaks every local build. Unblock: update the nixpkgs lock once it carries 1.26.6, then bump the `go` directives in every go.mod + go.work (CI follows via `go-version-file: go.mod`).

---

## P3 — Technical debt & future

- [ ] **Add cqrs-lint strict CI gate to GitHub Actions.** Run cqrs-lint `--strict` in CI to catch catalog/validation findings early. Blocked: cqrs-lint is a Nix-only binary; needs a Go-installable distribution or a Nix CI runner. The flake.nix `check-cqrs-lint` app exists for local use.
- [ ] **Cross-module dep version drift after v4.7.0 tagging.** Published submodule tags reference stale sibling versions (e.g., `adminui/v4.7.0` still depends on `usermgmt/v4.6.1`). Next release should bump all cross-module refs before tagging. Source: `docs/status/archived/2026-08-07_22-23_release-v4.7.0-and-retry-investigation.md`.
- [ ] **Re-investigate datastar/go-sse architecture decision.** The prior analysis (`docs/status/archived/2026-08-07_06-25_datastar-go-sse-analysis-self-review.md`) claimed go-sse cannot produce Datastar wire format — but go-sse has `KeyedLines`/`SendKeyed`/`SendLines` designed for Datastar. The exclusion is a design choice (Patch coupling to SDK), not a technical incompatibility. Needs either an ADR documenting the decision or a migration to go-sse.
- [ ] **Add golines alignment to `nix fmt` pipeline.** `golines` is available but not integrated into treefmt. Would catch alignment drift automatically. May need `pkgs.golines` from nixpkgs or a wrapper.
- [ ] **Consider a Go-based markdown link checker.** The current `check-docs-links.sh` uses awk regex which handles common cases but may miss edge cases. A Go checker using goldmark would be more robust. The awk checker + test suite is sufficient for now.
- [ ] **Rewrite `origin/v4` branch history to strip 3 remaining binary blobs (~27.7 MB).** The master branch was cleaned via `git filter-repo` (731.8 MB → 0 blobs), but the `v4` branch has independent binary contamination (`examples/basic/basic` 9.8MB, `examples/datastar-demo/datastar-demo` 8.9MB ×2) that does not share ancestry with master. Requires `git filter-repo` on the v4 branch + force-push. Source: `docs/status/2026-08-09_06-15_git-binary-cleanup-history-rewrite.md`.
- [ ] **Purge `examples/setup-demo/setup-demo` (27 MB) from pushed master history.** Removed from HEAD and from all unpushed commits 2026-08-14 (filter-branch; family tags re-pointed), but the blob still exists in already-pushed history (introduced `5604e810`, last carried by `73ff1556`). Requires filter-repo + force-push on master — same class as the v4 branch item above. Low priority: the repo already survived a 731.8 MB cleanup.

---

_For completed work, see [CHANGELOG.md](CHANGELOG.md) and [git log](https://github.com/larsartmann/cqrs-htmx/commits/master). For long-term vision, v5 plans, and rejected ideas, see [ROADMAP.md](ROADMAP.md)._
