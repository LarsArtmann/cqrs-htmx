# Post-Release Documentation Cleanup — Comprehensive Status Report

**Date:** 2026-07-08 12:20
**Session goal:** Execute all P0/P1 items from the release session status report, then deep-audit for missed issues
**Git:** master @ `45cc8ed` — pushed to origin
**BuildFlow:** 38/38 green

---

## a) FULLY DONE ✅

### Round 1 — P0 Fixes (commit `202edd1`)

1. **eventtest go.mod** — `go-error-family` moved from `// indirect` to direct `require` block. Verified with `go mod tidy`.
2. **README.md line 31** — `go-cqrs-lite v3.1.0` → `v3.7.4`
3. **usermgmt CHANGELOG migration guide** — Fixed "unchanged" methods list. Initially I made an error claiming `Authenticate` was removed (it survives — validates session tokens, not passwords). Caught by source code grep before commit.

### Round 1 — P1 Fixes (commit `202edd1`)

4. **4 CHANGELOGs created** from git history analysis:
   - `adminui/CHANGELOG.md` — v3.0.0 → v4.2.0 (6 versions)
   - `usermgmt/totp/CHANGELOG.md` — v4.0.0 → v4.0.2 (3 versions)
   - `usermgmt/webauthn/CHANGELOG.md` — v4.0.0 → v4.0.2 (3 versions)
   - `usermgmt/oauth2/CHANGELOG.md` — v4.0.0 → v4.0.2 (3 versions)
5. **README full audit** — Also fixed line 1177 (`go-cqrs-lite v3.5.0` → `v3.7.4`) and line 1179 (`go-error-family v0.5.1` → `v0.6.1`)
6. **CONTRIBUTING.md** — Audited, confirmed current (11 modules correct)
7. **DOMAIN_LANGUAGE.md** — Fixed MySQL ref on SQLEventStore (only Postgres+SQLite), UserID "branded ULID" → "branded string"
8. **AGENTS.md gotcha #14** — Was "Max password length 128 / bcrypt CPU abuse" but passwords were removed in v4.0.0. Rewritten to reflect email-only registration.
9. **v3-to-v4 migration guide** — Audited, verified accurate
10. **Full docs scan for removed APIs** — Searched all `.md` files for ChangePassword, bcrypt, LoginRequest, etc. Historical references in CHANGELOGs and status archives are correct as-is.
11. **flake.lock** — Checked; already current (nixpkgs up to date)

### Round 2 — Deep Audit (commit `45cc8ed`)

After self-reflection, a deeper grep found **8 more issues** I had missed:

12. **usermgmt CHANGELOG v4.0.0 section** — 4 factual errors:
    - Line 45: `ServiceConfig.WebAuthnConfig` → `ServiceConfig.WebAuthn` (WebAuthnConfig was removed in auth strategy extraction)
    - Line 46: `golang.org/x/crypto` → note about WebAuthn deps moving to sub-module
    - Line 54: `WebAuthnConfig` → `WebAuthnProvider` interface
    - Line 66: Deleted entirely ("Password hashing happens in Service layer" — contradicts "ALL password code removed")
    - Line 67: `MemoryBus` → `watermill.EventBus` with `BlockPublishUntilSubscriberAck`
13. **DOMAIN_LANGUAGE.md** — `MemoryBus` entry → `EventBus` (the actual type is `watermill.EventBus`)
14. **README catalog version** — `go-cqrs-lite/catalog v3.2.0` → `v3.7.1` (5 minor versions behind!)
15. **Migration directory consolidation** — `docs/migration/` (singular) and `docs/migrations/` (plural) both existed. Moved v2-to-v3.md into `docs/migrations/`. Updated README link.
16. **TODO_LIST.md** — Version header updated `v4.0.0 (go-cqrs-lite v3.5.0)` → `v4.2.1 (go-cqrs-lite v3.7.4)`. Stale `UserStore` reference annotated with removal note.

### Verification

- All 7 modules: build ✅ + test with `-race` ✅
- Root lint: 0 issues ✅
- Errorfamily: 0 violations (root + usermgmt) ✅
- BuildFlow: 38/38 ✅

---

## b) PARTIALLY DONE ⚠️

### 1. Docs are clean but not bulletproof

- All **active** documentation files have been audited and corrected.
- **Historical** docs (status reports, planning docs, ADRs) intentionally reference old APIs — they're point-in-time snapshots.
- No automated check exists to prevent stale version refs from creeping back in.

### 2. usermgmt CHANGELOG v4.0.0 section still has rough edges

- The section was written incrementally as features shipped (June-July). Some entries reflect intermediate states that were later changed by the auth strategy extraction (v4.0.0 → v4.2.0 evolution).
- A full rewrite of the v4.0.0 section would make it cleaner, but it's historically accurate enough now.

### 3. TODO_LIST.md "Open Items" section is mislabeled

- Section header says "Open Items" but ~90% of items are marked `[x]` DONE
- Only genuinely open items: Phase 2b persistent offline queue (line 174) and the deferred god-package split (line 191)
- Needs reorganization into "Completed" vs "Open" sections

---

