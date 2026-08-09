# WebSocket Removal — Status Report

**Date:** 2026-08-05 12:15
**Branch:** master (2 commits ahead of origin, uncommitted working tree clean)
**Task:** Remove all WebSocket code from cqrs-htmx root module; SSE only

---

## a) FULLY DONE

### Source-level WS removal (root module)

- **Deleted 4 dedicated WS source files** (via `git rm`):
  - `ws.go` (171 LOC) — `WSMessage`, `ParseWSMessage`, `ParseWSMessageInto`, `WSOOBHTML`, `parseWSHeaders`
  - `ws_broadcaster.go` (96 LOC) — `WSBroadcaster`, `NewWSBroadcaster`, `BroadcastHTML`, `BroadcastOnSuccessWS*`, `BroadcastOnErrorWS*`
  - `ws_dispatch.go` (289 LOC) — `WSCommandDecoder`, `WSQueryDecoder`, `DecodeWSJSON`, `DecodeWSJSONQuery`, `App.DispatchWSCommand`, `App.DispatchWSQuery`, `wsContext`, `wsCallContext`, `withDispatchTimeout`, `decodeWSMessage`, `isWSValueNil`, `wrapWSDispatchErr`
  - `ws_encoder.go` (96 LOC) — `WriteWSMessage`, `WriteWSMessageInto`
- **Deleted 5 dedicated WS test/example files** (via `git rm`):
  - `ws_test.go` (213 LOC)
  - `ws_dispatch_test.go` (276 LOC)
  - `ws_encoder_test.go` (321 LOC)
  - `ws_end_to_end_integration_test.go` (325 LOC)
  - `example_ws_test.go` (31 LOC)
- **Deleted embedded JS asset** `extensions/ws.min.js` (299 LOC) — the sole reason `extensions/` needed `*.min.js -text` gitattributes workaround

### Embedded extension surface

- Removed `HTMXExtWS` constant from `htmx_extensions.go`
- Removed `extWSJS` `//go:embed` directive
- Removed `HTMXExtWS` from the `htmxExtensions` registry and `htmxExtensionCDNURLs` map
- Removed WS comments from `HTMXExtensionHandler`/`HTMXExtensionsHandler` docstrings
- Updated `htmx_extensions_test.go` to drop the `ws` `DescribeTable` entries, the `POST /ext/ws.js` 405 test, the WS bundle ETag expectation, the WS CDN URL entry, and the `ws` name in `HTMXExtensionNames`/`HTMXExtensionVersion`

### ack.go surgical removal

- Removed `BroadcastOnAckWS()` and `BroadcastOnAckWSFunc()` methods on `WSBroadcaster`
- Removed `ackToWSMessage` helper
- Comment updates: "SSE/WS" → "SSE" in `defaultAckEventName` docstring and "when parsing client-sent acks (e.g. over WebSocket)" → "when parsing client-sent acks"

### Shared test cleanup (surgical)

- `ack_test.go` — removed `TestBroadcastOnAckWS_Success` and `TestBroadcastOnAckWS_NoCommandID`
- `bdd_realtime_test.go` — removed the "HTMX WebSocket form submissions into typed Go structs" `Describe` block (3 `It` blocks) and the `&` closing of the outer `Describe`; renamed outer from "Realtime (SSE & WebSocket)" → "Realtime (SSE)"
- `integration_transport_test.go` — removed both `Describe` blocks (`WS dispatch round-trip...` with 4 `It` blocks + `WS message encode/decode round-trip...` with 1 `It` block); dropped unused `query` import
- `benchmark_server_test.go` — removed `BenchmarkWSBroadcasterBroadcastStress` and `BenchmarkWSBroadcasterConcurrentSubscribe`; dropped unused `strconv` + `sync` imports + the `var _ = sync.WaitGroup{}` placeholder
- `example_sse_test.go` — removed `ExampleWSBroadcaster` and `ExampleWriteWSMessage`
- `fuzz_test.go` — removed `FuzzWriteWSMessage`; dropped unused `bytes` import
- `partial_test.go` — removed `WSOOBHTML (delegates to OOBHTML)` `Describe` block

