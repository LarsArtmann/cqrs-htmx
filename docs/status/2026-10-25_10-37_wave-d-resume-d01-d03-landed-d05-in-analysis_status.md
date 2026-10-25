# Status: Wave D resume — D01 closed (co-owned with the sibling session), D03 landed, D05 in analysis; concurrent session detected on D02

**Written:** 2026-10-25 10:37 CET · **Session:** "READ, UNDERSTAND, RESEARCH, REFLECT… Execute and Verify… Repeat until done" against the 08-05 plan's Wave D (Wave R + D04 were already closed by the prior continuation — pushed HEAD `a5183fff`, CI `40179492715` all-success).
**Plan of record:** [`docs/planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md`](../planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md) (per-milestone scope: the 06-20 file; S-item detail: the 10-24 master plan t-tables).
**Supersedes:** [`2026-10-25_10-15_green-tree-receipt_status.md`](archived/2026-10-25_10-15_green-tree-receipt_status.md) (Wave R + D04 story unchanged — carried here in one line: battery 4× rc=0, pushed through strict gates, CI terminal green, D04 = golangci 2.15.0 everywhere; its §g owner-gate tables are carried below).

## §a FULLY DONE this session (committed)

| Work | Receipt |
| --- | --- |
| **Resume-state correction** — the handoff summary was STALE (it described HEAD `3edd893b`, ahead 12, R04/R05/R06 open); the tree had already moved to `18bca31a` with Wave R + D04 CLOSED, pushed, CI green. Detected before any duplicate execution; the D-wave remaining set (D01–D16) re-derived from the three plans | `git log` verification; this report |
| **D01 delivery receipt — co-owned.** The CHANGELOG receipts owed (S35/S41/S44) were verified ALREADY PRESENT (root CHANGELOG `Unreleased` + dashboardui line 16 — a later session added them; the 05-50 report's "owed" list was stale). The sibling session's `dacae5c2` landed Appendix D reconciliation + its own S27 harvest + the 08-05 §8 execution receipt WHILE I worked; I did not race it (my master-plan multiedit aborted on the file-changed check) | `dacae5c2` (theirs) |
| **D01 S27 harvest remainder — the truth-pass neither receipt finished.** Struck the executed rows with receipts (S33 root honesty, S34+S43 openapi, S35 dashboardui, S41 setup appkit health, S44 sync DTO, S32 WebAuthn counter — all receipted in CHANGELOGs); rewrote the sync follow-through row to the open watches (N5/N4/demand-gated tails); replaced the two dead P1 rows with ONE pointer row naming the Wave D code-fidelity milestones AND the two milestones both successor-plan inventories dropped by omission — **S36** (loginpage `NewPageData` drops Nonce/Theme — verified live at `loginpage/handler.go:229-296`, `ServeHTTP` populates them but the exported constructor never does; adminui AccentColor unvalidated + partial-render 200s) and **S39** (transport replay/live dedup, unknown-cursor semantics, auditlog shutdown, health `Recorder(nil)`, datastar `NewEventBridge(nil)`/`MappedEventTypes` unsorted — verified at `datastar/event_bridge.go:38,106-119`; oauth2 empty `sub`; totp negative Window) | `b51a4a33`; `check-todo-hygiene` 72 rows / 0 violations |
| **D03 / S47 — `bump-dep --lint` leg.** After tidy/verify/build/vet, `--lint` runs hermetic golangci (own `.golangci.yml`, `GOEXPERIMENT=jsonv2 GOWORK=off`) in every swept module; a lint failure fails the sweep naming the module + log; missing binary is a loud failure; `BUMP_DEP_LINT` env override exists for fixtures. Fixture self-test 7→9 cases (stubbed exhaustruct failure propagates; clean modules PASS) | `bash scripts/selftests/test-bump-dep.sh` 9/9 green |
| **D03 / S48 — the formatter→templ-generate chain closed.** Decision: EXCLUDE `*.templ` from buildflow (its bundled templ binary is version-skewed from the nix-pinned one — the exact class of the load-bearing templ-generate skip, gotcha 35), and close the chain at commit time with a repo-native gate: `scripts/checks/check-templ-codegen-staged.sh` wired into `scripts/hooks/pre-commit.template` + the active `.githooks/pre-commit` (MANUAL EDIT block, restorable via install-git-hooks). Regenerates module-dir (canonical bare `FileName:`) and refuses a staged `.templ`/`_templ.go` pair that drifted; hand-edits to `_templ.go` caught too; fast-path = one git call | `.buildflow.yml` + hooks; 4-case fixture self-test green |
| **D03 / S48 — live proof (d03.4).** Staged a real comment edit to `adminui/layout.templ` → gate RED naming `adminui/layout_templ.go` + the `nix run .#gen` fix hint → regenerated + staged → GREEN → tree restored byte-clean | shell transcript; ambient templ v0.3.1020 == CI's pinned generator == pkgs.templ (version agreement verified) |
| **D03 — wiring parity.** New flake app `.#test-check-templ-codegen-staged`, check-modules stage (list + body), CI step beside the bump-dep self-test; `nix fmt` clean (one SC2164 fixed); ci.yml YAML-parses; flake evaluates | `nix run .#test-check-templ-codegen-staged` green via the app |

## §b PARTIALLY DONE

- **D05 / S37 (usermgmt tails) — analysis complete, no edits yet.** Verified against source, three of six sub-items are ALREADY FIXED by intervening sessions: **S37c** cookie Max-Age (sessionMaxAge wired, `http.go:291-293` — test still owed), **S37e** MySQL doc truth (`sql_event_store.go:35-37`), and the hard-coded 30s rebuild drain half of **S37b** is gone. Genuinely open: **S37a** TenantDeleted cascade (`es_casbin_projection.go:111-117` still calls `RemoveAllRolesForUser(tenantStreamID)` which removes NOTHING — member policies survive; `RemoveAllRolesInDomain` exists at `identity-model/authz_policies.go:262` as the right primitive; `service_tenant.go`'s dispatch-error swallowing to audit), **S37b** failure-path (Stop → Reset/create fail → service left with ZERO live projections; recovery-host restart owed), **S37d** lockout (`EvictStale`'s doc LIES — claims zero-attempt eviction but only sweeps `lockedAt`; `RecordFailure` re-extends `lockedAt` on continued failures during an active lock, `lockout.go:110-120`), **S37f** empty-store fallback warning (site moved; re-locate).

## §c NOT STARTED (this session; the 08-05 Wave D remainder)

D02 remainder verification (sibling in flight — see §g.1), D06 (S38 identity-model tails), D07 (S42 systemadapter fidelity), D08 (S46 seams + S49 StateShape), D09 (S58 schema stamping + health mirrors), re-routed S36 + S39, D10 (S50+S51 scripts), D11 (S57 loginpage E2E), D12 (S40 file splits), D13 (S55+S56 owner packet + watches), D14a/D14b (harvested backlog), D15 (BuildFlow nix-fmt filing + bench window), D16 (parked-row audit). Battery + push checkpoints after every ~5 milestones (master-plan §9.9).

## §d TOTALLY FUCKED UP (honest)

1. **I duplicated sibling work before checking for a sibling.** I hunted the S35/S41/S44 CHANGELOG receipts and drafted 14 Appendix-D edits without first sniffing for concurrent sessions (last-commit age / target-file mtimes). `dacae5c2` landed mid-flight; my multiedit aborted on the file-changed guard (working as designed), but the duplicate ANALYSIS effort was avoidable — `git log -1 --format=%cd` costs one call.
2. **The daemon shredded D03's attribution twice.** Most D03 files were swept into `65665920` ("auto-commit 10 files") MIXED WITH the sibling's D02 files (credential memo, v4-to-v5 additions); my descriptive message landed only on the 3-file remainder (`ae3b80dd`). Content is fine; history readability suffered. Cause: I ran gates BETWEEN editing and committing — the fmt→add→commit chain must be tighter.
3. **First live-proof attempt used an HTML comment at templ top level** → parse error, one wasted cycle. Should have read the file shape first.
4. **Session-summary staleness tax.** The handoff summary was 3 commits + one whole receipt behind; I spent early context planning R04–R06 work that was already done. The env-block git snapshot was also from conversation start. Lesson: trust `git log`, not the summary.

## §e IMPROVE

- Pre-flight ANY multi-file docs/receipts work with a concurrency sniff: `git log -3 --format='%h %cd %s'` + `find <targets> -newer <marker>`; if anything is minutes-old, coordinate scope before editing.
- Commit feature work the moment its gates pass — every gap is a daemon window (twice proven this session).
- Sibling-coexistence protocol (proposed, see §g.1): disjoint file sets + immediate commits + `git log --stat` reconciliation before batteries.

## §f NEXT (execution order; V = verify command)

1. **D05 finish** — S37a TenantDeleted cascade: enumerate domain members from the casbin projection's own view (`Policies()` scan or membership read model), `RemoveAllRolesInDomain` per member + tenant role-hierarchy edges; service_tenant.go stops swallowing dispatch errors (log+audit instead). V: `nix run .#test-usermgmt`.
2. S37b recovery host: on Reset/create failure after Stop, recreate a host from surviving checkpoints before returning the error. V: stub-Reset test asserting `ProjectionStatuses()` non-empty after failed rebuild.
3. S37d lockout truth: `EvictStale` sweeps zero-attempt `attempts` entries older than Duration (make the doc honest); `RecordFailure` does not re-extend an ACTIVE lock (fixed expiry from the original lock — see §g.2). V: unit tests for both.
4. S37f: locate the empty-store journal fallback (moved from es_setup.go:285), add the warning. V: `go vet` + test.
5. S37c: add the SessionTTL→Max-Age header test (behavior already correct, pin it). V: `httptest` cookie assertion.
6. usermgmt CHANGELOG receipts for S37a/b/d/f in the SAME commits (published module).
7. **D06 / S38** a–g (GenerateUserID ordering via `brandid.NewID`; strict ParseTenantID/ParseBotID; Exists/IsActive split; g2-guard under custom model; NotConfigured → permanent family; accessor copies; kill internal deprecated NewUserID). V: `nix run .#test-identity-model` + `.#lint`; idm CHANGELOG receipts.
8. **D07 / S42** Enforce-ladder diff vs legacy Casbin grants; audit Action vocabulary parity (`change_email` vs `email_changed`); ActorID into declarative audit entries. V: declarative suite.
9. **D08 / S46** DocsHandler end-to-end mount test + `RecorderChain(Recorder(svc), auditPlugin)` proof (integration_test, published tags). V: `nix run .#test-integration_test`.
10. **D08 / S49** StateShape round-trip: SQL snapshot store + codec stamp trace; file upstream if dropped. V: round-trip test.
11. **D09 / S58** `WithSchemaVersion(1)` on the 29 constructors (21 idm + e2e/demo sites); delete health status mirrors + root require bump + integration_test half. V: `.#test-health` + `.#test-identity-model`.
12. **S36** loginpage `buildPageData` populates Nonce/Theme in the `r != nil` block; adminui AccentColor shared grammar + partial-render non-200. V: module suites + goldens; CHANGELOG receipts.
13. **S39** transport replay/live dedup + unknown-cursor doc/probe; auditlog shutdown story; health `Recorder(nil)` + status wording; datastar nil guard + `MappedEventTypes` sort; oauth2 empty-`sub` rejection + OIDC nonce; totp negative-Window clamp. V: module suites.
14. **D10 / S50** `check-upstream-treadmill.sh` (per-repo prefix map) + offline fixture. V: self-test.
15. **D10 / S51** `wave-tag.sh` dry-run→push→refresh→sweep→re-tag + ≥6-case offline self-test. V: self-test.
16. **D11 / S57** Playwright WebAuthn virtual-authenticator E2E + full-page render golden. V: e2e suite + golden.
17. **D12 / S40** five file splits (`setup/config.go`, `systemadapter/declarations.go`, `usermgmt/service_core.go`, `dashboardui/layout.go`, `e2e/server/main.go`). V: build + `check-large-files` + suites.
18. **D13 / S55+S56** owner packet (D1–D17 + upstream asks + PapDashboard + datastar-demo + E005 + GCL debt) + watch table. V: packet doc + `check-docs-links` + corpus baseline bump.
19. **D14a** loginpage AutoRedirect follow-through; dashboardui 09-48 tails (rune-safety, PathEscape, data-base-path, audit streaming, neighbors early-stop, snapshot 404/500, boost guard); root biome.json prove-or-remove; sweep-plan M18/M20/M21. V: per-row suites.
20. **D14b** art-dupl r3 accept-baseline decision + 2 extractions; `check-skill-claims.sh` skeleton; snippet compile gate + README split-brain scan; tooling canaries a–g; upstream ask drafts batch. V: each script + self-test.
21. **D15** BuildFlow nix-fmt loop source verification (`buildflow history --step nix-fmt --last-error`) + upstream filing per verify-before-filing; bench-spike ONLY at load < 6 (currently ~20). V: issue link + bench rc or honest refusal.
22. **D16** one dated verdict per parked row (billing S00, DataStar Tier-4, ReadLastN, SidebarNav, BuildFlow BF1, cqrs-lint fleet swap, check-cqrs-lint CI, GCL debt) — recommendations only where owner-gated. V: TODO diff + hygiene gate.
23. **Battery checkpoint** after every ~5 milestones: `nix run .#test`, `.#lint`, `.#coverage-gate`, `.#check-modules -- --report` (now includes both new self-tests). V: 4× rc=0.
24. **Push checkpoint**: `git push` through both strict pre-push gates + `gh run watch --exit-status` + `.#verify-remote-tags`; ahead-count 0.
25. **Sibling D02 verification** once their session ends: S45 EventHandler section + idm README "when do I import", S52 mandate (their AGENTS/release-playbook edits are in the tree), S53 credential memo (draft seen in `65665920`), S54 v5 skeleton (seen); docs gates + corpus baseline green. V: `check-docs-links` + `check-docs-freshness`.
26. **Final receipt** report (supersede this one; archive ritual; TAIL_BASELINE stays 1; CORPUS_BASELINE measured after mutations).
27. CURRENT_FOCUS rewrite at Wave D completion.

## §g Owner gates (cannot figure out myself)

1. **Concurrency protocol.** A sibling session is ACTIVELY executing D02 in this working tree right now (their `dacae5c2` + in-flight AGENTS.md/release-playbook edits; daemon commits mix our files). I cannot see their remaining scope. Options: (a) I continue D05+ in parallel on disjoint files (current mode — daemon absorbs interleaves, batteries reconcile at checkpoints), (b) I pause code waves until they finish, (c) I take a different lane order. A plain "continue" = (a).
2. **Lockout re-extension semantics (S37d).** Today, continued failed logins during an active lock RE-EXTEND the lock indefinitely (a probing attacker keeps the legitimate user locked forever). Fixing per the config's documented semantics = fixed expiry (`Duration` after the ORIGINAL lock; probes during lock are ignored). That is a published-module behavior change — bounded-DoS posture vs maximal-lockdown posture. Recommend fixed expiry; say "keep re-extend" to preserve current behavior (then S37d shrinks to the EvictStale doc-lie fix only).
3. **The dropped S36/S39.** Both successor plans (06-20 §1 "nothing outside these buckets is open" and 08-05) omit S36/S39, but the master plan routed them NOW and I verified both still open in source today. Execute them inside this Wave D (my §f items 12–13) or park them back to TODO as deliberately deprioritized? A plain "continue" = execute (they are small, consumer-facing correctness items).

---

**Carried owner-gate tables from the 10-15 receipt (unchanged):** tag hygiene = leave-and-document (3 red-era tags documented there; `usermgmt/v4.16.0` healthy); minor-version ratification = ratified as-documented per the standing continue-rule, both flagged into the v5 migration plan (S54).
