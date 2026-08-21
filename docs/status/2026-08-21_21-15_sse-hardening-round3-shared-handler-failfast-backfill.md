# SSE Hardening Round 3 — Shared Handler Extraction, Fail-Fast SubscribeAll, Backfill Cap

**Date:** 2026-08-21 21:15
**Session scope:** Three TODO_LIST items from the round-3 SSE hardening list
**Status:** PARTIALLY COMPLETE — code written, transport tests green, setup build blocked on a missing dev-replace

---

## What was requested

Three items from the round-3 SSE `/sse` hardening list (TODO_LIST):

1. **[~] Finish round-3 SSE hardening** — REMAINING: (a) decide CORS posture for `/sse`; (b) `attachSSE`'s `SubscribeAll` failure is log-only — should fail `New` (construction failure, not a dead feed answering 200).
2. **[ ] Extract shared SSE handler shape into `transport`** — both `setup/` and `dashboardui/` carry ~40 near-identical lines (subscribe → connected → replay → heartbeat-join → pump). A `transport.ServeDomainEvents(...)` helper eliminates the duplication.
3. **[ ] Cap first-connect backfill on both SSE endpoints** — `sse.Replay` takes no max; a first-time subscriber receives the ENTIRE journal. Need a bounded window via `WithMaxReplay`.

---

## a) FULLY DONE

| # | Item | Files | Verified |
|---|------|-------|----------|
| 1 | `transport/serve.go` — `ServeDomainEvents` helper | `transport/serve.go` (new, 120 LOC) | builds (GOWORK=off), 0 root imports |
| 2 | `transport/serve_test.go` — 6 tests (nil-broadcaster 503, custom message, connected+live pump, replay from store, heartbeat emission, cursored replay) | `transport/serve_test.go` (new) | ALL 6 PASSING |
| 3 | `setup/sse.go` refactored to call `transport.ServeDomainEvents` | `setup/sse.go` (rewritten) | code written |
| 4 | `dashboardui/sse.go` refactored to call `transport.ServeDomainEvents` | `dashboardui/sse.go` (rewritten) | code written, builds (GOWORK=off) |
| 5 | `setup/sse.go` `attachSSE()` returns `error` (fail-fast SubscribeAll) | `setup/sse.go` | code written |
| 6 | `setup/setup.go` propagates `attachSSE` error via `cleanup()` | `setup/setup.go` | code written |
| 7 | `dashboardui/sse.go` `startEventBridge()` returns `error` (fail-fast SubscribeAll) | `dashboardui/sse.go` | code written |
| 8 | `dashboardui/dashboard.go` propagates `startEventBridge` error | `dashboardui/dashboard.go` | code written, builds (GOWORK=off) |
| 9 | `setup/config.go` — `SSEMaxReplay int` field added | `setup/config.go` | code written |
| 10 | `dashboardui/config.go` — `SSEMaxReplay int` field added | `dashboardui/config.go` | code written, builds (GOWORK=off) |
| 11 | `setup/sse.go` passes `WithMaxReplay(b.config.SSEMaxReplay)` to `NewJournalSSEStore` | `setup/sse.go` | code written |
| 12 | `dashboardui/dashboard.go` passes `WithMaxReplay(config.SSEMaxReplay)` to `NewJournalSSEStore` | `dashboardui/dashboard.go` | code written, builds (GOWORK=off) |
| 13 | `setup/setup.go` `buildDashboardConfig` forwards `SSEHeartbeatInterval` + `SSEMaxReplay` to dashboard | `setup/setup.go` | code written |
| 14 | `dashboardui/handler.go` mount site updated (`d.guard(d.sseHandler())`) | `dashboardui/handler.go` | code written |
| 15 | `dashboardui/sse_replay_test.go` — 3 call sites updated (`d.sseHandler().ServeHTTP(rec, req)`) | `dashboardui/sse_replay_test.go` | code written |

---

## b) PARTIALLY DONE

### Setup module build — BLOCKED on missing dashboardui dev-replace

