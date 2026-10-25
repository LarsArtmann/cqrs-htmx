# Status — S32 credential-use persistence (sign counter + honest UP/UV + challenge namespacing), gate triage, and a concurrent-session collision

> **ANNOTATED 2026-10-25** — ~~every open item in this report is resolved~~ **ARCHIVED 2026-10-25 (round-23 (2026-10-25)).**
> The next session in the same arc is [`2026-10-25_09-45_wave-r-rescue-executed_status.md`](2026-10-25_09-45_wave-r-rescue-executed_status.md), which S32 proven with receipts; red-state evidence superseded by the healing tree.
> Nothing here is a live work item: it is kept as the session's evidence.

> **Point-in-time report · 2026-10-25 07:43 CET (Sunday).** Session started as a continuation of the
> master plan after the previous report (`2026-10-24_12-52_archive-ritual-corpus-ratchet-and-authz-hierarchy`),
> whose two red gates (`module-isolation`, `branching-flow`) and uncommitted report were the first items.
> The tree moved under this session repeatedly: another agent committed this session's in-flight work,
> bumped and tagged three modules, archived/deleted 617 docs files, and briefly broke the root module.
> Everything below is what THIS session authored or verified, with the concurrent actions named as such.

---

## 0) Headline

**S32 is functionally complete and locally green in the four modules it touches — but the repository is
still RED, and four of the five red causes are this session's own incomplete verification, not the feature.**

- Sign-counter persistence (the security bug: clone detection was dead) is implemented end to end and
  proven by a round-trip test + a regression-rejection test.
- UP/UV are no longer fabricated `true` on reload; they are persisted and reloaded honestly.
- Registration and login challenges no longer share a key (`reg:` / `login:`).
- The declarative system path initially rejected the new event as a **dangling subscription** — caught by
  the coeffect gate and fixed; the exhaustive schema/decoder test then caught the second half. That gate
  pair is the best evidence in this session that the wiring is now complete.
