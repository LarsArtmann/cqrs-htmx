# Status Report: encoding/json/v2 Revert & Build Fix

**Date:** 2026-07-09 05:26
**Session focus:** Fix broken `test-race` and `hierarchical-errors` CI failures caused by automated `encoding/json/v2` migration

---

## a) FULLY DONE

### Root Cause Identified & Fixed

The `go-auto-upgrade` buildflow tool migrated `encoding/json` (v1) to `encoding/json/v2` + `encoding/json/jsontext` across **26 source files**. These are experimental stdlib packages excluded by build constraints in Go 1.26.4, causing:

```
package github.com/larsartmann/cqrs-htmx/v4
    imports encoding/json/jsontext: build constraints exclude all Go files in ...
    imports encoding/json/v2: build constraints exclude all Go files in ...
FAIL    github.com/larsartmann/cqrs-htmx/v4 [setup failed]
```

### Actions Taken

1. **Reverted all 26 source files** across all modules:
   - **Root (11 files):** `ack.go`, `csrf_helpers.go`, `decoder.go`, `errors.go`, `httputil.go`, `logging.go`, `response.go`, `structured_error.go`, `ws.go`, `ws_dispatch.go`, `ws_encoder.go`
   - **usermgmt (11 files):** `es_events.go`, `es_upcaster.go`, `http.go`, `import_export.go`, `service_oauth2.go`, `sql_session_store.go`, `sql_view_marshal.go`, `user.go`, `verification_totp_http.go`, `webauthn_service.go`
   - **usermgmt submodule providers (2 files):** `oauth2/provider.go`, `webauthn/provider.go`
   - **adminui (1 file):** `render.go`
   - **examples/datastar-demo (3 files):** `domain_cqrs.go`, `domain_store.go`, `domain_types.go`

2. **API call reversions included:**
   - `json.MarshalWrite(w, v)` → `json.NewEncoder(w).Encode(v)` / `json.NewEncoder(&buf).Encode(v)`
   - `json.UnmarshalRead(r, &v)` → `json.NewDecoder(r).Decode(&v)`
   - `json.MarshalEncode(enc, v)` → `enc.Encode(v)`
   - `jsontext.NewEncoder(w)` → `json.NewEncoder(w)`
   - `jsontext.Value` → `json.RawMessage`

### Verified

| Check                                    | Status                    |
| ---------------------------------------- | ------------------------- |
| `go build ./...` (root)                  | PASS                      |
| `go build ./...` (usermgmt)              | PASS                      |
| `go build ./...` (adminui)               | PASS                      |
| `go build ./...` (all 4 examples)        | PASS                      |
| `go test ./... -race` (root)             | PASS (4.0s)               |
| `go test ./... -race` (usermgmt)         | PASS (2.7s)               |
| `go test ./... -race` (integration_test) | PASS (1.0s)               |
| `go test ./... -race` (adminui)          | PASS (1.1s)               |
| `go test ./... -race` (totp)             | PASS (1.0s)               |
| `go test ./... -race` (webauthn)         | PASS (1.0s)               |
| `go test ./... -race` (oauth2)           | PASS (1.1s)               |
| `branching-flow errorfamily .`           | PASS (0 violations)       |
| `buildflow -s hierarchical-errors -v`    | PASS (0 violations, 6.4s) |
| `buildflow -s test-race -v`              | PASS (1/1, 5.0s)          |

### Legitimate Auto-Fixes Kept

These were applied by other buildflow tools and are correct — kept in the working tree:

- **`nix-flake-update`:** Bumped transitive deps across all modules (`golang.org/x/sys` v0.46→v0.47, `golang.org/x/sync` v0.21→v0.22, `golang.org/x/text` v0.39→v0.40, `modernc.org/libc` v1.74.0→v1.74.1, `templ-components` v0.9.0→v0.10.0)
- **`goimports`:** Merged separate import groups into single groups in 3 example files (`domain_commands.go`, `handlers_helpers.go`, `admin-demo/main.go`) — cosmetic, removes blank-line-separated import blocks

### Remaining Working Tree Changes (20 files)

All are go.mod/go.sum dependency bumps + 3 cosmetic import reordering in examples. No source logic changes remain.

---

## b) PARTIALLY DONE

