# Status Report — 2026-08-05 02:26

## Deep Architecture Review + Skills Sweep + Code Changes

---

## Session Summary

What started as a single "DEEP architecture-review!" grew into a five-skill audit sweep followed by three code-change directives. This report covers everything done, partially done, missed, and what remains.

**Skills executed:** architecture-review, architecture-visualization, go-modularize, data-model-review, naming-review

**Code changes shipped:** 160 deprecation markers, 656 cfg→config renames, OAuth2Service extraction prototype, 3 shadowing bug fixes, 1 pre-existing bug fix

**Verdict:** 4.0/5 Good architecture. No P0. No novel debt. Two ADR'd deferrals validated. One extraction prototype shipped.

---

## A) FULLY DONE

### 1. Architecture Review (architecture-review skill)

- **Output:** `docs/architecture-understanding/2026-08-05_01-40_deep-architecture-review.html`
- **Score:** 4.0/5 Good (Coupling 4, Cohesion 3, Modularity 4, Composability 5, Scalability 4, Service Orientation 4, Dependency Direction 4)
- **Key finding:** SDP-perfect dependency profile (stable core I=0, volatile leaves I=1, zero cycles). The two largest debts (god-package + god-object) are already ADR'd and deferred.
- **Corrected:** Initially flagged Casbin as an "open item" — user clarified Casbin is the foundation. ADR-0044 already records this as ACCEPTED. Both reports corrected to reference ADR-0044.

### 2. Architecture Visualization (architecture-visualization skill)

- **Output:** `docs/architecture-understanding/2026-08-05_01-50-current.d2` + `.svg`, `2026-08-05_01-50-improved.d2` + `.svg`
- Current state diagram shows the SDP-perfect profile with usermgmt flagged as god-module (red).
- Target (v5) diagram shows decomposed usermgmt with 4 aggregate sub-modules + 4 focused services.
- **Bug fixed:** d2 v0.7.1 doesn't support `shape: component` — changed to `shape: rectangle`.

### 3. Go Modularize Analysis (go-modularize skill)

- **Output:** `docs/modularization/2026-08-05_PROPOSAL.html`
- **Key result:** 0 of 3 decomposition triggers met. Independently confirmed ROADMAP's deferral decision via co-change analysis (zero cross-aggregate coupling) and dep-tree analysis (0% divergence).
- **Value-add:** Quantitative validation of ADR-0019/0038, plus identification of clean file-prefix seams (6 clusters) that make v5 mechanical.

### 4. Data Model Review (data-model-review skill)

- **Output:** `docs/reviews/2026-08-05_data-model-review.html`
- **Score:** 4.5/5 Exemplary. 0 critical findings.
- identity-model uses branded phantom-type IDs, typed enums with Valid(), constructor-enforced validation, closed unions via interfaces. Nine distinct Go type-system features used deliberately.
- 4 low-severity findings all on boundary types (Provider, Session.Token, AttestationType, Transports) that mirror external protocol data.

### 5. Naming Review (naming-review skill)

- **Output:** `docs/reviews/2026-08-05_naming-review.html`
- **Score:** 4.5/5 Clean. Zero Impl suffixes, zero I-prefix interfaces, zero Manager/Processor trash-can classes.
- Error naming exemplary: 248 errors all follow ErrXxx + errorfamily classification.
- One systemic nit: `cfg` abbreviation (656 occurrences — fixed this session).

### 6. Deprecation Headers on Re-export Layer

- **What:** 160 `// Deprecated: Import github.com/larsartmann/cqrs-htmx/identity-model/v4 directly.` markers added across all 26 usermgmt re-export files.
- **Coverage:** Every exported type alias, var alias, const alias, and constructor wrapper. Single-line and multi-line funcs handled. Existing deprecations (NewUserID) merged, not duplicated.
- **Tests:** All pass. Build clean.

### 7. cfg → config Rename

- **What:** 656 occurrences of `cfg` renamed to `config` across the entire workspace.
- **Collisions found and fixed:** 3 variadic-parameter shadowing bugs where the renamed parameter `config ...T` collided with an existing local variable `config :=`:
  - `identity-model/authz.go` — local renamed to `resolved`
  - `usermgmt/http.go` — local renamed to `resolved`
  - `usermgmt/lockout.go` — local renamed to `resolved`
- **Tests:** All pass. Build clean. gofmt clean.