- What is still red: the branching-flow ratchet (+47 unadjudicated), root `golangci-lint` (44 issues, all
  in the concurrent session's files), and the fact that **master CI failed on a tree that had already been
  version-tagged** (run `40167243876`: `checks`, `lint (.)`, `test`, `module-architecture`,
  `lint (identity-model)`; `architecture-gate` skipped).

---

## a) FULLY DONE (verified, with evidence)

### a1. The two inherited red gates, diagnosed rather than papered over

| Gate | Previous verdict | Root cause found | Resolution |
| --- | --- | --- | --- |
| `module-isolation` | ❌ "2 of 55 stages failed" | **Not code.** `/mnt/buildcache` was at 1444 MB free, under `go-cache-env.sh`'s 2048 MB floor (gotcha 12). The stage failed before compiling anything. | Re-ran the stage directly with the correct command: `bash scripts/checks/check-module-isolation.sh` → **15/15 modules OK (GOWORK=off)**. Disk had recovered to ~64 GB free (another session freed it). |
| `branching-flow` | ❌ `+3 added, −3 removed, ~133 modified` | A pure **transpose**: the same three `PHANTOM-TRANSPOSE` findings on `identity-model/authz.go` moved `:68/:112/:136 → :73/:117/:141` because S31's `NewAuthz` builds the `Authz` value before seeding (+5 lines). | Proved it with a SARIF key diff (`(rule, file, line, message)` set difference: exactly 3 adds, 3 removes, same file, same message), wrote a dated `2026-10-25` amendment row in `docs/analysis/triage-decisions.md`, re-pinned the baseline `168 → 168`. Gate green at that moment. |

That evidence chain (diff the SARIF, not the summary line) is reusable: the tool's `+N/−M` counts are
anchors, not facts.

### a2. S32 — identity-model domain (the source of truth)

| Change | File | Why |
| --- | --- | --- |
| `CredentialUse` value type (`CredentialID`, `SignCount`, `UserPresent`, `UserVerified`, `BackupState`) | `identity-model/credential.go` | One record shape for "what one assertion observed", used by the command, payload, fold and read model. |
| `(WebAuthnCredential) CounterRegressed(next)` | same | The WebAuthn §7.2 step-17 rule in ONE place: `next <= stored && (next != 0 || stored != 0)`. Counter-less authenticators (0 → 0) are not regressions. |
| `(WebAuthnCredential) ApplyUse(use)` | same | Copy-on-write application of a use; the caller must reject a regression first. |
| `UserPresent` / `UserVerified` on `CredentialCore` | same | The flags are now persisted domain data, not a provider-side guess. |
| `CredentialUsedPayload` | `identity-model/events.go` | Event payload: credential ID, counter, UP/UV/BS. |
| `RecordCredentialUseCmd` + `CmdRecordCredentialUse` / `EventCredentialUsed` | `commands.go`, `constants.go` | New command and event: **25 events / 21 commands** (was 24/20). |
| Fold case `EventCredentialUsed` | `fold.go` | Applies the use to the named credential; unknown credential → `Corruption` ("the stream cannot use a credential it never added"). Extracted to `applyCredentialUsed` / `applyCredentialRemoved` helpers. |
| `WebAuthnProvider.FinishLogin` returns `([]byte, error)` | `interfaces.go` | **The actual S32 bug**: the interface had no way to return the advanced credential, so every implementation had to drop it. The doc now states that dropping it is what kills clone detection. |

**Verified:** `go build ./...` OK · `golangci-lint run ./...` **0 issues** · `go test ./...` **ok**.

### a3. S32 — usermgmt (application wiring)

| Change | File |
| --- | --- |
| `decideRecordCredentialUse` — monotonic counter (regression → `Conflict "…the authenticator may be cloned"` with stored/asserted counts in context), unknown credential → rejection, deleted user → rejection, and a **no-op suppression** (0→0 + unchanged flags emits nothing, so counter-less authenticators do not fill the journal) | `usermgmt/es_decide_credentials.go` |
| `RegisterTyped` for the new command | `usermgmt/es_dispatch.go` |
| Read-model handler `handleCredentialUsed` (updates the in-memory credential; the SQL view persists the whole user JSON, so the counter rides along) | `usermgmt/es_readmodel.go` |
| `AuditActionCredentialUsed = "credential_used"` + `auditActions` mapping | `usermgmt/audit_log.go` |
| Event-catalog entry (payload fields documented) | `usermgmt/es_event_catalog.go` |
| Command/type/payload re-exports (`RecordCredentialUseCmd`, `NewRecordCredentialUseCmd`, `CredentialUsedPayload`, `CredentialUse`) | `es_commands.go`, `es_events.go`, `credential.go` |
| `Service.recordCredentialUse` — decodes the provider's returned credential and dispatches the command **before** the session is created, so a regression fails the login | `usermgmt/webauthn_service.go` |
| `reg:` / `login:` session-key namespaces + doc comment explaining the collision | same |
| UP/UV in `webAuthnUserCred` + `marshalWebAuthnUser` | same |

**Verified:** `go build ./...` · `go vet ./...` · `golangci-lint run ./...` **0 issues** · `go test ./... -count=1` **ok (17.6 s)**.

### a4. S32 — webauthn provider (the strategy module)

- `FinishLogin` now **returns** `fromWebAuthnCredential(credential)` as JSON instead of discarding it.
- `credentialData` and `toWebAuthnCredential` carry and use the stored `user_present` / `user_verified`
  values; `fromWebAuthnCredential` captures them. The former `UserPresent: true, UserVerified: true`
  fabrication is gone.
- Both broken test call sites (`provider_test.go:273`, `:372`) updated for the new signature; the
  spec-vector login test now also asserts the returned credential's ID survives the round trip.

**Verified:** `go build ./...` · `golangci-lint run ./...` **0 issues** · `go test ./...` **ok**.

### a5. S32 — systemadapter (declarative path) — where the gate earned its keep

1. `EventTypeDecoder()` registered `CredentialUsed`.
2. `CredentialView` gained `UserPresent` / `UserVerified`.
3. New `updateCredentialUsed()` fold wired into **both** `userLookup()` and `userScan()`.
4. Audit projection gained `auditFold(EventCredentialUsed, …, "credential_used")`.
5. `EventSchemas()` gained the event.

Without (5) the first test run failed with:
`system: dangling event subscription (coeffect gate): event types consumed but not declared in DomainConfig.Events: [CredentialUsed …]`.
That is the gate doing exactly its job. After (5), the exhaustive test failed on its own hardcoded count
(`All24EventTypesRegistered … count = 25, want 24`), which was then corrected to 25 and renamed.
**Verified:** `go test ./... -count=1` **ok** · `golangci-lint run ./...` **0 issues**.

### a6. S32 — tests written for the behaviour, not the implementation

`usermgmt/credential_use_test.go` (7 tests):

| Test | What it pins |
| --- | --- |
| `TestFinishLogin_PersistsSignCounterAndFlags` | Full ceremony through the stub: stored counter becomes 1, UP and UV are true, and a `CredentialUsed` event exists on the stream. **This is the S32 regression test.** |
| `TestFinishLogin_RejectsCounterRegression` | Stored 5, asserted 1 → login fails, error names the clone signal, stored counter stays 5, no event emitted. |
| `TestCeremonySessions_NamespacedPerCeremony` | Registration and login begun for the same user produce `reg:<id>` and `login:<id>`, and BOTH sessions survive. |
| `TestDecideRecordCredentialUse_NoOpForCounterlessAuthenticator` | 0→0 with unchanged flags emits 0 events. |
| `TestDecideRecordCredentialUse_Rejections` | Unknown credential, deleted user, missing user. |
| `TestDecideRecordCredentialUse_AdvancesCounter` | A real advance emits exactly one `CredentialUsed`. |
| `TestProvider_…` (webauthn, updated) | The returned credential is the one the caller can persist. |

### a7. Documentation/count drift caused by the new event/command — repaired

`check-domain-counts.sh` went from **2 drifts** to `OK`:

- `AGENTS.md`: 24 events / 20 commands → **25 events (22 lifecycle + 3 auth-audit) / 21 commands**.
- `FEATURES.md`: User payloads 12 → **13**; commands 20 → **21**.
- `systemadapter/domain_config.go` doc comments: 20 commands / 21 event types → **21 / 25**.

### a8. Two defects in my own change, found and fixed before reporting

1. **Impure fold.** `next := state` is a shallow copy, so writing `next.Credentials[i] = …` mutated the
   caller's backing array. `applyCredentialUsed` now allocates a fresh slice before writing. `applyCredentialRemoved`
   got the same treatment (it already allocated; the helper makes the rule explicit).
2. **`FoldUser` cyclomatic complexity 34 (> 30)** — `gocyclo` red in CI's `lint (identity-model)`. Extracted
   `applyCredentialUsed` and `applyCredentialRemoved`; now 0 lint issues.

---

## b) PARTIALLY DONE

1. **S32 end-to-end verification.** The four touched modules are green individually, but there has been no
   full `nix run .#check-modules` (55 stages) since the change, no `.#coverage-gate`, and no fresh CI run.
   Module-level green is not repository green.
2. **branching-flow.** Green after the a1 re-pin, then RED again at `+47 added, −32 removed, ~107 modified,
   =22 unchanged` (179 findings vs the 168 pinned). Composition of the 47 adds:
   - **Mine (10):** 4 × `BOOL_BLIND` — `CredentialCore`, `CredentialView`, `credentialData`,
     `webAuthnUserCred` now have 4 bools each; 6 × `COMPOSITION_mixin` on genuinely new twins
     (`CredentialUse` ↔ `CredentialUsedPayload`, `CredentialCore` ↔ `CredentialView`, plus payloads that
     the new field counts pushed over the shared-fields threshold).
   - **Not mine (≥2):** `EXPLICIT_PANIC` ×2 from the concurrent root session's validator panic.
   - **The rest:** anchor drift from that session's root rewrites and the docs eviction.
   No adjudication row written yet, so the gate stays red. This is the single most mechanical red left.
3. **Version/tag receipts.** identity-model `v4.14.0`, usermgmt `v4.15.0`, webauthn `v4.13.0` were cut by
   the concurrent session while `usermgmt/webauthn/provider_test.go` did not compile. The CHANGELOG entries
   exist (`f165854f`, `819be075`) but the release train does not prove the tag's test suite builds.
4. **Root `CHANGELOG.md`** has no entry for the `CredentialUsed` event kind; only the module CHANGELOGs carry
   receipts. The root catalog surface changed (event count, catalog endpoint content), so it should.
5. **`CURRENT_FOCUS.md`** still says S32 is "the next thread"; nothing records it as executed.
6. **Statements of "done" in the plan.** The master plan's S32 row is untouched (no `**DONE**` receipt with a
   commit hash), unlike S30/S31.

