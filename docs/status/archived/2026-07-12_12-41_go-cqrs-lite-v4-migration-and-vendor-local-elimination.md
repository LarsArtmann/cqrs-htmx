# Status: go-cqrs-lite v3→v4 Migration & .vendor-local Elimination

**Date:** 2026-07-12 12:41
**Session scope:** Migrate all go-cqrs-lite dependencies from v3.7.4 to v4.0.0, eliminate `.vendor-local`, fix all docs, self-review
**Commits:** 10 (all pushed to master)
**Files changed:** 213 files, +1291 / -2667 lines (net reduction of ~1376 lines)

---

## A) FULLY DONE (verified green)

### go-cqrs-lite v3→v4 Migration

- All 12 Go modules migrated: root, usermgmt, usermgmt/totp, usermgmt/webauthn, usermgmt/oauth2, adminui, loginpage, integration_test, examples/basic, examples/admin-demo, examples/catalog-demo, examples/datastar-demo
- All Go source imports updated (0 v3 imports remaining in any `.go` file)
- All go.mod/go.sum files cleaned (0 v3 indirect dependencies remaining)
- `usermgmt/.golangci.yml` exhaustruct exclusions updated (`kv/v3` → `kv/v4`)
- Source comments updated in 7 files (sse_event.go, idempotency.go, errors.go, sql_event_store.go, sql_session_store.go, es_materialize_adapter.go, es_setup.go)
- Fixed pre-existing bug in `.vendor-local/eventtest/fake_store.go` (missing `errorfamily` import) before deleting it

### .vendor-local Elimination

- Deleted `.vendor-local/eventtest/` entirely (14 files, ~1600 lines)
- Deleted `.vendor-local/README.md` (orphaned after directory removal)
- Removed `eventtest` replace directives from all 8 go.mod files
- Removed `.vendor-local/eventtest` from `go.work`
- Verified `go mod tidy -e` works as the replacement workflow (root module only; submodules clean)

### Documentation Cleanup

- **README.md**: import paths, version refs, catalog refs (all v3→v4)
- **CONTRIBUTING.md**: event/v3 → event/v4
- **usermgmt/README.md**: signing/v3, encryption/v3 → v4
- **usermgmt/AGENTS.md**: id/v3 → id/v4
- **usermgmt/CONTRIBUTING.md**: event/v3 → event/v4
- **adminui/README.md**: cqrs-htmx/v3 → v4
- **AGENTS.md**: 6 v3→v4 edits (dep table, eventtest, publishing bug, module deps)
- **FEATURES.md**: all go-cqrs-lite version refs updated
- **TODO_LIST.md**: header version updated
- **VERSIONING.md**: catalog/v3 → catalog/v4
- **SKILL.md**: import path examples updated
- **docs/migrations/v3-to-v4.md**: added full "go-cqrs-lite v3 → v4" section with import changes, publishing bug, and .vendor-local cleanup explanation
- **CHANGELOG.md**: added [Unreleased] entry for v4 migration + .vendor-local removal

### Verification (all green)

- **Build**: all 12 modules build standalone (GOWORK=off)
- **Tests**: all 8 test suites pass with `-race`
- **Examples**: all 4 examples build
- **Lint**: 0 issues (root + usermgmt + adminui)
- **Module isolation**: all modules pass GOWORK=off build+vet
- **Dependency budgets**: all modules within budget
- **Version drift**: clean (all v4.0.0)
- **Replace directives**: all relative paths
- **errorfamily**: 0 violations (root + usermgmt)
- **Root coverage**: 93.0%

---

## B) PARTIALLY DONE

### Historical Docs (intentionally left as point-in-time records)

- `docs/status/` — 30+ status reports reference v3 (they describe what was true at the time)
- `docs/adr/` — ADRs reference v3 (decisions were made against v3)
- `docs/planning/` — planning docs reference v3
- `ROADMAP.md` version history table — entries describe v3.x releases (correct as history)
- `CHANGELOG.md` historical entries — describe v3.x changes (correct as history)
- **Decision needed**: should these be left as-is or retroactively updated? (See Questions)

### Evaluation Docs (3 files with current v3 refs)

- `docs/observability-wiring.md`: 3 refs to `otel/v3`, `middleware/v3`, `prometheus/v3` — this is a **current** guide, should be v4
- `docs/evaluations/relational-projection-evaluation.md`: 1 ref — evaluation doc, borderline
- `docs/evaluations/catchup-subscriber-evaluation.md`: 1 ref — evaluation doc, borderline

