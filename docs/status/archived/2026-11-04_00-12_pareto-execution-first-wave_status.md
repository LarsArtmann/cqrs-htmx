# Status: Pareto execution, first wave — six tasks shipped, push still gated

> **ANNOTATED 2026-11-04** — ~~every open item in this report is resolved~~ **ARCHIVED 2026-11-04 (round-23 (2026-11-04)).**
> The next session in the same arc is [`2026-11-04_04-01_pareto-execution-second-wave_status.md`](2026-11-04_04-01_pareto-execution-second-wave_status.md), which superseded: second wave landed R3/R9/R10 + issue 113; questions refined to push-train-daemon-lintposture.
> Nothing here is a live work item: it is kept as the session's evidence.

**Date:** 2026-11-04 00:12 CET · **Session:** continuation of the go-cqrs-lite
leverage follow-through (plan of record:
[`docs/planning/2026-11-03_20-02_SUPERB-gocqrslite-leverage-maximization-pareto-plan.md`](../../planning/2026-11-03_20-02_SUPERB-gocqrslite-leverage-maximization-pareto-plan.md))

## What was done (all committed, hooks green)

1. **R1 CBOR — ADR 0059 shipped, corrected to FIVE payloads** (`1b9e3444`).
   CredentialAdded/CredentialRemoved/CredentialUsed/TOTPEnabled/BotRegistered
   now stamp `event.WithCodec(codec.CBORCodec{})`. Execution-time research
   caught the audit's fifth payload (CredentialAdded's `[]byte` fields arrive
   via embedded CredentialCore — missed by the line-scan). Safety surfaces all
   pre-verified: decode is encoding-stamp-aware everywhere, SQL store
   persists `payload_encoding`, sync pull reports "cbor"+base64 honestly,
   SSE envelope is metadata-only. Tests: mixed-journal fold+readmodel
   round-trip byte-exact, size win pinned by assertion.
2. **R2 derived event catalog** (`88aeec54`). PayloadFields reflection-derived
   from the payload structs (hand lists had drifted: CredentialAdded
   documented 3 of 12 wire fields). Inventory gate both directions against
   the canonical `all*EventTypes` lists; pilot field-set pins; integration
   recipe pinned via upstream `catalog.AssertEventInventory`. "24 events"
   comment was drift — true count 25.
3. **R4 durable SQLite DLQ default** (`dd31a716`). `durableDeadLetterStore`
   probes the event store's dialect: SQLite → `projection_dead_letters` on
   the same DB (restart-survival pinned by test); postgres/mysql keep memory
   (no dialect-correct SQL DLQ upstream at the pin — creating the
   SQLite-dialect table there would fail startup). The plan's "three SQL
   setup paths" were one factory all along.
4. **R7 command lifecycle opt-in** (`92f75e4b`). `ServiceConfig.CommandLifecycle`
   wires the commandlifecycle outer+attempt pair adjacent inside the audit
   chain; received/completed/rejected/failed/dead-lettered pinned by three
   dispatcher-level tests. Consumer retry via CommandMiddleware lands inside
   the pair as upstream documents.
5. **R8 ProjectionHostOptions** (`21ba82c6`). Exposed through
   ServiceConfig AND setup.Config, appended last-wins after curated
   defaults; nil-default equivalence pinned; async-projections guide
   documents the batching tradeoff.
6. **R5 guide sections** (`34aa58dd`). §2.7 OTLP export (verified against
   tagged source: OTLP/HTTP-only, 4318 not gRPC 4317) and §2.8 flight
   recorder (real options; no invented query variant). Corpus baseline
   39682→39770.
7. **Import completion for the concurrent session's R3**: their serve.go
   dedup.Ring edit was missing only the import + root go.mod require; I added
   both (uncommitted, in tree) so the workspace compiles and my commits
   could pass test-compile. Their code, their commit to make.

## What is NOT done — the honest list

- **The push.** 14 commits unpushed. Blocked all session: first by their
  in-flight route-posture files, then by their mid-edit transport, now by an
  uncommitted family sweep (go.mod/go.sum edits across ~25 modules, unstaged
  — almost certainly the concurrent session's train ritual running right
  now; their last commit says "train-armed").
