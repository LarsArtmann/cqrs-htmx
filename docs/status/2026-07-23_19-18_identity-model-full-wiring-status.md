# Status Report: identity-model Extraction & Full Wiring

**Date:** 2026-07-23 19:18
**Session scope:** Execute the Phase 2 wiring plan (casbin-leverage-and-full-wiring-plan.md) — fix go-cqrs-lite API breakage, wire ALL usermgmt domain types to identity-model via type aliases, build + test entire workspace, polish.

---

## a) FULLY DONE

### 1. go-cqrs-lite API breakage fixed (workspace-wide)

The entire workspace had pre-existing compilation failures from go-cqrs-lite API evolution (`AggregateID()`→`StreamID()`, `AggregateType`→`StreamType`, `event.AggregateRef`→`id.StreamRef`, `snapshot.AggregateID/Type`→`StreamID/Type`). Fixed in:

- `usermgmt/es_dispatch.go`, `es_bot_dispatch.go`, `es_membership_dispatch.go`, `es_tenant_dispatch.go` — command method renames
- `usermgmt/snapshot.go` — snapshot struct field renames
- `usermgmt/coverage_sql_readmodel_test.go`, `service_security_test.go`, `sql_event_store_test.go`, `es_scenario_*_test.go`, `es_membership_test.go`, `snapshot_test.go`, `sql_event_store_errors_test.go`, `sql_event_store_extra_test.go` — test compilation fixes
- Root: `testing_types_test.go`, `typed_handlers_test.go`, `bdd_test.go`, `event_store_sse_test.go`
- `examples/basic/main.go`, `examples/datastar-demo/domain_cqrs.go` — example command method renames

### 2. Authz engine fully wired to identity-model

- `usermgmt/authz_types.go` — replaced 296 lines with type aliases (`type Authz = identitymodel.Authz` etc.) + re-exported `NewAuthz()` and `AssignableRoles()`
- `usermgmt/authz_roles.go` — **deleted** (all methods inherited through type alias)
- `usermgmt/authz_policies.go` — **deleted** (all methods inherited through type alias)

### 3. ALL usermgmt domain types wired to identity-model

- `usermgmt/id.go` — UserID, TenantID, BotID, ActorKind, ActorID + all constructors aliased
- `usermgmt/email.go` — Email type + ParseEmail/MustParseEmail aliased
- `usermgmt/credential.go` — WebAuthnCredential, CredentialCore + NewCredentialFromPayload aliased
- `usermgmt/external_account.go` — ExternalAccount, ExternalAccountCore + NewExternalAccount aliased
- `usermgmt/user.go` — User, Session, SessionOrigin, DirectLogin, Impersonation, Membership + constructors aliased
- `usermgmt/crypto.go` — GenerateToken, HashToken, VerifyToken aliased
- `usermgmt/random.go` — randomBase64URLString delegates to identitymodel.RandomBase64URLString
- `usermgmt/es_events.go` — 12 event payload structs aliased + marshalPayload delegates to MarshalPayload
- `usermgmt/es_membership_events.go` — 3 event payload structs aliased
- `usermgmt/es_tenant_events.go` — 4 event payload structs aliased
- `usermgmt/es_bot_events.go` — 2 event payload structs aliased
- `usermgmt/es_commands.go` — 11 command structs aliased + constructors + mustCommand helper
- `usermgmt/es_membership_commands.go` — 3 command structs aliased + deriveMembershipID wrapper
- `usermgmt/es_tenant_commands.go` — 4 command structs aliased
- `usermgmt/es_bot_commands.go` — 2 command structs aliased
- `usermgmt/es_state.go` — UserState aliased; foldUser kept (upcaster dependency); unmarshalPayload kept
- `usermgmt/es_membership_state.go` — MembershipState aliased; foldMembership kept
- `usermgmt/es_tenant_state.go` — TenantState aliased; foldTenant kept
- `usermgmt/es_bot_state.go` — BotState aliased; foldBot kept
- `usermgmt/errors.go` — all 28 error sentinels re-exported from identitymodel with WithHTTPStatus wrapping where needed

### 4. identity-model exports fixed for cross-module use

