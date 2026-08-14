# Status Report — go.work repair, tag-existence guard, self-review

**Date:** 2026-08-15 01:34 CEST
**Session scope:** Continuation of the v4.8.0 post-push follow-through. Executed the planned backlog (living-docs pass, version-drift guard, CI assertion, GOWORK audit) PLUS an unplanned repair: go-cqrs-lite landed its ADR-0128 module extraction mid-gap, which broke every workspace-mode build in cqrs-htmx. Ends with this user-requested self-review (a–g).

---

## a) FULLY DONE

1. **State re-verification at open:** user has NOT pushed (remote still `56068b16`); `datastar/v4.8.0` local tag intact at `f128072d` (tag object `89f8f143`, single signature); go-datastar sibling clean/synced at `5b70bb1`.
2. **Discovered + root-caused the ADR-0128 breakage:** go-cqrs-lite commit `5127039da` (pushed) deleted the in-repo `codec/`, `retry/`, `flightrecorder/`, `idempotency/` shim modules (extracted to external published repos `go-codec` v0.3.0, `go-retry` v0.2.0, `go-flightrecorder` v0.2.0, `go-idempotency` v0.1.2). Four cqrs-htmx go.work replaces pointed at the deleted dirs → `replacement module directory does not exist` on every workspace-mode build. Hermetic GOWORK=off builds were unaffected (published tags still resolve).
3. **Fixed go.work (`29bf201f`):** removed the 4 dead replaces with a documenting comment block (why removed, why external-path migration must wait: published `event/v4.6.0` APIs still take `codec/v4` types, so the module families cannot mix until post-extraction go-cqrs-lite tags). Corrected the stale go-idempotency replace comment. Verified: `go build ./...` across all 27 workspace modules → exit 0, zero errors. go.work.sum churn (1 line: proxy sums for newly-unreplaced modules) inspected and included in the commit.
4. **Published-tag existence guard (`d63799c7`):** `check-version-drift.sh --strict` now verifies every `github.com/larsartmann/*` require resolves to a tag that exists on the module's remote repo (`git ls-remote`, one fetch per repo, cached). Catches the phantom-bump class (e.g. buildflow's historical `totp v4.8.0`). Details: pseudo-versions skipped; requires satisfied by a local `replace` in the same go.mod exempted (replace proves local satisfiability — `examples/datastar-demo`/`integration_test` depend on the unpushed `datastar/v4.8.0` this way); one-line `require x vY` entries now parsed (previously block-only); tag-mapping handles bare major-suffixed root modules (`cqrs-htmx/v4@v4.8.0` → `v4.8.0`, not `v4/v4.8.0`) and sub-module paths (`usermgmt/totp/v4@v4.7.0` → `usermgmt/totp/v4.7.0`). Baseline: **645 requires checked, all resolve; 12 legitimately skipped.** Negative-tested twice (phantom `totp/v4 v4.8.0` → caught by drift leg; isolated phantom `go-finding v9.9.9` → caught by the tag leg, exit 1).
5. **CI template-restore assertion (`d63799c7`):** the `SQL setup template compile check` step now fails if `check-templates.sh` leaves `usermgmt/` modified (trap-restore leak or untidy go.mod detection). YAML validated. The CI version-drift step stays advisory (`|| true`) with a comment explaining why (https ls-remote auth for private repos unverified in CI; strict gate runs in `nix run .#check-modules`).
6. **GOWORK=off audit of all flake gate apps (`d63799c7`):** goApp family hermetic via shared `goEnv`; isolation/scripts hermetic per-invocation; **one gap found and fixed: `errorfamily` (raw writeShellApplication) was the last gate without `GOWORK=off`** — `go run scripts/errorfamily_scanner.go` loaded the workspace graph and would break whenever the go-cqrs-lite sibling is mid-refactor. Fixed and verified green.
7. **Living-docs pass (`368dd717`):** runbook §3 go-datastar marked DONE (user tagged `static/v0.2.0`); §4 item 8 marked executed-locally with exact remaining steps; AGENTS.md — new ADR-0128 bullet, replace-pile bullets updated (datastar train state; remaining pile = metaengine pair + 2 datastar-local family replaces), 4 stale claims fixed (lint exclusion now excludes only e2e/examples; release-train remaining-work; check-templates hermetic rework description; integration_test SA1019 migration recorded), new bullet for the tag guard + all-gates-hermetic note; CHANGELOG Unreleased gained 4 Added + 3 Fixed entries; TODO_LIST P1 item 1 and P2 CI item updated (check-templates no longer "blocked").
8. **Gates verified this session (6):** workspace build all 27 modules; `check-modules` (isolation + dep-budgets + strict drift+tags + replace-directives + docs-freshness + docs-links); `check-templates` (+ clean restore); `errorfamily`; `nix flake check --no-build`; HEAD re-verification after the `--no-verify` commits (build + drift both green). golangci-lint ran green across all modules inside the (failed) commit-hook attempts.
9. **3 commits landed:** `29bf201f` (go.work), `d63799c7` (checks), `368dd717` (docs). Tree clean. **8 total unpushed commits + 1 unpushed tag.**

