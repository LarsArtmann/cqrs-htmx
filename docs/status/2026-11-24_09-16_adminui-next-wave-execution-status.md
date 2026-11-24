# Status — Adminui Next-Wave Execution (Pareto Plan T01–T32)

**Date:** 2026-11-24 09:16 · **Author:** Crush · **Session window:** 06:31 → 09:16
**Task:** execute `docs/planning/2026-11-24_06-31_SUPERB-adminui-next-wave-pareto-plan.md` (the 100-idea adminui audit turned into 32 coarse / 229 fine tasks) via concurrent subagents, per the owner directive ("spawn subagents, keep 3 concurrent, do not stop until finished and verified").
**Repo state at report time:** master ahead of origin by **53 commits** (push BLOCKED — see §d/f), working tree clean, adminui race suite green, adminui lint 0 issues.

---

## a) FULLY DONE (implemented + verified this session)

| Task | What landed | Verification receipt |
| --- | --- | --- |
| **T01 honest auth-state surface** | `authMethods` struct from Service probes (`HasWebAuthn`/`HasTOTP`/`HasOAuth2`/`ConfiguredOAuth2Providers`); Passkeys/External cards + stat tiles capability-gated (dead cards never render); "Cannot sign in" badge + blast-radius copy when no credential AND no external AND no TOTP; `userDetailDefinitionItems` refactor; supervised-enrollment design note (`docs/decisions/2026-11-24_adminui-supervised-enrollment-note.md`, feeds T02 packet) | `user_auth_state_test.go` (253 lines: probe snapshot, 4-row truth table at helper + render level, badge-clears-after-real-AddCredential); `user_lifecycle_test.go` dead-card absence pins; `oauth_views_test.go` gate flipped; suite green ×2 (`ok … 9.8s/9.7s -race`) |
| **T02 decision packet** | `docs/decisions/2026-11-24_adminui-d1-d3-and-upstream-asks.md`: verbatim D1–D3 + (h)(1)–(5) quotes; 8 API sketches (A `RevokeAllSessions`, B `SessionsForUser`, C admin TOTP bypass, D `LockoutStatus`, E bot listing, F `RenameTenant`, G batched member counts, H tenant-attributed audit) each with file:line evidence; recommendation blocks with dismissed alternatives; TODO_LIST pointer appended to the D-row | Every ask verified against usermgmt/identity-model source; two upstream premises CORRECTED: F rename vocabulary already exists in identity-model (decide/dispatch/Service glue missing), D lockout is an interface not UserState fields; `check-todo-hygiene` clean (131 rows, 0 violations) |
| **T06 dashboard quick actions + Config.PageSize** | Quick-actions row (Create user / Create tenant) inside the polled region — page and 30s stats partial keep parity BY CONSTRUCTION; super-admin only, tenant-admin renders nothing; linked to the EXISTING `/users/new` + `/tenants/new` routes (no route changes needed — investigated first); `Config.PageSize` (default 50, clamped 10..200, clamped-not-rejected) adopted at all `pageBounds` call sites | `TestDashboardQuickActions_ModeGate` (both surfaces, both modes), `TestConfig_PageSizeClamped` (7 boundary cases), `TestPanel_PageSizeShapesList` (11 users → 2 pages at 10); suite green |
| **T09 audit filter depth** | Inclusive from/to date range (YYYY-MM-DD, whole-day bounds, server-local); actor-kind chips (user/bot/system/service); honest issue strips (malformed bound names its input + ignores that half; reversed range announced + all rows kept); empty state distinguishes "nothing recorded" from "nothing matches filters"; retention caption from the window's own facts (holds/limit/dropped); full filter preservation through pager/chips/HTMX swaps | `audit_filters_test.go`: range boundary matrix, issue+ignore semantics, chip degrade, state round-trip, partial swap, caption truth |
| **T10 audit intelligence** | Per-user activity load-more: `GET /users/{id}/activity?skip=N` (route-table row, super-admin posture, manifest golden updated); fragment replaces its own trigger row (appends in place); honest "All N events in the audit window are shown." end note; cursor tampering degrades to 0. Diff renderer (`auditDiff`: old struck, new bold) shipped as the single display source — currently renders the honest "values not recorded" note because the audit projection carries NO old/new payload (verified: `AuditEntry.Detail` holds only login-failure classes; payloads hold only NEW values; panel seam has no journal access). Correlation column: data-blocked (entries carry no correlation IDs) — honestly skipped | Full-stream coverage test (card 8 + fragment 3 = all 11 events exactly once), exhausted/mid-stream/overrun cursors, tampered-cursor degrade, invalid-id 400, fragment-not-page swaps, no-fabricated-diff pin |
| **T09+T10 completion + hardening (done by the orchestrator, not an agent)** | Fixed the crashed agent's broken tree (nil-string compile error, stale generated templ, undefined fragment references); wrote the ENTIRE missing test file; fixed all 9 golangci findings to **0 issues** (cyclop 13→ via extracted `validateAuthorizerPosture`; exhaustruct ×5 exhaustive literals; gochecknoglobals → `auditActorKinds()` func; goconst → `labelEmail` const; gosmopolitan → `time.Now().Location()` with explanatory comment); CHANGELOG bullets ×3 | `go test ./adminui/... -count=1 -race` → ok (10.3s); `golangci-lint run` → **0 issues**; gofmt clean |
| **T26 demo seeds** | `examples/admin-demo/seed.go` (266 lines): 2 bots with show-once token logging (+`TokenPepper` config the pin requires); passkey user via real `AddCredential` + TOTP enroll with the demo playing the authenticator (RFC 6238, ~15 lines); suspended + deleted tenants; REAL `login_failed`/`login_succeeded` audit events via actual wrong/right TOTP verifies; role-change seeds; locked-out user (mallory, zero channels); deterministic cast | `seed_test.go` (162 lines) green in workspace AND `GOWORK=off` hermetic build against the published pins (root-tag-gap check); hermetic binary smoke-ran clean; honest skips documented: `LinkExternalAccount` does not exist at pin v4.21.0 (only Unlink), `roles_updated` legacy event has no emitters at pin so role changes don't appear on the audit page (README + test both say so) |
| **T28 e2e expansion** | WebKit project corrected (launch-wrapper bug: `MiniBrowser` shell script clobbers `LD_LIBRARY_PATH` — config now execs the real ELF with derived `WEBKIT_EXEC_PATH`/bundle/merged libs); `e2e/webkit-forest.sh` (reproducible NixOS launch-libs forest from the flake.lock pin, shellcheck clean, 4 live findings documented); provisioning via the repo's OWN pinned driver (webkit-2388, re-run = clean no-op, no copied trees); `admin-csrf.spec.ts`: token-less POST → 403 with `maxRedirects: 0`, refused tenant proven absent, paired success (303) + real in-page fetch variant; CI advisory leg verified in place (`continue-on-error: true`, own server set per gotcha 59); e2e README WebKit section rewritten | **First-ever webkit-green run: 10 passed (27.8s), reproduced 10 passed (29.6s)** from a fresh forest; CSRF spec chromium 2 passed with the LIVE server-log refusal line captured (`WARN httputil: CSRF validation failed …`); chromium re-verified 10 passed after config edits; WebKit stays advisory per K-T28 |
| **T29a docs truth pack (part 1)** | adminui/README.md: 46-row route table (programmatically verified 1:1 against `routes.go` — 42 super-admin rows match the manifest golden, 15 served in tenant mode), gate legend + QUERY/override placement + freshness paragraph naming the enforcing tests; Security section (setup's `session(csrf(panel))` composition + `Config.CSRF` knob, `KeyedRateLimiterMiddleware` recipe, cookie-flag truth: what usermgmt's stack already sets vs hand-rolled guidance); Server-Timing paragraph (panel emits none; go-appkit emits `total;dur` under `RunWithAppkit`, tri-state knob, self-hosted alternative) | Route counts stated and matched (46 = 46); every claim verified in source; F213 audit verdict: **no dishonest empty last-seen rendering exists** (all IsZero-guarded with honest dashes); flagged adjacent unguarded `RelativeTime` renders (CreatedAt/OccurredAt would render "Jan 1, 0001" if ever zero) — listed as follow-up |
| **T31 gated-surface UI specs (paper)** | `docs/decisions/2026-11-24_adminui-gated-surface-ui-specs.md` (474 lines): §0 panel conventions (verified: confirm modal, CSRF meta flow, route-table contract, T01 honesty rules); impersonation kit (D1-gated, reason field, acting-as banner, self-refusal); bots pack (show-once reveal contract, rotate composed from published Register+Delete — NOT a new ask); sessions + MFA-bypass (mandatory reason + audit disclosure) + lockout badge in T01's amber family; tenant rename/counts/attribution both-variants; full A–H/D1–D3 mapping table; every dependency labeled `gated: needs <ask>`; one pointer line appended to the T02 packet | Paper only — zero code, zero unpublished-API assumptions; item 74 declared NOT covered (needs an API the packet does not sketch) |
| **Phase 0 (plan + baseline)** | Plan file + CORPUS_BASELINE (66291→67696) committed (daemon swept them within minutes — verified content in `2b5b05f3`/`e8ddc7f8`); push attempted → BLOCKED by the pre-push release-train gate (33 fleet train lags — see §d) | Gate output captured; fix recipe recorded (3 bump-dep sweeps) |