### 8. OAuth2 Sub-Service Extraction Prototype

- **What:** Extracted OAuth2 logic from `*Service` into a focused `OAuth2Service` struct.
- **Files:**
  - `usermgmt/service_oauth2_extracted.go` — new file with `OAuth2Service` struct, all OAuth2 logic (BeginLogin, FinishLogin, Unlink + 5 private helpers), `dispatcher` interface, `errorClassifier` function type
  - `usermgmt/service_oauth2.go` — rewritten as thin delegates (3 methods → `s.oauth2Svc.BeginLogin/FinishLogin/Unlink`)
  - `usermgmt/service_core.go` — added `oauth2Svc *OAuth2Service` field + wiring in `NewService`
- **Design pattern validated:** Focused service holds own deps + shared deps via constructor injection + shared logic via function fields. This is the ADR-0038 pattern for v5.
- **Tests:** All OAuth2 tests pass. Public API unchanged (delegates preserve `svc.BeginOAuthLogin(...)` signature).

### 9. Living Docs Harvested

- `TODO_LIST.md` — added P1 section with two items: OAuth2 prototype (done), re-export removal v5 (scheduled)
- `ROADMAP.md` — updated trigger status (0/3 met, method count tracking at 74), added re-export retirement section, cross-referenced prototype

### 10. Pre-existing Bug Fix

- `dashboardui/handlers_aggregates.go` — missing `strconv` import (pre-existing, unrelated to session work). Fixed.

---

## B) PARTIALLY DONE

### 1. OAuth2 Extraction — Pattern Validated, Not Fully Propagated

The OAuth2Service prototype works and proves the pattern, but the decomposition is incomplete:

- Only OAuth2 extracted. The other 5 domains (User, Membership, Tenant, Bot, Auth) still live on `*Service` (71 remaining methods).
- The `dispatcher` interface is defined locally in `service_oauth2_extracted.go` — it should be promoted to a shared location if other sub-services adopt it.
- The `errorClassifier` function type is co-located with OAuth2Service — same issue.
- No ADR written for the extraction pattern itself (though it validates ADR-0038).

### 2. Deprecation Markers — Coverage Complete, Enforcement None