---

## c) NOT STARTED

1. Fold unit test in `identity-model` for `EventCredentialUsed`: the unknown-credential corruption path and
   the **purity** property (folding must not mutate the input state). The fix is in, the test is not.
2. `identity-model` test for `ApplyUse` / `CounterRegressed` as a table (0→0, 0→5, 5→5, 5→1, 5→0).
3. webauthn unit test for `toWebAuthnCredential` ↔ `fromWebAuthnCredential` preserving UP/UV/counter.
   The long spec-vector test asserts the ID only.
4. systemadapter declarative test that actually folds a `CredentialUsed` into `UserView` and reads the
   advanced counter back (the coeffect gate proves declaration, not computation).
5. `nix run .#check-modules`; `nix run .#coverage-gate`; `nix run .#lint`; `nix run .#test` as one battery.
6. The master-plan S05 tail work is finished by another session (tail 38 → 1); S16 and the eviction (S13/S14,
   Q2 = delete+manifest) were also executed by that session. **Not verified by me.**
7. Any element of Lane D beyond S32.

---

## d) TOTALLY FUCKED UP

1. **A version bump and three tags landed on a tree whose tests did not compile — and I contributed to that.**
   I changed the provider interface and had not yet compiled the webauthn module's own tests when the
   concurrent session committed and tagged. Master CI is red (`40167243876`). The two call sites were a
   2-line fix; the window between "interface changed" and "all implementers' tests compiled" was the failure.
   A Principal Engineer's rule: *never leave an interface change without every implementer compiled in the
   same step.*