**Coarse-task score: 9 of 32 fully done (T01, T02, T06, T09, T10, T26, T28, T29-part, T31).**
**Fine-task score: ~57 of 229 (25%).**

## b) PARTIALLY DONE

1. **T29 docs truth pack** — F209/F210/F212/F213-audit done; F211 (DashboardURL deep links + health strip) deliberately deferred until T14 (ReadOnly) and T24 (health strip) land so the docs don't lie; the optional members.templ zero-dash `title` nit from the F213 audit is unfixed.
2. **The wave overall** — 9/32 coarse done; the rest queued or in flight (see §c).
3. **Push** — 53 commits ahead, hard-blocked by the release-train gate (see §d1).
4. **T28 WebKit promotion** — advisory green ×2; blocking promotion deliberately NOT attempted (K-T28 requires firefox-class evidence first).

## c) NOT STARTED (queued, dependency-ordered)

- **In flight RIGHT NOW (dispatched, reports pending):** T03+T04+T05 users-list cluster (sort + pager windowing + filtered exports) and T27 gates/test housekeeping (coverage_gaps splits + `check-adminui-surface` gate; CI wiring deferred to a follow-up step).
- **Queued:** T07+T30 (per-render caches + render benchmark), T11 (dashboard intelligence), T12 (member-safety batch), T13 (global membership directory), T14 (security posture: ReadOnly, authztest.Suite, header assertions), T15 (a11y completion), T16 (i18n seam), T17 (client pack A), T18+T20 (command palette + saved views), T19+T24 (bulk actions + live updates), T21+T22 (tenant pack + ownership transfer), T23 (user-detail pack 2), T25 (visual pack), T29b (F211 docs), T32 (receipt battery + true-ups).
- **Never dispatched:** the family alignment pass (train-lag fix) — deliberately deferred from early-session to avoid go.mod churn colliding with agent waves; now a blocking debt for push.

