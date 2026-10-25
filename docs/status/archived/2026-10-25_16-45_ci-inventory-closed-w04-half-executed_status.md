# Status — round 30: CI lint inventory closed + receipt written; W04 half-executed (2 of 7 tails landed)

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-25_18-28_round-31-w04-w05-w06-landed-ci-green_status.md`](2026-10-25_18-28_round-31-w04-w05-w06-landed-ci-green_status.md), which W04 finished and W05/W06 landed since; all open questions and next lists are carried forward in the round-31 report. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-25)).**

**Date:** 2026-10-25 16:45 CET · **Session:** resume of round 29 WAIT state
**Plan of record:** `docs/planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md` (W01–W17)
**Owner directive in force:** "READ, UNDERSTAND, RESEARCH, REFLECT → Execute and Verify → Repeat" / "GET SHIT DONE!"

---

## a) FULLY DONE (this session)

1. **CI triage of the red run `40203801203` (commit `5e0fb368`) — all 7 lint findings fixed, verified, pushed, CI GREEN.**
   The previous session's round-29 report said "CI queued, expectation green" — it was red. Three failing lint jobs, seven findings, ALL from my W03 work:
   - root `transport/serve.go`: gci (mis-indented `replayed` block), **gocognit 32 > 25** on `ServeDomainEvents` (the dedup seam grew the closure), 2× wsl_v5 (blank lines above `if`);
   - root `transport/serve_test.go`: wsl_v5 (no blank line above `go`);
   - `auditlog/auditlog.go:64`: wrapcheck on `Setup.Shutdown`;
   - `usermgmt/oauth2.go:64`: exhaustruct (state literal missing `nonce`), `service_oauth2_extracted.go:113`: golines (118-col interface method).
   **Fixes (root cause, not suppressions):** extracted `pumpLiveEvents` + `isReplayDuplicate` + `startHeartbeat` helpers out of the closure (complexity well under 25, dedup semantics unchanged — mutation-proven test still passes); own-line `//nolint:wrapcheck` passthrough on Shutdown (repo precedent: sentinel passthroughs stay unwrapped so `errors.Is(err, context.Canceled)` works); `nonce: ""` listed in the legacy Save literal; interface methods wrapped under 100 cols. Local per-module lint: **0 issues × 3 modules**; tests green. Daemon interleaved (banked the first 5 edits as `59fd9f0a` mid-flight); my commit `948825e5` carried the rest. **CI run `40205954025` on `948825e5`: SUCCESS.**