## b) PARTIALLY DONE

1. **Full 13-gate suite at HEAD:** NOT re-run this session (see d.3). Only the 6 above. `lint`/`test`/`test-fuzz`/`test-flake`/`coverage-gate`/`check-codegen`/`check-cqrs-lint` last verified green at `b6020272`; this session's diff (go.work, flake.nix, scripts, CI, docs) is low-risk for them (per-module gates run GOWORK=off; go.work only feeds module discovery) — but "verified" they are not.
2. **Runbook §3 tag-mapping validity post-extraction:** I annotated `5127039da` as a valid base for the projectionadapter/sqliteengine tags but did NOT re-verify that their go.mod files (now requiring external `go-codec` etc.) still strip/build/publish per the runbook's exact instructions. Must be checked before cutting those tags.
3. **gopls workspace health:** restarted once after the go.work fix, but tool diagnostics still showed ~7,287 stale errors afterwards. Builds are truth and are green; the LSP view may need another restart or has another problem. Unverified.
4. **Pre-commit hook:** works for content, but this session it failed twice on infra (dprint crash, see d.2) and re-mutated `adminui/styles.css` twice (tailwind regenerating the minified CSS with a trailing-newline diff on every run, regardless of staged files). I reverted both times manually — whack-a-mole, root cause untouched.

## c) NOT STARTED (this session, from the plan or discovered)

1. Post-push datastar remainder: strip the 2 `cqrs-htmx/datastar/v4 => local` family replaces (examples/datastar-demo, integration_test) + verify proxy serves the tag — blocked on user push.
2. totp/oauth2 v4.8.0 tag cuts — blocked on Q1.
3. systemadapter train — blocked on upstream metaengine tags (§3).
4. cqrs-htmx migration to external `go-codec`/`go-retry`/`go-flightrecorder` import paths — blocked on post-extraction go-cqrs-lite tags (event/v4.7+ etc.).
5. Triage of the 28 govulncheck findings surfaced by the hook (incl. GO-2026-6090, crypto/tls) — noticed, logged here, not triaged, no TODO_LIST entry yet (fix follows this report).

## d) TOTALLY FUCKED UP (honest ledger)

1. **The PIPESTATUS trap — repeat offender.** The prior session's summary explicitly documents "pipes mask exit codes; use `cmd; echo $?`". I still used `${PIPESTATUS[0]}` three times this session (mvdan/sh does not support it reliably). Worst instance: the FIRST workspace build FAILED but my echo printed `BUILD-EXIT: 0` — I only caught the failure because I actually read the error output above it. A lazier read would have shipped "workspace green" while it was red. Documented gotcha, ignored, repeated.
2. **Blind retry on the failed commit.** The first `--no-verify`-less commit failed on dprint exit 14. The root-cause hint (`SQLite database ... is busy` — nix eval-cache lock contention, likely racing the auto-git daemon) was in that first log; I grep'd for FAIL patterns, missed it, and retried unchanged — violating "never repeat a failure expecting a different result". Second identical failure, then I read properly and used the documented `--no-verify` fallback (with justification in the commit message and HEAD re-verification). One commit cycle wasted.
3. **Overclaimed todo completion.** I marked "Run gates, commit docs/scripts changes" completed after 6 of 13 gates. Prose was precise, but the checklist said done. The full suite is the standing bar before calling a session's verification complete.
4. **Shallow existence check for replace targets.** I first tested `[ -d dir ]` — `idempotency/` still exists as a directory (only `kvstore/`/`sqlstore/` submodules remain) while its go.mod was deleted. The correct check `[ -f dir/go.mod ]` cost an extra build round-trip after a false "only 3 dead".
5. **First negative test was inconclusive, reported as passed.** The phantom `totp/v4.8.0` was caught by the drift leg (v4.7.0 vs v4.8.0) BEFORE the tag leg ran — I initially said "caught", then isolated the tag leg with `go-finding v9.9.9` to prove the actual new code path. Correct final state, sloppy first claim.

