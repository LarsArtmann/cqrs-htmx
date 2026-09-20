# Status Report — Command-Utilization Deep Dive (Session Scope)

**Date:** 2026-09-17 20:18 CEST
**Scope:** THIS session only — the go-cqrs-lite `command/v4` utilization audit across cqrs-htmx. Per instruction, no unrelated research was done; everything below comes from what this session actually ran and observed.
**Branch state at report time:** `master`, ahead of origin by 4+ commits, a **concurrent sibling session is actively committing** (heuristic daemon commits + a toolchain tug-of-war, see §d/§e). Deliverable: `docs/research/2026-09-17_go-cqrs-lite-command-deep-dive.html` (1659 lines, committed at `473bbf29`).

> **ANNOTATED 2026-09-20** (docs-health sweep): the audit's two headline findings and the top-3 remediation items were implemented 2026-09-18.
> - **§b:** b2 DONE (bijection scripted); b1/b3/b4/b5 remain partial/notes.
> - **§c:** c1–c3 DONE (structural `ApplyOptions` fix; built-in audit chain; `CommandMiddleware` seam); c4–c8 remain open.
> - **§f:** struck rows confirmed done (the audit-trail chain, bijection script, AGENTS gotchas, toolchain fix, train-lag zero, harvest); unmarked rows remain open → `TODO_LIST.md` / `ROADMAP.md` (idempotency, validation, journal producer, prior-art cross-check, BuildFlow upstream asks).
> - **§g:** Q1 answered (implemented); Q3 resolved (coordinated 1.27.1 bump 2026-09-19); Q2 (prior-art relationship) remains an owner call.

---

## a) FULLY DONE

1. **Skill-gated process.** Loaded `cqrs-htmx` + `library-deep-dive` (both references: research-methodology, output-guide, html-output-guide) before starting; loaded `status-report` + `brutal-self-review` before writing this report.
2. **Phase 1 — usage inventory across all 27 modules.** Import-level grep of `go-cqrs-lite/command/v4`: direct users = root (App bridge, handler.go:151/243/285/326), identity-model (20 command structs, all embedding `*command.BasicCommand`), usermgmt (internal dispatcher, service_core.go:331, ~25 dispatch sites), dashboardui (journal read-side, handlers_audit.go:26-58), systemadapter (declarative `system.DomainConfig`), 6 examples. `totp` reaches commands via the service (totp.go:80,110). adminui/loginpage/datastar/setup/health/auditlog have none — architecturally correct for UI/transport/bridge layers.
3. **Phase 2 — full capability surface of `command/v4` v4.10.0**, read from the local sibling repo (not training data): Dispatcher/RegisterTyped/BasicCommand/PersistedCommand/TypedCommandStore[P]/MemoryBus/MetadataCarrier, 7 metadata options, 13 `Command*` middleware factories. Signatures used in the report's before/after code (`CommandIdempotency`, `CommandValidation`, `ActorEnricher`, `ApplyOptions`) were verified against source before citing.
4. **Version currency verified:** root go.mod:9 pins `command/v4 v4.10.0`; highest upstream tag = `v4.10.0`. Zero gap — all findings are adoption gaps, not upgrade gaps.
5. **Gap analysis + weighted scoring:** 4 fully-leveraged / 3 partially-used / 4 missed / 3 not-applicable → **62/100 ("Moderate")** with the rubric's weighting (critical 40 / advanced 25 / config 15 / best-practice 10 / version 10).
6. **Key discovery (the report's headline):** the upstream audit-trail chain (`middleware.CommandActorContext()` → `decider.WithEnricher(event.ActorEnricher)`, plus `CommandCausalityEnricher`) has **zero call sites** in this repo; commands are constructed without metadata and `repositoryOptions` (usermgmt/snapshot.go:88-91) wires only snapshot+state-cache. Every event the library's own domain emits is anonymous. Second discovery: root's `enrichCommandFromContext` (handler.go:347-351) asserts the *concrete* `*command.BasicCommand`, so every embedded-command wrapper — including all 20 identity-model commands — silently skips HTTP metadata enrichment (the skip is even documented in the comment; upstream's `MetadataCarrier` capability interface exists precisely to avoid this).
7. **HTML report written and verified:** 1659 lines, 8 sections, 7 finding cards, 2 tables; automated structural checks (all 8 section IDs present exactly once, div/section/table/tr/pre/code/p tag balance all equal, zero template placeholders); title placeholder fixed.
8. **Commit integrity verified.** Working tree == HEAD for the report (`git diff HEAD` empty); committed version contains the complete final content (1659 lines, title match, 15 finding/callout markers).
9. **Concurrent-session hygiene.** 5 foreign dirty files (`go.mod`, `setup/bundle.go`, `systemadapter/go.mod`, `systemadapter/go.sum`, `examples/system-demo/go.mod`) identified and left untouched; the go `1.27.1` vs `go.work 1.26.7` tug-of-war correctly diagnosed from the log (commit `68750ef8` "restore go 1.26.7 directive") and cited as the root cause of the pre-commit failures — not fought, per AGENTS.md guidance.