2. **RESUME-FIRST item closed: `usermgmt/CHANGELOG.md` S39 receipt written** (in `948825e5`). The root CHANGELOG pointer is no longer dangling. Receipt covers: service-seam empty-subject rejection (`usermgmt.oauth_no_subject`, covers custom providers), additive `NonceStateStore`, capability-detected OIDC nonce round-trip on the default path, pointer to `usermgmt/oauth2/CHANGELOG.md`.
3. **W04 findings re-verified in-source (all 7 S38 tails confirmed LIVE, none pre-fixed):**
   - S38a: `GenerateUserID` routed fresh ULIDs through SHA-256 (`SyntheticUserID`) — **destroys ULID time-ordering**;
   - S38b: `FoldMembership` reconstructs TenantID from raw payload with zero validation (fold.go:242); no strict `ParseTenantID`/`ParseBotID` exist;
   - S38c: `UserState`/`BotState` have no `IsActive`; `TenantState` has the split; `MembershipState.Exists` folds `Removed`;
   - S38d: `NewAuthz` seeds `g2` hierarchy edges unconditionally — a custom model without a g2 section fails (authz.go:59-66);
   - S38e: `ErrEmailVerificationNotConfigured`, `ErrTOTPNotConfigured`, `ErrOAuthNotConfigured` are **Transient (503)** — a missing config never becomes available by retrying (WebAuthn's is already Rejection);
   - S38f: 7 command accessors (`Roles` ×3, `Scopes`, `Secret`, `TokenHash`, `CredentialID`) return backing slices/arrays uncloned;
   - S38g: 3 internal uses of deprecated `NewUserID` (id.go:162 `ActorIDAsUserID`, authz_roles.go:113/124).
4. **W04 code landed so far (daemon-banked as `cbe3aea4`/`5a4674a1`/`de63a9f9`, build+vet+tests verified after the fact):**
   - **S38a:** `GenerateUserID` → `brandid.NewID[id.UserMarker](ulid.Make())` — timestamp preserved, ordering intact;
   - **S38g:** `ActorIDAsUserID` → strict `ParseUserID(actorID.Raw())` (malformed raw ⇒ `(zero, false)`); `RolesForActor`/`ImplicitRolesForActor` → strict parse with error propagation;
   - **S38b (parsers only):** `ParseTenantID`/`ParseBotID` + `parseBrandedComponent` helper added (strip exact `Tenant:`/`Bot:` display prefix; reject empty/whitespace-padded; `identitymodel.*.invalid` Rejection codes). **NOT yet wired into FoldMembership.**
5. **Test fallout from S38g fixed and pushed (`9bccf0cd`):** 4 authz actor tests seeded non-ULID subjects ("u1", "seed-user") that only matched via the deleted hash-masking path. `userActorPair()` now mints fresh ordered ULIDs; 21 call sites across 2 test files converted; identity-model suite green.

## b) PARTIALLY DONE

- **W04 (S38 a–g): 2.5 of 7 tails implemented.** Remaining: S38b fold wiring (`FoldMembership` → `ParseTenantID`), S38c (`UserState.IsActive` + `BotState.IsActive` additive; Membership documented as no-split-needed — see question 3), S38d (g2 guard: skip seeding when `model["g"]["g2"]` absent + custom-model test), S38e (3× NewTransient → NewRejection + family tests), S38f (`slices.Clone` on 7 accessors + mutation tests), plus the ordering test for S38a, the strict-parse test battery, and the identity-model CHANGELOG receipt.

## c) NOT STARTED

W05 (S42 systemadapter declarative fidelity), W06 (S46+S49 consumer seam proofs), W07 (S58 `WithSchemaVersion(1)` ×29 — blocked on W04), W08 (verify sibling's D02 receipts S45/S52/S53/S54), W10 (loginpage Playwright WebAuthn E2E + golden), W11 (five file splits past 350 lines), W12 (CHECKPOINT: full battery 4× rc=0, push, CI watch, verify-remote-tags, book reconciliation), W14/W15 (backlog I+II), W17 (FINAL battery + receipt superseding the 15-16 interim + CURRENT_FOCUS rewrite).

## d) TOTALLY FUCKED UP (honest ledger)

1. **The 7 lint findings existed at all.** Round 29 declared W03 "landed, CI green" on the strength of per-module TEST suites and covering CI runs — but I never ran the per-module LINT gate locally after the final edits. The sibling's push carried my daemon-banked files; master went red on MY debt. The cover fix is now mechanized habit: per-module `golangci-lint run` before "landed".
2. **Unverified edits left in the tree while the daemon is active.** My W04 strict-parse edits were banked before I ran the suite; 4 tests broke and a queued CI run (`40207861689` on `de63a9f9`) went red-bound. Caught locally within minutes and superseded by `9bccf0cd`, but master briefly had a red-bound tip. Lesson already in AGENTS gotchas — I violated it anyway: verify IMMEDIATELY, commit fast.
3. **CI watch background job failed silently-to-me:** the `gh run list --commit` lookup raced run registration (empty ID → 404). The verdict was only recovered by manual re-query. Should poll-until-registered, then watch.
4. **Minor:** helper-name typo (`parseBranedComponent`) shipped into a daemon commit (fixed by sed); one edit attempted without prior View (tool rejected, round trip wasted); W04 coding started before the fix-push CI verdict was confirmed (pre-push gates were green; verdict later confirmed success).

## e) WHAT WE SHOULD IMPROVE

- **Per-module lint is part of "landed"** — test suites alone are not the gate; the CI matrix runs lint per module and so must the session.
- **Daemon-aware edit discipline:** after ANY code edit, run build+test in the SAME working window; the daemon does not wait for verification.
- **CI watch helper:** poll `gh run list --commit <sha>` until the run exists, then `gh run watch <id> --exit-status`; never trust a single early lookup.
- **Strict-parse rollouts** (S38g class): grep test fixtures for non-ULID seed strings BEFORE pushing the strict path — the fallout class is predictable.

## f) NEXT (up to 50, priority order)

**W04 finish (now):** 1) wire `ParseTenantID` into `FoldMembership` + fold-rejection tests; 2) `UserState.IsActive` + `BotState.IsActive` + tests; 3) MembershipState no-split doc note; 4) g2 guard in `NewAuthz` + custom-model-without-g2 test; 5) 3× NotConfigured → Rejection + family tests; 6) `slices.Clone` ×7 accessors + mutation tests; 7) S38a ordering test (2ms-apart ULIDs strictly increase); 8) ParseTenantID/ParseBotID accept/reject vectors; 9) idm CHANGELOG receipt; 10) `.#test-identity-model` + lint + grep-clean `NewUserID` internal uses.
**Then:** 11) W05 S42a Enforce-ladder parity test (declarative vs legacy grants — the silent-widening risk); 12) W05 S42b audit Action vocabulary parity (`usermgmt.AuditAction*`); 13) W05 S42c ActorID into declarative audit entries; 14) systemadapter CHANGELOG; 15) W06 S46a DocsHandler/DocsPath end-to-end mount test; 16) W06 S46b `RecorderChain(Recorder(svc), auditPlugin)` proof; 17) W06 S49a SQL snapshot `StateShape` round-trip test; 18) W06 S49b codec round-trip verdict (file upstream if stamp drops); 19) W07 `WithSchemaVersion(1)` across 29 constructors; 20) W07 delete health status mirrors + root require bump; 21) W07 receipts (idm + health); 22) W08 verify sibling's S45/S52/S53/S54 receipts against code; 23) W10 Playwright WebAuthn virtual-authenticator harness; 24) W10 registration+login E2E through the rendered page; 25) W10 nonce-insensitive full-page render golden; 26) W11 split `setup/config.go`; 27) W11 split `systemadapter/declarations.go`; 28) W11 split `usermgmt/service_core.go`; 29) W11 split `dashboardui/layout.go` + `e2e/server/main.go`; 30) W12 full battery (`.#test`/`.#lint`/`.#coverage-gate`/`.#check-modules -- --report`) 4× rc=0 ≤2 attempts; 31) W12 push + `gh run watch` + `.#verify-remote-tags` + ahead-0; 32) W12 `git log --stat` book reconciliation vs sibling's 15-16 receipt; 33) W14 loginpage AutoRedirect follow-through; 34) W14 dashboardui 09-48 tails; 35) W14 root declutter; 36) W15 art-dupl r3; 37) W15 skill-claims skeleton; 38) W15 canaries; 39) W15 upstream ask drafts; 40) W17 FINAL battery + receipt superseding 15-16 interim (archive ritual + both baselines); 41) W17 CURRENT_FOCUS rewrite; 42) watch CI verdict for `9bccf0cd` (run on tip, queued at report time); 43) re-check `40207861689` final state (superseded/cancelled expected).