## d) TOTALLY FUCKED UP (honest failure ledger)

1. **Push is blocked and stayed blocked all session.** The pre-push hook runs `check-release-train --strict-lag 0`: 33 fleet train lags (go-codec v0.3.3→v0.3.4 across ~24 modules, go-atomic-write v0.6.2→v0.7.0 ×4, go-legalparty v0.2.0→v0.3.0). This is PRE-EXISTING drift (other repos published newer tags), not caused by this session — but I deferred the alignment sweeps to the end, so every push attempt failed. Consequence: 53 unpushed commits.
2. **Provider rate limits wrecked the 3-concurrent directive.** ~10 agent crashes across the session (`alejandra-t01`, `mindy-t06`, `benedict-t28`, `candy-t28`, `amy-t28`, `xochitl-t09-t10`, `benoit-t09-t10` mid-work, `alexander-t27`, `jacopo-t27`, plus one earlier). Three concurrent glm-5.3-flash agents reliably exhausted the provider quota; even SINGLE agents died mid-work. I dropped to 1–2 concurrent and retried; every crashed task eventually landed, but the wall-clock cost was heavy.
3. **The worst crash left adminui NON-COMPILING.** `benoit-t09-t10` died after 70 tool calls with: a nil-as-string compile error, stale generated templ (undefined `auditPanelContent`/`userActivityFragment`), and zero tests. I stabilized by hand (typo fix, regen, full test file, lint to zero). Lesson absorbed: after ANY crash, verify `go build` BEFORE dispatching completion agents.
4. **Partial-work detection missed daemon-swept state.** The first crashed T28 attempt's `playwright.config.ts` edit was already daemon-committed, so my working-tree-only check called the tree "clean" and the completion agent started with a mystery. Detection must include recent daemon commits, not just `git status`.
5. **The auto-commit daemon ate two authored commit messages.** The T31 batch and the T09/T10 batch landed as `chore: auto-commit N changed file(s)` — the daemon swept my staged files between my `git add` and `git commit` (and once even between `git status` and `git add` in the same command). Content is all committed and verified; the detailed messages are lost to history.
6. **One self-inflicted edit error.** A multiedit on `audit_filters_test.go` replaced the actor-kind-presence block with a broken loop (out-of-scope variable) — caught on the next test run and fixed within one edit. No damage escaped the tree.
7. **Not yet run: the heavy repo gates.** `check-modules`, `coverage-gate` (adminui ≥66% ratchet unverified), the full chromium e2e battery, docs-gates battery, erraudit advisory sweep — all deliberately scheduled for T32; the wave is NOT yet "verifiably no worse" end-to-end.
8. **User directive not fully honored:** "3 concurrent at all times" was physically impossible against the provider's rate limiter; 2 concurrent also died repeatedly. Final working pattern: 1–2 concurrent + immediate retries.

