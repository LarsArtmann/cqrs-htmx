# Round 41 — continuation: sweep-collision hardening executed and verified; the gotcha-31 near-miss caught; three commit races lost; corpus/tail trued under an active sibling

> **ANNOTATED 2026-10-28** — ~~this report~~ **SUPERSEDED** by [`2026-10-27_21-57_round-43-issue-pr-merge-sweep-usermgmt-release-in-flight_status.md`](../2026-10-27_21-57_round-43-issue-pr-merge-sweep-usermgmt-release-in-flight_status.md), which Round 43 carried the sweep-collision hardening into shipped tooling (bump-dep receipt, gate self-tests) and round 44 closed the queue sweep with a release tag, so the round-41 self-critique has no live consumer.. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-28 (round-44 (2026-10-28)).**

**Date:** 2026-10-27 16:38 CET · **Reporter:** Crush continuation session (round-38 §f execution, "execute until done") · **Head at continuation start:** `ebfcd5e2` (pushed, CI green) · **Head at close:** `a4144806`+ (local-only, ~10 ahead of origin; NOT pushed — owner cadence) · **Trigger:** "READ, UNDERSTAND, RESEARCH, REFLECT. Break this down into multiple actionable steps… Execute and Verify them one step at a time."

**Scope note:** session-scoped. The sibling session (now round-39/40, 16:17/16:37) ran CONCURRENTLY throughout — it archived round-36 (resolving round-38's §g1 tail breach), harvested BOTH reports into TODO/ROADMAP, implemented issue #32's bounded AuditLog, wrote its own report, and won two of my three commit races. Division of labor was honored by detection, not coordination. **Renumber note:** the sibling answered the same owner status-demand one minute earlier with its own "round-40" file — this report renumbered 40→41 (title + filename) to keep the round series unambiguous.

## a) FULLY DONE (this continuation, receipts)

1. **Research before action:** loaded the docs-health ANNOTATE/ARCHIVE canon; correctly concluded round-36/37 were NOT archivable by me (open owner questions cannot be resolved on another session's behalf) — the tail breach stayed owner-gated until the sibling resolved it.
2. **vendorHash alarm → verified FALSE POSITIVE with the deciding receipt:** `vendorHash.nix`/`hash.nix` are benchstat-derivation-only ("unrelated to this repo's own go.sum files" per the file header); gotcha 31 documents the exact misattribution class; `nix build .#benchstat` rc=0. NO hash edited (the gotcha's rule: never edit a hash off the checker's finding). Round-38's wrong claim ("most likely red thing on master") corrected in place, §c1/§f1 struck as RESOLVED.
3. **buildflow's 9 stale step failures discharged:** `--failed-only` reran them (first invocation executed; second said "no failures recorded in the last run" — the decisive receipt). The rc=1 on a further no-op is buildflow's `pipeline.empty` rejection quirk; recorded as a deliberate non-fix (arguably correct-by-design).
4. **FIX RECIPE hardened:** `check-release-train.sh` now prints `bump-dep … --commit --lint` with prose explaining the daemon-shredding class; BOTH exact-string self-test pins (F1 root + F2 submodule) updated in the same change; self-tests 3/3 and 5/5 green; shellcheck clean.
5. **bump-dep collision forensics:** names a sweep log written <15 min ago BEFORE the dirty-tree refusal (fixture-safe under `BUMP_DEP_ROOT`); first draft hit SC2012 (`ls` in command substitution) — rewritten with `find -mmin -15 -printf`, which also eliminated the age math; self-test all green; shellcheck clean.
6. **Runbook trued:** dependency-train-bump gained the gotcha-31 FP row and the mid-sweep-collision row (both with fixes), the stale 28→29 module count fix, and a `--commit --lint` example.
7. **CHANGELOG receipt** under [Unreleased]/Added for all of the above; the risky prefix-substring edit verified byte-intact (grep receipt `S50+S51).** (a)`).
8. **Harvest verified (not executed — sibling's work):** TODO rows cite round-38 §d1/§e1/§e2; todo-hygiene green (68 rows, 0 dead).
9. **Final gate sweep green:** corpus 33563/33563 (later re-trued by the sibling to 33738 for round-39), tail within budget, todo-hygiene clean. All my content verified in HEAD by grep after the daemon landed it (recipe flag ×1, `recent_sweep_log` ×3, runbook rows ×2, self-test pin ×1).
10. **This report + corpus baseline re-truth for its own lines.**

## b) PARTIALLY DONE

1. **Commit hygiene — the irony:** the change that makes sweeps land as ONE reviewable commit could not land as one reviewable commit. Three attempts: (1) cqrs-lint repair timed out at its 2m cap under load 45+ (a repair-mode run over zero staged Go files — pure load); (2) buildflow PASSED in 56s but `git commit` lost the optimistic HEAD lock to a concurrent committer (`cannot lock ref 'HEAD'`); (3) by retry time the daemon had banked my staged files as chores (`d761e5cf..a4144806`). Content verified; messages lost. The failure class my patch prevents was reproduced while shipping the patch.
2. **The deferred composite:** `check-modules` was NOT run — deliberate (sibling mid-flight with code changes; the composite would have validated its in-flight state, not mine). It is a deferral owed before the NEXT push, not a skip.
3. **Round-38 report corrections used the annotation dialect on a LIVE file** (struck `~~§c1/§f1~~ RESOLVED` markers). Defensible — it pre-resolves items for the future archive pass — but it mixes dialects: the canon reserves markers for archived files; live reports get plain corrections. Harmless (rows gate binds archived only), canon-sloppy.

## c) NOT STARTED

1. **The push** (~10 local commits, gates green locally; owner cadence holds).
2. **The composite check-modules on a quiescent tree** (see §b2 — must run before whoever pushes).
3. **Load-gating as a HABIT:** `wait-tree-quiet` exists in this repo's toolbox, was named in round-38 §e3, and I still retried a heavyweight commit under load 45 without an `uptime` check. The improvement is harvested (sibling's ROADMAP row) but the behavior didn't change in-session.
4. **buildflow upstream filing** for the `--failed-only` empty-selection rejection (documented in-session, filed nowhere).
5. **The dead `/mnt/buildcache/golangci-lint` cache** (mount alive, subdir gone; buildflow env-guard rewrote caches to `~/.cache`): observed, root-caused as the true source of the +100–1400% timing-regression noise, recorded nowhere durable.

## d) TOTALLY FUCKED UP (honest ledger)

1. **The gotcha-31 near-miss was closer than it looked.** Round-38 §f1 said "run `nix-hash-fix --fix` — cheapest possible master-red prevention" and this session's step 1 was queued to do exactly that. Only the read-before-edit discipline (grep flake.nix first) prevented editing a live hash off a checker warning — the EXACT documented anti-pattern, one flag away. And the root cause was round-38's own §b3 hedging: "suspected, unverified — deliberately, per 'report, don't research'". That instruction never applied to a warning generated by THIS session's own tool run; the one-command receipt was always in scope. Warnings are findings, not facts.
2. **Three commit attempts, zero reviewable commits, no protocol.** After the first timeout the correct move was a quiescence check or "stage, verify-by-grep, let the daemon bank it" as a PLAN. Instead: blind retry → HEAD-lock loss → daemon swallow. The end state is what the pragmatic move would have produced, arrived at by luck instead of decision.
3. **Corpus baseline arithmetic without a ledger.** My CHANGELOG edit was +1 line by my arithmetic; the measured actual came back UNCHANGED (33563) because the sibling's tree state had moved under me. I set the baseline to the measured value without itemizing the why — the exact rubber-stamp shape round-37 §d3 warned about, repeated by me in the same repo two hours later.
4. **`--failed-only` verdict cost three invocations** because I filtered output with grep/tail and lost the verdict line twice before capturing the exit code and the error block properly. Read the whole output the first time; slices of a summary are not a verdict.

## e) WHAT WE SHOULD IMPROVE (durable)

1. **Receipt-before-claim for checker warnings (the gotcha-31 rule, generalized):** any doctor/linter WARNING that becomes a report claim or a queued action gets its one-command receipt FIRST. "Don't research" scopes to foreign topics, never to the evidence trail of one's own session.
2. **Hot-tree commit protocol:** after ONE failed heavyweight commit (timeout or HEAD race): check quiescence (`uptime`, `git log -1` recency); if another actor is active, switch to stage + verify-by-grep + let the daemon bank it. Fighting for the message on a hot tree is churn with three observed losses.
3. **One correction dialect for live reports:** plain inline corrections (with a dated correction note) while a report is in the live tail; strikethrough verdicts only at ANNOTATE/archive time. One line in `docs/status/README.md` makes it canon.
4. **Baseline bumps carry a growth ledger** (round-37 §d3, now mine too): the corpus gate could PRINT per-file deltas on failure — turning the rubber stamp into a documented decision mechanically.
5. **Cache-mount health is gate health:** a dead cache subdir silently doubles every lint step and pollutes timing analytics with fake regressions. Worth a doctor-level check (buildflow's env-guard already detects it — surface it as a standing warning row, or fix the mount).
6. **Read full tool summaries before acting on them** (§d4) — verdict lines live at the END of buildflow output, after the noise.

## f) Up to 50 things next (ordered by impact; ★ = owner answer needed)

1. ★ Push decision: ~10 local commits (both sessions' work) sit unpushed with local gates green; per standing rule the owner's cadence decides — then `.#push-and-watch`.
2. Run `check-modules` composite on a quiescent tree before that push (§b2; covers the sibling's AuditLog code too).
3. Restore `/mnt/buildcache/golangci-lint` (or re-baseline buildflow timings to cold-cache reality) — every lint step currently pays for it (§e5).
4. File the buildflow `--failed-only` empty-selection rejection upstream (warn-not-error; fleet repo, §e close-the-loop).
5. Add the live-report correction-dialect line to `docs/status/README.md` (§e3).
6. Encode the hot-tree commit protocol (§e2) — one paragraph in the session-start runbook or a hint in the pre-commit failure path.
7. ★ Tail budget policy (§g1): this report makes the live tail 4 again.
8. Corpus gate per-file delta ledger (§e4) — small `scripts/checks` enhancement + fixture.
9. Watch the FIX RECIPE telemetry (sibling's ROADMAP row): do the next real sweeps use `--commit --lint`?
10. bump-dep log filenames carrying the sweep pattern (round-38 §f11, still open, cosmetic).
11. Sibling lane (their rows, unchanged): usermgmt train for the bounded AuditLog (issue #32 closure), PR #33 badge defects, architecture-gate `SSH_PRIVATE_KEY` decision.
12. If a train happens: `.#train-preflight` mandatory (S52) — the e2e battery remains un-run this session (correctly; no train).

(12 items; the sibling's round-39 §f and the standing TODO_LIST (68 rows) remain the backlog of record.)

## g) Questions I CANNOT answer myself (3)

1. **Tail budget policy:** three sessions wrote reports within 70 minutes today; each new one breaches the absolute budget of 3 and someone archives to compensate (round-36 today, then 15-06). Archive round-37 next when its §g questions resolve, or raise the budget to 4–5 to match the real multi-session cadence? The budget encodes a serial-session assumption that no longer holds, but raising it is policy and the archives have been cheap.
2. **Commit authority on a shared tree:** when the daemon/sibling keep winning races, is "stage + verify-by-grep + let the daemon bank it" an acceptable NORM for docs/scripts changes on hot trees (history readability sacrificed), or do you want reviewable commits defended (quiescence waits, or a lock/announce mechanism)?
3. **The `/mnt/buildcache/golangci-lint` subdir:** machine state outside the repo — is its absence intentional (mount trimmed) or drift to repair? I cannot know the mount's intended shape from inside this repo.
