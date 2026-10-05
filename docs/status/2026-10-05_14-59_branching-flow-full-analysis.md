# Status Report — branching-flow Full Analysis Pass

**Date:** 2026-10-05 14:59 CEST
**Session:** Ad-hoc static-analysis sweep using `branching-flow` (v0.2.0, build `05d8209`)
**Scope:** Run every `branching-flow` analyzer except `phantom` against the whole repo, then triage what is actually worth acting on.
**Report format note:** The `status-report` skill's canonical output is a styled HTML dashboard; the operator explicitly requested `.md` at `docs/status/<YYYY-MM-DD_HH-MM_WELL-NAMED>.md`, so this report honors that override. The override is recorded here so it is not mistaken for a new default.

---

## Session Stat Summary (verbatim from `branching-flow stats .`)

| Linter | Issues | Severity breakdown |
|---|---|---|
| Phantom Types | 499 | 281 critical, 117 high, 36 medium, 65 low — **EXCLUDED per request** |
| Strong ID Types | 76 | 4 high, 5 medium, 67 low |
| Mixins | 51 | 1 medium, 50 low |
| Anti-Patterns | 8 | 8 low |
| Duplicate Types | 8 actionable | 21 groups total (8 actionable, 3 intentional, 10 false-positive) |
| samber/do Anti-Patterns | 5 | 5 medium |
| Flag Parameters | 4 | 2 medium, 2 low |
| Interface Completeness | 3 | 3 low |
| Boolean Blindness | 1 | 1 critical |
| Context Propagation | 0 | — |
| Naked Return Guard | 0 | — |
| Panic Conditions | 0 | — |
| samber/ro Anti-Patterns | 0 | — |
| Split-Brain Interfaces | 0 | — |
| **Total** | **655** (incl. phantom) | **~156 excluding phantom** |

- Root walk: **292 files**, ~3.4s.
- Coverage probe: total non-test `.go` files in the tree = 310; `health`=3 files, `auditlog`=2 files both individually walkable and returned **zero** findings. So the 292-file root walk is broad (module-recursive), and health/auditlog's clean result is genuine, not a skipped-module false green.
- Raw log: `/tmp/bf-all.txt`.

---

## a) FULLY DONE

> Evidence = tool output in this session. Nothing was committed (no code changed).

- **All 14 non-phantom analyzers executed** against `.` — `anti-patterns`, `boolblind`, `contextguard`, `do`, `dupe`, `flagparam`, `ifacecomplete`, `mixins`, `nakedreturn`, `panic`, `ro`, `splitbrain`, `strong-id`, plus `stats` for the summary. Evidence: `/tmp/bf-all.txt`, `stats` output above.
- **Clean analyzers confirmed:** Context Propagation, Naked Return, Panic, samber/ro, Split-Brain → 0 findings each.
- **`do` (5) root-caused → false positive.** All five `do.Invoke` findings are in `examples/samber-do-demo/container.go:308-335`, inside *typed accessor methods* (`Service()`, `Broadcaster()`, `App()`, `Logger()`, `HealthDashboard()`) that deliberately centralize resolution. This is the canonical samber/do provider pattern, not service-locator abuse — the in-file comment says so.
- **`boolblind` `Capabilities` (1, critical) root-caused → reject.** `dashboardui/core/capabilities.go:61` has 11 bools that are a *named-field projection* of `Config` (presence flags). Bit-flags would destroy readability; the "critical" severity is overstated for a self-documenting projection.
- **`strong-id` bulk (76) triaged.** Verified most are intentional wire/boundary strings: `identity-model/events.go` ActorID/TenantID (serialized event payloads), `openapi/builder.go` operationID (spec field), `htmx.go` TriggerID, `ack.go` `commandId` (JS-facing JSON contract), `systemadapter/queries.go` (strings go straight into the engine lookup key).
- **`mixins` (51) triaged → reject.** Would fragment the flat, golden-pinned 21-event payload structs.
- **`anti-patterns` (8) triaged → reject.** All `large-struct` on configs / composition roots (`setup.Config`, `usermgmt.ServiceConfig`, `cqrshtmx.handlerConfig`), which is their job. `core.DefaultPayloadRenderer` `base-naming` is a default-impl name, fine.
- **`ifacecomplete` (3) triaged → reject.** `LockoutStore`, `TOTPProvider`, `WebAuthnSessionStore` are the extension seams satisfied structurally by the `totp`/`webauthn`/`oauth2` strategy modules. Single-impl is intentional.
- **`dupe` group 1 root-caused.** `commandOptionApplier` is duplicated in `handler.go:343` and `usermgmt/audit_context.go:19` — identical interface + near-identical doc comment, duplicated *because both are unexported and root↔usermgmt cannot share*. Documented pattern; nothing guards drift.
- **Subprocess hygiene verified.** Long-running `stats` probe was correctly auto-backgrounded and drained via `job_output`.