## b) PARTIALLY DONE

1. **Deliberate commit with narrative message — lost to the daemon race.** First `git commit` attempt ran the full BuildFlow hook (36s, red on pre-existing workspace failures). My drafted message never landed; when I attempted the documented `--no-verify` fallback, staging was already gone — the daemon had committed the file as `473bbf29 chore: auto-commit 1 changed file(s) (heuristic)`. Content intact, attribution lost.
2. ~~**"20/20 typed" proof.** Verified by counting `RegisterTyped` occurrences (11+4+3+2 = 20) against AGENTS.md's stated 20 commands — but the 1:1 bijection (each `Cmd*` constant ↔ one command struct) was not scripted/proven.~~ done (scripts/check-command-bijection.sh enforces the 20/20 bijection)
3. **"0/13 middleware in the core path" supporting check.** One grep (`benchmark_middleware_test.go`) ran against a wrong path (`root/` subdir that doesn't exist), silently returned nothing, and the thread was dropped. The conclusion still holds structurally (root's `App` consumes consumer-supplied dispatchers; usermgmt's internal dispatcher demonstrably has no `Use()`), but that particular check was never completed.
4. **Self-review integration.** The `brutal-self-review` skill prescribes its own HTML at `docs/reviews/`; per your single-file instruction it is folded into this report's §d/§e instead (format/location override, flagged here).
5. **Overlap check with prior research.** `docs/research/go-cqrs-lite-feature-audit.html` exists (seen in the directory listing); per "do not research unrelated stuff" it was never opened — whether my deep-dive duplicates or contradicts it is **unknown**.

## c) NOT STARTED

All eight remediation items from the audit remain unimplemented (they are findings, not changes):

1. ~~Structural `ApplyOptions` enrichment fix in root (priority 20) + regression test~~ done (structural ApplyOptions fix landed)
2. ~~Wire `ActorEnricher` + `CommandActorContext` + causation into usermgmt (priority 20)~~ done (audit chain built-in 2026-09-18)
3. ~~Expose `ServiceConfig.CommandMiddleware` and `Use()` it (priority 16)~~ done (CommandMiddleware seam shipped)
4. `CommandIdempotency` on auth mutations (priority 12)
5. Document the recommended production middleware chain (priority 8)
6. Optional SQL command journal producer for dashboardui's audit view (priority 6)
7. `CommandValidation` for uniform syntactic 400s (priority 6)
8. `Dispatcher.Close()` in `Service.Close` (priority 5)

Also not started (session follow-through): filing the 8 items into `TODO_LIST.md` + a CHANGELOG entry (docs-health HARVEST); cross-checking/cross-linking the prior feature-audit report; linking the report from docs index/AGENTS.md; the skill's Phase-6 cross-skill referrals.

## d) TOTALLY FUCKED UP

1. **Commit attribution lost — a *foreseeable* instance of the documented loss class.** AGENTS.md: "the auto-commit daemon polls faster than a long verification tail … 8+ deliberate narrative commits were lost." I knew the rule, staged the file, launched a 36-second hook, and then fired the retry without re-checking `git status --short` first. Result: the deep-dive — this session's entire deliverable — sits in history under a meaningless heuristic message, and the drafted narrative (which included the `--no-verify` justification) is gone. The failure mode was in my own project memory and I executed it anyway.
2. **One wasted round trip.** The `--no-verify` attempt failed with "nothing staged" — a direct consequence of #1, avoidable with one `git status` before retrying.
3. **Everything else is intact.** No source code touched, no foreign changes reverted, no force-push, report content byte-verified at HEAD. Nothing needs repair.

**What I forgot (explicit list):** (a) re-check `git status --short` immediately before ANY `git commit` (the rule exists; I skipped it once); (b) the prior `go-cqrs-lite-feature-audit.html` overlap check; (c) TODO_LIST/CHANGELOG filing of my own findings; (d) compile-verifying the report's before/after snippets (signatures verified, snippets not compiled); (e) explaining an empty grep result before moving on (happened once, §b3 — my own memory rule: "if two sources disagree, trust the one you just ran" requires noticing when a check silently returned nothing).

## e) WHAT WE SHOULD IMPROVE

**Process (this session's mistakes, generalized):**

1. **Pre-commit fast-path for docs-only changes.** BuildFlow ran its full 104-step pipeline — including 11 golangci-lint module runs destined to fail on the tug-of-war — for a single self-contained HTML file. A staged-file-type gate (skip Go steps when no Go files staged) would have made the deliberate commit land in seconds. This is a BuildFlow upstream improvement worth reporting (same bucket as the known-tool-bug notes).
2. **Execute the daemon-race rule, don't just know it.** `git status --short` immediately before every `git commit` attempt, especially after background jobs complete.
3. **Empty tool output is a finding, not a pass.** One grep returned nothing because the path was wrong; that must terminate in "path corrected, re-ran" or "claim weakened", never in silence.
4. **Label unverified code in reports.** The before/after snippets were signature-verified but never compiled. Reports should mark such snippets "API-verified, not compiled" — or a scratch `go build` should back them.
5. **Script counting claims.** "20/20" should come from a scripted bijection (command constants ↔ registrations), not occurrence counting.

**Artifact quality (the audit's own targets, in priority order):** items 1–8 in §c — top two are each roughly a dozen lines and unlock the audit story the library already sells (actor-attributed events end-to-end).

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

*A brainstorm sorted by impact — not a commitment list; most items below #8 are ROADMAP/HARVEST fuel.*

**Implement the audit's findings (highest impact):**
1. ~~Fix `enrichCommandFromContext` to a structural `ApplyOptions` interface + regression test (P20)~~ done (structural ApplyOptions fix landed (enrichment-skip footgun fixed 2026-09-18))
2. ~~Wire `CommandActorContext` + `decider.WithEnricher(event.ActorEnricher)` + causation enricher into usermgmt (P20)~~ done (audit chain built-in in usermgmt 2026-09-18 (ActorEnricher + CommandActorContext + causation))
3. ~~Add `CommandMiddleware []command.Middleware` to `ServiceConfig`/`EventSourcedConfig`, thread through `setup.Config` (P16)~~ done (ServiceConfig.CommandMiddleware + setup.Config threading shipped)
4. Wire `CommandIdempotency` for register/verify/OAuth paths (P12)
5. Document the recommended production middleware chain for usermgmt consumers (P8)
6. Optional SQL command journal producer so dashboardui's command audit is live by default (P6)
7. `CommandValidation` for uniform syntactic 400s at the dispatch layer (P6)
8. Close the dispatcher in `Service.Close` (P5)

**Session follow-through (cheap, do first):**
9. ~~File items 1–8 into `TODO_LIST.md` (docs-health HARVEST) + append a CHANGELOG entry for the audit~~ done (harvested + CHANGELOG entry)
10. Open and cross-check `go-cqrs-lite-feature-audit.html`; merge, supersede, or cross-link
11. Link the new report from the docs index and AGENTS.md research pointers
12. ~~Script the 20/20 bijection proof (command constants ↔ `RegisterTyped` targets)~~ done (scripts/check-command-bijection.sh enforces the bijection)
13. Re-run the "0/13 middleware in core" grep with corrected paths and settle it
14. Compile-verify the report's before/after snippets in a scratch module; annotate the report
15. ~~Add the embedded-`BasicCommand` enrichment-skip footgun to AGENTS.md gotchas (consumer-facing)~~ done (AGENTS.md enrichment-skip footgun gotcha recorded)
16. Add pointer + one-line summary of the deep-dive to AGENTS.md so fresh sessions know it exists
17. ~~Decide ownership of the 5 foreign dirty files once the sibling session settles (not mine to touch now)~~ done (toolchain resolved; foreign-file ownership moot)

**Repo-wide issues noticed this session (report-only, not mine):**
18. ~~Resolve the toolchain tug-of-war: `GOTOOLCHAIN=go1.26.7` pin for root-module commands OR a coordinated flake+go.work+27-module bump to 1.27.1 (policy decision, AGENTS.md-documented)~~ done (coordinated 1.27.1 bump landed 2026-09-19)
19. Report/fix BuildFlow docs-only pre-commit cost (full pipeline for one HTML file; 36s + guaranteed red during tug-of-war)
20. Restore `fail_on` gating in `.buildflow.yml` when go-structure-linter ships suppression config + BuildFlow bumps its pin (already documented restoration condition)
21. Report gomod-check double-count upstream (gomod-check + go-mod-ignore-check report the same 51 findings)
22. Commit or gitignore the untracked `examples/middleware-showcase/vendor/` dir (25 vendor-consistency findings)
23. Fix the stale README version claim flagged by cqrs-lint (`README v4.6.0` vs `go.mod v4.10.0` — a real two-line drift)
24. Clean the AGENTS.md "v4.10.0+ vs v4.10.0" cqrs-lint warning (D005-adjacent phrasing)
25. ~~Route the 56 train-lag entries (templ-components v1.18.0 wave, go-appkit v0.5.1, go-retry v0.7.1, go-health v0.2.0, …) into the next family train~~ done (train-lag swept to ZERO 2026-09-20)
26. Dependabot cap: 28 Go modules vs 20-entry generation limit
27. Investigate samber-linter's 88% failure rate (61/69) flagged by preflight
28. nix-checker vendorHash staleness warnings (go.mod modified after hash was set)
29. ~~govulncheck workspace-mode failure while the go-directive inconsistency persists (self-heals after #18)~~ done (self-healed after the toolchain fix)
30. Purge the `setup-demo` 27MB blob from pushed history (AGENTS.md carry-over I re-confirmed exists in the gotchas, untouched)

**Deeper follow-ups the audit suggests:**
31. `CommandCausalityEnricher` adoption so every event links back to its command ID
32. Decide the idempotency store backend (memory vs SQL) before #4 lands in production shape
33. ADR for the command journal schema (type, stream ref, payload, metadata) if #6 is pursued
34. Benchmark: does the enricher chain measurably cost anything on the dispatch path (b.Loop + benchstat, per repo bench policy)
35. E2E test asserting actor-attributed events appear in auditlog/dashboard views after #2
36. Evaluate `MemoryBus` SubscribeAll for a command-metrics side channel (currently N/A, revisit if needed)
37. Document per-module command posture in the cqrs-htmx skill (which modules touch commands and why)
38. Consider exporting a `Service.Dispatcher()` accessor vs keeping the dispatcher private (consumers currently cannot reach it at all)
39. Sweep examples for the manual `CommandOptionsFromContext` pattern (basic/main.go:316) once root enrichment is fixed — they become redundant
40. Add the report's 12-row capability matrix to the go-cqrs-lite leverage guide as a checklist

*(Stopped at 40 — the remaining space is deliberately left for HARVEST routing decisions rather than padding.)*

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Implement or report-only?** Should I file the 8 findings into `TODO_LIST.md` now and implement the top-2 fixes (each ~a dozen lines + test) in this repo — or hold all code changes until the concurrent session and the toolchain tug-of-war settle?
2. **Prior-art relationship.** Is the existing `docs/research/go-cqrs-lite-feature-audit.html` the canonical audit I should update/merge into, or is my command deep-dive a standalone snapshot that should just cross-link it? (You told me not to open it, so only you can rule.)
3. **Toolchain policy.** Is the sibling session's `go 1.27.1` bump on root `go.mod` intentional (a coordinated 27-module bump is coming) or accidental (should be pinned back to 1.26.7 / `GOTOOLCHAIN`)? This decides whether my future commits keep hitting a red pre-commit hook and need the `--no-verify` fallback.

---

**WAITING FOR INSTRUCTIONS.**
