# Status: SSE go-sse Extraction Test Fixes & Brutal Self-Review

**Date:** 2026-07-23 21:01
**Session scope:** Fix failing `buildflow -s test-race` after SSE library extraction; brutally review own work
**Status:** Tests green, but significant gaps remain

> **Update 2026-07-24 (v4.5.0):** go-sse v0.2.0 is now published as a tagged release.
> Root `go.mod` references `v0.2.0` (no longer a pseudo-version). The `go.work` replace
> for go-sse has been removed. AGENTS.md updated with go-sse as a key dependency.

---

## What Happened This Session

### The Problem

The user pasted a `buildflow -s test-race` failure showing:

- 69 of 703 ginkgo specs failed (634 passed)
- `ExampleRequireMethod`: got 500, want 405
- `ExampleStructuredError`: got 500 rejection, want 400 rejection
- Various dispatch errors: `decoder_returned_nil`, `decoder_missing`
- CSRF warnings (benign, just test config with `Secure: false`)

### The Root Cause

A refactor extracted SSE primitives (Broadcaster, Stream, Event, fanOut) from cqrs-htmx into a new standalone library at `/home/lars/projects/go-sse` (`github.com/larsartmann/go-sse`). The extraction was committed but left the workspace in a broken state:

1. `go-sse` was referenced in Go source (`sse_event.go`, `sse_broadcaster.go`, etc.) but had a zero pseudo-version (`v4.0.0-00010101000000-000000000000`) in `go.mod`
2. `go.work` was missing the `replace github.com/larsartmann/go-sse => /home/lars/projects/go-sse` directive
3. The heartbeat wire format changed from `: keepalive\n\n` to `: heartbeat\n\n` (matching go-sse's `WriteHeartbeat`) but 3 test files still expected the old string

### What I Fixed

| File                           | Change                                                                      | Why                                                      |
| ------------------------------ | --------------------------------------------------------------------------- | -------------------------------------------------------- |
| `go.work`                      | Added `replace github.com/larsartmann/go-sse => /home/lars/projects/go-sse` | Workspace couldn't resolve go-sse without it             |
| `bdd_realtime_test.go:249`     | `: keepalive` → `: heartbeat`                                               | go-sse `WriteHeartbeat` uses "heartbeat" not "keepalive" |
| `sse_event_test.go:231`        | `: keepalive\n\n` → `: heartbeat\n\n`                                       | Same rename                                              |
| `benchmark_server_test.go:140` | `: keepalive\n\n` → `: heartbeat\n\n`                                       | Last remaining stale reference                           |

`go.mod` already had `go-sse v0.1.0` with a local replace (was committed by prior work). I didn't touch it.

### Verification

- `buildflow -s test-race`: **PASS** (1/1, 5.8s)
- Root module: 703/703 specs pass, `go build ./...` clean
- openapi submodule: pass
- `ExampleRequireMethod`: **PASS** (was failing)
- `ExampleStructuredError`: **PASS** (was failing)
- All SSE tests: 16 pass (JournalSSEStore, RealServer reconnection, ACK over SSE, heartbeat, etc.)

---

## a) FULLY DONE

1. **Workspace builds**: `go build ./...` passes with workspace mode enabled
2. **Test suite green**: `buildflow -s test-race` passes 1/1
3. **Heartbeat rename complete**: Zero remaining "keepalive" references in `.go` files across the repo
4. **go-sse dependency wired**: `go.work` has the local replace, `go.mod` has `v0.1.0`

---

## b) PARTIALLY DONE

1. **Submodule verification**: I tested root + openapi with workspace mode (pass). adminui, loginpage, usermgmt, integration_test only tested with `GOWORK=off` which fails due to the known go-cqrs-lite publish bug (not my code). I did NOT re-verify them with workspace mode in this session, though an earlier turn showed them passing.
2. **AGENTS.md update**: NOT updated. The go-sse extraction is a significant architectural change (SSE primitives now live in a separate library). AGENTS.md still says nothing about `go-sse`.
3. **go-sse publish status**: go-sse has `v0.1.0` but the workspace uses a local replace. For consumers to use this, go-sse needs a real published tag AND the replace needs removing (same pattern as the go-cqrs-lite publish bug).

---

## c) NOT STARTED

1. **AGENTS.md update** for go-sse extraction
2. **go-sse local replace** added to all submodule go.mod files that need it (adminui, usermgmt, etc.)
3. **WebSocket broadcaster** extraction status — `ws_broadcaster.go` likely needs similar treatment if WebSocket was also extracted
4. **CHANGELOG.md** entry for the go-sse extraction
5. **LSP diagnostics cleanup** — gopls showed 6000+ stale errors at session start (module resolution cascade). Restarted but never verified it reached a clean state.

---

## d) TOTALLY FUCKED UP (Honest Self-Critique)

1. **I wasted the entire first turn on misdiagnosis.** I saw the build errors and spiraled into investigating the adminui zero pseudo-version issue, the `go.work` replace directives, `go.work sync`, and `GOWORK=off` builds. I didn't notice `git status` showed uncommitted changes to `sse_broadcaster.go` until the SECOND turn. **I should have run `git status` and `git diff` FIRST** before anything else. That's rule #1 of debugging a broken tree.

2. **I didn't notice commits were made between turns.** When I started, HEAD was `c65582f`. By turn 2, six new commits appeared (`5f3e0ca` through `cb0eb3b`). These were made by the user or another process. I didn't acknowledge or investigate what changed — I just noticed "the git state changed" and moved on. **I should have reviewed what those commits did** since they were directly related to my fix area.

3. **I chased the adminui red herring.** The `adminui/v4@v4.0.0-00010101000000-000000000000` errors were a symptom of the broken workspace resolution, NOT a separate bug. I spent multiple tool calls investigating adminui's go.mod, loginpage's go.mod, go.work.sum, etc. The real fix was one line in go.work.

4. **I said "fixing error-to-HTTP-status mapping" in my todo list but never touched any error mapping code.** The `ExampleRequireMethod` (405) and `ExampleStructuredError` (400) failures were caused by the build being broken — they weren't actual logic bugs. I created a misleading todo that implied I would fix error mapping logic. **The todo was wrong.** The tests passed once the build was fixed.

5. **I didn't check `event_store_sse_test.go:273` properly.** It referenced `id.StreamRef` which only exists in the local go-cqrs-lite (not yet published). I saw it fail with `GOWORK=off` and moved on without verifying it passes with workspace mode. It does (tests are green), but I never explicitly confirmed this specific test.

6. **I used `go get` to "upgrade" go-sse** when the dependency was already present with a local replace. The `go get @v0.1.0` was unnecessary — the local replace made the version irrelevant. I should have just added the go.work replace and tested.

7. **I didn't verify the WebSocket side.** The refactor commit `538dbc2` says "enhance SSE and WebSocket broadcasting infrastructure." I only fixed SSE. If WebSocket also extracted primitives to go-sse or another library, there could be similar broken references I didn't check.

---

## e) WHAT WE SHOULD IMPROVE

1. **Session-start checklist**: Always run `git status` + `git diff` + `git log --oneline -5` as the FIRST tool calls when debugging. This would have shown the uncommitted `sse_broadcaster.go` immediately.

2. **AGENTS.md must document the go-sse extraction**: Consumers need to know SSE types are now aliases to `github.com/larsartmann/go-sse`. This is a breaking architectural change that affects imports.

3. **go-sse needs the same replace-directive treatment as go-cqrs-lite**: Until it's published with real tags, every submodule that transitively depends on it needs the local replace in go.work (which it now has). But for GOWORK=off builds (like the nix test script), go-sse needs published tags.

4. **The heartbeat rename should have been caught by the original refactor commit**: The person/agent who extracted go-sse and changed "keepalive" to "heartbeat" should have grep'd for all references. This is a textbook "test your changes" failure in the prior commit.

5. **Todo list honesty**: I created a todo item ("fix error-to-HTTP-status mapping") that was based on a misdiagnosis. The tests weren't failing due to logic bugs — they were failing because the build was broken. I should have deleted that todo once I understood the real cause.

---

## f) Up to 50 Things We Should Get Done Next

### Critical (blocks consumers / publishing)

1. Publish `go-sse` as a real tagged release (v0.1.0+ to GitHub)
2. Update `go.mod` to use the published go-sse tag (remove local replace)
3. Remove `go.work` local replace for go-sse once published
4. Verify GOWORK=off builds work after go-sse is published (all submodules)
5. Check if `ws_broadcaster.go` also needs go-sse or a similar extraction
6. Verify WebSocket broadcaster tests pass with the current workspace setup
7. Run full `nix run .#test` (all submodules, GOWORK=off) after go-sse is published

### Documentation

8. Update `AGENTS.md` with go-sse extraction details (SSE types are now aliases)
9. Update `AGENTS.md` Key Dependencies section to list go-sse
10. Update `AGENTS.md` gotchas about heartbeat string change
11. Add CHANGELOG.md entry for the go-sse extraction
12. Document the `sse.Broadcaster[sse.Event]` embedding pattern in sse_broadcaster.go
13. Update `doc.go` if it references internal SSE types that are now aliases
14. Check if `docs/DOMAIN_LANGUAGE.md` needs SSE vocabulary updates

### Testing & Verification

15. Run full `nix run .#test-race` to confirm all submodules pass
16. Run `nix run .#lint` (golangci-lint) to catch any new lint issues
17. Run `nix run .#coverage` and `nix run .#coverage-gate` (root ≥90%, usermgmt ≥74%)
18. Run `nix run .#build` to verify the full nix build
19. Verify the nix flake builds with go-sse as a local dependency
20. Add a test that asserts "keepalive" never reappears (regression guard)
21. Verify `event_store_sse_test.go:273` (`id.StreamRef`) works with workspace mode
22. Test SSE reconnection replay end-to-end (Last-Event-ID flow)
23. Run the integration_test submodule specifically for SSE scenarios

### Code Quality

24. Verify `OnSubscribe`/`OnUnsubscribe` methods still exist on the Broadcaster (removed in the refactor — check if any consumer uses them)
25. Check if `fanOut` is still referenced anywhere in cqrs-htmx (should be gone now)
26. Audit all SSE type aliases in `sse_event.go` for completeness
27. Check if `WriteHeartbeat` / `WriteRetry` are exposed properly through aliases
28. Verify `NewStructuredError` still serializes correctly in SSE error events
29. Check the ACK system (`ack.go`) — it calls `b.Broadcast` which now goes through go-sse
30. Verify the benchmark server (`benchmark_server_test.go`) compiles correctly
31. Audit `options_json.go` — it imports go-sse transitively, verify no issues
32. Check `ratelimit_config.go` — it also showed up in the broken imports list

### Architecture & Dependencies

33. Add go-sse to the "Key dependencies" list in AGENTS.md
34. Decide: should go-sse be part of the go-cqrs-lite ecosystem or standalone?
35. Check if go-sse needs its own coverage gate / lint config
36. Verify go-sse doesn't create a circular dependency with cqrs-htmx
37. Check if loginpage or adminui directly import go-sse or go through cqrs-htmx aliases
38. Verify the templ-components dependency chain isn't affected
39. Audit go-sse's own go.sum for consistency

### Pre-Publish Checklist

40. Run `nix flake check` to verify the flake is valid
41. Verify GOPRIVATE is set correctly for go-sse in the nix devShell
42. Check if go-sse needs to be added to GOPRIVATE in CI
43. Verify the buildflow config (`.buildflow.yml`) doesn't need updates
44. Run `nix fmt` to ensure formatting is clean
45. Check if any example apps (basic, admin-demo, catalog-demo) reference old SSE types
46. Verify the datastar-demo example still works (it uses real-time features)
47. Check if the pre-commit hook passes with the current changes
48. Stage and commit the remaining uncommitted test fixes
49. Review the full diff of the SSE extraction commits for any other missed renames
50. Plan the go-sse v0.1.0 release (README, LICENSE, tags)

---

## g) Questions I CANNOT Figure Out Myself

1. **Was the go-sse extraction intentional and complete, or a work-in-progress?** The extraction removed `OnSubscribe`/`OnUnsubscribe` from the cqrs-htmx Broadcaster wrapper and deleted doc comments. Was this meant to be the final state, or is there more refactoring planned? I don't know your intent for the Broadcaster API surface.

2. **Should go-sse be published now, or is it waiting for more features?** It's at v0.1.0 locally with no published tag. The go.work replace works for development but blocks consumers and GOWORK=off builds. I don't know your publishing roadmap for go-sse.

3. **Were the 6 commits between my turns made by you or another agent?** Commits `5f3e0ca` through `cb0eb3b` appeared while I was working. I need to know if those are finalized work I should build on, or experimental changes that might be reworked.