---

## b) PARTIALLY DONE

- **Credential-shape duplication (dupe g4/g5) — flagged, not adjudicated.** Four declarations of the same WebAuthn credential shape:
  - `identity-model/credential.go:7` `CredentialCore`
  - `systemadapter/views.go:35` `CredentialView`
  - `usermgmt/webauthn/provider.go:65` `credentialData`
  - `usermgmt/webauthn_service.go:21` `webAuthnUserCred`
  Shared fields: `AAGUID, AttestationType, BackupEligible, BackupState, ID, PublicKey, SignCount, Transports`. **What works:** identified precisely with line numbers. **Open:** whether any can alias/embed vs redeclare without coupling domain↔adapter. **Blocker:** needs a domain-intent decision (question g2). **Effort:** M.
- **`strong-id` severity split — partially triaged.** The `stats` line reports 4 high / 5 medium / 67 low, but the finding table carries no severity column, so I could not attribute severity per row and lumped all 76 together before selecting the few real candidates. **Open:** re-run with a severity filter to isolate the 4 high items. **Effort:** S.
- **Section (f) harvest — not performed.** Per the `status-report` skill + AGENTS.md gotcha 20, the Top-N list must be routed into `TODO_LIST.md`/`ROADMAP.md` via `docs-health` HARVEST. Deferred pending the operator's "wait for instructions". **Effort:** S.

---

## c) NOT STARTED

- **Any code fix.** This was an analysis-only pass; zero source edits.
- **Gate integration for `branching-flow`.** The tool is not referenced in `flake.nix`, `check-modules`, `.githooks`, or CI. It could become a gate over select analyzers (like `cqrs-lint` did), but no decision made. Priority: operator-dependent.
- **`plainBodyWriter` options refactor** (`errors.go:236`, two positional bools). Identified, not started. S.
- **`pendingTOTPStore` typed-ID refactor** (`usermgmt/totp.go:166,177`). Identified, not started. S.
- **Drift guard for the `commandOptionApplier` split** (shared home or cross-link comment). Identified, not started. S.
- **Baseline/ratchet config for `branching-flow`** (e.g. an `.branching-flow.yml` if supported, or a suppression baseline). Not started.

---

## d) TOTALLY FUCKED UP!

Honest read: **nothing is broken in the repo.** No test/build harm was done — this session only read. But the *analysis itself* has real defects worth naming rather than hiding:

- **Severity-blind triage (self-inflicted).** I ranked findings by eyeballing file/symbol importance instead of by the tool's own severity. The `stats` output says `strong-id` has **4 high**, `do` has **5 medium**, `flagparam` has **2 medium**, `boolblind` has **1 critical** — I dismissed the two "medium/high" clusters (`do`, `boolblind`) as false positives *after* reading source (correctly), but I never proved which 4 `strong-id` rows are "high". **Severity: blocks nothing, degrades report precision. Root cause: the finding table omits a severity column and I didn't compensate with a filtered re-run.**
- **Coverage asserted from a file count, not from a load log.** I inferred "all modules walked" from `292 vs 310` files, plus a 5-module spot probe. That is circumstantial; a module that failed to type-load could hide here. **Severity: false-green risk (the AGENTS.md gotcha-2 class). Root cause: I did not capture a per-module load/candidate count.** Mitigation: the spot probe (health/auditlog/oauth2/totp/setup) all returned sane file counts, so the risk is low but not zero.
- **One recommendation (`boolblind` = critical) as literally printed is misleading.** If a consumer greps for "critical" findings they'll hit `Capabilities` 11-bool first — the loudest item in the report is a non-issue. **Mitigation: this report labels it explicitly rejected.**