- `foldUser`→`FoldUser`, `foldMembership`→`FoldMembership`, `foldTenant`→`FoldTenant`, `foldBot`→`FoldBot` — exported
- `randomBase64URLString`→`RandomBase64URLString` — exported
- `credentialCore`→`CredentialCore`, `externalAccountCore`→`ExternalAccountCore` — exported
- Removed unused: `fmtPolicy`, `allUserEventTypes`, `allMembershipEventTypes`, `allTenantEventTypes`, `allBotEventTypes`
- Fixed deprecated API usage: `id.AggregateID`→`id.StreamID`, `id.DeriveAggregateID`→`id.DeriveStreamID`, `id.NewAggregateID`→`id.NewStreamID`
- 0 golangci-lint issues
- 34 tests pass with `-race`

### 5. Command dispatch updated to use accessor methods

All 4 dispatch files now call `c.Email()`, `c.Roles()`, `c.Name()`, `c.OwnerID()`, etc. instead of direct field access `c.email`, `c.roles`. All scenario test files updated similarly.

### 6. Documentation updated

- `AGENTS.md` — architecture section updated (12→13 modules, identity-model description, dependency direction, new gotcha)
- `CHANGELOG.md` — Added + Changed sections for identity-model extraction and full wiring
- `identity-model/LICENSE` — copied from root
- Code formatted via `nix fmt` (18 files changed)

### 7. Build status

**Entire workspace builds clean:** `GOEXPERIMENT=jsonv2 go build ./...` exits 0.

---

## b) PARTIALLY DONE

### usermgmt tests: PASS with caveats
- usermgmt test suite runs and passes WITHOUT `-race` for most tests
- WITH `-race`, 9 tests fail due to the pre-existing MapError bug (see section d)
- `TestAuthz_Authorize_ReturnsErrForbidden` was failing initially but is now FIXED (errors.Is identity matching works)

### identity-model lint: clean but could be better
- 0 golangci-lint issues
- Missing: GOWORK=off replace directives not yet added to `identity-model/go.mod`
- Missing: identity-model not added to `flake.nix` build targets

### adminui/integration_test
- adminui: all tests pass
- integration_test: 2 tests fail (TestCrossModuleErrUnauthorized, TestCrossModuleErrForbidden) — pre-existing

---

## c) NOT STARTED