### Source comment cleanup

- `doc.go` — removed "WebSocket variant for real-time push" code block, removed the entire `# WebSocket` section (encoder/broadcaster/dispatch bridge/AfterDispatch examples); updated 3 inline `SSE or WS`/`HTTP/SSE/WS`/`SSE and WS` references
- `errors.go` — "SSE and WebSocket error broadcasts" → "SSE error broadcasts"; "HTTP, SSE, and WS transports" → "HTTP and SSE transports"
- `structured_error.go` — "SSE, and WebSocket transports" → "SSE, and HTTP transports"; "SSE event data or WebSocket message payloads" → "SSE event data or HTTP response bodies"; "SSE/WS/HTTP clients" → "SSE and HTTP clients"; "SSEEvent.Data or WS message payloads" → "SSEEvent.Data or HTTP response bodies"
- `logging.go` — "preserving SSE, WebSocket, and HTTP/2 capabilities" → "preserving SSE and HTTP/2 capabilities"
- `partial.go` — "Works for both HTTP responses and WebSocket messages" → "Works for HTTP responses and SSE event data"
- `responsewriter.go` — "preserve SSE, WebSocket, and HTTP/2 capabilities" → "preserve SSE and HTTP/2 capabilities"; "Hijack delegates to the underlying Hijacker so WebSocket upgrades work through wrappers" → "Hijack delegates to the underlying Hijacker. Used by long-lived connection upgrades"

### Verification

- `GOEXPERIMENT=jsonv2 go build ./...` — **PASS** (clean, root + `openapi/` sub-package)
- `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` — **PASS** in 4.04s (root) + 1.01s (openapi)
- `GOEXPERIMENT=jsonv2 go test ./... -cover` — **PASS** root 93.1% (gate 90%), openapi 99.0%
- `golangci-lint run` — **37 issues** vs pre-change baseline of **38 issues** (I introduced 0 net lint regressions; the 1-issue drop is `gci` cleanups from `gofmt -w`)
- Auto-git daemon captured my file-deletion work in two clean commits: `9869c7f4 chore(ws): remove WebSocket module and untrack binary assets` and `557a6ccd refactor(test): remove orphaned WebSocket test coverage, benchmarks, and examples`

### Documentation

- **ADR-0046** written: `docs/adr/0046-drop-websocket-sse-only.md` (Accepted). Documents context, decision, positive/negative consequences, and migration recipe for each removed symbol
- ADR INDEX updated (pending — see PARTIALLY DONE)

---

## b) PARTIALLY DONE

> **Resolution (2026-08-05 docs-health pass):** ALL of the items below were completed by the phase-7 follow-up session (`2026-08-05_12-43_ws-removal-phase7-docs-status.md`). They are struck through below.

- ~~**ADR INDEX** (`docs/adr/INDEX.md`)~~ done — ADR-0046 row added; ADR-0004 + ADR-0010 marked Superseded by ADR 0046 in the INDEX. (The ADR _bodies_ still need inline supersession notes → TODO_LIST P1.)
- ~~**`docs/guides/realtime.md`**~~ done — the file never existed as a guide; the WS content was stripped from SKILL.md + doc.go + structured_error.go + logging.go + partial.go + responsewriter.go instead. The broken `references/realtime.md` SKILL.md reference is tracked in TODO_LIST P1.
- ~~**`.agents/skills/cqrs-htmx/SKILL.md`**~~ done — WS removed from YAML description, module matrix, decision tree, realtime section, embedded-extension list; HTMXExtWS removed.
- ~~**`AGENTS.md`**~~ done — new gotcha "WebSocket transport removed in v5 (ADR 0046)" listing all 14 removed symbols + migration recipe.
- ~~**`README.md`**~~ done — removed ~108 lines of WS content (tagline, WebSocket Helpers/API sections, tree entry, extension row); replaced with a one-line SSE-only callout.
- ~~**`FEATURES.md`**~~ done — "Real-Time — WebSocket" section replaced with a 9-row REMOVED table; updated metrics + lint notes (2026-08-05 pass).
- ~~**`CHANGELOG.md`**~~ done — `### Removed` section added to `[Unreleased]` with full per-symbol migration recipe + ADR supersession chain.
- ~~**`TODO_LIST.md`/`ROADMAP.md`**~~ done — both rewritten 2026-08-05 (TODO_LIST: WS items routed; ROADMAP: WS-removal subsection marked DONE-pending-tag).