---

## C) NOT STARTED

- `docs/observability-wiring.md` v3→v4 update (should be done — it's a current guide)
- Upstream issue in go-cqrs-lite to properly publish `eventtest` module (eliminates the `go mod tidy -e` workaround)
- CI workflow review — need to verify CI passes with the new go.mod/go.sum state (CI doesn't run `go mod tidy`, so should be fine, but untested)

---

## D) TOTALLY FUCKED UP

**Nothing was totally fucked up.** The migration went smoothly after identifying the root cause (the vendored eventtest module was the source of all v3 transitive leakage).

### What I did poorly (honest self-critique):

1. **First-pass doc blindness**: I checked Go source files and go.mod files but completely missed consumer-facing docs (README.md, CONTRIBUTING.md, FEATURES.md, etc.) in the initial migration. The user had to ask "what did you forget?" for me to catch 7+ doc files with stale v3 references. This was a **systematic miss** — I should have grepped ALL files for v3 references from the start.

2. **The `.vendor-local/README.md` miss**: When I deleted the `.vendor-local` directory via `trash`, I only checked if the directory was gone. I didn't check `git status` to see if a tracked file was orphaned. The user's question prompted me to find it.

3. **No CHANGELOG entry**: I completed an entire major dependency migration without adding a CHANGELOG entry. This is basic release hygiene — any consumer upgrading to v4 needs to know what changed.

4. **No migration doc update**: The `docs/migrations/v3-to-v4.md` file existed but said nothing about go-cqrs-lite v3→v4. Consumers upgrading would be confused.

5. **Initial eventtest approach was wrong**: My first instinct was to update the vendored eventtest module to v4 (which I did correctly). But the better question was "do we even need this?" — nobody imports it, and `go mod tidy -e` handles the resolution. The user had to suggest eliminating `.vendor-local` entirely.

---

## E) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Migration checklist**: For any version migration, always grep ALL files (not just source code): `rg "oldversion" .` across every file type. Source, docs, config, CI, examples.
2. **CHANGELOG-first**: Any dependency version change should get a CHANGELOG entry as part of the migration, not as an afterthought.
3. **"Does this need to exist?" test**: Before updating a workaround (like `.vendor-local`), ask if the workaround is still needed at all.
4. **Git status after deletions**: Always check `git status` after deleting directories to catch orphaned tracked files.

### Technical Improvements

5. **`go mod tidy -e` is a papercut**: The root cause is go-cqrs-lite's publishing bug. We should file an upstream issue to publish the `eventtest` module or remove it from `event/v4`'s published go.mod.
6. **Historical doc policy**: Need a clear policy on whether historical docs (status reports, ADRs, planning) should be retroactively updated or left as point-in-time records.
7. **Coverage gate verification**: The coverage gate (`nix run .#coverage-gate`) wasn't run this session — need to verify it still passes.

---

## F) Next 50 Things to Get Done

### High Impact, Low Effort (quick wins)

1. **Fix `docs/observability-wiring.md`** — 3 v3→v4 refs (otel/v3, middleware/v3, prometheus/v3)
2. **Run `nix run .#coverage-gate`** — verify coverage thresholds still pass
3. **Verify CI passes** — push triggered CI; check if any CI step references v3 or `.vendor-local`
4. **File upstream issue** in go-cqrs-lite to properly publish eventtest (eliminates `-e` workaround)
5. \*\*Check `docs/evaluations/` — decide if the 2 evaluation docs need v4 updates
6. **Run `nix flake check`** — verify flake still works after all changes
7. **Check `.github/workflows/ci.yml`** for any stale v3 or .vendor-local references
8. **Check `flake.nix`** for any stale v3 or eventtest references

### Medium Impact, Medium Effort

9. **Run full coverage report** (`nix run .#coverage`) and update AGENTS.md coverage stats
10. **Audit historical docs** — decide policy: leave as-is vs add "(historical)" headers
11. **Add a `go mod verify` step** to CI to catch checksum issues early
12. **Consider a `make check-versions` script** that greps for stale version references across all files
13. **Update `docs/adr/INDEX.md`** if any ADR needs a v4 addendum
14. **Review the v3-to-v4 migration doc** with fresh eyes — is it complete enough for a consumer?
15. **Check `admin-demo` example** — does it still run correctly with v4?
16. **Check `basic` example** — does it still run correctly with v4?
17. **Verify `catalog-demo` example** still works with catalog/v4

### Architecture & Type Model Improvements

18. **Review v4 compat aliases** — go-cqrs-lite v4 has `event.AggregateType = id.AggregateType` etc. Should we use `id.AggregateType` directly now?
19. **Audit `kv.ViewQuery` usage** — v4 added `OrderBy`, `Desc`, `Limit`, `Offset`, `Values` fields. Are we using them where beneficial?
20. **Check if v4's `metadata` package** offers capabilities we should adopt (was extracted from event)
21. **Review if v4's module restructure** enables any dependency isolation improvements
22. **Audit `projection` package usage** — v4 extracted it from event. Any import cleanup needed?
23. **Consider adopting `stack.TypedRepository`** from v4 for compile-time command type binding
24. **Review `dedup.Ring` usage** — verify it's still the right dedup approach in v4
25. **Check `codec.ForEncoding`** — verify v4 codec resolution is still correct

### Testing Improvements

26. **Add a migration test** — verify no v3 imports can sneak back in (grep-based guard)
27. **Add a `go mod tidy -e` smoke test** to CI (verify root module tidies without fatal errors)
28. **Cross-module integration test** — verify all modules resolve correctly under GOWORK=off
29. **Verify `scenario/v4` tests** still pass if we have any
30. **Add test for datastar-demo** — it had the most fragile go.sum (publishing bug workaround)
31. **Run benchmarks** — verify v4 doesn't regress performance vs v3

### Documentation Improvements

32. **Write a "v4 upgrade checklist"** for consumers (1-page quick reference)
33. **Update `docs/adr/0016-go-cqrs-lite-v3-migration.md`** with a v4 addendum
34. **Review all example READMEs** for v3 references
35. **Check `loginpage/README.md`** for v3 references
36. **Update dependency table in README.md** — verify all versions are current
37. **Add a "Troubleshooting" section** to the migration guide (common errors + fixes)
38. **Document the `-e` flag workaround** in CONTRIBUTING.md (how to add new deps)

### Developer Experience

39. **Create a `just tidy` or nix app** that runs `go mod tidy -e` for root and `go mod tidy` for submodules
40. **Add a pre-commit hook** that rejects v3 go-cqrs-lite imports
41. **Create a `scripts/check-no-v3.sh`** script that fails if any v3 references are found in source
42. **Consider a Go workspace cleanup** — do we still need all 12 modules in go.work?
43. **Review GOWORK=off workflow** — can any modules be consolidated?

### Security & Maintenance

44. **Run `govulncheck`** against v4 dependencies
45. **Run `gosec`** against the codebase
46. **Check for any CVEs** in new v4 transitive dependencies
47. **Review the otel/v4 dependency** — any new transitive deps?
48. **Audit the `metadata/v4` package** — any security-relevant changes?
49. **Verify `snapshot/v4` API** hasn't changed in ways affecting our snapshot store
50. **Review `storage/v4` API** — any breaking changes in SQL store wrappers?

---

## G) Top 2 Questions

### Question 1: Historical docs policy

**Should we retroactively update historical docs (status reports, ADRs, planning docs, ROADMAP.md version table) to remove v3 references, or leave them as accurate point-in-time records?**

There are 81 files with v3 references outside of go.sum. Most are in `docs/status/`, `docs/adr/`, `docs/planning/`, `ROADMAP.md`, and `CHANGELOG.md` historical entries. Updating them all would be significant effort and arguably dishonest (they describe what was true at the time). But leaving them means anyone searching for "v3" finds stale info.

**My recommendation:** Leave historical docs as-is. Add a note to CONTRIBUTING.md: "Historical docs in `docs/status/`, `docs/adr/`, and `docs/planning/` reference previous versions. They are point-in-time records and intentionally not updated." Then just fix the 3 current evaluation/guide docs.

### Question 2: Should we file the upstream issue?

**Should I create a GitHub issue in `go-cqrs-lite` requesting that the `eventtest` module be properly published (or removed from `event/v4`'s published go.mod)?**

This would eliminate the `go mod tidy -e` workaround for every consumer, not just us. The root cause: `event/v4`'s published go.mod has `require github.com/larsartmann/go-cqrs-lite/event/v4/eventtest v0.0.0-00010101000000-000000000000` with a replace directive that doesn't resolve at publish time. Every consumer importing `event/v4` hits this during `go mod tidy`.

**My recommendation:** Yes — this is a 5-minute fix upstream (either publish eventtest as a real module or move it to a test-only build tag) that saves every downstream consumer from the `-e` workaround.