- All 160 markers are in place, but there's no CI check that catches new re-exports being added without markers.
- No migration guide written for consumers (the markers say what to do, but there's no `docs/migrations/` guide).

### 3. Review Artifacts — Written but Not Cross-Referenced

- 8 HTML/SVG/D2 artifacts produced. They reference each other but are not linked from README, FEATURES, or AGENTS.md.
- No CHANGELOG entries for the review work or code changes (auto-git daemon committed, but CHANGELOG not updated — that's its convention).

---

## C) NOT STARTED

1. **CHANGELOG entries** — convention says completed work goes to CHANGELOG.md. Not done.
2. **ADR for OAuth2 extraction pattern** — validates ADR-0038 but no dedicated ADR recording the prototype decision.
3. **Consumer migration guide** — `docs/migrations/re-export-deprecation.md` or similar.
4. **CI check for unmarked re-exports** — a grep-based check that new aliases in usermgmt carry `// Deprecated:`.
5. **AGENTS.md update** — the deprecation markers, cfg rename, and OAuth2 extraction are not reflected in AGENTS.md.
6. **Integration test module test** — not explicitly run (only per-module tests).
7. **Lint check** — `nix run .#lint` not run after changes (gofmt done, golangci-lint not verified).
8. **Coverage gate** — `nix run .#coverage-gate` not run.
9. **check-templates** — `nix run .#check-templates` not run after touching usermgmt SQL setup files.

---

## D) TOTALLY FUCKED UP

### 1. Blind sed for cfg→config caused 3 build failures

The `sed -i 's/\bcfg\b/config/g'` was applied blindly across the workspace without checking for variable shadowing. Three functions had a variadic parameter `config ...T` that collided with an existing local variable also named `config` (previously `cfg`). This caused compile failures in `identity-model/authz.go`, `usermgmt/http.go`, and `usermgmt/lockout.go`.

**Root cause:** Applied a mechanical find-replace without reading the affected code first. Violated the critical rule "never edit files without reading first." Should have used LSP rename or at minimum checked for the variadic-parameter pattern before sed.

**Fix:** All three fixed by renaming the local variable to `resolved`. But this wasted time and could have been avoided.

### 2. Initially mischaracterized Casbin as "open"

The architecture review flagged Casbin as the "one genuinely open item" — a dependency-direction blemish. ADR-0044 (ACCEPTED 2026-07-23) already records this as a deliberate decision with alternative #3 (interface) explicitly rejected. I didn't read the ADRs thoroughly enough before writing the report. User corrected this directly ("Casbin is the base all this is build on top of and will stay").

### 3. First architecture-review report was too narrow

User explicitly called out ("That's all? Other skills!??!?!") that a single architecture-review skill was insufficient for a "DEEP" request. Should have proactively loaded complementary skills (go-modularize, data-model-review, naming-review, visualization) from the start rather than needing to be prompted.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **Track Service method count as a metric.** It's at 74 (+22 in 6 weeks). Add a CI check or script that fails when it crosses a threshold (e.g., 80). This is the leading indicator for v5 trigger.
2. **Promote the `dispatcher` interface.** It's currently local to `service_oauth2_extracted.go`. When extracting the next sub-service (User? Membership?), promote it to a shared file so both sub-services use the same interface.
3. **Unify `errorClassifier`.** Same as dispatcher — it's a shared concern that should live in a shared location when the second sub-service is extracted.
4. **The re-export layer needs a migration guide.** Markers say "import identity-model directly" but there's no document showing the before/after mapping (26 types, which paths change, which stay the same).
5. **Consider a `//cqrs-lint:ignore` or `//nolint:staticcheck` directive strategy for the deprecated re-exports.** They now emit SA1019 warnings in consumer code. Document whether this is expected.

### Process

6. **Always run `nix run .#lint` after code changes**, not just `go build`. The lint catches things gofmt doesn't.
7. **Always run `nix run .#coverage-gate`** — coverage regression is silent without it.
8. **Read ADRs before writing architecture reports.** The Casbin miss was avoidable — ADR-0044 is right there in `docs/adr/`.
9. **Load all relevant skills proactively for "DEEP" requests.** The user had to prompt for more skills.
10. **Write CHANGELOG entries immediately after completing work**, not at the end.

### Code Quality

11. **The OAuth2Service has duplicated logic from `*Service`** — `createSession`, `logAuth` are copied. In v5 these unify, but in v4 they're parallel implementations. Acceptable as a prototype, but document the duplication as intentional.
12. **`service_oauth2_extracted.go` has a long struct field list** — 10 fields. A `OAuth2Deps` struct could group the shared dependencies. Defer to when the second sub-service is extracted.

---

## F) Up to 50 Things to Get Done Next

### Immediate (this session's loose ends)

1. Run `nix run .#lint` across all modules
2. Run `nix run .#coverage-gate`
3. Run `nix run .#check-templates` (usermgmt SQL files were touched)
4. Write CHANGELOG entries for: deprecation markers, cfg→config rename, OAuth2Service extraction, dashboardui strconv fix
5. Update AGENTS.md with: cfg→config rename note, OAuth2Service field on Service struct, deprecation marker count
6. Run `integration_test` module tests explicitly

### Short-term (P1-P2)

7. Write ADR-0046: OAuth2Service extraction pattern (records the prototype decision, references ADR-0038)
8. Write consumer migration guide: `docs/migrations/re-export-deprecation.md`
9. Add CI check: grep for unmarked re-exports in usermgmt (any new `= identitymodel.` without `// Deprecated:` on the preceding line)
10. Add CI check or script: count `*Service` methods, fail if > 80
11. Extract UserService prototype (next domain after OAuth2, validates pattern with a second extraction)
12. Promote `dispatcher` interface and `errorClassifier` type to a shared file (`service_types.go` or similar)
13. Cross-link review artifacts from README or AGENTS.md
14. Run `nix run .#test-flake` to verify Nix hermetic build
15. Update FEATURES.md if the OAuth2 extraction changes any feature descriptions

### Medium-term (P2-P3)

16. Extract MembershipService prototype (3 methods)
17. Extract TenantService prototype (8 methods)
18. Extract BotService prototype (4 methods)
19. Extract AuthService prototype (4 methods — auth_methods)
20. Once all sub-services extracted, evaluate whether `*Service` can become a thin facade
21. Add deprecation `// Deprecated:` on the `oauth2` field of `Service` (now that `oauth2Svc` exists)
22. Consider removing the old `oauth2`, `oauth2States`, `oauth2StateTTL` fields from `Service` (they're redundant with `oauth2Svc`)
23. Document the `cfg`→`config` rename in the v4.x CHANGELOG
24. Audit whether any example code or docs still reference `cfg`
25. Review whether the dashboardui strconv bug indicates a missing `nix run .#build` step in the auto-git daemon's pre-commit

### Long-term (v5 preparation)

26. Draft v5 migration guide (aggregate module paths, Service API changes)
27. Design the `usermgmt/user/v5` module boundary (User aggregate extraction)
28. Design the `usermgmt/membership/v5` module boundary
29. Design the `usermgmt/tenant/v5` module boundary
30. Design the `usermgmt/bot/v5` module boundary
31. Evaluate whether `usermgmt/store/v5` should be a separate module or stay in core
32. Plan the v5 release process (batch tag, migration tooling)
33. Evaluate go-cqrs-lite publishing bug status (are the broken pseudo-versions fixed?)
34. If go-cqrs-lite is fixed, remove the go.work local replaces
35. Consider whether the re-export layer can be partially removed before v5 (type aliases are non-breaking to keep, but some could be removed in a minor if no consumers use them)

### Polish & Tooling

36. Run `nix run .#check-docs-links` to verify all cross-references in the new review artifacts
37. Add the architecture-review HTML to `docs/architecture-understanding/README.md` or index if one exists
38. Verify D2 SVGs render correctly in a browser (only verified file size, not visual output)
39. Consider adding the current vs improved architecture diagrams to README.md
40. Audit whether the naming review's `cfg` finding is fully resolved (grep for any remaining abbreviations: `mgr`, `cnt`, `tmp`)
41. Run `nix fmt` to ensure treefmt alignment is applied
42. Check if the `service_oauth2_extracted.go` file triggers any cqrs-lint findings (it has dispatch calls)
43. Add `//cqrs-lint:ignore` directives if needed on the extracted OAuth2 dispatch calls
44. Review whether the `OAuth2Service` needs its own coverage test (currently tested via `*Service` delegate tests)
45. Consider extracting a `ServiceDeps` struct that groups the shared dependencies passed to all sub-services

### Documentation

46. Update `docs/DOMAIN_LANGUAGE.md` if the OAuth2 extraction introduces new domain vocabulary
47. Document the sub-service extraction pattern in a guide (`docs/guides/sub-service-extraction.md`)
48. Update `docs/guides/leveraging-go-cqrs-lite.md` if the dispatcher interface affects the middleware composition story
49. Review all 8 review artifacts for internal consistency (do they reference each other correctly?)
50. Consider a `docs/reviews/INDEX.md` that lists all reviews by date and scope

---

## G) Questions I Cannot Answer Myself

### Q1: Should the old `oauth2`/`oauth2States`/`oauth2StateTTL` fields be removed from `*Service` now that `oauth2Svc` holds them?

The extraction duplicated the OAuth2-specific state: `*Service` still has `oauth2 OAuth2Provider`, `oauth2States OAuth2StateStore`, `oauth2StateTTL time.Duration`, AND `oauth2Svc *OAuth2Service` (which holds copies of the same). The old fields are still set in `NewService` for backward compat. Removing them is cleaner but might break consumers who access them directly (they're unexported, so probably safe). I can't determine if any test or external code relies on these fields being on `*Service` without a deeper audit, and I don't know your preference on cleanup-timing.

### Q2: Should the extracted `dispatcher` interface and `errorClassifier` type be promoted NOW, or wait until the second sub-service extraction?

Promoting them now (to a shared `service_types.go`) means the OAuth2Service file is self-contained but the shared types are premature (only one consumer). Waiting means the second extraction will require touching OAuth2Service to import from the new location. I can't decide this because it depends on your timeline for extracting the next sub-service (UserService etc.) — if that's soon, promote now; if that's months away, wait.

### Q3: The dashboardui `strconv` missing-import bug was pre-existing (the file used `strconv.ParseUint` at lines 131 and 230 but never imported `strconv`). How did this pass CI?

This is either (a) CI doesn't build dashboardui, (b) the auto-git daemon committed this without building, or (c) the file was recently modified (the session snapshot showed `handlers_index_test.go` modified). I can't determine the root cause without checking CI config and recent git history on that file. This might indicate a gap in the auto-commit pre-commit hook's build verification.
