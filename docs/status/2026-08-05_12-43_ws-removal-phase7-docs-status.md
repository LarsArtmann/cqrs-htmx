# Status Report — WebSocket Removal Phase 7 (Documentation)

> **Session:** docs-phase-7-only
> **Date:** 2026-08-05 12:43 CEST
> **Scope:** Documentation updates only — picked up where prior session left off after phases 1-6 (source/test deletions, comment cleanup, build/test/lint verification) were already complete.
> **Outcome:** Phase 7 complete. WebSocket removal documentation is now consistent across all consumer-facing surfaces. No code changes in this session.

---

## a) FULLY DONE ✅

### Doc updates shipped (7 files modified)

| File                                | Lines    | Change                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| ----------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `docs/adr/INDEX.md`                 | +5/−2    | Added ADR-0046 row (Accepted); marked ADR-0004 + ADR-0010 as `Superseded by ADR 0046`                                                                                                                                                                                                                                                                                                                                                                                  |
| `.agents/skills/cqrs-htmx/SKILL.md` | +13/−12  | Removed WS from YAML description, module matrix row, decision-tree prompt, "Realtime (SSE / WebSocket)" → "Realtime (SSE)" section, embedded HTMX extensions list (3→2: SSE + idiomorph only), removed WS mirroring paragraph, removed `HTMXExtWS` from example bundle, `references/realtime.md` summary                                                                                                                                                               |
| `AGENTS.md`                         | +3/−2    | Root module description: "HTMX/SSE/WS helpers" → "HTMX/SSE helpers" + bolded "**WebSocket transport removed in v5** — SSE only (see ADR 0046)"; new gotcha entry "WebSocket transport removed in v5 (ADR 0046)" listing all 14 removed symbols + migration recipe + file deletions + supersession chain (ADR-0046 supersedes ADR-0004 and ADR-0010); SSE re-export note: removed stale `/WSBroadcaster` reference (was a self-contradiction with the new removal note) |
| `README.md`                         | +11/−108 | Removed intro tagline WS mention, "WebSocket dispatch" shape line, entire "WebSocket Helpers" section (~50 lines), entire "WebSocket API" table (~20 lines), `ws.go` tree entry, "WS (2.0.4)" row from Embedded HTMX Extensions table; replaced with a one-line SSE-only callout pointing to ADR 0046                                                                                                                                                                  |
| `FEATURES.md`                       | +30/−36  | Replaced "Real-Time — WebSocket" section (header + 7-row table) with "Real-Time — WebSocket (removed in v5)" (header + 9-row table — added `HTMXExtWS` and `WS Dispatch` rows; all rows 🔴 `REMOVED` with migration notes); updated summary table row, Embedded HTMX Extensions description, StructuredError note, "Not Planned" entry, removed "SSE/WS" from split rationale                                                                                          |
| `CHANGELOG.md`                      | +4       | Added `### Removed` section to `[Unreleased]` with full migration recipe (symbol-by-symbol) + ADR supersession chain + rationale for choosing SSE-only                                                                                                                                                                                                                                                                                                                 |
| `ROADMAP.md`                        | +7/−1    | Added "WebSocket Transport Removal (v5)" subsection to v5 retirement list (between "httputil Re-export Retirement" and "Proposed Module Boundaries"); updated "Not Planned" entry to reflect removal                                                                                                                                                                                                                                                                   |

### Pipeline verifications (all passed)

| Check                                                                                                   | Result                                                                                                                                                                                                                                      |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOEXPERIMENT=jsonv2 go build ./...`                                                                    | PASS (silent)                                                                                                                                                                                                                               |
| `GOEXPERIMENT=jsonv2 go vet ./...`                                                                      | PASS (silent)                                                                                                                                                                                                                               |
| `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race` (workspace mode, isolated `GOCACHE=/tmp/gocache-$$`) | PASS — root 4.08s, openapi 1.02s                                                                                                                                                                                                            |
| `GOEXPERIMENT=jsonv2 go test ./... -cover` (same isolation)                                             | PASS — root **93.1%** (gate 90% ✅), openapi 99.0%                                                                                                                                                                                          |
| `GOEXPERIMENT=jsonv2 golangci-lint run --timeout=5m`                                                    | **37 issues, zero in any file I touched** — all in `security.go`/`security_test.go`/`doc.go`/`htmx.go`/`errors.go`/`response.go` (pre-existing httputil v0.8.0/v0.9.0 split-brain)                                                          |
| Final grep for all 14 removed WS symbols across `*.go` + `*.md`                                         | All remaining matches are intentional: ADR-0046, status report, ROADMAP v5 retirement note, FEATURES.md removal table, CHANGELOG Removed section, or **pre-2026-08 archived status snapshots** that describe past work as historical record |

### Git state

- 2 commits ahead of origin/master: `4c427979 chore(deps): switch to local module versions for development` + `87c9500b docs(agents-md): remove stale WSBroadcaster reference from SSE re-export note`
- Working tree clean
- The auto-daemon captured all my in-progress doc edits as separate `docs(...)` commits during the session (similar to the prior session's behavior on source files)

---

## b) PARTIALLY DONE ⏳

### `nix run .#test` (hermetic GOWORK=off build) — NOT verified green