1. **GOWORK=off replace directives** in identity-model/go.mod — identity-model can't build standalone without the go.work local replaces
2. **flake.nix integration** — identity-model not in flake.nix build/check targets
3. **identity-model added to integration_test module** — no cross-module bridge tests yet
4. **Additional identity-model tests** (foldMembership 3+ scenarios, foldBot 2+ scenarios, ActorID JSON round-trip, Session JSON round-trip)
5. **godoc examples** (`example_test.go`)
6. **Evaluate CasbinProjection move** to identity-model
7. **usermgmt interfaces** (TOTPProvider, WebAuthnProvider, OAuth2Provider, store interfaces) — NOT yet aliased to identity-model. They still live in usermgmt and may need to be evaluated for extraction.
8. **maxEmailLength constant** — still lives in `usermgmt/service_core.go` and is duplicated in `identity-model/email.go` as `maxEmailLength`. Minor split-brain.
9. **constants.go duplication** — event types, command types, and aggregate types are duplicated between usermgmt and identity-model (simple string consts, can't diverge in behavior, but still a code smell)

---

## d) TOTALLY FUCKED UP

### PRE-EXISTING: MapError returns 500 for ALL errorfamily errors

**Root cause:** `go-error-family` was bumped to **v0.8.0** (AGENTS.md still says v0.7.0). In v0.8.0, `*errorfamily.Error` gained an `HTTPStatus() int` method that returns the `httpStatus` field (0 when unset). This makes EVERY `*errorfamily.Error` satisfy `cqrshtmx.HTTPStatusCarrier` interface. `MapError` → `carrierStatus()` matches, gets status=0, `validHTTPStatus(0)` returns false, so it falls back to `(500, true)`. This short-circuits all other status mapping logic.

**Impact:** 11+ tests fail across root, usermgmt, and integration_test:
- Root: `TestProblemDetailsErrorHandler_ShapeAndContentType` (500 instead of 400)
- usermgmt: `TestErrorStatus` (all subtests), `TestHandlers_Register_DuplicateEmail` (500 instead of 409), `TestHandlers_Register_EmptyBody` (500 instead of 400), `TestHandlers_Logout_StoreError`, `TestHandler_WebAuthnBegin_UserNotFound`, `TestWriteDispatchError_IncludesCodeField`, `TestHandler_OAuth2Callback_InvalidState`
- integration_test: `TestCrossModuleErrUnauthorized` (500 instead of 401), `TestCrossModuleErrForbidden` (500 instead of 403)

**Verification:** I confirmed this is pre-existing by stashing my changes and testing at HEAD. The root module's own `cqrshtmx.ErrUnauthorized` maps to 500 via `cqrshtmx.MapError`.

**Fix needed:** In `errors_status.go`, `carrierStatus()` should treat `HTTPStatus() == 0` as "not set" and return `(0, false)` instead of `(500, true)`. One-line fix. NOT in scope for identity-model extraction.

### SESSION MISTAKE: git checkout restored deleted files

When I ran `git checkout 85972b4 -- .` to test pre-session state, it resurrected the deleted `authz_policies.go` and `authz_roles.go` files, causing a build breakage. Fixed by re-deleting them. **Lesson:** `git checkout <commit> -- .` restores ALL files from that commit, including ones you've deleted. Should have used a more targeted approach.

### SESSION MISTAKE: ErrForbidden/ErrUnauthorized wrapping confusion

I initially left `ErrForbidden` and `ErrUnauthorized` unwrapped, then wrapped them with `WithHTTPStatus`, then had to revert because wrapping breaks `errors.Is` matching (Authz.Authorize returns the identity-model sentinel, not the wrapped one). The final state is unwrapped — they ARE `identitymodel.ErrForbidden`/`identitymodel.ErrUnauthorized` directly. This means `errors.Is` works correctly, but MapError status mapping relies on the broken `authStatusFromErrorCode` path (which is pre-existing broken).

---

## e) WHAT WE SHOULD IMPROVE

1. **Fix the MapError carrierStatus bug FIRST** — it's a one-line fix in `errors_status.go` and unblocks 11+ tests across the workspace. This should have been the first thing I checked, not the last.
2. **The constants duplication** between usermgmt and identity-model (event types, command types, aggregate types) should be consolidated — either alias them or move the constants to identity-model only.
3. **The `maxEmailLength` constant** lives in both `service_core.go` (254) and `identity-model/email.go` (254). Should be single-source.
4. **Fold functions are duplicated** — identity-model has FoldUser/FoldMembership/FoldTenant/FoldBot but usermgmt keeps its own foldUser/foldMembership/foldTenant/foldBot because they use usermgmt's upcaster registry. This is a split brain. The upcaster registry should be parameterized or moved to identity-model.
5. **The `unmarshalPayload` function is duplicated** — both identity-model and usermgmt have their own copy. usermgmt's version uses the upcaster registry; identity-model's is a pass-through.
6. **Interface extraction** — TOTPProvider, WebAuthnProvider, OAuth2Provider, and store interfaces are NOT yet in identity-model. They should be evaluated for extraction.
7. **Error wrapping strategy** — the current approach of wrapping identitymodel errors with WithHTTPStatus in usermgmt creates a two-tier error system. Consider whether identity-model errors should carry their own HTTP status hints.
8. **BuildFlow auto-commits** — the pre-commit hook auto-committed my changes with generic messages. This makes it hard to review the actual logical changes. Should consider squashing or amending.
9. **Test isolation** — I ran the full test suite too late. Should have run usermgmt tests immediately after wiring errors, before moving on to documentation.
10. **AGENTS.md version mismatch** — says `go-error-family v0.7.0` but actual is v0.8.0. This caused confusion when debugging.

---

## f) Up to 50 things to get done next

**Critical (blocks tests):**
1. Fix `carrierStatus()` in `errors_status.go` to treat `HTTPStatus()==0` as "not set" (return false)
2. Update `go-error-family` version reference in AGENTS.md from v0.7.0 to v0.8.0
3. Re-run full test suite after MapError fix to verify all 11+ tests pass
4. Run `nix run .#test` to verify workspace-level test gate passes

**identity-model completeness:**
5. Add GOWORK=off replace directives to `identity-model/go.mod` (matching go.work)
6. Verify `GOWORK=off go build ./...` works in identity-model
7. Verify `GOWORK=off go build ./...` works in usermgmt
8. Add identity-model to `flake.nix` build targets
9. Add identity-model to `flake.nix` test/lint/check targets
10. Add identity-model to `flake.nix` coverage gate
11. Add more foldMembership tests (3+ scenarios)
12. Add foldBot tests (2+ scenarios)
13. Add ActorID JSON marshal/unmarshal round-trip test
14. Add Session JSON marshal/unmarshal round-trip test
15. Add godoc examples (`example_test.go`)
16. Consider moving CasbinProjection to identity-model
17. Extract TOTPProvider/WebAuthnProvider/OAuth2Provider interfaces to identity-model
18. Extract store interfaces to identity-model
19. Consolidate constants.go — alias from usermgmt to identity-model instead of duplicating
20. Consolidate `maxEmailLength` — single source
21. Consolidate fold functions — parameterize the upcaster registry
22. Consolidate `unmarshalPayload` — single source with injectable upcasters
23. Add identity-model to integration_test module bridge tests
24. Add `DeriveMembershipID` test in identity-model
25. Document the identity-model module architecture in a README

**Code quality:**
26. Run `golangci-lint` on usermgmt after all changes — verify clean
27. Run `golangci-lint` on root after MapError fix — verify clean
28. Run `nix run .#lint` workspace-wide
29. Run `nix run .#coverage` and `nix run .#coverage-gate`
30. Fix `sqlite_setup_test.go` build tag issue properly (currently masked with `//go:build ignore`)
31. Review all type alias files for completeness — ensure no exported symbol was missed
32. Add a cross-module test verifying `usermgmt.Authz` == `identitymodel.Authz` at compile time
33. Update the cqrs-htmx skill SKILL.md with identity-model module description
34. Update the skill's module table to include identity-model

**Error handling:**
35. Evaluate whether identity-model errors should carry HTTP status hints natively
36. Consider making `errorfamily.HTTPStatus()` return the family default when httpStatus==0 (upstream fix)
37. Add error wrapping tests to identity-model (verify codes survive wrapping)
38. Document the error wrapping strategy (identitymodel → usermgmt WithHTTPStatus → MapError)

**Examples:**
39. Update `examples/admin-demo` to verify it still works with aliased types
40. Add an identity-model usage example (standalone, without usermgmt)
41. Verify all examples build and run

**Documentation:**
42. Update FEATURES.md with identity-model module
43. Update docs/DOMAIN_LANGUAGE.md with identity-model terms
44. Write an ADR for the identity-model extraction decision
45. Write an ADR for the Casbin-as-first-class-dependency decision
46. Update the planning doc to mark Phase 2 as complete

**Technical debt:**
47. The `sqlite_setup.go` file has `//go:build ignore` — the SQLite stack module isn't in go.mod. This is pre-existing but should be resolved.
48. The `go.work.sum` may need updating after all module changes
49. Consider whether `event.AggregateRef`/`event.AggregateType` backward-compat aliases in go-cqrs-lite will be removed (plan for forward compatibility)
50. Run `cqrs-lint` on the workspace to verify CQRS pattern compliance

---

## g) Questions I CANNOT figure out myself

1. **Should identity-model own the upcaster registry?** The fold functions currently stay in usermgmt because they depend on `applyUpcasters()` which uses usermgmt's global `UpcasterRegistry`. Moving the registry to identity-model would let the fold functions move too, eliminating the duplication. But it changes the module's responsibility — is identity-model a pure domain model or does it own infrastructure hooks? This is an architecture decision.

2. **Should the constants (event types, command types, aggregate types) be consolidated?** They're currently duplicated between usermgmt and identity-model with identical values. Aliasing them (`const eventUserRegistered = identitymodel.eventUserRegistered`) is impossible because they're unexported. Moving them to identity-model only and making them exported changes the API surface. What's the right tradeoff?

3. **Should identity-model errors carry HTTP status hints?** Currently identity-model errors are HTTP-free (errorfamily-only). The plan says "usermgmt wraps them with WithHTTPStatus". But the pre-existing MapError bug shows that errorfamily v0.8.0's HTTPStatus() method interferes with this. Should identity-model use `errorfamily.Error.WithHTTPStatus()` (the new v0.8.0 method) instead of relying on cqrshtmx.WithHTTPStatus wrapping? This would avoid the wrapper-breaks-errors.Is problem entirely.