## c) NOT STARTED ❌

1. **GitHub Releases for 6 tags** — Tags pushed but no `gh release create` run. Consumers browsing GitHub see no release notes.
2. **Go module proxy verification** — Did not verify `go get github.com/larsartmann/cqrs-htmx/v4@v4.2.1` works from a clean consumer perspective.
3. **Pre-release verification script** — No `nix run .#release-checklist` or equivalent to prevent stale docs in future releases.
4. **Release process documentation** — CONTRIBUTING.md has no section on how to cut a release (tag naming, CHANGELOG update order, GitHub release creation).
5. **TODO_LIST.md reorganization** — "Open Items" section needs splitting into completed vs open.

---

## d) TOTALLY FUCKED UP 💥

### 1. I marked `Authenticate` as removed when it wasn't

**What happened:** In my first fix of the usermgmt CHANGELOG migration guide, I wrote:

> `Login`, `ChangePassword`, `Authenticate` — replaced by WebAuthn ceremonies

**The problem:** `Authenticate` **still exists** (`service_login.go:18`). It validates session tokens, not passwords. I assumed it was removed because it was in the same list as `Login` and `ChangePassword`.

**How I caught it:** Before committing, I grepped the source code: `grep 'func (s \*Service) Authenticate'`. Found it immediately.

**Lesson:** Always verify against source code before making claims about API removal. The CHANGELOG said "unchanged" and I "corrected" it to say "removed" — the opposite of truth for one of the three methods.

### 2. Round 1 missed 8 issues that Round 2 caught

**What happened:** I declared the docs audit "complete" after Round 1, but a deeper grep (prompted by the user asking "what did you forget?") found 8 more stale references.

**What I missed specifically:**

- `MemoryBus` (removed type) still defined in DOMAIN_LANGUAGE and CHANGELOG
- `WebAuthnConfig` (removed type) referenced as current API in CHANGELOG
- Password hashing line that directly contradicts "ALL password code removed"
- Catalog version 5 minor versions behind
- Split migration directories

**Lesson:** My Round 1 search patterns were too narrow. I searched for `ChangePassword|bcrypt|LoginRequest` but didn't search for `MemoryBus|WebAuthnConfig` — types that were renamed/removed in later v4 releases. I should have searched for ALL type names that appear in the v4 breaking changes section.

### 3. I didn't verify against the actual Go source code early enough

The pattern in both mistakes (`Authenticate` and the missed `MemoryBus`/`WebAuthnConfig`) is the same: I trusted documentation and changelog entries to tell me what the current API looks like, instead of grepping the actual Go source. The source code is the source of truth.

---

## e) WHAT WE SHOULD IMPROVE 🔧

### Process Improvements

1. **Pre-release doc scanner script** — A script that extracts all type names from Go source (`grep -rn 'func\|type\|var' *.go`) and checks that every type referenced in `.md` files still exists. Would catch `MemoryBus`, `WebAuthnConfig`, etc. automatically.

2. **CHANGELOG-first, always** — Write CHANGELOG entries BEFORE the commit that ships the feature, not weeks later from git log archaeology. Retroactive CHANGELOGs are error-prone.

3. **Search the breaking changes section** — When documenting breaking changes, extract every type name mentioned and grep for stale references. The v4 breaking changes removed `WebAuthnConfig`, `MemoryBus`, `UserStore`, etc. — all of these should have been search targets from the start.

4. **Source code is truth** — Before claiming an API was removed, grep the source. Before claiming an API exists, grep the source. This would have prevented the `Authenticate` error.

5. **TODO_LIST hygiene** — Move completed items out of "Open Items" into a "Completed" archive section. The current 200-line wall of `[x]` items is noise that hides the 2 genuinely open items.

### Technical Improvements

6. **`nix run .#check-docs-freshness`** — Scan all `.md` files for version strings that don't match `go.mod`. Would prevent `v3.1.0` → `v3.7.4` class of bugs.

7. **Type registry from Go source** — Extract all exported type/function names from `.go` files into a list, then flag any `.md` reference to a name NOT in the list. Catches removed types (`MemoryBus`, `WebAuthnConfig`, `InMemoryUserStore`).

8. **Single migration directory** — Now fixed, but should be enforced. Only `docs/migrations/` should exist.

---

## f) Up to 50 Things to Get Done Next

### Immediate (P0 — this week)

1. Create GitHub Releases for `v4.2.1`, `usermgmt/v4.2.0`, `adminui/v4.2.0`, `totp/v4.0.2`, `webauthn/v4.0.2`, `oauth2/v4.0.2`
2. Verify `go get github.com/larsartmann/cqrs-htmx/v4@v4.2.1` resolves from Go proxy
3. Check if pkg.go.dev picked up v4.2.1

### Short-term (P1 — next 2 weeks)