## e) WHAT WE SHOULD IMPROVE

1. **Run the family alignment pass EARLY** (bump-dep sweeps are go.mod-only and don't collide with adminui source work) so pushes aren't blocked all session.
2. **Single-stream dispatch as default** for this provider; concurrency only across provably crash-isolated tasks (docs-only, e2e-only).
3. **Post-crash protocol:** check tree AND `git log --stat -5` for daemon-swept partials; run `go build ./adminui/` before any completion dispatch; include that build check in every agent's prompt preamble.
4. **Agents should write tests BEFORE the templ-regen step** — benoit died with implementation but no tests; test-first ordering preserves more value per crash.
5. **Commit authored work the instant an agent lands** — and accept that the daemon may still win; prefer per-task `git commit -F <file>` messages prepared in advance to minimize the race window.
6. **Lint discipline belongs in every agent prompt** (this session proved it: T09/T10 landed 9 lint findings I had to fix by hand; T03-05's prompt now carries the exact linter constraints).
7. **Keep the honest-skip culture** — T26's pin-limit skips and T10's data-blocked verdicts are exactly right; every skip names its evidence and lands in CHANGELOG/README so docs never lie.
8. **Gate noise:** BuildFlow pre-commit emits ~150s of ANSI noise per commit in this transcript — fine for hooks, hostile to orchestration logs; consider `--quiet` for orchestrator-driven commits.

## f) NEXT 50 (priority-ordered; ✱ = needs owner input or another session's work)

1. Verify + commit T03+T04+T05 when celeste reports (users sort, pager windowing, filtered exports).
2. Verify + commit T27 when andre reports (test splits, surface gate, flake wiring).
3. Apply T27's deferred CI step diff to `.github/workflows/ci.yml`.
4. T07 per-render caches (GroupPolicies batch with the gotcha-45 hierarchy filter, AllTenants map, lastSeen TTL cache, TenantMembers batched counts) + kill-point K-T07 discipline.
5. T30 render benchmark (200-user page, `b.Loop`, machine-pinned baseline, idle machine).
6. T11 dashboard intelligence (Change/Trend cards, sparkline, login-failure card, adoption cards, operators + bus-factor).
7. T12 member-safety batch (role distribution, last-admin DOWNGRADE guard, suspended banner + reactivate).
8. T13 global membership directory (`/members`, super-admin, mode-scoped nav).
9. T14 security posture (`Config.ReadOnly`, authztest.Suite over the route table, header assertions).
10. T15 a11y completion (axe sweeps, contrast pass shrinking the ~115-finding ACCEPTED ledger, aria-live counts, focus restore, keyboard nav).
11. T16 i18n seam (labels table, `Config.Labels`, lang/dir, locale dates, reduced-motion guards).
12. T17 client pack A (skeleton shimmer, client-side sortable, unsaved-changes guard, copy-row-JSON).
13. T18 command palette (Ctrl/Cmd+K, `?` help).
14. T19 bulk select + actions (resend/export/grant v1, per-row outcome honesty, CSRF + confirm).
15. T20 saved views + prefs (localStorage presets, column visibility, density).
16. T21 tenant surface pack (slug suggest, role help, tombstone read-only view, roster CSV + pagination, switcher search).
17. T22 ownership transfer (two durable commands under one confirm, mid-sequence failure honesty).
18. T23 user-detail pack 2 (OAuth card enrichment, verification-TTL, self-service card, credential restore).
19. T24 live updates (stats ETag/304, SSE-triggered refresh, idiomorph adopt-or-close verdict, hx-boost closure).
20. T25 visual pack (favicon, sticky theads, avatar color hash, recently-active, print sheet, skip link).
21. T29b: F211 docs (DashboardURL deep links + health strip) — after T14/T24.
22. T32 receipt battery: CHANGELOG sweep, DOMAIN_LANGUAGE additions, coverage ratchet re-true, full battery, corpus/tail baselines.
23. Family alignment sweep: `bump-dep 'larsartmann/go-codec$' v0.3.4 --commit --lint`.
24. Family alignment sweep: `bump-dep 'larsartmann/go-atomic-write$' v0.7.0 --commit --lint`.
25. Family alignment sweep: `bump-dep 'larsartmann/go-legalparty$' v0.3.0 --commit --lint`.
26. Re-run `check-release-train --refresh-cache --strict-lag 0` → green.
27. PUSH (finally), then `push-and-watch` + `verify-remote-tags`.
28. Fix the members.templ zero-dash `title` nit (T29a's only actionable F213 finding).
29. Guard the adjacent unguarded `RelativeTime` renders (CreatedAt/OccurredAt IsZero) that T29a flagged.
30. Append the F063 finding to the T02 packet: diff extraction needs audit payload widening (strengthens ask H's evidence).
31. Full chromium e2e battery after the template-heavy waves; `git restore` byte-churn on screenshot PNGs unless the render changed.
32. `nix run .#check-modules` after each wave (surface gate, CSS gates, shellcheck, conflict markers).
33. `nix run .#coverage-gate` — verify the adminui ≥66% ratchet holds after T27's test splits.
34. `check-css-bundles` + `check-css-bundle-classes` — confirm no agent wave introduced phantom/stale tokens (no CSS rebuild has been needed so far; keep it that way or rebuild in the same change).
35. Scoped `nix fmt` over agent-edited files (treefmt roundtrip discipline).
36. erraudit advisory re-run for adminui (new code adds findings; the fleet hold window closes 2026-12-21).
37. cqrs-lint per-module re-run for adminui (`--scope=module --strict`).
38. Append per-task receipts to the plan's § Execution log (T01–T28 entries, one block each).
39. Corpus baseline re-true after T32's DOMAIN_LANGUAGE additions (same-commit rule).
40. Tail baseline bump for new status reports (same-commit rule).
41. AGENTS.md adminui section true-up once the wave ships (auth-state surface, audit filters, activity route, PageSize, quick actions).
42. Verify the activity route's tenant-mode absence in the manifest (mode-scoped rows).
43. e2e admin specs for users sort/windowing/export after T03-05 lands.
44. ✱ Decision packet hand-off: D1–D3 + asks A–H await owner signatures; T31 specs are ready the day they land.
45. ✱ WebKit blocking-promotion evidence path (firefox-class 3-clean-runs rule) once advisory holds across runs.
46. Watch co-tenant usermgmt churn (`es_decide_merge.go` +385 lines landed mid-session from another session) — re-verify adminui's Service usage compiles after their next sweep.
47. `go mod tidy` the 30 stale modules flagged by BuildFlow preflight (or `nix run .#deps`).
48. VACUUM the 1.09 GB BuildFlow state DB (recurring preflight warning).
49. `nix-hash-fix` for the stale vendorHash FOD (recurring preflight warning).
50. Final owner-facing TABLE report (Pareto tiers + Table A + commit/push receipts) once the wave + alignment + push complete.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Push vs plan ordering:** pushes are hard-blocked on 33 fleet train lags; the fix is 3 bump-dep sweeps (go-codec, go-atomic-write, go-legalparty) that touch every module's go.mod and could surface real API breakage. Should I run the alignment sweeps NOW (mid-wave, pausing new dispatches while their lint legs run), or finish the plan first and leave the branch unpushed until you run the alignment?
2. **The auto-commit daemon** swallowed two authored commit messages this session (content landed fine). Do you want me to keep racing it, or can you pause it for the remainder of the wave so per-task detailed messages land?
3. **Owner gates D1–D3 + asks A–H** are packeted with verified evidence and matched UI specs. Do you want to sign any of them this session (which would un-block the gated implementation tasks), or should the wave finish exclusively on non-gated work and leave the packet for your review?

---

**Verification receipts for this session:** adminui `go test ./adminui/... -count=1 -race` → ok ×6 runs · `golangci-lint run` (adminui) → 0 issues · e2e webkit smoke 10/10 ×2 · e2e chromium 10/10 + CSRF spec 2/2 · hermetic examples build (GOWORK=off) green · check-todo-hygiene clean · route manifest golden green (activity route) · corpus gate green at each bump.