---

## e) WHAT WE SHOULD IMPROVE!

**Process (this session):**
1. **Rank by the tool's severity, not by intuition.** Use `stats` severity buckets to drive the shortlist; re-run with filters to isolate high/medium rows when the table is severity-blind. Concrete: for `strong-id`, get the 4 high rows before writing the verdict.
2. **Prove coverage with a count, don't infer it.** Record per-module candidate counts (the AGENTS.md "PRINT ITS CANDIDATE COUNT" rule) so a skipped module fails loudly instead of silently.
3. **State the false-positive verdict next to the finding**, as done here for `do`/`boolblind` — a raw analyzer dump without an adjudication table is noise.
4. **Close the loop to HARVEST.** Section (f) dies in a timestamped file unless routed to `TODO_LIST.md`/`ROADMAP.md`.

**Tooling/design (repo):**
5. **`branching-flow` is not gated.** It is a fast, zero-config, 14-analyzer pass (3.4s root walk) — a natural complement to `cqrs-lint`. If the org wants it, it needs the same treatment `cqrs-lint` got: flake app + `check-modules` stage + CI step + fixture self-test + README line (atomic per AGENTS.md gotcha 19). But it MUST be run in a way that loads the 1.27.1 workspace (same `GOTOOLCHAIN=local` + goPkg trap as `cqrs-lint`), or it will false-green.
6. **The `commandOptionApplier` duplication is a latent split brain.** Same concept, two files, unexported so unfixable by import today. Add a cross-reference comment or a shared internal helper + a test that asserts both stay shape-identical.
7. **Credential DTO sprawl** is the one data-model smell with real depth — needs a deliberate Published-Language vs anti-corruption-boundary call.
8. **`strong-id` candidates that ARE real** are few: `usermgmt/totp.go` `pendingTOTPStore` (UserID available in scope), and the `usermgmt` internal fn params. The events/openapi/htmx/ack hits are wire contracts and should be whitelisted in a baseline so signal isn't drowned.

**Self-critique (asked explicitly):**
9. **What did I forget?** — (i) the per-module candidate-count proof; (ii) isolating the 4 high `strong-id` rows; (iii) checking whether `branching-flow` has a config/suppression mechanism to build a baseline; (iv) running `dupe`/`mixins` at module scope to confirm no cross-module walk artifacts; (v) HARVEST.
10. **What could I have done better?** — Driven the shortlist off `stats` severity; produced an adjudication table (finding → verdict → evidence) for every analyzer instead of prose; recorded raw + parsed JSON if the tool supports it.
11. **What could I still improve?** — Build a `.branching-flow` baseline committing current verdicts so future runs diff against a known-good set, turning a 655-line firehose into "new findings since baseline = N".

---

## f) Top next tasks (up to 50; ranked; later items are ROADMAP fuel)

### Act-on shortlist (the findings worth investigation)
1. Investigate credential 4× duplication → decide alias/embed vs keep — **High / M / Quality** (g4/g5)
2. Add drift guard or cross-link comment for `commandOptionApplier` split brain — **Medium / S / Cleanup**
3. Type `pendingTOTPStore.Save/Consume` with `identitymodel.UserID` — **Medium / S / Quality**
4. Refactor `plainBodyWriter(includeInternal, includeRequestID bool)` → options struct/enum — **Medium / S / Quality**
5. Re-run `strong-id` with severity filter; enumerate the 4 high rows — **Medium / S / Analysis**
6. Whitelist the intentional strong-id wire/boundary hits in a baseline — **Medium / S / Cleanup**

### Analysis hardening
7. Produce per-module candidate/load counts for a `branching-flow` run — **High / S / Analysis**
8. Build `.branching-flow` baseline + diff workflow — **High / M / Tooling**
9. Capture machine-readable (JSON) output for CI consumption — **Medium / M / Tooling**
10. Add `branching-flow` cross-check to AGENTS.md gotcha list (tool exists, how to run) — **Medium / S / Documentation**
11. Document the severity-blind-table pitfall in this repo's analysis runbook — **Low / S / Documentation**

