# Status Report — 2026-07-04 14:18 CEST

## Project Vitals

| Metric                     | Value                                                                                              |
| -------------------------- | -------------------------------------------------------------------------------------------------- |
| **Module**                 | `github.com/larsartmann/cqrs-htmx/v4`                                                              |
| **Branch**                 | `master` (clean, up to date with origin)                                                           |
| **Go modules**             | 11 (root + usermgmt + 3 auth strategies + adminui + integration_test + 3 examples + datastar-demo) |
| **Production LOC**         | ~23,883 (excluding tests)                                                                          |
| **Test files**             | 204                                                                                                |
| **Errorfamily violations** | **0** (was 32)                                                                                     |
| **Context quality score**  | **96.6/100** (was 95.4)                                                                            |
| **Error paths**            | **14** (was 24)                                                                                    |
| **Critical losses**        | **0**                                                                                              |
| **High losses**            | **2** (was 5)                                                                                      |
| **Lint**                   | 0 issues across all 6 linted modules                                                               |
| **Tests**                  | 7/7 modules pass with `-race`                                                                      |
| **Module isolation**       | All modules build standalone (GOWORK=off)                                                          |
| **Dep budgets**            | All within budget (totp/webauthn/oauth2 at capacity — 0 slots remaining)                           |

---

## a) FULLY DONE

### Error Family Adoption (32 → 0 violations)

All 32 `errors.New`/`fmt.Errorf` calls in the 3 auth strategy modules replaced with `go-error-family` constructors. The library (`v0.5.1`) is now a direct dependency of `usermgmt/totp`, `usermgmt/webauthn`, and `usermgmt/oauth2` — chosen over the `event/v3` re-export because strategy modules must stay decoupled from `go-cqrs-lite`.

**Family assignments (semantically correct):**

- **Rejection** — config validation (`ProviderConfig.Validate`), provider not found, WebAuthn ceremony failures
- **Transient** — OIDC discovery, token exchange, userinfo fetch, id_token verification, WebAuthn begin ceremonies
- **Infrastructure** — JSON marshal/unmarshal failures, HTTP request construction, TOTP key generation
- **Corruption** — base32 decode failure, stored session/user JSON unmarshal failures

**Files changed:** `oauth2/provider.go` (16 sites), `totp/provider.go` (2 sites), `webauthn/provider.go` (14 sites + full rewrite after multiedit damage)

### Error Context Enrichment (24 → 14 error paths)

Domain-specific context variables attached via `.WithContext()` chaining:

| Area                    | Variables added                                                                                  | Files                                                         |
| ----------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------- |
| OAuth2 strategy         | `provider`, `state`, `code`, `pkce_verifier`                                                     | `oauth2/provider.go`                                          |
| TOTP strategy           | `account`                                                                                        | `totp/provider.go`                                            |
| `classifyDispatchError` | Variadic `kv ...string` parameter — call sites pass `new_email`, `new_name`, `reason`, `subject` | `service_register.go`, `service_misc.go`, `service_oauth2.go` |
| SQL stores              | `dialect`, `token`                                                                               | `sql_session_store.go`, `sql_event_store.go`                  |

The `classifyDispatchError` signature changed from `(err, userID)` to `(err, userID, kv ...string)` — fully backward compatible.

### Integration Test Fix