2. **I shipped an impure fold.** Mutating through `next := state`'s shared slice is a real correctness bug:
   a caller folding a *rejected* branch could have seen its input state change. It was caught only on
   re-reading the code, not by a test. That is the second time this session that "it compiles" was mistaken
   for "it is right".
3. **I broke `FoldUser`'s complexity budget and only noticed via CI's job list**, not by running the module
   linter after the edit. `lint (identity-model)` failed for a reason I could have seen in 20 seconds.
4. **I forgot that systemadapter is an exhaustive-schema consumer.** The declarative path required four
   separate registrations (decoder, schema, two view folds, audit fold) and I found them one failing test
   at a time. The coeffect gate caught it, which is the system working; a `rg CredentialRemoved` sweep
   across the repo would have found all five sites in one command before the first test run.
5. **I corrupted a file with a bad edit.** Inserting the decider dropped a newline and fused two lines
   (`…error) {\treturn func(…`). I noticed by re-reading, not by the tool. Edit hygiene: read back the
   edited region every time.
6. **I wrote checks in the wrong directory twice** (`go vet ./usermgmt/` from the repo root builds the ROOT
   module, not usermgmt). Wasted minutes and, worse, produced a misleading error attributed to another
   session's file.

---

## e) WHAT WE SHOULD IMPROVE