---

## c) NOT STARTED

- Update ADR-0004 and ADR-0010 inline text to reference ADR-0046 (link cross-reference)
- Update `.golangci.yml` if any lint exclusion was WS-specific (audit pending)
- Verify no submodule (`adminui`, `loginpage`, `dashboardui`, `datastar`, `usermgmt`, `totp`, `webauthn`, `oauth2`) imports any removed symbol — initial grep showed 0 WS references outside the root module, but full verification not re-run after the removal
- Verify no `examples/*` directory references removed WS symbols — initial grep was clean but should be re-confirmed
- Verify the nix hermetic build (`nix run .#build`, `nix run .#test`, `nix run .#lint`) all still pass — only ran `go build`/`go test`/`go vet`-equivalent directly; nix scripts add templ generation, coverage gate, and check-templates steps
- Update `flake.nix` if any module list hardcodes WS-named files (unlikely — module lists use directory names, not file names)
- Update CI workflow (`.github/workflows/ci.yml`) — unlikely affected
- Run `nix fmt` to confirm `gofmt`-style formatting on remaining edits

---

## d) TOTALLY FUCKED UP

Nothing is broken. Two minor process issues:

1. **LSP cache staleness repeatedly misled me during the session.** After deleting files and editing others, gopls/golangci-lint-ls reported diagnostics with line numbers from the old (pre-deletion) state. I had to `lsp_restart` multiple times and cross-check with `grep`/`wc -l` to know what was real vs stale. Net wasted ~5 minutes chasing phantom errors.
2. **The auto-git daemon committed my intermediate work-in-progress in two commits without me noticing.** I was planning to do a single coherent commit at the end of phase 7 with full context, but the daemon split the work into "source removal" (`9869c7f4`) and "test removal" (`557a6ccd`). The split is logically reasonable (source vs test) but I didn't review the commit messages before they landed. Both commits look correct from the diff stats, but I didn't author them — they're the daemon's interpretation of my `git rm` and edits. This is documented behavior in `AGENTS.md` ("auto-git commit daemon runs continuously") so it's a known constraint, not a bug, but worth flagging.
3. **Sed regex miss:** my first attempt to delete the WS section from `doc.go` via `sed '/^WebSocket variant.../,/^... Submodule: usermgmt/d'` did NOT match because the pattern spanned doc-section boundaries. I had to use a precise line-range `sed -i '175,206d'` instead. Mild waste, no correctness impact.

---

## e) WHAT WE SHOULD IMPROVE

### Process

- **Explicit LSP-restart discipline.** After batch edits to a Go workspace, the first instinct should be `lsp_restart` + `go build ./...` rather than reading LSP diagnostics that may be stale. Add this to project workflow notes.
- **Grep-before-edit across the module graph.** When removing a public symbol, grep should include `examples/`, `adminui/`, `dashboardui/`, etc. — not just root. I did the initial grep but didn't re-verify after each deletion phase.
- **Commit-on-finish preference over auto-commit-daemon work-in-progress.** The daemon's commits were fine but unlabeled with the ADR context. A single, well-crafted final commit referencing ADR-0046 would carry more meaning for future archaeology. Consider whether the daemon's commit cadence should be reduced for in-progress refactors (maybe `--no-verify` workflow for refactors that touch >10 files).

### Code quality observations I made along the way

- **`ack.go` `defaultAckEventName`** was documented as "SSE/WS event name" — the WS path was already unreachable from external callers since `ackToWSMessage` returned a plain string (not an `sse.Event`), and the matching `BroadcastOnAckWS` was the only consumer. The docstring lie was a small split brain; fixed.
- **`responsewriter.go` `Hijack`** method docstring claimed "WebSocket upgrades work through wrappers" — but `Hijack` is also used by SSE long-lived responses and by any HTTP upgrade (WebDAV, raw TCP over HTTP, etc.). The corrected docstring ("Used by long-lived connection upgrades") is more honest about its general purpose.
- **`delegatingWriter`** was named generically but its purpose expanded to cover SSE and HTTP/2 alongside the original WS intent. The new docstring ("preserve SSE and HTTP/2 capabilities") is more accurate but the type name itself is still generic — not a problem, just an observation.

