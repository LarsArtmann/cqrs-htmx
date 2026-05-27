# Bug Fix Session Status — 2026-05-27

**Date:** 2026-05-27_15-40 | **Commits:** 10 | **All tests pass (race-safe)** | **0 lint issues**

---

## Completed (10 commits)

| Commit | Fix | Impact |
|--------|-----|--------|
| `00b7187` | GetUser/UpdateRoles: don't wrap domain errors (ErrUserNotFound) as Transient | **Bug: was returning 500 instead of 404** |
| `994e9e0` | integration_test: go mod tidy | Build fix |
| `2d872bc` | Rate limiter: update lastUsed + push fresh heap entry on TTL re-check | **Bug: hot keys getting evicted prematurely** |
| `4f653b4` | CSRFTokenHXHeaders: use json.Marshal instead of string concat | **Bug: malformed JSON if token contains `"` or `\`** |
| `fe9f8a1` | Store: return defensive copies from FindByID/FindByEmail | **Correctness: callers can no longer mutate store internals** |
| `14b4478` | SessionMiddleware: log auth failures at Debug level | **Observability: auth errors no longer invisible** |
| `485dfd9` | Register: log rollback errors instead of `_ =` | **Correctness: cleanup failures now visible** |
| `2262d8e` | Extract validatePassword shared function (DRY) | Code quality |
| `f34b81e` | Authz.Apply: add-before-remove ordering | **Security: prevents permission gaps during role updates** |
| `ac80978` | WriteJSON: buffer before WriteHeader + rate limiter heap consistency | **Correctness: failed encode no longer commits success status** |

## Not Done (deferred — lower impact or need discussion)

| Item | Reason |
|------|--------|
| Replace Chain with justinas/alice | Low impact, adds dependency |
| Replace decodeFormValues with gorilla/schema | Medium impact, breaking change for consumers |
| Deduplicate handleCommandDispatch/handleQueryDispatch | Needs architecture discussion |
| Replace StatusRecorder with httpsnoop | Low impact, adds dependency |
| Hash session tokens before storage | Medium effort, breaking change |
| Add clock abstraction | 2h effort, test improvement only |

## Verification

- Root tests: ✅ (race-safe)
- usermgmt tests: ✅ (race-safe)
- integration_test: ✅ (race-safe)
- datastar-demo build: ✅
- Root lint: 0 issues
- usermgmt lint: 0 issues