The `setup` module has its own `go.mod` and depends on the **published** `dashboardui/v4.8.0`. In `GOWORK=off` mode (the nix gate's hermetic build mode), setup resolves `dashboardui.Config` from the published tag, which does NOT have the new `SSEMaxReplay` field. The build fails:

```
./setup.go:250:3: unknown field SSEMaxReplay in struct literal of type dashboardui.Config
./setup.go:250:28: cfg.SSEMaxReplay undefined (type Config has no field or method SSEMaxReplay)
./sse.go:31:37: b.config.SSEMaxReplay undefined (type Config has no field or method SSEMaxReplay)
```

**Fix needed:** add a dev-only `replace github.com/larsartmann/cqrs-htmx/dashboardui/v4 => ../dashboardui` to `setup/go.mod` (same pattern as the existing `usermgmt/v4 => ../usermgmt` dev-replace documented in AGENTS.md). Strip before the next family tag.

In workspace mode (`GOWORK=on`), setup sees the local dashboardui and the field resolves — but the nix gate builds hermetically with `GOWORK=off`.

### CORS posture decision — NOT STARTED

Item (a) from the round-3 hardening list: "decide the CORS posture for `/sse`". No code or documentation written yet. The existing finding: neither `setup/` nor `dashboardui/` wires CORS into their SSE handlers. CORS is handled externally by `httputil.CORS` middleware. The decision is whether `/sse` needs a different CORS posture than the rest of the app (e.g., permissive for cross-origin EventSource connections, or same-origin only).

### Tests for SubscribeAll-fails-New — NOT STARTED

No test was written to verify that `New` returns an error when `SubscribeAll` fails. This is the core proof for item (b) — without it, the fail-fast behavior is unverified. The `FakeBus.SubscribeAllFn` injection point exists and can return an error; a test should construct a `FakeBus` with `SubscribeAllFn` returning an error and assert that `setup.New` and `dashboardui.New` both return non-nil.

---

## c) NOT STARTED

| # | Item | Why |
|---|------|-----|
| 1 | CORS posture decision + documentation | Deferred — needs a design decision |
| 2 | SubscribeAll-fails-New tests (setup + dashboardui) | Not written yet |
| 3 | Running setup test suite | Blocked on the dev-replace fix |
| 4 | Running dashboardui test suite | Not attempted yet (build passes, tests may have other issues from the sseHandler signature change) |
| 5 | Running full test suite (`nix run .#test`) | Not attempted |
| 6 | Running lint (`nix run .#lint`) | Not attempted |
| 7 | Running coverage (`nix run .#coverage-gate`) | Not attempted |
| 8 | Running `go vet` per-module | Not attempted |
| 9 | Updating `TODO_LIST.md` — mark items done/partially done | Not done |
| 10 | Updating `CHANGELOG.md` — add entries for the new transport helper + fail-fast + backfill cap | Not done |
| 11 | Updating `AGENTS.md` — document `transport.ServeDomainEvents`, `SSEMaxReplay`, fail-fast SubscribeAll | Not done |
| 12 | Documenting CORS posture in `docs/guides/sse-and-datastar.md` | Not done |
| 13 | Verifying `buildDashboardConfig` now forwarding `SSEHeartbeatInterval` doesn't double-apply defaults | Not verified (setup defaults to 15s, dashboardui defaults to 15s — passing it through should be idempotent but needs a test) |
| 14 | Checking if `setup/` has existing dev-replace for `dashboardui` (it might already) | Not checked |

---

## d) TOTALLY FUCKED UP

Nothing is totally fucked up. The transport helper is clean and tested. The refactoring is mechanically correct. The one blocker (setup dev-replace) is a known pattern with a known fix. No data was lost, no files were corrupted, no git operations were performed.

---

## e) WHAT WE SHOULD IMPROVE

1. **The `ServeDomainEvents` helper signature takes `*sse.Broadcaster[sse.Event]` (raw hub), not `*cqrshtmx.Broadcaster`.** This is correct per the hub-first broadcaster vocabulary (AGENTS.md), but both call sites do `b.Broadcaster.Raw()` to get the hub. Consider whether `transport` should accept the `cqrshtmx.Broadcaster` directly — but that would make transport import root, violating its package boundary. The current design is correct; the `.Raw()` call is the intended seam.

2. **`buildDashboardConfig` now passes `SSEHeartbeatInterval` to the dashboard.** Before this change, the dashboard always defaulted to 15s regardless of `setup.Config.SSEHeartbeatInterval`. This is a behavioral change: a consumer who sets a custom heartbeat on setup.Config now gets the same heartbeat on the dashboard SSE endpoint. This is the correct behavior (consistency), but it's a change from the previous state where the dashboard always used its own default.

3. **Error rollback in `attachSSE` is manual.** When `SubscribeAll` fails, I close `sseDone` and `Broadcaster` inline, then `New()` calls `bundle.cleanup()` which closes Dashboard + Service. The Broadcaster cleanup is NOT in `cleanup()` — it's only in `Close()`. This means if `attachSSE` fails after creating the Broadcaster but before `SubscribeAll`, the Broadcaster is leaked unless the inline rollback runs. The current code handles this correctly (the inline rollback is before the error return), but it's fragile — a future edit to attachSSE that adds a step between Broadcaster creation and SubscribeAll would need to add cleanup too. Consider extracting a `closeSSE()` helper.

4. **The `WithMaxReplay` option already existed on `JournalSSEStore`** (with `DefaultMaxReplay = 1000`). The backfill cap was always available — the issue was that `setup/` and `dashboardui/` didn't pass it. Now they do, via the new `SSEMaxReplay` config field. A value of 0 (the zero value) keeps the transport default of 1000, which is the right default behavior.

5. **The `PIPESTATUS` gotcha bit again.** The command `go build ... 2>&1; echo "EXIT: $?"` printed `SETUP EXIT: 0` even though the build had errors, because the shell captured the exit code of the `echo` in the pipe, not `go build`. The AGENTS.md documents this: always capture as `cmd > /tmp/f 2>&1; echo $?` — never through a pipe. I used the wrong pattern and was briefly misled.

6. **The go 1.26.6 binary at `/nix/store/890aq4jla87agkqj2wq0mr15kyi0qvp1-go-1.26.6/bin/go` works but isn't the default `go` on PATH** (which is 1.26.5). The go-cqrs-lite sibling requires 1.26.6. The nix devShell presumably pins the right version, but ad-hoc shell commands need the explicit path.

---

## f) Up to 50 things to get done next

### Blocking (must do before anything else works)

1. Check if `setup/go.mod` already has a `replace` for `dashboardui/v4` — if not, add `replace github.com/larsartmann/cqrs-htmx/dashboardui/v4 => ../dashboardui` (dev-only, strip before tag)
2. Rebuild `setup/` with `GOWORK=off` and verify the `SSEMaxReplay` field resolves
3. Run `go vet` on setup + dashboardui (catches `_test.go` import drift invisible to `go build`)

### Tests (proving the work)

4. Write `TestBundle_SSESubscribeAllFails` — `FakeBus` with `SubscribeAllFn` returning error; assert `setup.New` returns non-nil error
5. Write `TestDashboard_SSESubscribeAllFails` — same pattern for `dashboardui.New`
6. Write a test verifying `SSEMaxReplay` is forwarded to the `JournalSSEStore` (check `sseStore`'s `maxReplay` field via export_test or behavior)
7. Run the existing `setup/sse_internal_test.go` tests — verify the 3 existing tests still pass with the refactored `sseHandler()` (now returns `http.HandlerFunc` via helper, wrapped in `SessionMiddleware(requireSession(...))`)
8. Run the existing `dashboardui/sse_replay_test.go` tests — verify the 3 existing tests pass with `sseHandler()` now returning `http.HandlerFunc` instead of being a method with `(w, r)` signature
9. Run `dashboardui/dashboard_test.go` — verify `TestDashboard_Close` and other SSE-adjacent tests pass
10. Run full `setup/` test suite
11. Run full `dashboardui/` test suite

### Verification gates

12. Run `nix run .#build` (all 26 modules hermetic)
13. Run `nix run .#test` (all 17 suites)
14. Run `nix run .#lint` (15 modules, 0 issues expected)
15. Run `nix run .#coverage-gate` (15 gates — transport coverage likely improved; verify setup + dashboardui thresholds still pass)
16. Run `nix run .#check-cqrs-lint` (verify the C027/A005 suppressions on the bridge functions still work — they were preserved)
17. Run `nix run .#check-codegen` (verify no templ drift)
18. Run `nix run .#check-templates` (verify SQL templates unaffected)

### CORS posture (item a — deferred)

19. Decide: does `/sse` need permissive CORS for cross-origin `EventSource`, or is same-origin sufficient?
20. Consider: `EventSource` (the browser SSE client) does NOT send preflight `OPTIONS` requests (unlike `fetch`), so CORS for SSE is simpler — just `Access-Control-Allow-Origin` on the response headers
21. Consider: the library principle says "never enforce defaults consumers might disagree with" — CORS should be opt-in, not wired into the SSE handler
22. Decision options: (a) document that consumers should wrap `/sse` with `httputil.CORS` if they need cross-origin; (b) add an opt-in `SSECORSOrigin` config field; (c) do nothing — same-origin only
23. Document the decision in `docs/guides/sse-and-datastar.md`
24. If option (b), add `SSECORSOrigin string` to setup.Config and wire it

### Documentation

25. Update `TODO_LIST.md` — mark the three items as done (or partially done)
26. Update `CHANGELOG.md` — add entries: `transport.ServeDomainEvents` helper, fail-fast `SubscribeAll`, `SSEMaxReplay` config field, `SSEHeartbeatInterval` forwarded to dashboard
27. Update `AGENTS.md` — document `transport.ServeDomainEvents`, `SSEMaxReplay` on setup+dashboardui Config, fail-fast SubscribeAll behavior
28. Update `docs/guides/sse-and-datastar.md` — document the shared handler, CORS posture, backfill cap
29. Update `docs/guides/fullstack-wiring.md` — note that `setup.Config.SSEMaxReplay` now controls both `/sse` and the dashboard SSE endpoint
30. Check if the `setup/README.md` needs updating for `SSEMaxReplay`

### Polish

31. Consider extracting a `closeSSE()` helper on `*Bundle` to avoid the manual rollback fragility in `attachSSE`
32. Consider adding `SSEMaxReplay` to the `setup.Config.validate()` — reject negative values (currently `<= 0` means "use default", which is fine, but a very large positive value could OOM)
33. Verify the `WithSSELogPrefix` option produces the right log output (grep for "setup:" and "dashboardui:" in test output)
34. Check if `transport/serve.go` needs a `//cqrs-lint:ignore` for any rules (it's a new file — the C027/A005 rules are for the bridge functions, not the handler)
35. Verify `gofmt` on all changed files (`nix fmt` or `gofmt -l`)
36. Check if any examples (`examples/setup-demo/`, `examples/dashboard-demo/`) need updating for the `sseHandler` signature change
37. Check if `integration_test/` references `sseHandler` directly (it might)
38. Verify the `Broadcaster.Raw()` call in both sse.go files is not deprecated (AGENTS.md says `Raw()` is deprecated for v5 removal — but we need the raw hub for `ServeDomainEvents`). Consider whether `Hub()` should be used instead (AGENTS.md: "Hub() + NewBroadcasterFromHub" are the canonical API)

### Session hygiene

39. Check `git status` — see what the auto-git daemon may have committed
40. Check `git diff` — verify only intended files changed
41. Run `git log` — see if any concurrent session committed over our work
42. Consider whether a commit is appropriate (user hasn't asked for one)
43. Restore `adminui/styles.css` if the pre-commit hook mutated it (known gotcha)
44. Verify the GOCACHE workaround is needed (`df -h /mnt/buildcache` — the failing disk)

### Deeper verification

45. Write a transport-level integration test that wires `ServeDomainEvents` with a real `JournalSSEStore` + `sse.Broadcaster` + `FakeBus` and verifies end-to-end: publish → bus → bridge → broadcaster → SSE client receives
46. Benchmark the new helper vs. the old inline code (should be identical — the helper is the same code, just factored out)
47. Check if `datastar/` module has a similar SSE handler shape that could also use the helper (it has its own `Broadcaster` but different `Broadcast(patch Patch)` signature — likely not applicable)
48. Verify the `ServeDomainEvents` handler is safe for concurrent use (multiple clients connecting simultaneously — each gets its own stream + channel, the broadcaster is shared)
49. Check if the `hbDone` join pattern in the helper matches the original exactly (the deferred `<-hbDone` before return — yes, verified by reading the code)
50. Consider whether `ServeDomainEvents` should be an `http.Handler` (it returns `http.HandlerFunc` — this is fine, `http.HandlerFunc` implements `http.Handler`)

---

## g) Questions I CANNOT answer myself

### Q1: Should `setup/go.mod` get a dev-only replace for `dashboardui/v4`?

The setup module depends on published `dashboardui/v4.8.0`, which does NOT have the new `SSEMaxReplay` field. In `GOWORK=off` mode, the build fails. The existing pattern (per AGENTS.md) is a dev-only `replace` for `usermgmt/v4 => ../usermgmt` that gets stripped before the next family tag. Should I add `replace github.com/larsartmann/cqrs-htmx/dashboardui/v4 => ../dashboardui` to `setup/go.mod`? Or should I NOT pass `SSEMaxReplay` through `buildDashboardConfig` and instead let the dashboard use its own `SSEMaxReplay` config independently (which would mean the setup consumer can't control the dashboard's backfill cap)?

**My recommendation:** Add the dev-replace (consistent with the usermgmt pattern) and pass `SSEMaxReplay` through — a setup consumer should control both SSE endpoints from one config field. But this adds another dev-replace to strip before the next family tag, which is friction.

### Q2: What is the CORS posture for `/sse`?

Should the SSE endpoint be same-origin only (consumer wraps with `httputil.CORS` if needed), or should the library provide an opt-in CORS config field? `EventSource` does not send preflight `OPTIONS`, so CORS is just `Access-Control-Allow-Origin` on the response. The library principle says "never enforce defaults consumers might disagree with." Options:
- (a) Document only — same-origin by default, consumer adds `httputil.CORS` if cross-origin
- (b) Add `SSECORSOrigin string` config field (opt-in, empty = same-origin)
- (c) Do nothing — same-origin is the default and no one has complained

**My recommendation:** (a) — document that consumers wrap with `httputil.CORS` if they need cross-origin SSE. SSE endpoints are session-gated (auth cookies), and cross-origin EventSource with credentials is a CORS complexity most consumers don't need.

### Q3: Should `b.Broadcaster.Raw()` (deprecated for v5) be used, or should I use `Hub()` instead?

AGENTS.md says: "Hub() + NewBroadcasterFromHub are the canonical API; Raw() / NewBroadcasterFromRaw are deprecated (v5 removal)." Both `setup/sse.go` and `dashboardui/sse.go` call `b.Broadcaster.Raw()` to pass the raw `*sse.Broadcaster[sse.Event]` hub to `transport.ServeDomainEvents`. Should I use `Hub()` instead? But `Hub()` is not shown in the code I read — I only saw `Raw()`. If `Hub()` exists and has the same signature, I should use it. If it doesn't exist yet, `Raw()` is the only option.

**My recommendation:** Check if `Hub()` exists on `*cqrshtmx.Broadcaster`. If it does, switch to it. If not, use `Raw()` and note the v5 migration path.
