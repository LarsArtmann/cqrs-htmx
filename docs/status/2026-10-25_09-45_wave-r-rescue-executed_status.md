# Status: Wave R executed — red-master rescue, S32 proof (found a real bug), D04 in flight

**Written:** 2026-10-25 09:45 CET · **Supersedes:**
[`2026-10-25_06-52_lane-d-plan-and-red-master-rescue-train_status.md`](archived/2026-10-25_06-52_lane-d-plan-and-red-master-rescue-train_status.md)
and
[`2026-10-25_07-43_credential-use-sign-counter-persistence_status.md`](archived/2026-10-25_07-43_credential-use-sign-counter-persistence_status.md)
(their open items are carried here).
**Plan of record:** [`docs/planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md`](../planning/2026-10-25_08-05_SUPERB-ci-rescue-s32-proof-and-full-micro-breakdown-pareto-plan.md)
**Tree:** clean, master **ahead 11, unpushed** — CI has NOT seen this session's work yet (R05 pending).

## §a FULLY DONE this session (all committed locally)

| Work | Receipt |
| --- | --- |
| **R01a triage** — CI run `40171760161` decomposed: 2 red jobs = `lint (.)` (44 root findings) + `test` root Coverage (88.1% < 90%); the other two red causes from the 08-05 plan (domain counts, identity-model gocyclo, webauthn compile) were already green at HEAD | finding list `/tmp/ci_lint_findings.txt` analysis |
| **R01b root lint 44 → 0** — err113 → `errorfamily` Corruption/Infrastructure constructors (sync_pull, sync_push); openapi keeps its documented dependency-free contract via static sentinels + `%w`; varnamelen/wsl_v5/recvcheck (Clone unified to pointer receivers)/gci/golines/revive/modernize/nolintlint all fixed at source, zero suppressions | root golangci 2.14.0 (== CI pin): **0 issues** |
| **R01b+ errorfamily scanner blind-spot FIXED** — `WalkDir(".")` self-skipped (name "." matches the hidden-prefix rule), so the ROOT module was never scanned while the security job stayed green. Root never skipped now; `openapi` (dependency-free by contract) and `cmd` (CLI diagnostics, mirrors its `//nolint:err113` markers) exempted with documented rationale. The fix surfaced **6 more real violations** in systemadapter → converted to `WrapInfrastructure`/`Wrapf` | scanner rc=0 on full walk; systemadapter lint 0 |
| **R01c root coverage 88.1% → 91.8%** — openapi clone tests (9 zero-covered helpers), canonical failing-writer error paths, deprecated SSE re-export delegation, `GenerateUserID`, `WithHTTPStatus`/`WithRetryAfter` message preservation, `StructuredError.Error` | `go test -coverprofile` total **91.8%** (gate 90%) |
| **R01d branching-flow re-pinned 168 → 179** — gate verdict +43/−32/~107/=22; dated ledger row buckets the adds (sibling S32 model / S43 canonical shadows / this session's anchor shifts) under existing verdicts; FIRST verdict for the panic class (openapi builder fail-fast guards: reject) | gate **OK +0/−0**; ledger + baseline committed together |
| **R02 S32 proof tests + a REAL BUG found and fixed** — fold apply / unknown-credential corruption / **purity**, `CounterRegressed` + `ApplyUse` tables, declarative end-to-end lift test. The declarative test exposed: **`RecordCredentialUse` was never registered in systemadapter's DomainConfig** — `system.New()` consumers could not record assertion state; the declarative UserView kept the registration-time counter forever (clone detection dead on the system path). Fixed: usermgmt `DecideRecordCredentialUse` export → **usermgmt/v4.16.0 tagged through verify-tag.sh guards** (content assertions + pre-push CI-parity green) → requires swept via bump-dep (all modules tidy+verify+build+vet green) → strict release-train **0/0/0** | `651fa40b`, `94b6e729`; 4 module suites green hermetic |
| **R03 S32 receipts** — root CHANGELOG Added entry (`CredentialUsed` + catalog 25 events); master plan S32 row → **DONE** with tags + proof; CURRENT_FOCUS rewritten (Wave R done, Wave D active/next); usermgmt README published-language section (25 events / 21 commands, both dispatch paths); provider-implementation guide de-staled to the real interface (`userJSON`/`credentialJSON` shape — it showed the PRE-v4.14 signatures) + assertion-state contract note; CORPUS_BASELINE 30732 → 30734 in the same commit | `59487eb8` + daemon `d0faae0f`; docs-links/status-rows/domain-counts gates green |
| **R04.1 test battery** — `nix run .#test` full race battery, all modules **rc=0** | job output captured |

## §b PARTIALLY DONE

- **R04 battery** — test ✓; **lint ✗ RED** (see §d.1); `.#coverage-gate` and `.#check-modules` NOT RUN yet.
- **D04 golangci alignment** — analysis complete, **no edits applied yet**: the flake's nixpkgs golangci is **2.15.0**, CI's action runs **2.14.0**; 2.14 = 0 issues repo-wide, 2.15 flags **201 wrapcheck findings — every one the `go-error-family` constructor class** (`New*`/`Wrap*`/`.WithCause`/`.WithContextAny` — the sanctioned wrapping layer; the existing ignore-sigs already exempt the stdlib analogues). The enumerated ignore-sigs set is ready; the CI up-pin decision is §g.1.

## §c NOT STARTED

R05 (push + CI watch + tag-hygiene), R06 (green-tree receipt), D01 (delivery receipt: Appendix D, CHANGELOG S35/S41/S44, S27 harvest), D02 (docs/doctrine S45/S52/S53/S54), D03 (tooling S47/S48), D05–D09 (code fidelity S37/S38/S42/S46+S49/S58), D10 (S50/S51 scripts), D11 (S57 loginpage E2E), D12 (splits), D13 (owner packet), D14a/b (harvested backlog), D15 (BuildFlow filing + bench window), D16 (parked-row audit).

## §d TOTALLY FUCKED UP (honest)

1. **I reported `LINT_RC=0` from a piped `tail` exit code** while the output said FAIL — the rc capture was wrong and could have masked the red verdict; caught by reading the output text, but the habit is the bug (fix in §e).
2. **Predicted instead of read, twice-ish**: asserted exact `String()`/`Error()` formats in tests without reading their implementations (SSEEventID carries a type prefix; errorfamily messages carry the `[family:code]` prefix) — three wasted test runs. Same class: `UserState.Exists` is a method not a field; `Operation{}.Clone()` literal not addressable under the new pointer receiver.
3. **A multiedit would have REPLACED the existing 2026-10-25 triage row** (destroying the S31 adjudication) instead of appending — the read-before-edit refusal caught it; then one sub-edit still failed and I verified only by grepping the file afterward.
4. **Two lint-fix rounds** because I hand-wrote what a formatter would do (long sentinel line, an over-120 `WrapInfrastructure` line) instead of running the formatter/module lint after each batch.
5. **Daemon races**: several descriptive commit messages were lost to heuristic auto-commits that captured the files first (content intact, receipts in this report + the plan).

## §e IMPROVE

- Capture gate rc with `PIPESTATUS[0]` or run unpiped — never from the last pipe segment.
- Read a type's `String()`/`Error()` before asserting its exact output; prefer `strings.Contains`.
- Run module lint immediately after each edit batch, not after the whole wave.
- Watch the CI run triggered by any push/tag immediately (`gh run watch`) — the v4.16.0 tag push's run is still unwatched.
- Verify the command-bijection gate after adding the 21st `RegisterTyped` (it runs in `check-modules`, which is still pending).

## §f NEXT (in execution order, ≤50)

1. **D04**: apply the wrapcheck ignore-sigs set (`go-error-family.New`, `go-error-family.Wrap`, `.WithCause(`, `.WithContext(`, `.WithContextAny(`), decide the pin direction (§g.1), re-run `.#lint` → 0, AGENTS quick-ref row.
2. **R04 remainder**: `.#coverage-gate` (expect floors to hold at 91.8 root), `.#check-modules` (55 stages; watch command-bijection + the scanner's new root walk).
3. **R05**: push the 11-commit range through both strict pre-push gates, `gh run watch` to terminal green, tag-hygiene table (§g.2).
4. **R06**: green-tree receipt (supersedes this report's red-state framing).
5. **D01**: master-plan Appendix D reconciliation; CHANGELOG receipts S35/S41/S44; S27 TODO harvest (strike executed rows, verify each).
6. **D02**: S45 EventHandler docs + identity-model README; S52 train-preflight mandate; S53 credential memo; S54 v5 migration plan skeleton.
7. **D03**: `bump-dep --lint` leg + fixture; templ fmt→generate chain in `.buildflow.yml`.
8. **D05–D09**: S37 usermgmt tails, S38 identity-model tails, S42 systemadapter fidelity, S46 seams + S49 StateShape, S58 schema stamping + health mirrors.
9. **D10**: `check-upstream-treadmill.sh` + `wave-tag.sh` (three new receipts from this session's protocol drift feed S51).
10. **D11**: loginpage Playwright WebAuthn E2E + render golden.
11. **D12**: the five file splits. 12. **D13**: owner packet + watches. 13. **D14a/b**: 14 harvested-row micro-batches. 14. **D15**: BuildFlow filing + bench window. 15. **D16**: parked-row verdicts.
16. New this session: scanner self-test fixture for the `.`-walk blind-spot class; verify `check-command-bijection` after the 21st registration; watch the v4.16.0 tag's CI run.

## §g Questions I cannot answer myself

1. **golangci direction (D04/Q1)**: up-pin CI's action to 2.15.0 + the errorfamily ignore-sigs (plan's recommendation — forward, kills the split brain, and the 201 findings are all the sanctioned-constructor FP class), or pin the flake's nixpkgs golangci down to 2.14.0 (fights nixpkgs forward motion; next bump re-breaks)?
2. **Tag hygiene (R05.3)**: the three tags cut on this morning's red tree (identity-model v4.14.0, usermgmt v4.15.0, usermgmt/webauthn v4.13.0) — now that the tree is healing green: leave-and-document, or retract/patch-release? (Importers are unaffected — tests are not compiled by consumers — but the tags' CI story is red.)
3. **Minor-version ratifications from the 06-52 report** (carried): `FinishLogin` interface change shipped as v4.14.0/minor (one official implementor; CHANGELOG notes technically-breaking) and the auth-audit journal necessarily riding usermgmt v4.15.0 — ratify or flag for v5?