## e) WHAT WE SHOULD IMPROVE (process, durable)

1. **Shell exit-code discipline is load-bearing:** always `cmd > file 2>&1; echo $?` — never `${PIPESTATUS[0]}`, never `cmd | head; echo $?`. Add to AGENTS.md gotchas (currently only implicit in prior status reports).
2. **Root-cause before retry:** on hook/pipeline failures, grep the FULL log for `busy|lock|timeout|transient` first; a retry without a hypothesis is a wasted cycle.
3. **Scope-verify after structural changes:** when a gate list exists, either run all of it or write explicitly which subset the diff can affect and why. "6/13, others unaffected because X" is honest; a green checkmark is not.
4. **Determinism of generated assets:** the tailwind/styles.css churn recurs every commit. Fix it once (commit the regenerated output so it stabilizes, or scope/skip the tailwind step in `.buildflow.yml` for non-CSS commits, or make the generator byte-stable). Manual `git restore` twice per session is unsustainable.
5. **Guard the dead-replace class:** this session's breakage (upstream deletes a replaced dir → workspace builds fail with a confusing per-file error) is now cheap to prevent: a gate step asserting every go.work replace target contains a go.mod. Same spirit as the tag guard: turn a silent landmine into a loud, named failure.
6. **New infra failure modes deserve memory:** the dprint/nix eval-cache crash and the styles.css churn are NOT yet in AGENTS.md gotchas (only in commit messages + this report). Memory rules say write it at discovery time — fix in the next docs touch.
7. **Version-drift tag leg in CI is advisory-only** because private-repo https auth for `ls-remote` is unverified there. Consider wiring SSH credentials into CI so the strict gate runs there too (currently local-only).

## f) NEXT — up to 40 things, ordered

**Unblocked now (no user action needed):**

1. Add the 28 govulncheck findings (GO-2026-6090 first) to TODO_LIST with a triage decision (Go toolchain bump vs accepted-risk documentation).
2. Run the remaining 7 gates at HEAD (`lint`, `test`, `test-fuzz`, `test-flake`, `coverage-gate`, `check-codegen`, `check-cqrs-lint`) — closes b.1.
3. Add "every go.work replace target has a go.mod" assertion to `check-modules` (e.5).
4. Add AGENTS.md gotchas: PIPESTATUS/mvdan-sh rule (e.1), dprint eval-cache flake + `--no-verify` fallback criteria (d.2), styles.css churn + chosen root fix (e.4).
5. Fix styles.css churn at the root (see e.4 options) and stop hand-reverting.
6. Re-verify runbook §3 instructions against the `5127039da` tree: read `metaengine/projectionadapter/go.mod` + `sqliteengine/go.mod`, confirm strip-and-tag steps still hold with external `go-codec` requires; update the runbook mapping.
7. Commit this status report (done as part of this step).
8. Migrate `examples/middleware-demo` off deprecated `cqrshtmx.SecurityHeadersMiddleware` (SA1019 the hook keeps reporting) to `httputil.SecurityHeaders`.
9. Fix the 3 known `e2e/server` lint findings (exhaustruct on itemStore, G114 no-timeout serve, `bc` param name) — the only lint-dirty non-example module.
10. Pin `github/codeql-action/upload-sarif` to a commit SHA (hook finding, ci.yml:629) — same for other tag-pinned actions.
11. Add `deriver`, `Unparseable` etc. to codespell's ignore list (130 findings are dominated by domain-word false positives).
12. Run buildflow's `go-mod-normalize` (currently skipped in pre-commit) to clear the ~50 "direct and indirect requires mixed" warnings across go.mods.
13. Investigate the gomod-check errors "sub-module cqrs-htmx/v4 required but has no replace directive" in systemadapter + examples/system-demo — likely needs the family replace spelled per sub-module or a documented exemption.
14. Add `meta.mainProgram` to the flake package (flake-meta-checker finding).
15. Configure/silence the buildflow tsconfig-check, type-check, vulnix steps (they run bare `tsc`/`vulnix` with no args in this repo — tool misconfiguration noise).
16. Verify gopls workspace health after another restart; if the ~7k diagnostics persist, diagnose (b.3).
17. Draft the go-cqrs-lite post-extraction tag ask (event/v4.7+, command, query, id, record, system, metaengine family) so cqrs-htmx can migrate to external codec/retry/flightrecorder paths — prepare the migration plan (which cqrs-htmx files import `codec/v4`: usermgmt ×9, identity-model, dashboardui ×2, root indirect).
18. Add a CI job that builds in WORKSPACE mode (the exact class this session repaired — currently nothing catches dead replaces on CI).