`integration_test/go.sum` was missing the `httputil v0.4.0` `h1:` hash (pre-existing from release v4.1.1 where the httputil bump wasn't propagated). Fixed via `go mod download`.

### Verification Gates (all green)

- `branching-flow errorfamily .` → **0 rows**
- `branching-flow context .` → **96.6/100**, 0 critical, 2 high, 15 medium
- `nix run .#test` → **7/7 modules pass**
- `nix run .#lint` → **0 issues** across all 6 modules
- `nix run .#check-modules` → isolation OK, dep budgets OK, no version drift, no absolute replace directives
- `nix run .#build` → all 10 buildable modules compile

---

## b) PARTIALLY DONE

### Remaining Context Losses (14 error paths, 2 high-severity)

The 2 remaining **high-severity** losses (score 90.0) are `error_propagation` sites where `providerName` and `state` flow through `service_oauth2.go` but aren't attached to the error. These are in the Service layer wrapping code — the provider errors come back with context, but the Service wraps them again and drops it.

The **medium-severity** losses (score ~96.7) are mostly in:

- HTTP handler `writeError(w, errorStatus(err), err.Error())` sites — `err.Error()` loses `status` context (8 sites across `http.go`, `webauthn_http.go`, `verification_totp_http.go`, `credential_http.go`, `oauth2_http.go`)
- `SafeDetail()` in `errors.go` — `event.Classify(err).DefaultMessage()` loses `status`
- `import_export.go` — `err.Error()` in import errors loses row context
- Impersonation service — `reason` not attached to all error paths

---

## c) NOT STARTED

1. **AGENTS.md update** — The error family section needs updating to reflect that strategy modules now use `go-error-family` directly (not exempt as previously documented)
2. **CHANGELOG entry** — No changelog entry for the error family + context enrichment work
3. ** CONTRIBUTING.md update** — Still references the old "strategy modules are exempt" rule
4. **Dep budget review** — totp/webauthn/oauth2 are at capacity (0 slots). Adding go-error-family consumed their last slots. If more deps are needed, budgets must be raised

---

## d) TOTALLY FUCKED UP

**Nothing.** All work landed cleanly. The only damage during the session was the `webauthn/provider.go` multiedit mishap — 2 edits matched duplicated patterns and mangled function boundaries. Fixed immediately by rewriting the entire file from scratch (290 lines), verified by build + 18 W3C-spec test vectors passing.

---

## e) WHAT WE SHOULD IMPROVE

1. **Strategy module dep budgets at capacity** — totp (2/2), webauthn (2/2), oauth2 (4/4) have zero headroom. Any future feature requiring a new dependency forces a budget increase. Consider raising budgets to 3/3/5.
2. **`writeError` pattern is repeated 15+ times** — `writeError(w, errorStatus(err), err.Error())` appears in every HTTP handler. This is a prime candidate for a `writeDispatchError(w, err)` helper that preserves context.
3. **`classifyDispatchError` type assertion** — Uses `errors.As` now (good), but the Rejection branch that adds context to pre-existing `*event.Error` values is subtle. If the inner error isn't `*event.Error`, context is silently dropped. Consider always wrapping.
4. **Error context is not logged** — `.WithContext()` enriches the error object, but the structured logger in `RequestLogging` doesn't extract `ErrorContext()`. Context is only visible if the consumer explicitly reads it.
5. **No error context in HTTP responses** — `writeError` sends `err.Error()` (the message), not the structured context map. The `StructuredError` / ProblemDetailsErrorHandler path does expose context, but it's opt-in.

---

## f) Top 25 Things to Do Next

### High Priority (error quality)

1. **Fix the 2 remaining high-severity context losses** in `service_oauth2.go` — attach `providerName`/`state` at Service-layer wrapping sites
2. **Add `writeDispatchError(w, r, err)` helper** — consolidate the 15+ `writeError(w, errorStatus(err), err.Error())` sites into one context-preserving function
3. **Wire `ErrorContext()` into structured logging** — `RequestLoggingSlog` should extract context map from classified errors and add slog attrs
4. **Wire `ErrorContext()` into `StructuredError`** — expose context fields in RFC 7807 `extension` fields

### Medium Priority (documentation)

5. **Update AGENTS.md** — remove "strategy modules exempt" claim, document go-error-family as direct dep
6. **Add CHANGELOG entry** for error family adoption + context enrichment
7. **Update CONTRIBUTING.md** — strategy modules now follow the no-stdlib-error-constructors rule
8. **Document `classifyDispatchError` variadic API** — show how to pass kv pairs from new call sites

### Architecture

9. **Raise strategy module dep budgets** — totp 2→3, webauthn 2→3, oauth2 4→5 (or document that they're intentionally at capacity)
10. **Review `SafeDetail` context loss** — `event.Classify(err).DefaultMessage()` drops the error's context map; consider including context in redacted detail
11. **Impersonation service error context** — attach `reason`, `callerID`, `targetID` to all error paths in `service_impersonation.go`

### Testing

12. **Add error context assertions to strategy module tests** — verify `.WithContext()` values survive through the full call chain
13. **Add `classifyDispatchError` kv propagation test** — test that kv pairs appear on Conflict and Transient branches
14. **Fuzz `classifyDispatchError`** with nil errors, empty kv, odd-length kv
15. **Integration test: error context crosses module boundary** — strategy module error → Service wrap → HTTP response

### Code Quality

16. **Extract error context helpers** — `contextPairs(err) map[string]string`, `mergeContext(err, kv...)` to reduce repetition
17. **Review `import_export.go` error handling** — `err.Error()` in import result loses row number / field context
18. **Audit all `err.Error()` calls in HTTP handlers** — 25 sites use `err.Error()` for response body; should use `SafeDetail` for 5xx
19. **Consider `errorfamily.Compose` for multi-error sites** — import/export accumulates errors; could use composed errors

### Infrastructure

20. **Add `branching-flow context` to CI** — fail on score < 95 or any new critical losses
21. **Add `branching-flow errorfamily` to CI as hard gate** — currently in flake but not in `.github/workflows/ci.yml`
22. **Pin `branching-flow` version** — currently unpinned, could break CI on upstream changes

### Polish

23. **Remove stale `httputil v0.3.0` entries from `integration_test/go.sum`** — both versions present, v0.3.0 is dead weight
24. **Consider `go-error-family` as direct dep in root go.mod** — currently `// indirect`; now that strategy modules use it directly, it's semantically direct
25. **Review error code naming conventions** — codes like `"internal"` vs `"usermgmt.sql_session.find_failed"` are inconsistent; establish a naming scheme

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `go-error-family` be added to the root `go.mod` as a direct dependency (removing `// indirect`), given that 3 strategy sub-modules now import it directly — or does the "indirect" label correctly reflect that the root module itself never imports the package?**

The root and usermgmt modules import `go-error-family` transitively through `event/v3` re-exports. The strategy modules import it directly. Go's module system marks it `// indirect` in every `go.mod` where the importing package is itself an indirect dependency. This is technically correct but semantically misleading — `go-error-family` is now a first-class architectural dependency, not an incidental transitive one. The question is whether to update the `// indirect` comments for documentation accuracy, even though `go mod tidy` will revert them.
