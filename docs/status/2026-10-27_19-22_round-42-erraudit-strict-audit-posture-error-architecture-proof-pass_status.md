# Round 42 — erraudit strict-audit remediation + error-architecture proof pass

**Session:** 2026-10-27, ~17:45–19:25 CET · **Branch:** `docs/readme-table-of-contents` (master diverges — see §d1)
**Trigger:** `erraudit lint ./... --max` → 15 `generic_return` warnings, exit 2. Mid-session owner challenge: *"You sure our error management is superb!??!?! … any your not just lazy?"* — which converted a suppress-and-move-on pass into a genuine error-architecture audit.

---

## a) FULLY DONE

| # | Item | Evidence |
|---|------|----------|
| 1 | **All 15 `generic_return` findings classified as posture mismatch, not defects** — decoder callbacks are identity-bound to `func(*http.Request) (T, error)` at 8 `options_decode.go` call sites (adapters would introduce typed-nil bugs); `WriteJSON`/`DecodePaginationStrict`/`Spec.Validate` must keep the `error` contract (public API freeze, openapi is stdlib-only by design, `errors.Join` IS the specific type); go-error-family's own contract is "share the protocol, not the implementation" — the dynamic type is already `*errorfamily.Error`, so `errors.AsType` works today | Per-site reading of all 15 functions |
| 2 | **Canonical strict audit established and verified workspace-wide:** `GOEXPERIMENT=jsonv2 erraudit lint ./... --max --disable generic_return` → rc 0, and a per-module sweep of all 29 go.work members → **zero findings** | Full sweep this session |
| 3 | **Empirical tool facts (fixture-proven):** `lint --max` enables `--no-suppress` → ALL nolint directives bypassed (probe file with statement-trailing AND function-level directives, both still flagged); `lint` rejects `--modules` (analyze-only flag; the root-module hint suggesting `--modules=all` for lint is wrong — sweep per module dir) | Probe + live rc checks |
| 4 | **Both `context_loss` criticals re-greened** (inventory was red on root + usermgmt before this session): SSE replay adapter propagates verbatim but the handler logs `lastEventID` on both replay-failure paths (serve.go:368/380); webauthn FinishLogin passthrough is safe because `recordCredentialUse` attaches `user_id` | `erraudit-inventory` TOTAL=0 across 29 modules |
| 5 | **The audit forced a REAL fix:** `recordCredentialUse`'s unmarshal branch did NOT attach `user_id` — my own HELPER annotation claim was false for that path (the linter was right in a way the first pass missed). Now attached on all three branches | usermgmt/webauthn_service.go:323-327 |
| 6 | **Both boundary claims mechanized by tests** so the annotations cannot rot: `TestRecordCredentialUse_AttachesUserIDContext` (unmarshal + decider-rejection branches, `errors.AsType[*errorfamily.Error]` + `HasContext`/`ContextValue`) and `TestServeDomainEvents_ReplayFailureLogsCursor` (failing store → WARN log must carry `lastEventID=evt-42`, stream must survive; non-parallel — swaps global slog default with cleanup) | Both green in-package |
| 7 | **cmd/cqrs-htmx-upgrade converted to the fleet error stack:** 13 banned `fmt.Errorf` sites → `errorfamily.Newf/Wrapf` with codes (`cqrshtmx.upgrade.*`) and reasoned families (parse=Rejection, env=Infrastructure, invariant=Corruption); **5 nolints retired** (4× err113 + 1× erraudit); the analyzer then demanded `qualToPath` context on the corruption error (it holds errorfamily constructors to a HIGHER standard than `fmt.Errorf`) — attached. Build + package tests + dry-run smoke green | git diff cmd/, tests ok |
| 8 | **Decoder error-UX hardened:** the 413 path now reports body bytes + `max_body_size` (`WithContextAny`) while preserving sentinel identity (413 via `MapError`'s `errors.Is` chain walk — integration-verified); ambiguous form key now carries `WithContext("form_key", key)` (client-actionable field name stays in prose AND machine context) | integration_advanced_test 413 entry green |
| 9 | **Stale suppression debt cleaned:** `oauth2/provider.go:428` nonce-mismatch nolint no longer suppressed a live finding — removed by hand (never blanket `nolint-audit --fix`: it deletes `legacyerrors` tokens `lint` still honors). `nolint-audit` now 0 stale in root (13/13 needed) and usermgmt (10/10) | nolint-audit output |
| 10 | **Docs recorded:** gotcha 44 (`--max` bypasses nolint; `generic_return` posture; canonical invocation; per-module sweep; mechanized-test receipts), gotcha 8 addendum (inventory went stale-red again, re-greened), AGENTS.md index row 44 + count 43→44, `CORPUS_BASELINE` bumped (34037), corpus ratchet + tail budget verified green | gates green |
| 11 | **Formatter canonical:** `buildflow format` over the repo; only delta was an import-order fix in the new test file — everything else was already canonical | git diff |

## b) PARTIALLY DONE

1. ~~**`golangci-lint` verification**~~ CLOSED at close-out (~19:45): scoped direct runs with TRUE exit codes — root `golangci-lint run` rc=0 and `usermgmt` rc=0, zero issues each (golangci's "unknown linters in //nolint directives: erraudit" warning is the expected two-token topology, gotcha 8). The buildflow `-s` qualifier-scoping quirk remains open as tooling trivia, no longer load-bearing. (The first confirmation attempt repeated the §d3 pipe-status lapse — `| tail; echo $?` — caught and re-run.)
2. **The owner's literal command `erraudit lint ./... --max` still exits 2** — by recorded design (posture disable required). The exact command cannot be made green without either the disable flag or harmful API churn (§a1).
3. ~~**Filtered-path replay logging (serve.go:368)**~~ CLOSED at close-out: `TestServeDomainEvents_ReplayFailureLogsCursor_FilteredPath` drives the `WithSSEFilter` branch through `filteredReplayStore` and pins the same `lastEventID` cursor context — mutation-proven (fails with the filtered log line's cursor stripped, passes restored); both halves of the §a4 annotation are now test-backed. The slog-prefix-as-constant suggestion (§f26) stays open (harvested).
4. **Milestone commits** — the auto-commit daemon swept the working tree (9292936f on master, later commits on this branch); no deliberate milestone commit was made by this session (harness forbids committing without explicit owner request).
5. **`buildflow format` verdict** — 328 steps green, but 5 steps were lost to spawn starvation (15m load ≈ 189) and the pipeline exited `environment_starvation`; the touched files are clean, but the run was not a full green.

## c) NOT STARTED

1. **Merging this branch with master** — required before anything else lands (see §d1).
2. ~~**TODO_LIST/ROADMAP harvest**~~ DONE at close-out (~19:50): §f routed into `TODO_LIST.md` (merge row in P1 owner-gated per §g1; strict-sweep cluster, error-context/CHANGELOG row, and test/docs tails in P2; owner-gated items marked in-row). Items already done by this same close-out were dropped per HARVEST rule 3 (golangci confirm, the harvest itself); §f18 + §f21 were EXECUTED in-session instead of routed (session-start runbook: branch-check-first command + strict-audit finish item; `docs/agents-notes.md`: dated arc entry; corpus baseline → 34155). Hygiene gate green (79 rows, 0 dead).
3. **CI/check-modules wiring for the canonical strict sweep** — the strict audit is currently a documented manual invocation; `erraudit-inventory`'s CI coverage is the offline stub self-test only.
4. **Structured error JSON:** `errorfamily` `ErrorContext()` values render into logs (logging.go:279) but NOT into the RFC 7807 problem-details body (`structured_error.go` carries only request-ID context) — machine context is invisible to API clients.
5. **Upstream erraudit feedback:** (a) project-level config file so `--disable generic_return` isn't per-invocation folklore; (b) `lint`'s misleading `--modules=all` hint; (c) `generic_return` FP class for callback-bound generic functions.
6. **Skill feedback loop:** the `go-error-modernization` skill documents the `legacyerrors` class but not the `generic_return`/`--max`-bypasses-nolint class — fleet-wide trap, belongs in the skill.

## d) TOTALLY FUCKED UP

1. **Master/branch split-brain on the serve.go annotation (worst item).** The daemon auto-committed this session's FIRST edit round to **master** (9292936f) — including the WRONG suppression form (own-line `//nolint:wrapcheck,erraudit` above the return, which erraudit does NOT honor — proven by the still-red critical after that edit) — plus an em dash in source (repo ban). The owner then moved the session to `docs/readme-table-of-contents`, which carries the corrected trailing form. **Until these branches merge, master's erraudit-inventory stays RED on that one finding, and master lacks the user_id fix, the CLI conversion, and the decoder hardening.** Root cause: editing before fully knowing the suppression-line rule, and not noticing the branch switch for two turns.
2. **The laziness the owner called out was real.** First pass classified both `context_loss` findings as false positives and nolint'd them WITHOUT mechanizing the claims; one claim was outright false (unmarshal branch). The correct order — prove the invariant with a test, then annotate — was only reached after the challenge. Lesson recorded: never let an annotation assert something a test doesn't back.
3. **Shell/rc verification hygiene lapses early in the session:** a `head`-piped erraudit run reported `RC=0` (pipe status, not tool status); the wrong-line nolint cost a verification cycle; the em-dash violation.

## e) WHAT WE SHOULD IMPROVE

1. **Mechanize-then-annotate, in that order** — every `//nolint:erraudit // FP|HELPER` claim should be born with a test citation (this session's two new tests are the template).
2. **Check `git branch --show-current` before every edit burst** — the daemon + owner branch switches silently relocated work this session.
3. **erraudit suppression mechanics should be in the edit checklist** (directive covers its own line + the statement-start line; trailing-on-return is the safe form; never own-line above a return).
4. **The posture disable should live in project config, not session folklore** — upstream erraudit feature request, then a one-line project file instead of a documented flag.
5. **Don't run repo-wide quality pipelines at load ≈ 189** — starvation loses steps non-deterministically; re-run `buildflow format` when the machine is quiet for a clean full green.
6. **The gotcha-class grep (`rg 'nolint:erraudit // (FP|HELPER|LIVE-SECRET)'`)** requires the reason ON the directive line — write it that way from the first edit, not as a cleanup.

## f) Next tasks (ranked; grounded in this session's observations)

| # | Task | Why it matters | Size |
|---|------|----------------|------|
| 1 | Merge `docs/readme-table-of-contents` ↔ master; resolve the serve.go nolint-form conflict; re-run `erraudit-inventory` on the result | Kills the master-side red (§d1); unblocks all other work | S |
| 2 | Confirm `golangci-lint@root` green (in flight) and run `golangci-lint@usermgmt` for the touched module | Complete the lint leg of verification | S |
| 3 | HARVEST this report into TODO_LIST (docs-health) | §f must not be entombed | S |
| 4 | Wire the canonical strict sweep (`lint --max --disable generic_return`, per module) into `check-modules` or a flake app | Posture becomes enforced, not remembered | M |
| 5 | Extend `erraudit-inventory.sh` with a strict mode (or sibling app) covering the lint-side audit | One durable primitive instead of ad-hoc loops | M |
| 6 | File upstream: erraudit project-config file support (`--disable` per project) | Removes the per-invocation folklore | S |
| 7 | File upstream: `lint`'s `--modules=all` hint is wrong for lint (analyze-only flag) | Saves every future session a failed run | S |
| 8 | File upstream/discuss: `generic_return` should exempt callback-bound generics (`func(T) (R, error)` identities) | The 8-site constraint is a systematic FP class | S |
| 9 | Render `ErrorContext()` into the RFC 7807 problem-details body | Machine context currently invisible to API clients | M |
| 10 | Extend `TestServeDomainEvents_ReplayFailureLogsCursor` to the filtered path (serve.go:368) | The annotation claims "both paths"; only one is tested | S |
| 11 | Run full `nix run .#check-modules` | Session only ran targeted gates | M |
| 12 | Run full `nix run .#test` battery | Same | M |
| 13 | Update the `go-error-modernization` skill with the `generic_return`/`--max` class + suppression-line mechanics | Fleet-wide trap, skill only covers `legacyerrors` | S |
| 14 | Decide BuildFlow erraudit-step posture: should the step pass `--disable generic_return`? | BuildFlow gates on erraudit criticals today; the strict mode is unchecked territory | S |
| 15 | CHANGELOG entries for the behavioral error changes (413 wording + context, CLI codes/families) before the next release train | Consumers see new error strings | S |
| 16 | Map `errorfamily.ExitCode` into the codemod CLI's `fatalf` (Rejection vs Infrastructure exits) | Exit codes currently always 1 | S |
| 17 | Re-run `buildflow format` on a quiet machine for a starvation-free green | 5 steps lost this run | S |
| 18 | Add the erraudit posture line to the session-start runbook's three commands ("check branch" + canonical audit) | This session's two process traps, mechanized | S |
| 19 | Consider `nolint-audit` CI wiring (currently local-only per gotcha 8's two-token topology) | Stale-suppression debt re-accumulates silently | M |
| 20 | Sweep sibling repos for the same `--max` trap (the skill fan-out carries the legacyerrors class only) | Same footgun exists fleet-wide | M |
| 21 | Long-form session note in `docs/agents-notes.md` (dated history of the master/branch split-brain) | Repo culture: provenance for the conflict | S |
| 22 | Evaluate `WithContextAny` value rendering for non-scalar context (maps like `qual_to_path` print as Go syntax) | Diagnosability of corruption dumps | S |
| 23 | Re-check the `DecideRecordCredentialUse` conversion-failure branch reachability (`aggIDFromUser`) for a third test case | Invariant test covers 2 of 3 branches | S |
| 24 | Probe whether erraudit's context_loss checker would accept a narrower fix than full context attachment (message-embedded vars) | Avoid future over-attachment noise | S |
| 25 | Add `--max --disable generic_return` + expected-candidate-count guard to `scripts/selftests/test-erraudit-inventory.sh` IF task 4/5 change the script | Self-test parity with any gate change | S |
| 26 | Consider exporting the SSE replay-failure slog prefix as a testable constant | Log-assertion tests currently match on string literals | S |
| 27 | Review `filteredReplayStore.EventsAfter` (serve.go:236, single-expression verbatim return) for symmetry with the annotated adapter | Same class, unannotated because branchless — verify intent | XS |
| 28 | Close the loop with the owner challenge: a brutal-self-review pass over this session's diff | The skill pair (status + self-review) is the repo's stated combo | M |

(Items beyond 28 would be speculation, not session-grounded work — the list stops where the evidence does.)

## g) Questions I cannot answer myself

1. **Merge direction/timing:** should `docs/readme-table-of-contents` merge INTO master now (killing the §d1 red immediately), or is the README TOC work still in flight and master should receive a forward-port of the fixes first?
2. **Gate policy:** should the canonical strict erraudit sweep become a BLOCKING gate (check-modules + CI, like the tail ratchet) or stay an audit tool per the linter-building skill's warning against gating on full strictness output?
3. **Fleet posture:** is `--disable generic_return` the right posture for the whole LarsArtmann fleet (skill/BuildFlow-level default), or deliberately per-project (this repo documents it; other repos decide their own)?

---

**Verification state at write time:** build green (root, usermgmt, cmd) · transport/usermgmt/cmd/root-targeted/integration tests green · `erraudit-inventory` TOTAL=0 (29 modules) · `lint --max --disable generic_return` rc 0 (root + all 28 nested modules) · `nolint-audit` 0 stale (root, usermgmt) · corpus ratchet + tail budget green (after round-40 archive + baseline 34141) · golangci green fleet-wide via buildflow (scoped-rc confirmation open) · full test battery NOT run.

**Close-out addendum (2026-10-27 ~19:55):** strict sweep re-verified live — 29/29 modules rc=0 · golangci scoped rc=0 confirmed (root + usermgmt, true exit codes) · TODO_LIST harvest landed (hygiene gate green) · §f18/§f21 executed in-session (runbook + agents-notes; corpus baseline 34141 → 34155, ratchet green) · battery: the first `.#test` run's visible tail was all-ok but BOTH the early-module output and the true rc were lost to a `| tail` pipe (the §d3 class, twice in one day) — re-running with full log + direct rc capture, verdict lands in the log at `/tmp/battery-round42.log`.