- **R3** (dedup.Ring seam): theirs, in flight, uncommitted. My part (import
  + require) done; their seam tests (B24) and C035 retirement not visible.
- **R9 kv.Cache, R10 v5 tail, R12 full-gate re-verify**: not started. R9/R10
  were next in my queue; R12 requires a settled tree to mean anything.
- **Hygiene lane** (L1/L2/L3 lint batches, T1 testutil, W1 waivers, U1
  systemcatalog tag ask, F1 FailureLog, X1–X3): not started.
- **`nix run .#check-modules` never ran once this session.** Per-task I ran
  module tests (hermetic usermgmt green 5×) and the pre-commit BuildFlow
  (green 6×), but the full aggregate gate — the plan's spine — was never
  affordable under the concurrent edits (adminui/transport red from their
  files at various points). The audit report's "gate green at HEAD" caveat
  remains unproven until R12.
- **One lost commit:** my "mark CBOR finding adopted" follow-up (report
  annotation + first corpus bump) lost the HEAD race to the auto-commit
  daemon during its 10-minute hook and was never retried as a commit. The
  corpus part later landed inside 34aa58dd; the 8-line report annotation is
  STILL STAGED after ~1.5 h. Small, but it means the committed report does
  not yet carry its own adoption note.

## What I would do differently (the brutal part)

1. **I wrote guide snippets with unverified API names first.** First R5
   draft claimed `WithHTTPOption`, port 4317, `New(Config{RingSize})`, and a
   `QueryFlightRecorder` — none exist at the pins. Caught in self-review
   BEFORE commit, but the correct order is verify-then-write, always; the
   session summary even flags this exact class ("verify external claims").
2. **`rg -rn` struck again — three times.** The hook blocked me three times
   in one session for the same known trap (it's in memory!). Muscle memory
   for `rg -n` needs to actually form.
3. **Commit-hook races cost ~40 minutes.** The daemon committed my staged
   work mid-hook (HEAD moved under my commit), and one hook run died on the
   concurrent session's broken transport after I'd already waited 10 min.
   Correct move both times was: check `git log` BEFORE the hook finishes
   when the daemon is active, and never let a follow-up commit go
   un-retried (see the still-staged annotation).
4. **Sloppy test-file iterations.** command_lifecycle_test.go took three
   edit rounds (wrong type name, dead helper, unreachable branch) and the
   DLQ restart test used a synthetic entry that List legitimately skipped
   before I used a real event. Each was cheap; the pattern (write fast,
   fix in test output) is fine for tests but I should have read the
   List-skip contract the first time — the `skipped` counter was documented
   in the very file I was reading.
5. **Pathspec-commit mechanics:** twice I forgot untracked files need
   `git add` before a pathspec commit can include them. Known git behavior,
   zero excuses at this point.

## Open questions for the human (blocking or judgment calls)

1. **The sweep in the tree**: ~25 go.mod/go.sum edits sit unstaged (codec +
   id/v4 family bumps — the train alignment). If that is your running train
   ritual, I will keep hands off and only push after it lands; if it is
   abandoned mid-run, say so and I will complete it via the sanctioned
   `bump-dep --commit --lint` sequence.
2. **R3 ownership**: the concurrent session's serve.go edit has been
   silent-but-uncommitted for ~1.5 h (only their docs commit since). Finish
   it myself (seam test B24 + C035 check + commit), or is that lane still
   live?
3. **Push timing**: once the tree is clean, straight `git push` of the 14
   commits, or hold for the family train to cut (their "train-armed"
   commit suggests tags are imminent — pushing mid-train risks CI
   evaluating a moving target)?

## Bottom line

Six of the plan's Table-A tasks are shipped, tested, and hook-green — the
1%→51% CBOR change, the catalog drift gate, the durable DLQ, lifecycle
events, the host-options seam, and the two guide sections. The score that
matters moved where the audit said it would. What's left is blocked on a
tree I share with a live train ritual, plus three work items (R9, R10, R12)
and the hygiene lane that are mine to do the moment the tree settles.