The flake.nix `test` app uses `GOWORK=off` which discards the `go.work` local replace for httputil. The published httputil v0.8.0 lacks the security headers fields that `security.go` uses (`RecommendedHSTS`, `RecommendedCSP`, `SecurityHeaderSkip`, `ContentTypeOptions`). Build fails before any test runs:

```
./security.go:18:32: undefined: httputil.RecommendedHSTS
./security.go:23:31: undefined: httputil.RecommendedCSP
./security.go:28:35: undefined: httputil.SecurityHeaderSkip
./security.go:63:12: config.ContentTypeOptions undefined
./security.go:64:10: config.ContentTypeOptions undefined
FAIL	github.com/larsartmann/cqrs-htmx/v4 [build failed]
```

This is **pre-existing and unrelated to WS removal** — same code path that existed before today. Documented in AGENTS.md "httputil leverage (v4.x)" gotcha and ROADMAP.md v5 retirement section. Workspace-mode `go test` (which respects the `go.work` httputil replace) passes. **Resolution requires publishing httputil v0.9.0**, then bumping cqrs-htmx `go.mod` and removing the `go.work` replace — out of scope for this session.

### CI lint pipeline not run

Did not invoke `nix run .#lint`, `nix run .#coverage-gate`, or the cross-module flake apps (`check-templates`, `check-codegen`, `check-dep-budgets`, `check-module-isolation`, `check-phantom-version`, `check-cqrs-lint`). Only the root-module `golangci-lint run` was run directly.

---

## c) NOT STARTED ❌

### Submodule deep audit re-verification

The prior session verified zero WS references in `adminui`, `usermgmt`, `loginpage`, `dashboardui`, `datastar`, and `examples/*` via structural grep. I did NOT re-run that grep this session — trusting the prior session's verification. If the auto-daemon or concurrent sessions introduced new WS references, they would not be caught.

### `examples/*` smoke tests

Did not run any example to verify the docs still compile against the examples (e.g. `examples/basic/`, `examples/admin-demo/`). Examples were also untouched in prior session's deletions — should be fine but unverified.

### Tag / release

Did not bump version, tag a release, or push to remote. Per the status report's open questions Q1 (v4.x vs v5.0.0 bump) — still pending user decision. The CHANGELOG `[Unreleased]` section is unversioned.

### `references/realtime.md` file

The status report's exact next steps mentioned creating/updating `docs/guides/realtime.md`. The file does NOT exist (`ls docs/guides/` returned 14 files, no `realtime.md`). The SKILL.md still references `references/realtime.md` (line ~514) as if it existed. **This is a doc health drift**: the SKILL.md is pointing at a file that doesn't exist. Should be either created (with WS-stripped content) or the SKILL.md reference should be removed/redirected.

---

## d) TOTALLY FUCKED UP / NEAR-MISSES 💥

### 1. Wasted ~5 minutes on Go test linker cache corruption

After clean verification of `nix run .#test` (which failed for the documented httputil reason), I tried `go test ./...` with the default `GOCACHE` and hit:

```
/nix/store/.../link: cannot open file /home/lars/.cache/go-build/8c/8c33...-d: open: no such file or directory
FAIL	github.com/larsartmann/cqrs-htmx/v4 [build failed]
```

This is a known race in the Nix Go installation: the GOCACHE directory is shared between concurrent invocations (likely the auto-git daemon or other parallel agents triggering `go` runs in the background). Fix was `GOCACHE=/tmp/gocache-$$` to isolate. Should have known to do this immediately from AGENTS.md "GOEXPERIMENT=jsonv2 is mandatory" gotcha and the broader pattern of Nix + Go cache corruption. **Lesson for future:** always use isolated `GOCACHE=/tmp/gocache-$$` in this environment when running tests after any concurrent activity.

