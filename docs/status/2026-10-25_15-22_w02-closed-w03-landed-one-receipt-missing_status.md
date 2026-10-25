# Round 29: W02 closed + pushed, W03 fully executed (landed via daemon commits, CI green), one usermgmt receipt missing at interrupt

**Created:** 2026-10-25 15:22 CET · **Session:** continuation of the module-code lane (10-42 plan W01–W08, W10–W12, W14–W15 + W17 battery/receipt; sibling per the `824b3dba` addendum owns W09/W13/W16 + verdicts). **Supersedes:** [`2026-10-25_13-30_w01-s37-closed-ci-red-fixed-w02-half_status.md`](archived/2026-10-25_13-30_w01-s37-closed-ci-red-fixed-w02-half_status.md) (archived same-commit; tail stays 2). **State at writing:** tree clean, origin at `471ab2b2` (sibling's 15:16 receipt on top of my daemon-banked W03 fragments); CI green on `0a527b8e` (my G124 fix, run `40197779843`) and on `3ff9e0de` (run `40198848723`); the tip commit's run may still be in flight at wait time.

## a) FULLY DONE (this session, receipts)

1. **CI triage + gosec G124 fix, pushed green (`0a527b8e`).** Both red runs from the wait (`a9e7d64c`, `1921e57c`) root-caused to three failures: (i) shellcheck SC1087/2068/2034 in the sibling's treadmill script — ALREADY fixed by their daemon commits (`2819e22d`/`e8442fda`), verified locally with a clean full-tree shellcheck pass; (ii) treadmill self-test T5 — already green at HEAD (23/23); (iii) **mine**: gosec G124 on the loginpage theme-cookie pin test (`handler_test.go:373`). Fixed with the repo's scoped `//nolint:gosec` pattern (rationale: simulates an incoming request cookie, never a Set-Cookie), golines-wrapped after the first form tripped the line-length lint. loginpage suite + vet + hermetic lint green; pushed through both strict pre-push gates.
2. **W02 CLOSED — S36 UI seams, adminui half done this session (`4773044e`, `bb0b5dbc`, `c57cc36d`, pushed).**
   - **w02.3 AccentColor validated at `New`** (owner Q3 defaulted to reject-at-`New`, fail-fast, assumption noted in the CHANGELOG): new `adminui/accent_color.go` ports dashboardui's exact CSS-color grammar (#hex 3/4/6/8, numeric rgb()/rgba()/hsl()/hsla(), named colors + transparent, 32-char cap); `withDefaults` rejects outside-grammar values with the same message shape as dashboardui. Pinned by `TestConfig_AccentColor_RejectsAtNew` (incl. `red;} body{background:url(x)}` injection vector), `...AcceptsGrammar` (incl. `#0ea5e9`), and `...DefaultStillValid` (guards the default against its own grammar). README table updated to the honest contract.
   - **w02.4 partial-render errors answer 500**: `renderPartial` (render.go) logged render errors but left a truncated 200 that htmx swaps into the DOM silently; error path now mirrors `renderPage`'s explicit 500. The STALE test pinning the buggy contract (`TestRenderPartial_RenderError` asserted "writes nothing") was updated in place to assert the 500 — one home per contract, my duplicate render_test.go deleted (`c57cc36d`). dashboardui checked: has no renderPartial twin, adminui-only defect confirmed.
   - **w02.5 receipts**: loginpage CHANGELOG gained the S36 NewPageData nonce/theme receipt (code from the prior session); adminui CHANGELOG gained both S36 entries. Flake suites `.#test-loginpage` + `.#test-adminui` rc=0, goldens stable, hermetic vet green.
3. **W03 EXECUTED END-TO-END — all six S39 tails fixed, tested, and landed (daemon-banked: `b31cc915`…`5f47d846`; CI green on the runs that cover them).**
   - **w03.5 datastar** (`event_bridge.go`): `NewEventBridge(nil)` panics at CONSTRUCTION with a message naming the fix (was: nil-deref on first mapped event); `MappedEventTypes` sorts; the lying test (`ReturnsSortedCopy` asserting only `Contains`) now asserts exact order; `TestNewEventBridgeNilBroadcasterPanicsAtConstruction`. Suite green.
   - **w03.7 totp** (`provider.go`): `Window <= 0` clamps to 1 — `ValidateCode` widens via `uint(Skew)`, so -3 silently became a ~4-billion-step (multi-year) skew. `TestNewNegativeWindowClampedToDefault` probes -1/-3/-100 + end-to-end current-code validation. Suite green.
   - **w03.6 oauth2 empty-`sub` + OIDC nonce** (`usermgmt/oauth2/provider.go` + `usermgmt/service_oauth2_extracted.go` + `oauth2.go`): (i) `exchangeAndExtractUser` rejects a provider returning neither `sub` nor a usable `id` (`oauth2.empty_subject`, Rejection) — the external-account identity key can no longer degrade to email-only matching; defense-in-depth check in usermgmt's `completeLogin` guards custom OAuth2Provider implementations (`usermgmt.oauth_no_subject`). (ii) **OIDC nonce as an ADDITIVE pair**: `BeginLoginWithNonce` (embeds nonce in the auth URL, OIDC Core §3.1.2.1) + `FinishLoginWithNonce` (rejects mismatched OR missing nonce claim, `oauth2.nonce_mismatch`); usermgmt's `OAuth2Service` type-asserts the capability + the new `NonceStateStore` (implemented by the default in-memory store: `SaveWithNonce`/`ConsumeWithNonce`) so the DEFAULT path runs the full nonce round-trip while custom providers/stores keep the PKCE-only baseline; both FinishLogin paths share one `completeLogin` tail (no duplication). Five oauth2-module tests (URL embedding, match, mismatch+missing, both empty-sub paths) + one usermgmt test. Both suites green hermetically.
   - **w03.4 health** (`probe.go`): `Recorder(nil)` panics at construction mirroring `NewProbe`'s reject (was: nil-deref at first health batch); unknown projection statuses worded honestly (`projection_unknown_status`, "treated as not-ready") instead of diagnosed "catching up" — known drain states (idle/running/backoff) keep the draining wording. Two pin tests. Suite green.
   - **w03.3 auditlog** (`auditlog.go`): `Setup.Shutdown(ctx)` + `Setup.SignalComplete()` passthroughs give the documented MOUNTED viewer mode a teardown story (flush forwarder + drain broadcaster — verified to work WITHOUT ListenServeDelist; upstream v0.11.0's Shutdown was already mounted-mode-capable, the gap was only the missing passthrough + docs, doc comment carries the call sequence). `TestSetup_ShutdownWorksInMountedMode`. Suite green.
   - **w03.1 transport replay/live dedup** (`serve.go`): the replay pass now records the IDs it writes (post-truncation, both the plain and filtered store wrappers) and the live pump skips exactly those — subscribe-before-replay correctly buffers live events during the replay window, but pre-fix an event published in that window was delivered TWICE. **Mutation-proven**: disabling the skip makes `TestServeDomainEvents_ReplayLiveSeamNoDuplicate` fail (gated-store harness broadcasts into the live channel mid-replay); restored, it passes. Suite green.
   - **w03.2 unknown-cursor semantics**: probe already existed and pins the journal store's unknown-but-valid cursor = full capped backfill (`TestJournalSSEStore_EventsAfterNotFound`); the missing piece was the CONTRACT doc — `ServeDomainEvents` now documents the store-defined Last-Event-ID behavior and the "only send IDs the same store issued" rule.
   - **w03.8 receipts (5 of 6)**: root CHANGELOG (transport + health/auditlog + usermgmt-pointer entries), `datastar/CHANGELOG.md`, `usermgmt/totp/CHANGELOG.md`, `usermgmt/oauth2/CHANGELOG.md` (Fixed empty-sub + Added nonce pair), all landed in `5f47d846`.

## b) PARTIALLY DONE

1. **usermgmt/CHANGELOG.md S39 receipt — MISSING (the one thing the interrupt cut off).** The usermgmt code changes (empty-subject defense + `NonceStateStore` + nonce-capable service flow) are landed and green, but their module-level CHANGELOG entry was not yet written when the interrupt arrived (verified: zero `empty_subject`/nonce mentions in the file). The root CHANGELOG's pointer entry says "receipts also in usermgmt/CHANGELOG.md" — that pointer is currently a dangling promise. Two-minute fix, first action on resume.
2. **W03 verify is per-module, not battery-grade.** Each touched module: suite + hermetic vet + (where applicable) lint green. NOT yet run: the full W12-class battery (`.#test`/`.#lint`/`.#coverage-gate`/`.#check-modules -- --report`), which is the plan's own kill-criterion checkpoint and also covers coverage thresholds (auditlog/health are 100%-gated; new code paths are tested but the gate number is unverified).
3. **Tip-commit CI verdict unwatched.** My W03 fragments landed in daemon commits between `3ff9e0de` and `471ab2b2`; the latest COMPLETED run (`40198848723`, head `3ff9e0de`) is green but predates `5f47d846` (my transport test + CHANGELOGs). The tip's run was not yet listed at wait time — needs a `gh run list` check on resume (shellcheck on the new test file is the main residual risk; it was gofmt'd but not shellcheck-relevant, so expectation is green).

## c) NOT STARTED (my lane, per the 10-42 plan)

- **W04 (S38 identity-model tails)** — ordering, strict parses, Exists/IsActive, g2-guard, NotConfigured family, accessor copies, kill internal NewUserID, idm CHANGELOG. Requires re-verifying each `file:line` first (some S38 tails may already be fixed).
- **W05 (S42 systemadapter declarative fidelity)** — Enforce ladder, audit vocabulary parity, ActorID into declarative audit.
- **W06 (S46+S49 consumer seam proofs)** — DocsHandler mount E2E, RecorderChain proof, StateShape round-trip.
- **W07 (S58 schema-version stamping)** — `WithSchemaVersion(1)` across 29 constructors (depends on W04).
- **W08 (D02 verification)** — verify the sibling's S45/S52/S53/S54 receipts (their docs lane).
- **W10 (S57 loginpage Playwright E2E + render golden)**, **W11 (S40 five file splits past 350 lines)**.
- **W12 CHECKPOINT** — full battery 4× rc=0 + push + CI watch + `.#verify-remote-tags` + book reconciliation (kill criterion: two attempts or STOP).
- **W14/W15 (backlog waves I+II)**, then **W17 final** battery/receipt/CURRENT_FOCUS rewrite (the sibling already delivered W17-verdicts + an interim 15:16 receipt; the FINAL closure receipt + focus rewrite remain).

## d) TOTALLY FUCKED UP / went sideways (recovered)

1. **I designed the OIDC nonce as BREAKING interface changes first.** Initial implementation rewrote `identitymodel.OAuth2Provider.BeginLogin` (3→4 returns), `FinishLogin` (+nonce), and `OAuth2StateStore.Save/Consume` — a hermetic `GOWORK=off` build of usermgmt correctly rejected it: in-repo consumers resolve PUBLISHED module versions, so published-interface changes need a coordinated train (forbidden inside a wave; gotcha 30/36 territory). Caught it via the isolation-gate reality check before any commit; reverted idm + store signatures and pivoted to the additive `*WithNonce` pair + `NonceStateStore` type-assertion design (the repo's own `FinishLoginWithToken` precedent). Cost ~30 minutes. Lesson: **check published-module boundary constraints BEFORE designing cross-module API changes**, not after.
2. **`rg -rn` host hook hit me TWICE more** (3rd and 4th strikes on this host — the trap is now the single most frequent tool failure in my sessions). The hook is right; my fingers are wrong. `rg -n pattern dir/` only.
3. **Edit-tool "read first" burned me once** (`coverage_gaps3_test.go`): a bash `sed` view does NOT count as a read — the edit tool requires a prior View-tool read of the exact file. Known rule, skipped it once under momentum.
4. **Chased stale LSP diagnostics twice** (`usermgmt/service_core.go:301 "undefined: fmt"` and `systemadapter "context imported and not used"` — both false; source verified fine). Gotcha 14 says exactly this; I still spent a vet run confirming. Under multi-module edit churn, trust builds, not the LSP pane.
5. **Daemon scattered W02 across three commits** (`4773044e` accent files, `8ccc2b4f` render files — including a render_test.go I had just decided to delete, forcing a follow-up deletion commit `c57cc36d`). Content intact, attribution mixed, accepted per the concourse rules — but my add→commit window is still too wide on this tree; the sibling's lesson (inspect diff BEFORE add, chain add+commit in ONE call) is now mine too.

## e) WHAT WE SHOULD IMPROVE

1. **Published-interface changes need a train — make that a planning-time constraint.** The nonce pivot was avoidable: the 10-42 plan should carry a "which tails touch PUBLISHED exported APIs" column so the additive-vs-train decision happens at plan time, not mid-implementation.
2. **Mutation-proof valuable tests.** The transport dedup test was mutation-proven (fails with the fix disabled); the datastar sort test previously existed in name only. A "can this test fail?" check belongs in the definition of done for every pin test written against an old bug.
3. **Write module CHANGELOG receipts IMMEDIATELY after the module's last test goes green**, not batched at milestone end — the daemon can bank code at any moment, and a daemon-banked code change without its receipt violates the same-commit discipline through nobody's fault but leaves the debt anyway (see b1).
4. **The lying-test class is worth a sweep.** `ReturnsSortedCopy` asserted weaker than its name for who-knows-how-long. A one-liner audit of tests whose names make claims their assertions don't check (sorted/atomic/idempotent/no-dup) would find more of these.
5. **Two-session tree: pull before every milestone.** I started W02 while the sibling pushed `2e64862b`; my working tree was one pull behind and the gosec run landscape shifted mid-triage. Cheap fix, do it religiously.

## f) NEXT (up to 50, ordered)

1. Write the missing usermgmt CHANGELOG S39 receipt (empty-subject + nonce flow) — 2 min, unblocks the dangling root pointer.
2. Check CI verdict for the tip commit (`471ab2b2`/`5f47d846`); triage if red (shellcheck on new test files is the only plausible risk).
3. W04.0: re-verify every S38 `file:line` in identity-model (3 of 6 S37 tails were already fixed by others; expect the same here).
4. W04: `GenerateUserID` → `brandid.NewID` call-order tail.
5. W04: strict `ParseTenantID`/`ParseBotID` (reject malformed prefixes).
6. W04: `Exists`/`IsActive` split on the authz surface.
7. W04: g2-guard under custom Casbin model (hierarchy seeding must not corrupt a consumer model that reuses g2).
8. W04: `NotConfigured` errors → permanent family (not transient).
9. W04: accessor methods return copies where they return aliased slices/maps.
10. W04: kill the internal deprecated `NewUserID` (identity-model).
11. W04: idm CHANGELOG receipts + `.#test-identity-model`.
12. W05: declarative Enforce ladder vs legacy Casbin grants — align or document honestly (the silent-widening-on-migration class).
13. W05: audit Action vocabulary parity with `usermgmt.AuditAction*`.
14. W05: ActorID carried into declarative audit entries.
15. W05: systemadapter CHANGELOG + declarative suite.
16. W06: DocsHandler/DocsPath end-to-end mount test.
17. W06: `RecorderChain(Recorder(svc), auditPlugin)` composition proof.
18. W06: `StateShape` round-trip (SQL snapshot + codec; file upstream if the stamp drops).
19. W07: `WithSchemaVersion(1)` on the 21 idm constructors.
20. W07: `WithSchemaVersion(1)` on e2e/demo sites (8 sites).
21. W07: delete health status mirrors + root require bump + integration_test half.
22. W07: receipts + idm/health suites.
23. W08: verify sibling's S45 EventHandler section + idm README receipts.
24. W08: verify S52 mandate row, S53 credential memo, S54 v5 skeleton (presence + docs gates + AGENTS truth pass).
25. W10: Playwright WebAuthn virtual-authenticator E2E (register + login through the rendered loginpage).
26. W10: full-page render golden (stable IDs, nonce-insensitive).
27. W11: split `setup/config.go` (1186 lines).
28. W11: split `systemadapter/declarations.go` (609).
29. W11: split `usermgmt/service_core.go` (576).
30. W11: split `dashboardui/layout.go` (518 → assets).
31. W11: split `e2e/server/main.go` (487); verify `check-large-files` + suites.
32. W12: full battery — `nix run .#test` (4× rc=0 target, two attempts max).
33. W12: `nix run .#lint` + `.#coverage-gate` + `.#check-modules -- --report`.
34. W12: push through strict pre-push gates + `gh run watch` + `.#verify-remote-tags` + ahead-0.
35. W12: `git log --stat` book reconciliation with sibling's 15-16 receipt (their "11 commits ahead" note vs current synced state — confirm nothing of theirs was lost in my pushes).
36. W14: loginpage AutoRedirect follow-through (NoWebAuthn asymmetry memo, AuthPrefix validation, guide, FEATURES row).
37. W14: dashboardui 09-48 tails — rune-safe truncate/initials.
38. W14: dashboardui PathEscape on path-derived values.
39. W14: dashboardui data-base-path on the layout shell.
40. W14: dashboardui audit streaming + neighbour early-stop.
41. W14: dashboardui snapshot 404/500 + boost-inheritance guard sweep.
42. W14: root declutter — biome.json prove-or-remove, allowlist sketch.
43. W14: sweep-plan M18/M20/M21 + RequireSession export-or-document.
44. W15: art-dupl r3 accept-baseline + 2 extractions.
45. W15: `check-skill-claims.sh` skeleton + snippet compile gate + README split-brain scan.
46. W15: tooling canaries a–g + upstream ask drafts batch (packet U1 approves drafting).
47. W17: FINAL battery + push + final receipt superseding both 13-30 (archived) and the sibling's 15-16 interim + CURRENT_FOCUS rewrite.
48. Next-train prep (owner-gated): promote the OIDC nonce into the BASE OAuth2Provider interface as a coordinated idm+oauth2+usermgmt train (the additive pair is the v4 bridge; see question 2).
49. Consider a one-liner "test-name claims vs assertions" audit script (improvement e4).
50. Consider archiving the 15-16 sibling interim when the W17 final receipt lands (keep tail ≤2; same-commit baseline moves).

## g) QUESTIONS (cannot figure out myself)

1. **Ratify the AccentColor default (Q3 from the 13-30 report, still unanswered).** I shipped reject-at-`New` (fail-fast; a typo'd accent surfaces at construction) with the sanitize-to-default alternative dismissed in the CHANGELOG. If you prefer silent-sanitize, it is a ~15-minute revert-and-retest; every legitimate color passes unchanged either way. Also folded into your D-table R1 ratification would close it cleanly.
2. **Nonce promotion policy.** The additive `*WithNonce` pair + `NonceStateStore` gives the default path full OIDC nonce protection in v4, but custom OAuth2Provider implementations stay nonce-less until they implement the capability. Promoting nonce into the base `identitymodel.OAuth2Provider` interface is a breaking change requiring a coordinated idm+oauth2+usermgmt train (the exact class the hermetic gates forced me to avoid mid-wave). Next train: promote, or keep the capability-detection design as the permanent shape?
3. **W12 battery under load.** The sibling recorded load avg 111–211 all session and skipped their lane-tail battery for noise reasons; my W12 checkpoint (the plan's hard kill-criterion gate) wants 4× rc=0. Run it now regardless of load (results are pass/fail, not benchmarks — noise mostly costs time), or hold W12 for a quiet window and continue W04–W11 code first? My inclination: run it now, benchmarks are the only load-sensitive gates and W12 runs none.
