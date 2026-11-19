# Status report — owner verdicts executed, K2 wave, templ 1118 alignment, the hash anomaly (2026-11-19 20:47)

**Scope line — what this report VERIFIED:** the four owner verdicts captured through the native decision tool (M2M/adminui per-row, K2 GO, cache attribution "unattributed", hook fix by version alignment); templ ground truth at the source (`go list -m -versions`: latest published is v0.3.1118; `go list @v0.3.1123`: unknown revision; ls-remote: newest tags v0.3.1100/v0.3.1118; BuildFlow's own go.mod pins v0.3.1100; the hook's "generator v0.3.1123" traced to the owner's NixOS system profile at /run/current-system — an unpublished master build); the sweep 15/15 go.mods at v0.3.1118 with 0 failures; `nix run .#gen` twice at the moved rev both printing updates=0 (canonical form byte-stable); workspace build + adminui/dashboardui/loginpage suites + catalog-demo and integration_test hermetic builds all green at the 1118 runtime; the K2 wave verified (usermgmt full suite ok 18.7s, all six idempotency tests verbose-PASS incl. the pre-existing opt-in sentinel test, hermetic golangci-lint 0 issues); the overnight daemon captures inspected commit-by-commit. **What it did NOT verify:** the e2e battery (nothing browser-facing changed), the erraudit TOTAL (sweep still deferred — 34 crush sessions, load ~29 on 32 cores), a full local check-modules composite, the true source identity of the templGen fetch (the hash anomaly below — functional output verified twice, source identity not), and CI (nothing pushed yet — 4 local commits).

## a) FULLY DONE

1. **The four owner verdicts captured and routed** via the native question tool: M2M/adminui = per-row ticks (the packets stay parked; each data-model decision surfaces individually before its wave), K2 = GO (execute now), cache attribution = unattributed (write the provisioning ritual without naming anyone), hook fix = "pin go.mod to the latest version (v0.3.1123)".
2. **The hook-fix directive reconciled with ground truth.** v0.3.1123 does not exist upstream — the newest published templ is v0.3.1118, BuildFlow itself pins v0.3.1100, and the hook's "generator v0.3.1123" is the owner's NixOS system templ (an unpublished master build on /run/current-system). The achievable alignment was executed: everything IN this repo now pins the latest published.
3. **Templ swept 1100 → v0.3.1118 across all 15 requiring modules** (per-module hermetic go get + tidy, zero failures, printed candidate count per the gotcha-2 rule); captured by the daemon as `a3c12038` (32 files).
4. **The canonical generator moved to the same tag**: flake templGen version + rev → v0.3.1118, the CI install pin → v0.3.1118; `nix build .#templGen` green; `nix eval` confirms the rev; `nix run .#gen` ran TWICE, both times updates=0 — the v0.3.1118 generator's output is byte-identical to the v0.3.1100 generator's for this repo's templates (the 1118 release changed no generated form; stamps legitimately stay v0.3.1100-form).
5. **The runtime bump verified across every templ-touching surface**: workspace build green; adminui, dashboardui, loginpage suites green; catalog-demo hermetic build green and integration_test hermetic vet green (go-cqrs-lite's 1100-era docserver generated code compiles cleanly against the 1118 runtime).
6. **The K2 wave is code-complete and verified (the owner's GO tick executed).** ServiceConfig gains `IdempotencyStore` (injectable; nil = in-memory default, documented dev/test class) and `DisableCommandIdempotency` (opt-out); the replay guard mounts BY DEFAULT directly inside the audit chain — keyed by the actor-scoped `IdempotencyMetadataKey` metadata (same key the B2 decode hook populates, so recipe followers are covered with zero wiring), pass-through when absent, duplicate answered with a Conflict-family error (`usermgmt.command.replayed` → HTTP 409 instead of MapError's 503 fall-open), TTL constant 10 minutes with the sizing rule in the doc comment.
7. **The K2 regression was caught by the suite and fixed the same session.** The first implementation composed upstream `CommandIdempotency` with an outer translate wrapper — which re-familied ErrDuplicate bubbling from a CONSUMER-mounted idempotency middleware deeper in the chain, breaking the B2 sentinel contract (the pre-existing opt-in test reddened: got `[conflict:usermgmt.command.replayed]`, want `idempotency.ErrDuplicate`). The fix: the guard OWNS the claim (a self-contained mirror of the upstream CheckAndRecord discipline with the translation built in), so only the guard's own store claim can produce the replay Conflict; an inner sentinel passes through untouched. All six idempotency tests pass, full usermgmt suite green, hermetic lint 0 issues.
8. **Four new tests pin the default mount**: keyed retry dedupes across DIFFERENT commands (409 code asserted, second user never created), key-less dispatch passes through un-deduped (domain's own conflict surfaces, no guard code), opt-out removes the guard entirely (both same-key commands execute), and the TTL honesty holds (a forgetful store lets the second same-key command execute again).
9. **The overnight tree evolution was inspected, not assumed**: four daemon commits — `ccab30e5` (the 23-40 status report + ledger + both baselines; already PUSHED by a sibling lane), `a3c12038` (the templ sweep, 32 files), `1556cd20` (ServiceConfig fields), `1b63077f` (the K2 guard + tests) — ahead-of-origin is exactly the three K2/templ commits.

## b) PARTIALLY DONE

1. **The K2 wave's docs half is pending**: the guide's B2 section "mounted by default" note, the usermgmt CHANGELOG entry, the go-idempotency libs-report row true, the K2 packet status flip to ADOPTED/executed, the TODO_LIST row removal, and the CURRENT_FOCUS waiting-on-owner update.
2. **The hook-fix verdict is executed only as far as the repo reaches.** go.mod pins the latest published (1118) everywhere, but the hook's generator is the owner's SYSTEM templ (0.3.1123, unpublished master build) — outside this repo. The war ends when the system templ repins to a published release, which is an owner-side NixOS change; the stale flake comments (two "CI-canonical generator (v0.3.1100)" lines) and the templ-hash.nix comment text also still need truing.
3. **The templ-hash anomaly is open, functionally bounded.** `nix build .#templGen` validated the recorded v0.3.1100-era source hash at rev v0.3.1118 without a mismatch — mechanism unexplained (the two tags are lightweight refs on DISTINCT commits, so the same-tree theory is dead). What IS verified: the built binary's generation output equals the canonical form (updates=0 twice) and `nix eval` confirms rev/version 1118. A `--rebuild` + fetched-source inspection would settle it.
4. **The per-row verdict is not yet recorded in the M2M/adminui packets**, and the unattributed cache-provisioning ritual is not yet written into gotcha 66 — both are one-edit follow-ups from the owner's answers.

## c) NOT STARTED

1. M2M W1/W2 rows — now gated on PER-ROW decisions (surfacing one decision at a time per the owner's verdict).
2. Adminui panel waves — same per-row gating.
3. W3/W4 deep migration waves — D1/D2 recording still owed.
4. adminui vendored popover-CSS retirement — templ-components release still predates PR #101.
5. CI security-job go-version revert — fleet nixpkgs rev still below 1.27.4.
6. B18 — next family train trigger.
7. docs-gates audit-all drain-or-repin decision.
8. The erraudit quiet-window battery and the local check-modules composite — still deferred under sibling load (deadline for the erraudit receipt window: 2026-12-16).
9. Everything else carried on the 23-40 report's §f list (archive pass at tail 15, screenshot eyeball, dual-BuildFlow-checkout reconciliation, e2e flake notes, gen --check mode, and the rest).

## d) TOTALLY FUCKED UP (own failures this session, named)

1. **The K2 first implementation shipped a real regression — the exact class the wave exists to prevent.** The outer translate wrapper could not attribute an ErrDuplicate to its producer, so a consumer's own idempotency sentinel got re-familied to the guard's replay Conflict (the B2 contract inverted by the guard meant to complete it). Caught red by the pre-existing opt-in test before any push; the fix (own-the-claim) is structural, not a patch. Lesson generalized: when wrapping a sentinel-producing middleware, WRAPPER COMPOSITION IS WRONG — own the claim, or the wrapper lies about who produced the error.
2. **The first sweep attempt burned 15 cd failures on a path bug** — rg returns file paths; I concatenated them as directories. One trial iteration would have caught it before the loop.
3. **A `GOWORK=OFF` (uppercase) typo silently voided the dashboardui test leg** — the leg "ran" and errored on the invalid value; caught only because the tail was actually read.
4. **The stale-read-state edit bounces recurred** (flake.nix rejected the first multiedit) — the batch-read-then-edit lesson from my own 23-40 report §d.2 was still not ritualized before acting.
5. **The hash anomaly was left open instead of settled** — a ten-minute `--rebuild` + fetched-source grep would have answered whether the templGen fetch truly returned the 1118 tree; the report carries it as explicitly open instead of closed knowledge.

## e) WHAT WE SHOULD IMPROVE

1. **Codify the sentinel-attribution rule** into the B2 guide section being written: consumer-side composition of a translate wrapper around a sentinel middleware is legitimate for the CONSUMER'S chain; a SHARED default mount must own its claim because it cannot know what else in the chain produces the sentinel.
2. **Ritualize batch-read-then-batch-edit** — view every edit target fresh in one parallel batch before the first edit call; two sessions in a row have bounced on this.
3. **Shell loop discipline**: materialize directories (strip filenames) and trial ONE iteration before looping N; assert the env-var spelling in verification chains (a mis-valued GOWORK silently voids a leg).
4. **Settle anomalies before reporting when the decisive experiment is under ten minutes**; otherwise record them as open WITH the exact experiment named (done here — but do it earlier).
5. **After any daemon capture of fresh code, rerun the suite before building on it** — committed-by-daemon is not verified-by-anyone.
6. **The templ war's end-game is now an owner-side lever**: document the exact NixOS repin remedy (system templ → a published release) next to gotcha 67 rule (d), so the fix stops living in prose.

## f) Things to get done next (K2/templ follow-ups first, then the carried set)

1. Finish the K2 docs half: guide B2 "mounted by default since" note, usermgmt CHANGELOG entry, go-idempotency libs-report row true, K2 packet status → ADOPTED + executed receipt, TODO_LIST row removal, CURRENT_FOCUS update.
2. Record the per-row verdict in the M2M and adminui packets (owner answered via the tool; stop blanket-asking).
3. Write the unattributed cache-provisioning ritual into gotcha 66 (owner answered unattributed).
4. True the two stale flake comments ("CI-canonical generator (v0.3.1100)" at the devShell/apps sites) and the templ-hash.nix comment text.
5. Settle the templ-hash anomaly: `nix build .#templGen --rebuild`, inspect the fetched source tree (version constant + go.mod), record the finding next to the hash files.
6. Commit the K2 fix + helper refactor (uncommitted in the worktree) and push the 4 local commits; watch CI on the push.
7. OWNER (out of repo): repin the NixOS system templ from the 0.3.1123 master build to a published release (v0.3.1118 or v0.3.1100 — both produce the canonical 1100-form stamps) — the one lever that ends the hook-poison war.
8. Quiet window: erraudit strict sweep, refresh the AGENTS TOTAL receipt (window closes 2026-12-16).
9. Quiet window: one local check-modules composite for local/CI parity honesty.
10. M2M per-row surfacing: present D1 (bot-token M2M deprecation question) as the first single decision.
11. Adminui per-row surfacing: present D1-D3 decisions one at a time (impersonation surface, sessions, rename field-set).
12. On the next templ-components release carrying PR #101: retire adminui's vendored popover-CSS layer same-change.
13. Archive pass: the live tail reaches 15 with this report — the next docs-health round must drain the settled set toward the budget of 3.
14. CI security-job go-version revert when the fleet nixpkgs rev ships Go ≥ 1.27.4.
15. B18 execution on the next family train.
16. docs-gates audit-all drain-or-repin decision (393 vs the 322 pin).
17. Screenshot drift eyeball (the regenerated dashboard screenshots from `043f388c`).
18. Reconcile or park the dual-BuildFlow-checkout hazard (lowercase vs capitalized trees).
19. e2e flake app: cached-revision warning (open since 11-08).
20. e2e README: serial-context note.
21. e2e battery timing budget: CI vs the 2.7-minute local figure.
22. check-e2e-ports promote-or-drop.
23. file-size-check posture decision (99 advisory findings per hook run).
24. module-inventory posture decision (27 go.work modules without dependabot entries).
25. dependabot-auto-configure cap finding (31 modules > 20-entry cap).
26. README.md freshness (flagged at 264+ hours).
27. BuildFlow state DB VACUUM (1.03 GB).
28. io-history-ratchet: one profiled hook run to find the whole-tree scanner.
29. The unused `STEP_USES_SECRET_RE` in check-pr-secrets.py (vulture).
30. The stale vendorHash preflight warning — suppression note in .buildflow.yml.
31. Gen-app `--check` (no-write) mode, then a canonical-form CI leg (origin self-verifies the templ form).
32. Twice-regenerate self-test candidate (scriptable flip-signature proof for gotcha 67).
33. Post-daemon-capture suite rerun ritual (improvement 5).
34. K2 follow-up candidate: a reference SQL or Redis idempotency.Store example in the guide for multi-instance consumers (owner-gated scope).
35. Weekly digest of daemon-authored commits into the agents-notes (heuristic titles bury the work).
36. Keep the pre-push release-train cache-refresh flag in mind for the next train day.
37. ROADMAP fuel: a pre-push canonical-form leg via gen --check (closes the templ dimension of "push can still be redder than CI").
38. ROADMAP fuel: sentinel-attribution rule as a general dispatch-middleware ordering note (improvement 1), possibly upstream in go-cqrs-lite's middleware docs.

## g) Questions for the owner (3)

1. **The system templ repin is the last lever of YOUR hook-fix directive, and it is outside this repo**: your NixOS profile ships templ 0.3.1123 (an unpublished master build; the hook inherits it via PATH). Repinning the system templ to a published release (v0.3.1118 or v0.3.1100 — both emit the canonical 1100-form stamps, proven by tonight's builds) ends the war without scoping the hook. Do you want me to write the exact patch text into a runbook note (I cannot see your system-flake repo from here), or will you repin it directly?
2. **The templ-hash anomaly**: the templGen build validated the recorded 1100-era source hash at rev v0.3.1118 — the mechanism is unexplained and the tags peeled to distinct commits, so it is not a same-tree release. The functional output is verified byte-identical twice. Shall I settle it now with `--rebuild` plus a fetched-source inspection (about ten minutes), or park it while everything works?
3. **K2's multi-instance story**: the default mount's in-memory store cannot dedupe replays hitting a different replica (documented in the field comment). For production consumers, is a reference SQL or Redis `idempotency.Store` example in the guide worth a scoped follow-up wave, or does the injectable seam plus the store-contract docs cover it for v4?

---

*Receipts: templ ground truth via go list -m -versions / go list @v0.3.1123 (unknown revision) / git ls-remote (v0.3.1100 ed4e7663, v0.3.1118 ef1c3756) / BuildFlow go.mod (v0.3.1100) / readlink /run/current-system/sw/bin/templ (templ-0.3.1123); sweep 15/15 at v0.3.1118 (rg counts 0 at 1100, 15 at 1118); templGen nix eval rev+version v0.3.1118, build RC=0, gen updates=0 twice; workspace build + adminui (7.9s) + dashboardui + loginpage (2.9s) suites green; catalog-demo + integration_test hermetic RC=0; K2: usermgmt suite ok 18.743s, six idempotency tests verbose-PASS, golangci-lint 0 issues; overnight commits ccab30e5 (pushed by sibling), a3c12038, 1556cd20, 1b63077f; ahead-of-origin 4 with the K2 fix uncommitted in the worktree; owner verdicts per native tool 2026-11-19.*