## g) QUESTIONS (cannot be answered from the repo)

1. **AccentColor grammar (carried from round 29):** ratify the shipped reject-at-`New` posture (invalid accent ⇒ construction error), or revert to sanitize-with-warning (~15 min)? Default if unanswered: keep reject-at-`New`.
2. **OIDC nonce promotion policy (carried from round 29):** keep the additive capability-detection design permanently, or promote nonce into the base `OAuth2Provider`/state-store interfaces on the next coordinated train (breaking, version-bumped)? Default if unanswered: keep additive.
3. **S38c interpretation:** the plan says "Exists()/IsActive split per state struct". Published-module stability mid-wave (no version bump; usermgmt deciders depend on current `Exists()` semantics — e.g. deleted tenants must stay `!Exists` in `es_tenant_decide.go`) forces the ADDITIVE reading: keep every `Exists()` semantic EXACTLY as-is, add `IsActive()` to `UserState`/`BotState`, document `MembershipState` as needing no split (`Exists` already folds `Removed`; there is no exists-but-inactive membership state). Ratify the additive reading, or do you want the full semantic split (`Exists` = ever-created) accepted as a behavior change riding the next train? Default if unanswered: additive.

---

**Commits this session (chronological):** `59fd9f0a` (daemon, banked lint fixes 1–5) → `948825e5` (lint inventory closed + usermgmt S39 receipt; CI `40205954025` SUCCESS) → `cbe3aea4`/`5a4674a1`/`de63a9f9` (daemon, banked W04 S38a/S38b-parsers/S38g) → `9bccf0cd` (S38g test fallout fixed; CI queued at report time; supersedes red-bound `40207861689`).
**Sibling lane (observed, not mine):** `258cfe50` owner-status receipt (docs lane closed, master-red-on-lint noted — now fixed by `948825e5`), plus their test-format tweaks banked alongside my W04 files.