4. Reorganize TODO_LIST.md — split "Open Items" into "Completed" (archive) vs "Open" (actionable)
5. Write `nix run .#check-docs-freshness` app — scan `.md` files for stale version refs
6. Write type-existence checker — scan `.md` for Go type names not in source
7. Add release process section to CONTRIBUTING.md (tag naming, CHANGELOG order, gh release)
8. Audit docs/adr/\*.md for accuracy against current codebase (ADRs 0006, 0016, 0022-0035)
9. Check if ADR-0006 references `MemoryBus` or `WebAuthnConfig`
10. Full rewrite of usermgmt CHANGELOG v4.0.0 section (currently incremental/historical)
11. Add `nix run .#release-checklist` app (validates CHANGELOG + version refs + migration guide before tagging)
12. Verify all docs/migrations/\*.md internal links resolve
13. Check docs/observability-wiring.md and docs/MIGRATION-v3-incremental.md for stale refs
14. Add CI step: `go work sync` produces no changes

### Medium-term (P2 — next month)

15. Automate GitHub Release creation via CI on tag push (`.github/workflows/release.yml`)
16. Add version banner to adminui (show version in dashboard footer)
17. Consider changelog generator tool (`git-cliff` or `chronicle`)
18. Add `nix run .#check-docs-types` — type existence checker across all `.md` files
19. Audit all ADRs for removed API references
20. Consider consumer-facing v3→v4 codemod
21. Evaluate admin-demo example needs updating for v4.2.x
22. Add integration test that imports the published version (not local replace)
23. Add contract tests between root module and usermgmt (RateLimiter boundary)

### Technical Debt (P3)

24. Fix 2 pre-existing golangci-lint usermgmt findings (exhaustive switch, maintidx)
25. Consider splitting usermgmt god-package (84 files) per Sollbruchstellen analysis
26. Consider extracting root module's 7 identified sub-package candidates
27. Evaluate projectionhost adoption for crash-restart semantics
28. Consider TypedRepository adoption to eliminate command type assertions
29. Add OPFS persistence for offline command queue (Phase 2b per ADR-0029)
30. Consider Redis adapters for SessionStore/OAuth2StateStore/IdempotencyStore
31. Evaluate MySQL support for event store (currently Postgres + SQLite only)
32. Add health check endpoints documentation
33. Consider OpenAPI spec generation for HTTP endpoints

### Quality & Testing (P3-P4)

34. Add property-based tests for event fold functions
35. Add chaos testing for projection replay (random event ordering)
36. Add load testing benchmarks for SSE broadcaster under high fan-out
37. Add fuzz tests for OAuth2 state token generation/parsing
38. Add integration test for full offline→online sync cycle
39. Add test coverage gate for adminui (currently no threshold)
40. Consider mutation testing (go-mutesting)
41. Add test that verifies all module go.mod replace directives are consistent
42. Add CI step that verifies `go work sync` produces no changes
43. Benchmark WSBroadcaster under high fan-out
44. Add fuzz tests for WS message parsing edge cases
45. Add stress test for rate limiter under sustained load
46. Add test for CSRF middleware under concurrent requests
47. Add test for Server-Timing header injection prevention
48. Add test for idempotency store under concurrent duplicate commands
49. Add test for event store replay correctness after schema migration
50. Add test for OAuth2 state store TTL eviction correctness

---

## g) Top 2 Questions I Cannot Answer Myself

### Q1: Should I create GitHub Releases for the 6 tags now, or do you want to review the CHANGELOGs first?

Tags are pushed. CHANGELOGs exist for all modules. I can run `gh release create` with CHANGELOG-derived notes in ~5 minutes. But you may want to review the CHANGELOG content first, especially the newly created adminui/totp/webauthn/oauth2 ones (written from git log archaeology, not from memory of the work).

### Q2: Should the TODO_LIST.md "Open Items" section be rewritten now, or is that a separate session?

The section has ~200 lines, 90% marked `[x]` DONE. Only 2 items are genuinely open. Rewriting it properly means reorganizing into "Completed" (archive by session) vs "Open" (2 items only). This is a 30-minute task but changes the file significantly — should I do it now or wait for explicit instruction?

---

## Session Metrics

| Metric                | Round 1                         | Round 2                         | Total      |
| --------------------- | ------------------------------- | ------------------------------- | ---------- |
| Commits               | `202edd1`                       | `45cc8ed`                       | 2          |
| Files modified        | 5 (existing)                    | 4 (existing) + 1 rename         | 10         |
| Files created         | 4 (CHANGELOGs)                  | 0                               | 4          |
| Lines changed         | +226, -44                       | +56, -57                        | +282, -101 |
| P0 items resolved     | 5 of 6                          | 0 (all P0 done in R1)           | 5 of 6     |
| P1 items resolved     | 12 of 14                        | 0 (all P1 done in R1)           | 12 of 14   |
| Issues found in audit | 0 (declared done)               | 8 (deep grep)                   | 8          |
| Tests                 | All 7 modules pass with -race   | All 7 modules pass with -race   | ✅         |
| Lint                  | Root 0, usermgmt 2 pre-existing | Root 0, usermgmt 2 pre-existing | ✅         |
| Errorfamily           | 0 violations                    | 0 violations                    | ✅         |
| BuildFlow             | —                               | 38/38                           | ✅         |
| Pushed                | ✅                              | ✅                              | ✅         |
| Time elapsed          | ~40 min                         | ~25 min                         | ~65 min    |