Nothing partial — the fix was binary (revert or don't).

---

## c) NOT STARTED

- **Preventing recurrence:** No buildflow config change to prevent `go-auto-upgrade` from re-applying the json/v2 migration on the next run
- **Actual encoding/json/v2 adoption:** The project docs note this is deferred to v4.1 (Go 1.26+ when the package stabilizes). Not started, intentionally.

---

## d) TOTALLY FUCKED UP

Nothing. The fix was clean — all changes were pure reverts of the broken migration, verified by the fact that every file's diff contained ONLY json/v2 changes (no legitimate code mixed in).

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **`go-auto-upgrade` should be configured to SKIP `encoding/json` migration.** This is a known trap: `encoding/json/v2` and `encoding/json/jsontext` exist in the Go 1.26 source tree but are behind `GOEXPERIMENT=jsonv2` build constraints. Any automated tool that sees `encoding/json` and upgrades to `/v2` will break the build. This WILL recur on every buildflow run unless the tool is configured to exclude this migration.

2. **Buildflow `--fix` mode is dangerous without build verification between steps.** The `go-auto-upgrade` tool applied a breaking change, then `golangci-lint` auto-fix ran on top of it, and the failure wasn't caught until `test-race` at the end. If build verification ran between auto-fix steps, the broken migration would have been caught immediately.

3. **Consider adding `encoding/json/v2` to a denylist.** Whether in `.buildflow.yml`, a CI lint rule, or a pre-commit hook — the import path `encoding/json/v2` should be rejected until explicitly opted in.

### Codebase Observations (noted during this session)

4. **Import grouping inconsistency:** Some files separate `go-error-family` imports from `go-cqrs-lite` imports with a blank line; others group them together. The `goimports` auto-fix normalized toward grouped, but this is cosmetic and inconsistent across the codebase.

5. **The `docs/planning/` references to encoding/json/v2** (in `2026-07-02_v4-auth-extraction-execution-plan.md`) correctly note this is deferred to v4.1 — the docs are accurate.

---

## f) Up to 50 Things We Should Get Done Next

### Immediate (this session's followup)

1. **Configure `go-auto-upgrade` to skip `encoding/json` → `encoding/json/v2` migration** — prevent recurrence
2. **Add a CI lint rule rejecting `encoding/json/v2` and `encoding/json/jsontext` imports** — defense in depth
3. **Commit the current working tree** (20 files: dep bumps + import reordering) — these are legitimate and ready
4. **Run `nix flake check`** to verify nix formatting is still clean after the dep updates

### Short-term (days)

5. **Review `templ-components` v0.9.0 → v0.10.0 bump** — check changelog for breaking changes
6. **Audit all go.sum changes** for unexpected new dependencies
7. **Run full `nix run .#test` (all modules in sequence)** to catch any module-interaction issues
8. **Run `nix run .#lint`** to verify golangci-lint passes on all modules with the dep changes
9. **Update `go.work.sum`** — verify it's consistent with all module go.sum files
10. **Check if `golang.org/x/sync` v0.22.0 has any API changes** affecting the rate limiter or fanout code

### Medium-term (weeks)

11. **Evaluate actual `encoding/json/v2` adoption timeline** — when Go stabilizes it, plan the migration properly (not via auto-upgrade)
12. **Standardize import grouping** across the entire codebase — pick one style (grouped or separated) and enforce via lint rule
13. **Add integration test that builds all modules with `GOWORK=off`** — catches build-constraint issues per-module
14. **Review the `go-auto-upgrade` tool's scope** — what other migrations does it attempt that could be similarly destructive?
15. **Document the `encoding/json/v2` trap in AGENTS.md** — so future sessions know about it
16. **Consider pinning `GOEXPERIMENT` in devShell** if any developer wants to experiment with json/v2 locally
17. **Audit `.buildflow.yml` for other risky auto-fix tools** — what else can break the build silently?
18. **Add a post-auto-fix build verification step** in buildflow config if supported
19. **Review whether the `templ-components` bump affects adminui rendering** — run the seed_render_test with visual inspection
20. **Check `golang.org/x/text` v0.40.0 changelog** — used transitively, but worth verifying
21. **Verify `modernc.org/libc` v1.74.1 doesn't break SQLite** — run SQL-backed read model tests specifically
22. **Consider adding `go vet` as a separate buildflow step** — it catches different issues than golangci-lint
23. **Review the `go.work.sum` growth** — 40 lines added, verify no orphaned entries
24. **Run the admin-demo example manually** — `go run .` and verify the UI renders correctly with updated deps
25. **Check if any new golangci-lint rules fire** with the updated dependencies

### Strategic (month+)

26. **Plan proper `encoding/json/v2` migration** (v4.1 target) — API changes, performance benchmarks, test strategy
27. **Consider `encoding/json/v2` performance benchmarks** — measure potential gains for decoder.go and response.go
28. **Evaluate `sonic` or `sonic/v2`** as alternative JSON libraries if performance is a concern before json/v2 stabilizes
29. **Review the buildflow toolchain** — is `go-auto-upgrade` adding value or just creating noise?
30. **Standardize dependency update cadence** — weekly? per-release? ad-hoc via buildflow?
31. **Consider a `renovate.json` or similar** for controlled dependency updates instead of buildflow auto-upgrade
32. **Document the module dependency budget enforcement** — verify dep count hasn't changed with the bumps
33. **Run `nix run .#check-modules`** — verify module isolation still holds after dep updates
34. **Review whether `x/sync` v0.22 adds errgroup improvements** usable in the SSE/fanout code
35. **Audit all `// indirect` deps** — some may have become direct with the new versions
    36–50. _(Reserved — no additional actionable items identified from this session's scope)_

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. How do we configure `go-auto-upgrade` to stop migrating `encoding/json` → `encoding/json/v2`?

This will recur on every `buildflow --fix` run. I don't know if the tool supports per-package exclusion lists, skip rules, or if we need to file a bug/feature request upstream. The `.buildflow.yml` file may have configuration options I'm not aware of. **This is the #1 priority to prevent recurrence.**

### 2. Should we commit the dependency bumps (`x/sys`, `x/sync`, `x/text`, `libc`, `templ-components`) now, or batch them with other work?

The 20 remaining changed files are all go.mod/go.sum bumps + 3 cosmetic import reorderings. They're legitimate and verified, but I don't know if there's a release window, a batching strategy, or if you prefer to review dependency changes separately from code changes. The user hasn't said "commit" so I haven't committed anything.