### What I forgot (the honest list)

- To run **module lint + module tests immediately after each module's edit**, before moving to the next.
  Every one of the four reds in section d would have been caught by that loop.
- That `usermgmt` is a **published module with its own go.mod** — `go vet ./usermgmt/` from the root tests
  root, not usermgmt.
- That changing a **struct's bool count** trips `BOOL_BLIND`, and that adding a **twin struct** trips
  `COMPOSITION_mixin`. The branching-flow ledger already had both as rejected classes; knowing that in
  advance would have made the adjudication paragraph part of the same change.
- That the **SQL read model needs no new column**: the user's credentials are serialized into
  `users_view.data`, so the counter persists automatically. I verified this rather than assuming, but only
  after the fact — it should have been step one of the design.

### What I could have done better

- **Smaller atomic steps.** "Domain event + app wiring + provider signature + declarative path" is four
  commits, not one working tree. Under concurrent sessions, one working tree is a race.
- **Verify before claiming.** My first instinct after the modules built was to move on; two of the three
  later failures were build-time, not reasoning-time.
- **Use the gates as a checklist, not an oracle.** `rg` for every other registration of the neighbouring
  event (`CredentialRemoved`) would have enumerated the five sites in one command.
- **Record the concurrency explicitly in the report from the start.** Half of this session's confusion came
  from mistaking another agent's commits for my own.

### What I could still improve (before calling S32 done)

1. Fold purity + unknown-credential tests. 2. `ApplyUse` table test. 3. Provider flag round-trip test.
4. Declarative fold test. 5. Adjudicate + re-pin branching-flow with a dated ledger row.
6. Full `check-modules` + coverage battery. 7. Root CHANGELOG receipt + plan/`CURRENT_FOCUS` receipts.
8. Re-run CI and confirm green before any further tag.

---

## f) Next — 50 concrete items, in order

1. Adjudicate the 47 branching-flow adds, write the dated `triage-decisions.md` row (my 10 under the
   existing `BOOL_BLIND` / `COMPOSITION_mixin` reject verdicts; name the foreign `EXPLICIT_PANIC` pair),
   re-pin, commit with the ledger row.
2. Commit the systemadapter schema/decoder/fold-count fixes (already green locally).
3. Add the `identity-model` fold test: `CredentialUsed` applies; unknown credential → error; **input state
   unchanged** (purity).