**Blocked on user push (Q-answers):**
19. Push master (8 commits) + `datastar/v4.8.0` tag.
20. Strip the 2 datastar-local family replaces; hermetic tidy/build/vet each; verify proxy serves the tag.
21. Cut `totp/v4.8.0` + `oauth2/v4.8.0` no-op tags (if approved), then bump the 2 modules' family requires accordingly.
22. After totp/oauth2/webauthn all at v4.8.0: confirm buildflow gomod-check stops flagging family drift.

**Blocked on upstream go-cqrs-lite:**
23. Cut `metaengine/projectionadapter/v4 v4.5.0` + `metaengine/sqliteengine/v4 v4.0.2` (after item 6 verification).
24. systemadapter train: strip metaengine replaces (systemadapter + examples/system-demo), bump requires, cut `systemadapter/v4.8.0`.
25. External codec-path migration (after item 17's tags land).

**Carried backlog (from prior reports, still open):**
26. v4-branch blob rewrite (3 blobs, ~27.7 MB) — force-push decision.
27. Master-history blob purge (`5604e810`..`73ff1556`, 27 MB) — force-push decision.
28. Delete `backup/pre-blob-purge` branch (user decision).
29. auditlog viewer polish (SSE UI).
30. golines integration into `nix fmt`.
31. goldmark-based link checker upgrade for docs.
32. cqrs-lint Go-installable distribution for CI (P3.1).
33. `usermgmt` expiry on durable scheduling (go-cqrs-lite `scheduling`) instead of in-process timers.
34. dashboardui remaining templ-components adoption (Table, ToastContainer, EmptyState, PolledRegion).
35. loginpage adoption (recipes.AuthLayout, forms, Alert, Button).
36. 55 go-structure-linter root-package-files findings: document as intentional (flat root package is the library's public API) or plan v5 restructuring.
37. CHANGELOG strategy for the next cut (v4.8.1 patch vs v4.9.0 minor) once the 8 commits are pushed.
38. Sweep `docs/guides/leveraging-go-cqrs-lite.md` recipes against post-extraction reality (middleware module now requires external go-retry/go-flightrecorder).

## g) Questions (cannot figure out myself)

1. **Push scope (carried):** Please push master (now 8 commits: `f128072d`..`368dd717`) and the `datastar/v4.8.0` tag — and do you also want `totp/v4.8.0` + `oauth2/v4.8.0` cut now (no-op cuts at existing commits; they only exist to satisfy the coordinated-family version check), or left for the next family train?
2. **Upstream hygiene (carried):** File the issue/retraction for broken `stack/postgres v4.2.0` in go-cqrs-lite (compiles only via local replaces; v4.3.0 fixed it)? Retract, or leave with a comment?
3. **History rewrite (carried):** Accept the 27 MB blob in pushed master history, or authorize the force-push purge (rewrite `5604e810`.., re-cut the 10 family tags)? Same decision needed for the v4 branch (3 blobs, ~27.7 MB).

---

**State at close:** tree clean; 8 unpushed commits (`f128072d`, `e7aeee97`, `f6599348`, `af090852`, `b6020272`, `29bf201f`, `d63799c7`, `368dd717`); local tag `datastar/v4.8.0` unpushed; remote master `56068b16`. All gates verified this session green; 7 carried gates last-green at `b6020272`. WAITING FOR INSTRUCTIONS.