### Gate / CI integration (decision-gated)
12. Decide whether `branching-flow` becomes a gate — **High / S / Decision**
13. If yes: flake app `.#check-branching-flow` with `GOTOOLCHAIN=local` + goPkg — **High / M / Tooling**
14. If yes: fixture self-test + `check-modules` stage + CI step + README line (atomic) — **High / L / Tooling**
15. If yes: choose the analyzer subset (likely `contextguard`, `splitbrain`, `dupe`, `strong-id`, `flagparam`, `boolblind`) — **Medium / S / Decision**

### Credential/DTO cleanup
16. Audit `systemadapter/views.go` views vs identity-model cores for alias opportunities — **Medium / M / Quality**
17. Audit `usermgmt/webauthn/provider.go` `credentialData` vs `CredentialCore` — **Medium / M / Quality**
18. Audit `webAuthnUserCred` (`usermgmt/webauthn_service.go:21`) for removal — **Medium / S / Quality**
19. Add a domain-language entry for "credential" boundaries — **Low / S / Documentation**

### Small quality wins
20. Investigate `navItem` dup (adminui/dashboardui) — likely intentional, add comment — **Low / S / Cleanup**
21. `ifacecomplete` — annotate the 3 seam interfaces as intentional single-impl — **Low / S / Cleanup**
22. `boolblind` — annotate `Capabilities` as an accepted projection — **Low / S / Cleanup**
23. `anti-patterns` — annotate the config large-structs as accepted — **Low / S / Cleanup**
24. Consider `Capabilities.Has*()` helpers to reduce bool reach — **Low / S / Quality**
25. Re-check `flagparam` `buildHandlerConfigChecked typeIsZero` (low, not-last bool) — **Low / S / Quality**

### Loop-closing / hygiene
26. Route items 1-25 into `TODO_LIST.md`/`ROADMAP.md` via docs-health HARVEST — **High / S / Documentation**
27. Mark this report annotated once harvested — **Low / S / Documentation**
28. Note `branching-flow` v0.2.0 in memory as a known tool — **Low / S / Documentation**

### Speculative / ROADMAP
29. Evaluate `phantom` findings separately with a tuned baseline (499 raw) — **Low / L / Analysis**
30. Evaluate `mixins` for the non-wire structs only (`adminui`, `dashboardui`) — **Low / M / Quality**
31. Evaluate `strong-id` fix for `systemadapter/queries.go` internal params — **Low / M / Quality**
32. Evaluate typed IDs for `usermgmt` internal readmodel params — **Low / M / Quality**
33. Add `branching-flow` to the devShell if not present — **Low / S / Tooling**
34. Compare `branching-flow` overlap with `cqrs-lint` to avoid double-reporting — **Low / M / Analysis**
35. Consider `branching-flow all --json` nightly digest — **Low / L / Tooling**
36. Add suppression-reason convention mirroring `//nolint:erraudit` classes — **Low / S / Convention**
37. Build a "verdict table" template for future analyzer passes — **Low / S / Process**
38. Track analyzer false-positive rates over runs to tune whitelists — **Low / M / Process**
39. Add `contextguard` as a cheap permanent gate (currently 0 findings) — **Low / S / Tooling**
40. Investigate whether `dupe` cross-module matches are real or walk artifacts — **Low / S / Analysis**

---

## g) Questions I cannot answer myself (Top 3)

1. **Gate intent:** Do you want `branching-flow` wired into this repo's gates (flake app + `check-modules` + CI, like `cqrs-lint`), and if so, which analyzer subset? I can't know whether the org wants another analysis gate or whether a Go-installable distribution exists (the same distribution caveat that keeps `cqrs-lint` local-only). I tried to answer it by checking `flake.nix`/CI/hooks — the tool is currently referenced nowhere.
2. **Credential duplication:** Is the 4× WebAuthn credential shape an intentional anti-corruption/Published-Language boundary set (keep), or unwanted sprawl to consolidate? I can see the field sets and the boundary roles, but the domain intent — "are these the same concept or four facets?" — is yours.
3. **Scope of action:** Which of the act-on shortlist (items 1-4) should I execute now, and are items 29-40 to be treated as ROADMAP fuel rather than worker tasks? I flagged them but cannot set your priority.

**Status:** WAITING FOR INSTRUCTIONS.