### Architecture observations

- The WS surface was large enough (14 symbols, ~1200 LOC, 1 embedded asset) that removing it is a meaningful simplification. The trade-off is the breaking-change blast radius for any consumer who used it. **No usage audit was done** before removal — that should happen before a release.
- ADR-0004 and ADR-0010 are now superseded but the codebase still has their inline doc references intact. The INDEX update should also mark these as Superseded-by-0046 so future readers trace the lineage correctly.
- The skill (`SKILL.md`) is the user-facing migration surface. Until it's updated, downstream consumers will hit compile errors with no in-repo migration guide.

---

## f) Up to 50 things to get done next

> **Resolution (2026-08-05 docs-health pass):** these items were re-enumerated and resolved inline in the follow-up report `2026-08-05_12-43_ws-removal-phase7-docs-status.md` §f. The canonical routing is now in **TODO_LIST.md** (P0/P1) and **ROADMAP.md** (Open Questions + v5 retirement). The docs items (#6–#18) are done; the critical-path blockers (#1 publish httputil v0.9.0, #2–#5 tag/release) remain open and gated.

1. Add ADR-0046 row to `docs/adr/INDEX.md`
2. Mark ADR-0004 status as "Superseded by ADR 0046" in INDEX
3. Mark ADR-0010 status as "Superseded by ADR 0046" in INDEX
4. Update ADR-0004 inline text to reference ADR-0046 in its supersession note
5. Update ADR-0010 inline text to reference ADR-0046 in its supersession note
6. Rewrite `docs/guides/realtime.md` — remove WS sections, add migration table (WS symbol → SSE/HTTP equivalent)
7. Update `.agents/skills/cqrs-htmx/SKILL.md` description — remove WebSocket from trigger phrase
8. Update SKILL.md "Realtime (SSE / WebSocket)" section header to "Realtime (SSE)"
9. Update SKILL.md "Where to look" section to describe SSE-only realtime
10. Update SKILL.md to remove all WS symbol entries from the cheat sheet
11. Add "Removed in v5" migration recipe to SKILL.md showing how to migrate each of the 14 removed symbols
12. Update `AGENTS.md` to note the WS removal in the "Key Patterns" or "Gotchas" section
13. Update `AGENTS.md` "Architecture" section to clarify SSE-only transport model
14. Update `README.md` — remove any WS mentions from the sales page
15. Update `FEATURES.md` — flip WS feature status to "Removed" with migration pointer
16. Add prominent "Removed" section to `CHANGELOG.md` for the next version (v5.0.0 or whichever releases this lands in)
17. Audit `TODO_LIST.md` for stale `[~]` items referencing WS symbols (none expected, but verify)
18. Audit `ROADMAP.md` to ensure no "WS improvements" or "WS parity" backlog items remain
19. Run `nix run .#build` to verify the nix hermetic build (currently only `go build ./...` was tested)
20. Run `nix run .#test` to verify nix-driven test execution (currently only `go test` was tested)
21. Run `nix run .#lint` to verify nix lint pipeline
22. Run `nix run .#coverage` to verify coverage gate still passes via the nix pipeline
23. Run `nix fmt` to confirm formatting consistency
24. Verify `nix run .#check-templates` still passes (no template changes expected)
25. Verify `nix run .#check-codegen` still passes (no `_templ.go` changes)
26. Verify no submodule imports removed symbols (re-grep across all modules)
27. Verify no `examples/*` directory imports removed symbols
28. Audit `.golangci.yml` for any WS-specific lint exclusion that should be removed
29. Audit `.cqrs-lint.json` (the cqrs-lint preset) for any WS-related suppression that's now stale
30. Audit the changelog precedent (the SSE re-export deprecation pattern) for whether WS removal should also use a single `Deprecated:` marker phase before final removal — current implementation skipped the deprecation phase entirely
31. Run `cqrs-lint version` to verify it's still 4.3.0
32. Run `cqrs-lint ./...` to verify the lint preset still passes after the removal
33. Verify `gofmt -l .` shows zero files needing formatting
34. Verify `go vet ./...` is clean (was not explicitly run during this session)
35. Run the example apps (`examples/basic`, `examples/admin-demo`, `examples/catalog-demo`, `examples/middleware-demo`, `examples/observability-demo`) at least at the build level to ensure none of them imported WS symbols
36. Audit `docs/snapshot-testing-options.md` for WS references
37. Audit `docs/runbooks/cross-repo-consolidation.md` for WS references
38. Verify no template files in `usermgmt/` (`sqlite_setup.go`, `postgres_setup.go`, etc.) referenced WS — they don't, but verify
39. Add a `docs/migrations/v5-websocket-removal.md` guide showing the per-symbol migration recipe
40. Cross-check the e2e tests (`e2e/server`) don't reference WS — initial grep showed `e2e/node_modules/playwright/lib/agents/copilot-setup-steps.yml` (not our code) but should re-verify our e2e code
41. Verify the SSE re-export deprecation pattern from AGENTS.md gotcha is still applicable post-removal (no, it's now irrelevant since the WS removal is unconditional)
42. Update `.github/workflows/ci.yml` if any path or step referenced WS file paths
43. Update the `flake.nix` `coverage-gate` if any module list changed (root module list is unchanged — covered)
44. Update the `flake.nix` `lint` script's module list (no change expected)
45. Run `find . -name '*.go' | xargs grep -l "WSBroadcaster\|WSMessage\|HTMXExtWS\|DispatchWSCommand\|ParseWSMessage\|WriteWSMessage\|BroadcastOnAckWS"` to confirm zero references remain anywhere
46. Run `find . -name '*.md' | xargs grep -l "WebSocket"` to surface any remaining doc references that need updating
47. Verify that `.agents/skills/cqrs-htmx/SKILL.md` description field (the YAML frontmatter) no longer mentions WebSocket
48. Update `docs/research/2026-06-18_web-communication-matrix.html` if it contains an outdated transport-parity matrix
49. Add a new status report under `docs/status/` documenting the WS removal (or merge this one into a future weekly review)
50. Verify the auto-git daemon commits (`9869c7f4` + `557a6ccd`) are well-formed and force a follow-up commit referencing ADR-0046 once the doc updates in tasks 1-30 land — single coherent commit > two work-in-progress commits

---

## g) Up to 3 questions I CANNOT figure out myself

**Q1: Release cadence.** This is a breaking change (14 public symbols removed). What is the next cqrs-htmx release version? Current HEAD shows the version isn't visible in the root module files I edited — I see `.cqrs-lint.json` references `4.3.0` but the actual `cqrs-htmx/v4` module could be on v4.x with the next release being v5.0.0, or v4.x.y. The CHANGELOG entry depends on this and I cannot determine it without scanning version tags or release notes I don't have direct access to.

**Q2: WS symbol deprecation phase.** The SSE re-export removal followed a deprecation-then-removal pattern (mark `Deprecated:` in v4.x, remove in v5). I removed WS unconditionally without a deprecation phase. Was that intentional from your perspective ("do it now in one breaking bump"), or should I retroactively add `Deprecated:` markers to the 14 symbols in a v4.x backport and remove them in v5? The choice affects whether downstream consumers can use v4.5 with deprecation warnings before being forced to migrate.

**Q3: Submodule audit depth.** My initial grep showed zero WS references in `adminui/`, `loginpage/`, `dashboardui/`, `datastar/`, `usermgmt/`, `usermgmt/totp`, `usermgmt/webauthn`, `usermgmt/oauth2`, or `examples/*`. Should I treat that as a verified clean signal (and skip task #26), or do you want me to do a deep audit (parse Go module imports, check transitive usage) before declaring the removal complete? The grep is structural but might miss symbol references through `cqrshtmx.*` indirection in tests or comments.
