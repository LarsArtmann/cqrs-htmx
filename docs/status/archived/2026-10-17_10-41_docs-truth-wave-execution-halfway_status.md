# Status Report 2026-10-17 10:41 — Docs-Truth Wave Execution (M01–M17 Landed, Halfway Point)

> **ANNOTATED 2026-10-25** — ~~this report~~ **SUPERSEDED** by [`2026-10-24_12-52_archive-ritual-corpus-ratchet-and-authz-hierarchy_status.md`](2026-10-24_12-52_archive-ritual-corpus-ratchet-and-authz-hierarchy_status.md), which second half landed (M18 goth-stack, M21/M12 packets, S04b/S05/S23 via the master plan + 12-52); M12-14 are ROADMAP OQ30-32; f-11 crush-config row; M10 codespell harvested to the tooling-canary row. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-25 (round-23 (2026-10-24)).**

**Session segment:** 09:42–10:41 (continues `2026-10-17_09-34_pareto-plan-push-saga-inherited-systemadapter-red.md`). Scope: executing the SUPERB plan (`docs/planning/2026-10-17_09-03_SUPERB-readme-followthrough-docs-truth-pareto-plan.md`, 22 medium / 69 fine tasks) + the 09-34 report's §f items. Dispatch: "GET SHIT DONE! The WHOLE TODO LIST!" — this report is ordered at the halfway checkpoint.

**TL;DR:** 11 of 22 medium tasks fully landed (M01, M02, M03, M04, M05, M06, M07, M08, M09, M17, M19 + M20's row + M22's tail), all verified green at their own gates. The inherited systemadapter red is now PROVEN externally blocked (hermetic-FAIL vs workspace-PASS contrast; the upstream fix is pushed but untagged, and the owning session is active in that repo). One branching-flow drift (+4 foreign findings) was adjudicated and re-pinned. Two honest formatter scope failures happened and were reverted cleanly — one via a near-miss `git restore .` that could have destroyed foreign work (verified after: it did not). **12 commits sit unpushed**; the push + CI watch + the remaining packets are the second half.

---

## a) FULLY DONE