### 2. Auto-git daemon committed work-in-progress 3 times during my session

Without asking or surfacing the commits to me first:

- `49a9f78a docs(documentation): reflect WebSocket removal in v5 across skill, agent, and ADR docs` (my SKILL.md + ADR INDEX changes)
- `1818a82f docs(status,chore): record WebSocket removal status and normalize struct tag spacing` (prior session's status report — actually pre-dated my session)
- `87c9500b docs(agents-md): remove stale WSBroadcaster reference from SSE re-export note` (my last edit)
- `4c427979 chore(deps): switch to local module versions for development` (auto-daemon's own commit)

The commits were correct, but they:

- Made my `git diff HEAD` checks return empty when I expected to see pending work
- Forced me to query `git log` instead of `git status` to find what was captured
- Risk: if I'd been mid-edit on a broken file, the daemon could have committed a half-done change

This matches the prior session's behavior. Should have set `GIT_AUTHOR_DATE` or coordinated with the daemon better. **Lesson for future:** at session start, check `git log origin/master..HEAD` (not `git status`) to see what's captured.

### 3. ROADMAP.md edit initially missed line 147

My first `edit` on ROADMAP.md targeted lines 64-69 (Datastar future scope table) and 142-148 (Not Planned section). The first edit succeeded (removed WebSocket row from table); the second edit FAILED silently because the surrounding context lines I included did not match exactly (the `lsp_replace_symbol` read returned a slightly different whitespace). I caught this only via post-edit grep. Should have re-read each file's exact line range before editing.

### 4. SKILL.md edit for "Embedded HTMX extensions" accidentally also collapsed a needed WS-mirroring paragraph

When I rewrote the "Embedded HTMX extensions" section, my `new_string` block consumed the preceding WS-mirroring paragraph (`WebSockets mirror this: NewWSBroadcaster()...`). That paragraph was supposed to be replaced separately with the SSE-only note in the "Realtime (SSE)" section. Result: the WS-mirroring paragraph is gone (correct), but the next paragraph was a duplicate "Embedded HTMX extensions" header which was then correctly collapsed. Net result is correct (no broken structure), but the edit was structurally bigger than intended — should have done 2 separate edits.

---

## e) WHAT WE SHOULD IMPROVE 🚀

### Immediate wins from this session's work

1. **Add a CI step that grep-checks for WS symbol leaks** — `find . -name '*.go' -o -name '*.md' | xargs grep -lE 'WSBroadcaster|WSMessage|HTMXExtWS|DispatchWSCommand' | grep -v 'docs/adr/0046\|docs/status/2026-08-05_12-15'` should return empty. This would have caught any drift across auto-daemon commits.

2. **Document the `references/realtime.md` SKILL.md drift** — either create the file or remove the broken reference. Listed as an item in TODO.

3. **Add `references/migration-v4-to-v5.md`** — the CHANGELOG.md Removed section is comprehensive but not as discoverable as a dedicated migration guide. Consumers upgrading from v4.6.x to v5 will hit the WS-removal first; a `docs/migrations/v4-to-v5.md` with copy-paste migration recipes would be higher-quality than the inline CHANGELOG entry.

4. **Add a `cqrs-lint` rule for "removed symbols"** — the cqrs-lint tool has a `B026` rule for deprecated-but-still-imported symbols. A parallel rule for "imported but removed in this version" would catch migration regressions.

### Process improvements

5. **Adopt the isolated-GOCACHE pattern repo-wide** — `.buildflow.yml` already has `max_concurrency: 1` to prevent GOCACHE races, but the `nix run .#test` flake app doesn't pass `GOCACHE=/tmp/gocache-$$`. Add it to the flake `goEnv`.

6. **Coordinate with auto-git daemon** — either surface daemon commits to the assistant via a pre-commit hook summary, or let the assistant commit explicitly. The current "daemon commits behind your back" pattern is hostile to deliberate, reviewed commits.

7. **Pre-flight grep before any deletion-style refactor** — the prior session's source-file deletions were perfect because they pre-flighted the symbol list. The doc updates were less disciplined — I edited docs files ad-hoc instead of running a comprehensive grep first to enumerate every doc location mentioning WS. Should have built a checklist first.

### Code quality improvements surfaced by the deletion

8. **`ack.go defaultAckEventName` doc comment** had a small split brain — it was documented as "SSE/WS event name" but `ackToWSMessage` returned a plain string, not an `sse.Event`. Fixed in the prior session. Worth a search for similar doc-code inconsistencies in `partial.go` (was "SSE/WS"), `responsewriter.go` (was "SSE/WS"), `logging.go` (was "SSE/WS"). I fixed those but did not grep for the inverse pattern (code says SSE-only but doc says SSE+WS).

9. **`fanOut[T]` generic** was originally introduced to share code between SSE and WS broadcasters (per CHANGELOG.md line 619). Now that WS is gone, `fanOut` is only used by SSE. Worth renaming to `sseFanOut` or inlining for clarity. Out of scope for this PR but worth noting.

10. **`extensions/ws.min.js` gitattributes workaround** is now unnecessary. AGENTS.md mentions `*.min.js -text` gitattributes existed solely for this file. Verify it can be removed. Not done in this session.

### Architectural improvements

11. **The "drop WebSocket" pattern itself** — once httputil v0.9.0 is published and the v5 cut happens, the same pattern (drop a transport/feature without a Deprecated phase because the consumer surface is small) should be codified as a policy. ADR-0046 implicitly justifies this; could be a follow-up ADR ("How we sunset features").

12. **Datastar adapter module** has a "Datastar WS support" entry in ROADMAP.md line 68 (now removed). Re-check whether Datastar upstream still has WS support, and whether the adapter needs an equivalent migration guide.

---

## f) NEXT STEPS — up to 50 things to do next 📋

> **Resolution (2026-08-05 docs-health pass):** items re-routed below. `~~struck~~` = done; **→ TODO_LIST / ROADMAP** = tracked there (open); **still open** = not yet actioned. The httputil publish (#1–2) and version decision (#3) are the critical-path blockers; everything else is gated on them or is independent polish.

### Critical (block v5 release)

1. **Publish httputil v0.9.0** — unblocks `nix run .#test` hermetic build. **→ still open — TODO_LIST P0.**
2. ~~**Bump cqrs-htmx `go.mod` to httputil v0.9.0** + remove `go.work` httputil replace.~~ bundled with #1 (TODO_LIST P0).
3. **Decide version bump: v5.0.0 or v4.7.0.** **→ still open — ROADMAP "Open Questions" #1.**
4. **Tag and release v5.0.0.** **→ still open — gated on #3.**
5. **Push to origin.** **→ still open — gated on #4 (needs explicit user approval).**
6. ~~**Update `identity-model` CHANGELOG`**~~ NOT-DO — identity-model is pure domain types; zero WS surface (structurally impossible). CHANGELOG needs no WS entry.
7. **Verify submodule CHANGELOGs.** **→ still open — quick verification when #4 lands.**

### Doc health (within next session)

8. **Resolve the `references/realtime.md` SKILL.md drift.** **→ still open — TODO_LIST P1.**
9. **Create `docs/migrations/v4-to-v5.md`.** **→ still open — TODO_LIST P1.**
10. **Add CI step to grep for WS symbol leaks.** _*→ still open — folded into TODO_LIST P2 ("wire check-* into CI")._*
11. **Audit the 2026-06 archived status reports.** **→ still open (low priority) — historical snapshots; leave as-is or annotate when touched.**
12. ~~**Verify `extensions/ws.min.js` gitattributes workaround is no longer needed.**~~ partially resolved — `extensions/` now contains only `sse.min.js` + `idiomorph-ext.min.js`; the `*.min.js -text` gitattributes entry is now harmless but unverified-removable. **→ still open (cosmetic).**
13. ~~**Update `examples/datastar-demo/` references.**~~ NOT-DO — datastar adapter never had a WS surface; demo is unaffected.
14. **Verify `examples/*` build against new public API.** **→ still open — run `go test ./examples/...` when convenient.**
15. **Cross-check the `httputil` re-export deprecation in v5.** **→ tracked — ROADMAP "httputil Re-export Retirement (v5)".**
16. **Cross-check the `usermgmt` identity-model re-export deprecation in v5.** **→ tracked — ROADMAP "Re-export Layer Retirement (v5)"; markers already in CHANGELOG Deprecated.**

### Verification (build / test / lint)

17. **Run `nix run .#test` after httputil v0.9.0 publish.** **→ still open — gated on #1.**
    18–26. **Run the full nix gate cycle** (`.#lint`, `.#coverage-gate`, `.#build`, `.#check-templates`, `.#check-codegen`, `.#check-dep-budgets`, `.#check-module-isolation`, `.#check-phantom-version`, `.#check-cqrs-lint`). **→ still open — these are the canonical post-publish verification; recompute once #1 lands.** (Note: lint was confirmed 37 pre-existing `security.go` issues — all from the httputil split-brain, zero in WS-touched files.)

### Submodule post-deletion audit

27–34. **Re-grep submodules for WS symbols.** **→ low priority** — WS lived only in the root module; the prior session's structural grep was clean. Re-verify if any doubt arises, but no action is blocking.

### Cosmetic / nice-to-have

35–36. **Rename/inline `fanOut[T]`.** **→ tracked — CHANGELOG notes it is now SSE-only; rename is a cosmetic follow-up.** 37. **doc-comment header on `ack.go`.** **→ still open (cosmetic).** 38. **"v5 Migration" badge on README.** **→ still open — gated on #4.** 39. ~~**Verify `examples/admin-demo/` works without WS extension.**~~ NOT-DO — the WS `<script>` tag was removed with `HTMXExtWS` (the embedded asset is gone); admin-demo cannot reference it. 40. **Update `templ-components` docs.** **→ NOT-DO — templ-components has no WS surface.**

### Documentation consistency

41. **Update `docs/adr/0004-sse-websocket-support.md` inline.** **→ still open — TODO_LIST P1.**
42. **Update `docs/adr/0010-transport-parity.md` inline.** **→ still open — TODO_LIST P1.**
    43–48. **Annotate older docs that reference WS** (research, archived status reports). **→ low priority** — these are historical snapshots describing past work; leaving them is acceptable (they are timestamped). Annotate only if a reader is misled.

### Process / meta

49. **Follow-up ADR "How we sunset features".** **→ low priority — ADR-0046 already captures the rationale.**
50. ~~**Run the docs-health skill end-to-end.**~~ **done (2026-08-05)** — this pass rewrote TODO_LIST/ROADMAP/FEATURES, appended CHANGELOG, and annotated this report.

---

## g) QUESTIONS FOR USER (≤3, must be unresolvable by me) ❓

### Q1. Release version bump: v5.0.0 (major) or v4.7.0 (minor)?

The WS removal deletes 14 exported symbols (`WSBroadcaster`, `NewWSBroadcaster`, `WSMessage`, `WSOOBHTML`, `ParseWSMessage`/`ParseWSMessageInto`, `WriteWSMessage`/`WriteWSMessageInto`, `DispatchWSCommand`/`DispatchWSQuery`, `DecodeWSJSON`/`DecodeWSJSONQuery`, `BroadcastOnSuccessWS*`, `BroadcastOnErrorWS*`, `BroadcastOnAckWS*`, `HTMXExtWS`). Per SemVer, deleting public API requires a major bump. But the prior session's CHANGELOG entry, AGENTS.md gotcha, FEATURES.md table, ROADMAP.md, and SKILL.md description ALL frame this as "v5." The httputil re-export retirement is also bundled into the same v5 cut per ROADMAP.md. I assume v5.0.0, but want confirmation before tagging.

### Q2. Should the AGENTS.md "httputil leverage (v4.x)" gotcha be rephrased given httputil v0.9.0 isn't published yet?

The gotcha claims the consolidation is "complete" and only the publish step remains, but as of 2026-08-05 the hermetic `nix run .#test` is BROKEN because the local replace doesn't survive `GOWORK=off`. AGENTS.md framing implies it's done; reality is "almost done." I did not rephrase it because the user's framing may be intentionally optimistic, but the gap between docs and reality is large enough to be misleading to future agents.

### Q3. The `references/realtime.md` SKILL.md drift — create the file or remove the reference?

The SKILL.md at line ~514 references `references/realtime.md` which does not exist (`ls docs/guides/` returns 14 files, no realtime.md; the file would have lived at `.agents/skills/cqrs-htmx/references/realtime.md` which also doesn't exist). Either I should write the file (~200-400 lines: SSE broadcaster patterns + ACK protocol + idempotency + reconnection/replay + heartbeat + event filtering, WS-stripped) or update the SKILL.md to remove the broken link. I cannot infer which the user prefers without asking — the SKILL.md description line strongly implies the file should exist, but the project convention "don't add filler docs" suggests removing the reference.