4. Add the `CounterRegressed` / `ApplyUse` table test (0→0, 0→5, 5→5, 5→1, 5→0, 5→0 with flags change).
5. Add the webauthn conversion round-trip test (UP/UV/BS/counter/aaguid/transports).
6. Add a systemadapter declarative test that folds `CredentialUsed` and reads the advanced counter.
7. Run `nix run .#test` and record the receipt.
8. Run `nix run .#coverage-gate` (15 gated modules).
9. Run `nix run .#check-modules -- --report` to 55/55.
10. Run `nix run .#lint` and reconcile with CI's golangci version (2.14 vs 2.15 drift).
11. Add the root `CHANGELOG.md` receipt for the new event kind + catalog content change.
12. Mark S32 done in the master plan with the commit hash + verification line.
13. Update `CURRENT_FOCUS.md`: S32 done, next thread named.
14. Decide the audit-log question (Q3) and implement the verdict.
15. Decide the tag question (Q1); if retag: bump patch, re-tag, verify the module proxy.
16. Investigate the 44 root lint issues **with** the session that authored them (do not fix another
    session's in-flight code unilaterally).
17. Re-run CI (`gh run watch`) and confirm all jobs green.
18. Verify `.#check-css-bundles` / class-set gates still pass after the docs eviction (no CSS touched, cheap check).
19. Confirm the docs-eviction session's `CORPUS_BASELINE` (30101) matches a fresh `--ratchet` run.
20. Confirm `TAIL_BASELINE` (1) equals the live tail and that this report's addition raises it to 2.
21. Re-check `check-docs-links` after 617 files vanished (inbound links from live docs).
22. Re-check `docs/ARCHIVE_EVICTED.md` keep-list against the new tree.
23. Add a `CredentialUsed` row to any event documentation/guide that enumerates events.
24. Grep for other exhaustive event lists (`rg 'CredentialRemoved'`) and add `CredentialUsed` where the
    list is meant to be complete.
25. Add `CredentialUsed` to `integration_test` actor-attribution coverage if it enumerates event kinds.
26. Check `setup`/`adminui`/`dashboardui` suites for anything counting events or credentials.
27. Run the `usermgmt` race suite (`-race`) on the touched package.
28. Bench-check the extra event on the login path (`BenchmarkSpikeBaselineVsAppkit` style) — expected noise,
    but prove it.
29. Decide whether `CredentialUsed` should be emitted when the counter is 0 but flags change (currently yes)
    and document the rule.
30. Document the clone-rejection UX: what a consumer shows a user whose counter regressed.
31. Consider `CloneWarning` persistence (rejected vs degrade-softly) — a decision record.
32. Add a migration note: pre-upgrade credentials have `user_present:false`/`user_verified:false`; the
    first post-upgrade login fills them.
33. Check `crypto.go`/backup flags for the same fabrication class.
34. Add `CredentialUsed` to the `usermgmt` README's event/command table.
35. Update `docs/guides/` where the WebAuthn flow is described.
36. Add the `record_credential_use` command to the cqrs-lint-visible command list (if enumerated).
37. Verify `check-command-bijection` passes with 21 commands.
38. Verify `check-session-route-wrappers` still passes (no route changed).
39. Re-run `check-status-rows` / `check-todo-hygiene` after this report + any TODO additions.
40. Add a TODO row for "audit-log noise: CredentialUsed vs LoginSucceeded" if Q3 stays open.
41. Add a TODO row for the tag-hygiene gap ("tags must not outrun a compiling test suite").
42. Propose a `verify-touched-modules.sh` helper: build + vet + lint + test for a module list (the loop this
    session should have run).
43. Consider making `module-isolation`'s low-disk failure message *name the stage as environmental*, so a
    future session does not read it as a code failure.
44. Re-run `#check-branching-flow` after the re-pin and record the verdict line.
45. Re-run the webauthn spec-vector suite with `-count=1 -race`.
46. Spot-check that the SQL read model really round-trips the counter (hydrate → login → hydrate).
47. Confirm the audit log's `credential_used` action appears once per login in a test (assert or drop).
48. Review the `CredentialUse` naming against the domain language doc; adjust if "use" is wrong.
49. Once green: push through the strict pre-push gates and watch CI.
50. Write the next status report from a green tree.

---

## g) Three questions I cannot answer myself

1. **The tags.** identity-model `v4.14.0`, usermgmt `v4.15.0`, and webauthn `v4.13.0` were cut while the
   webauthn test suite did not compile (it does now, untagged). Module consumers are unaffected (tests are
   not compiled by importers), but CI on those tags is red. Do you want a retraction/re-tag, a patch release
   with the fix, or leave them and let the next train carry the fix?
2. **branching-flow adjudication depth.** The 47 adds are 10 mine under standing reject verdicts, ≥2 the
   concurrent root session's `EXPLICIT_PANIC`, and the rest anchor drift. Do you want the documented
   **provisional acceptance** (one dated ledger row, the 2026-10-12 precedent) or per-finding adjudication
   before the re-pin?
3. **Audit-log placement of `CredentialUsed`.** I put it in `allUserEventTypes`, which means the in-memory
   audit log records one `credential_used` entry per passkey login, duplicating `LoginSucceeded`. Is that
   desirable security telemetry, or should the event be read-model-only (excluded from the audited set)?

---

_Report scope: this session. Concurrent-session actions are named as such and are not claimed as mine._