| # | Item | Evidence |
| --- | --- | --- |
| 1 | **systemadapter red investigated to its true boundary (f-1/g-1).** Root cause CONFIRMED unchanged at current pins: hermetic `GOWORK=off` FAIL (`system: not found`, 5.01s) vs workspace PASS (0.009s) on `TestHarnessPilot_UserLifecycle`. The upstream fix (`b30156686`, projectionadapter `String()`→`Get()`) is NOW PUSHED on go-cqrs-lite origin/master — only the `projectionadapter/v4` v4.5.4+ TAG CUT remains, and a foreign session is active in that repo (dirty tree, ahead 1, architecture-reference work) → foreign-train doctrine: the tag is theirs. The 5-module sweep recipe (systemadapter, usermgmt, integration_test, dashboardui, examples/system-demo) is recorded in the TODO row | TODO_LIST systemadapter row (RE-VERIFIED 2026-10-17 marker); this session's contrast runs |
| 2 | **M01 HARVEST** — 08-55 §f (45) + 09-34 §f (17) routed: TODO rows for the quarterly README↔FEATURES claim-sweep ritual (M20), the usermgmt `EventHandler` docs gap (grep-verified: code in `usermgmt/events.go`, zero doc mentions anywhere), identity-model README review, README TOC decision; ROADMAP OQ29–32 (CI test matrix, reference-catalogue home, demo asset, old-anchor stubs — each with a recommendation) + a raw-ideas section (7 parked ideas); systemadapter row updated; `Updated:` header refreshed | commit `40b06c81` |
| 3 | **M02 README quick-start compile gate — full atomic bundle.** `scripts/checks/check-readme-quickstart.sh` (extract the `## Quick start` go fence → throwaway consumer module via path-replace → `GOWORK=off go mod tidy` + build; zero-fence = loud fail; deterministic toolchain-floor raise from the target module, never downgrading; mktemp + trap + timeout; rc outside pipes) + 4-case offline fixture self-test (clean/broken-go/missing-fence/poisoned) + flake app `.#check-readme-quickstart` + BOTH check-modules branches + 2 CI steps in the `checks` job + AGENTS Quick-Reference row. Verified: self-test 4/4 (incl. `CI=true` env-leak variant), real gate GREEN on the current README (106-line fence compiles hermetically), nix app green | commits `dd8fa114`+`cb622e44`+`0106064c` (daemon-chunked; message amended on HEAD) |
| 4 | **M04 residual-claims sweep** — lockout (`usermgmt/lockout.go` + `ServiceConfig.Lockout`), ACK protocol (`ack.go`: X-Command-Id-gated `BroadcastOnAck`, `AckStatus`), `JournalSSEStore` mapper contract (`transport/journalsse.go` doc comment) all verified TRUE; "21 events / 20 commands" KEPT (mechanically pinned by `check-domain-counts.sh` — CI-fresh, cannot drift silently); "all 27 middleware factories" DE-NUMBERED to "every" in README + AGENTS + the leveraging guide (the module exports 67 package-level funcs; no definition of "factory" yields 27) | commits `260a2dde`+`0ea488e5` |
| 5 | **M07 link hygiene** — CHANGELOG.md linked from the README Reference group; ADR 0046 is now a link (matches 0055/0056 posture); both internal anchors machine-verified with a GFM slugger (`#ready-made-uis--the-one-call-setup` ×2 correct) | same commits |
| 6 | **M22 SECURITY.md** — already existed (supported versions + posture); the missing "Reporting a Vulnerability" section added (private GitHub security advisories, no public issues) | `0ea488e5` |
| 7 | **M05+M06 READMEs** — `usermgmt/{totp,webauthn,oauth2}/README.md` (what it provides + constructor snippet + inject-as field + config pointer; every snippet symbol/field grepped from provider source — the webauthn `New() (*Provider, error)` + `RPOrigins []string` and oauth2 `New(ctx, …)` shapes were CORRECTED from first-draft guesses); `examples/{admin-demo,dashboard-demo,middleware-showcase,system-demo}/README.md` (ports extracted from each main.go; system-demo honestly a console demo); `examples/README.md` 13-demo index with ports column; root README's four bare-dir links now real | commit `bd012700` |
| 8 | **M08+M09+M19** — v4→v5 guide gained the full `cqrs-htmx-upgrade` codemod section (R1/R2/R3/R9/R10 rule table extracted from the cmd's doc comment, module-graph-derived symbol tables note, dry-run/`-apply`/`-v` usage); README migrations paragraph cross-links it; CONTRIBUTING's Release Process opens with the release-playbook + train-preflight pointer; the README↔setup/README quick-start duplication collapsed to teaser + link (one copy remains) | commit `c5580bf4` |
| 9 | **branching-flow +4 foreign findings adjudicated + baseline re-pinned** (the full `check-modules` battery's only red): `loginpage/handler.go:24` large-struct `PageData` = deliberate exported view-model (reject class extended); `rewrite.go:393/482` INDEX_OUT_OF_RANGE = range-bound + early-return-guarded indices; `rewrite.go:131` NIL_POINTER_DEREF = `*[]string` out-param contract — all safe-by-construction, all from concurrent-session Go edits since 10-15. Baseline 165→165 (minified), amendment logged, gate re-run GREEN | commit `d9cdd55d` |
| 10 | **M03 markdown formatter wired** — nixpkgs prettier in treefmt: `.prettierrc` (`proseWrap: preserve`, printWidth 120 — table pipes only, prose never re-wrapped); `lib.mkForce` scoping to `["*.md" "**/*.md"]` (BOTH globs load-bearing); 11 historical corpora excluded via `settings.global.excludes` (status/planning/research/ADR/feedback/analysis/modularization/agents-notes/CHANGELOG/testdata stay byte-identical); one-time normalize over 25 sub-tree + 6 root md files; post-normalize gates ALL green (docs-links 587 OK, docs-freshness, domain-counts, status-rows, full `nix fmt` rc=0); AGENTS gotcha 10 rewritten (the morning's "NO md formatter" correction superseded) | commits `96cfe1c5`+`c3a2ff3e`+`6f71fd65` |
| 11 | **M17 smoke harness extracted** — `scripts/selftests/lib/smoke.sh` (unique-port picker above 24100, spawn-with-log, `smoke_wait_http` with the log-first dead-server rule, stdlib-only Go prober — curl stays banned — compiled once per run, idempotent EXIT-trap teardown) + `test-smoke-harness.sh` 5/5 green (pick/spawn+readiness/GET+POST/dead-server-log-first/cleanup-idempotent), shellcheck-clean, treefmt-clean | commit `0bbd25b5` |
| 12 | **M11 claims audit (the runnable surface)** — `check-templates`, `check-go-toolchain`, `check-large-files --all`, `check-docs-links`, `check-docs-freshness`, `check-domain-counts`, `check-status-rows` all rc=0 LIVE; the FULL `nix run .#check-modules` battery ran to its final stage (sole red = the branching-flow drift, since fixed); `nix build .#benchstat` green (the nix-checker warning = gotcha-31 misattribution, deliberate non-fix); pkg.go.dev v4.14.1 rendered (f-35 CLOSED); `/htmx.js` route unanimous across the 5 serving examples (f-32 CLOSED) | this report + run logs cited inline |

## b) PARTIALLY DONE

| # | Item | What's done | What remains | Effort |
| --- | --- | --- | --- | --- |
| 1 | Master CI green | 2 of 3 inherited reds were healed by the PREVIOUS segment; the third (test job / systemadapter) is externally blocked (a-1) | The 12-commit local pile is UNPUSHED (pre-push strict gates + CI watch pending); CI on origin still shows the inherited red until then | S |
| 2 | M11 AGENTS claims audit | Every runnable Quick-Reference command verified live or by same-day evidence (a-12); falsifiables fixed ("27 factories", gotcha 10 formatter claim) | The coverage-gate row's evidence dates to 2026-10-01 (15/15) — re-run in the final quiet-window battery rather than now | M |
| 3 | The wave itself | 11+ of 22 medium tasks landed and gate-verified | Second half: M21, M12+M13+M14 packets, M15, M18, M10, report-#2 leftovers (f-4/f-6/f-7/f-11/f-14/f-15), final battery + push + CI watch + render check + inline annotations + TODO wave row | M |

## c) NOT STARTED

| # | Item | Why |
| --- | --- | --- |
| 1 | M21 module-README bloat packet; M12 reference-catalogue packet (+ M13 anchor annex, M14 asset recommendation folded in) | Ordered next; packets only — owner Q1/Q3 gate the BUILDS, not the packets |
| 2 | M15 docs-status bookkeeping (live-tail count 27→46 + advisory budget run) | Queued |
| 3 | M18 GOTH explainer; M10 codespell pass | Queued |
| 4 | Report-#2 leftovers: f-4 upstream release-note reads (go-codec v0.3.2, catalog v4.9.0), f-6 pre-push local checklist → release playbook, f-7 foreign-artifact adjudication (`work/` ignore scope, `.eventcatalog-hub.json` keep-tracked decision), f-11 lessons → crush-config repo, f-14 stray-binary scan, f-15 gotcha-40 addendum | Queued |
| 5 | f-16/f-17 tails: post-heal CI confirmation (externally blocked), bench/coverage quiet-window battery | Blocked on the upstream tag / a quiet window |

## d) TOTALLY FUCKED UP!

| # | Failure | Severity | Root cause | Mitigation |
| --- | --- | --- | --- | --- |
| 1 | **Blind `git restore .` after the prettier over-scope** — the tree held 5 foreign-session files (`.agents/skills/*`, `.config/metadata.yaml`, `.eventcatalog-hub.json`) alongside the formatter churn, and I restored EVERYTHING before diffing. Verified after via commit timestamps that the daemon had committed all foreign state by 09:17 — **no foreign work was lost** — but the check came AFTER the destructive reflex, and my own uncommitted flake edits WERE destroyed (M03 partially redone) | High as a near-miss (zero actual foreign loss) | Treated "all churn is prettier's" as fact; the pre-restore status listing the foreign files was already in my hands | New personal rule: `git stash` (recoverable) NEVER `git restore .` when the tree holds anything not authored this second; diff before any whole-tree destructive op (report #2 d-4's lesson, now with a near-miss to prove it) |
| 2 | **Prettier scope failure, twice** — (i) bare `*.md` include matches ONLY root-level files in treefmt, so the first full pass was unscoped in effect at depth… no: the inverse — `**/*.md` alone misses ROOT files; and (ii) plain `settings.formatter.prettier.includes` MERGES with the module's default extension list → a 98-file js/yaml/json churn wave (reverted). Fix: `lib.mkForce [ "*.md" "**/*.md" ]` | Medium (two revert cycles, ~10 min) | Validated the scoping on ONE file — but `examples/README.md` sits at depth 2, which masked the root-gap; and I didn't check that the ONE file exercised the includes ALONE | Both foot-guns durably documented in AGENTS gotcha 10 (glob semantics + merge + mkForce); formatter onboarding ritual: single-file dry-run at BOTH depths before any full pass |
| 3 | **rc-through-pipe, 3rd documented recurrence** — `bash test-smoke-harness.sh \| tail; echo $?` printed rc=0 beside three FAIL lines, in the SAME session that cited gotcha 3 in a commit message | Low (the FAIL text was unmissable) | Pipe muscle-memory again | The rule needs the file-redirect reflex (`> /tmp/x.log; rc=$?`), not memory — candidate for the crush-config lessons lane (f-11) |
| 4 | **Edit-then-commit race left a broken intermediate commit** — the SC1091 fix edit FAILED (shfmt had just reformatted the file, invalidating my read-state) but the chained bash continued through fmt→test→amend, committing WITHOUT the fix; caught and re-applied on the second attempt | Low (amended within a minute) | Chained commands assume earlier steps succeeded; the edit tool's failure didn't stop the chain | Verify the fix is IN the diff before amending; the chain pattern needs an existence check between edit and commit |
| 5 | **Daemon chunking despite chaining** — M02 (3 chunks), M03 (2 chunks + a stray `.prettierrc` capture), the M04 README (1 chunk) all landed as heuristic commits mid-verify | Low (content identical; HEAD messages amended) | The multi-second nix-run/verify windows inside ONE tool call still beat the daemon's poll | Accepted debt per gotcha 4; messages amended on HEAD, deeper chunks left as-is |

## e) WHAT WE SHOULD IMPROVE!

| # | Pattern | Impact | Concrete fix |
| --- | --- | --- | --- |
| 1 | **Destructive-reflex discipline** (d-1): restore/stash decisions made from assumption instead of diff | Near-miss on foreign work; actual loss of my own edits | Stash-over-restore default; diff-before-restore; the preflight-tree-check tool exists for exactly this and went unused AGAIN (report #2 d-5 repeat) |
| 2 | **Formatter onboarding is a scope trap** (d-2): treefmt-nix's include semantics differ from intuition at both ends | Two churn waves; could have been a pushed mangled tree | The gotcha-10 rewrite carries the semantics; single-file dry-runs at root AND depth before full passes (new ritual row candidate) |
| 3 | **The rc-through-pipe rule keeps losing to muscle memory** (3rd recurrence) | Masked exit codes | Make the file-redirect the ONLY pattern in transcripts; add to the crush-config lessons lane with the recurrence count |
| 4 | **The wave's own guardrails held** — worth recording: the historical-corpus excludes saved every annotation table from the normalize; the quick-start gate's candidate-count guard worked; the uncommitted-baseline guard fired correctly; the branching-flow ratchet caught foreign drift the lint jobs cannot see | Positive reinforcement | None — these are the mechanized wins the plan bet on |

## f) Next tasks (second half of the wave)

| # | Route | Task | Impact | Effort |
| --- | --- | --- | --- | --- |
| 1 | T | M21 bloat-review packet (`docs/proposals/2026-10-17_module-readme-bloat-review.md` — measurements + verdicts, NO edits) | M | S |
| 2 | T | M12 reference-catalogue packet (+ M13 old-anchor annex + M14 asset recommendation folded in) | H | S |
| 3 | T | M15: fix `docs/status/README.md` live-tail count (27 → actual 46+); run `.#check-docs-tail-budget` advisory; route findings to the round-19 row | L | S |
| 4 | T | M18 GOTH explainer `docs/goth-stack.md` + README badge/callout link | L | S |
| 5 | T | M10 `buildflow -s codespell` over the md set + markdownlint availability check (note-only if absent) | L | S |
| 6 | T | f-4: read + classify upstream deltas (go-codec v0.3.2, catalog v4.9.0) | H | S |
| 7 | T | f-6: pre-push local checklist into `docs/guides/release-playbook.md` §1 (fetch + large-file --all + docs gates [+ check-modules when feasible]) | H | S |
| 8 | D | f-7: adjudicate `work/` (ignore entirely vs `work/bin/`) + `.eventcatalog-hub.json` (keep tracked?) — my recommendation: ignore `/work/` wholesale (nothing tracked under it), keep the hub file tracked (tiny, referenced by the catalog work) | M | S |
| 9 | T | f-11: cross-project lessons → crush-config repo (`references/lessons.md`: stash-over-restore, rc-through-pipe recurrence #3, treefmt include semantics) | M | S |
| 10 | T | f-14: `git status --ignored` scan for other daemon-risky build outputs | M | S |
| 11 | T | f-15: AGENTS gotcha-40 addendum (healing one masked red reveals the next — proven twice) | L | S |
| 12 | T | Final: `nix run .#check-modules` + docs gates + treefmt check; PUSH the 12+ commits through the strict pre-push gates; `.#push-and-watch`; M16 render spot-check; annotate reports #1/#2 inline where items closed; TODO_LIST wave row with receipts | C | M |

## g) Questions (3 — blocking only the named lanes)

1. **Push timing:** 12 verified commits sit unpushed (docs + gates + the baseline re-pin). Default plan: ONE wave push after the remaining packets (avoids two CI cycles and the tag-cache dance). Confirm, or push now?
2. **The upstream projectionadapter tag:** leaving the cut to the active go-cqrs-lite session per foreign-train doctrine is my read — confirm, or authorize an exception (I would then verify their tree is quiet first).
3. **Q1/Q2/Q3 (asset / voice / catalogue)** remain unanswered from the plan — the packets will carry my recommendations (code-only; keep current register; pkg.go.dev-canonical) so the builds can start the moment you say GO.

---

*Point-in-time snapshot at the halfway checkpoint, ordered by the user. Annotations when verified: inline strikethrough + dated `> ANNOTATED YYYY-MM-DD` blockquote. The full backlog for the second half: the plan file + §f above.*
