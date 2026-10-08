# Status: Push Unblock + Signing Regression Root-Cause — 2026-10-04 05:51 CEST

> **ANNOTATED 2026-10-04 (docs-health round 17):** the arc closed within the hour — upstream landed the #52 fix themselves (`de2987576`, tag `signing/v4.3.3`, issue closed), the re-pin landed (`a4eec94b`, battery green), and CI now shows the signing class dead (test job green, run 37178906141). The workspace-mode failure set (§b3) is sharpened: it lives in the LOCAL sibling-tree state, not the published pins, and a concurrent session is actively fixing it (uncommitted brand-prefix strip in identity-model/id.go). The "patch-tag cqrs-htmx" thread (§c/§g3/f2) is MOOT — only integration_test requires signing, so published-tag consumers were never exposed. Still open (routed): the loginpage 7/5 dep-budget decision (the last red on master), the agents-notes long-form narrative, the #52 verification comment + tag-diff regression-test check, and the upstream changelog-accuracy mechanism. Archived this pass.

> Point-in-time snapshot (session window ~04:45–05:55 CEST). Append-only per `docs/status/README.md`.
> Scope: this session's work only — release-train push unblock, the pre-existing `integration_test` red, and its upstream root cause. No unrelated research.

**TL;DR:** The blocked push is unblocked and landed (`606d6186..caeba8e0`, all pre-push gates green, 831 requires / 0 lag). The pre-existing `integration_test` red was root-caused to an upstream bug — `signing/v4.3.2`'s `CloneEvent` silently re-stamps event encodings to CBOR — verified end-to-end at source level and filed as [go-cqrs-lite#52](https://github.com/LarsArtmann/go-cqrs-lite/issues/52). Master CI is now red on three fronts: the 3 signing tests (upstream #52, fix path known), a loginpage dependency-budget overrun, and loginpage lint — the latter two landed via my push but belong to a concurrent session's work.

---

## a) FULLY DONE

| #   | Item                                                                                                                                                                                                                                                                                                                                                  | Evidence                                           |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| a1  | Release-train unblock: broadcast v0.6.2 alignment completed (`integration_test` was the last laggard); sweep PASS 6/6 modules with per-module tidy+verify+build+vet green                                                                                                                                                                             | `d27600c5`, sweep PASS table                       |
| a2  | Release-train gate green: 831 internal requires, 0 unpublished, 0 replace-exempted, 0 train lag (`--refresh-cache --strict-lag 0`)                                                                                                                                                                                                                    | gate output, pre-push hook                         |
| a3  | Push landed through pre-push gates; interrupted `git town sync` finished cleanly                                                                                                                                                                                                                                                                      | `606d6186..caeba8e0`, "sync finished successfully" |
| a4  | Version-drift gate green; all 831 larsartmann requires resolve to published tags                                                                                                                                                                                                                                                                      | pre-push output                                    |
| a5  | dependabot.yml curated posture comment restored (auto-configurator had stripped it); gomod coverage survived; npm `/e2e` addition kept                                                                                                                                                                                                                | `3a68c32f`                                         |
| a6  | Root cause of the pre-existing `integration_test` red established at source level: `signing/v4.3.2` `CloneEvent` drops `WithEncoding` on its `event.NewEvent`→`event.New` migration; `event.New` stamps DefaultCodec CBOR (`event_new.go:63-70`) over usermgmt's JSON payloads (`es_decide.go:48`) → projection silently skips → `read_model_missing` | go-cqrs-lite#52 body                               |
| a7  | Failure scoped precisely: EXACTLY 3 tests (`TestSigningEncryption_{StoreEncryptionAndBusSigning,BusLevelCrypto,Ed25519AsymmetricSigning}`), 6/6 deterministic under published pins; usermgmt's own suite has no signing tests (`ok, no tests to run`)                                                                                                 | `/tmp` run logs, local run                         |
| a8  | Upstream issue filed, voice-checked (0 FAIL / 0 WARN): [go-cqrs-lite#52](https://github.com/LarsArtmann/go-cqrs-lite/issues/52) with one-line fix, regression test, and verify plan                                                                                                                                                                   | issue URL, `docs/drafts/`                          |
| a9  | Enduring lesson recorded: AGENTS.md gotcha 24 (wave tags carrying unlabeled code; isolation pitfalls; worktree-baseline invalidation under go.work replaces)                                                                                                                                                                                          | `caeba8e0`                                         |
| a10 | Workspace build green across all 28 modules with the alignment applied (`nix run .#build`)                                                                                                                                                                                                                                                            | "All modules built successfully"                   |
| a11 | Investigation artifacts cleaned (2 worktrees, all `/tmp` scripts/logs); concurrent session's tree changes untouched                                                                                                                                                                                                                                   | `git status`                                       |
| a12 | Session-external context honored: flake apps run `GOWORK=off` (published pins) — the CI-parity world; go.work's 35 local-path replaces only affect ad-hoc local runs                                                                                                                                                                                  | flake.nix:46, ci.yml env                           |

## b) PARTIALLY DONE

| #  | Item              | State                                                                                                                                                                                                                                                                            |
| -- | ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~ | ~~b1~~                | ~~Master CI green~~                                                                                                                                                                                                                                                                  |
| b2 | Wave verification | Only the composite battery + the signing cluster were tested; the other 40+ wave members were never individually exercised. The battery also short-circuits at first failure, so modules after `integration_test` (loginpage, setup, systemadapter, usermgmt) never ran locally. |
| ~~ | ~~b3~~                | ~~Workspace-mode failure set (~16 tests)~~                                                                                                                                                                                                                                           |
| ~~ | ~~b4~~                | ~~Documentation of the episode~~                                                                                                                                                                                                                                                     |
| ~~ | ~~b5~~                | ~~Upstream fix~~                                                                                                                                                                                                                                                                     |

## c) NOT STARTED

~~- `signing/v4.3.3` re-pin here (blocked on upstream fix+tag)~~ done — a4eec94b; regression verified dead against the published pin
~~- Upstream PR for #52 (draft not started — coordination question, see g2)~~ MOOT — upstream landed the fix themselves and closed #52
~~- CHANGELOG entry for the pushed alignment (broadcast v0.6.2 + wave + dependabot npm)~~ done — added to CHANGELOG [Unreleased] this pass
~~- HARVEST of section (f) into `TODO_LIST.md` / `ROADMAP.md`~~ done — this pass
~~- Triage of the two loginpage CI failures (budget + lint)~~ done-half — lint fixed (CI green); the 7/5 dep-budget breach is the last red on master (owner call)

- Changelog-accuracy mechanism upstream (directive-wave entries enumerating code deltas)

## d) TOTALLY FUCKED UP

1. **The wave-bisect conclusion was WRONG mid-flight and I briefly reported it as fact.** The 43-pair binary split "concluded" `scheduling/v4.5.1` — but scheduling's main module is code-identical to v4.5.0 (only submodules changed), so reverting it cannot change the effective graph; round 7's PASS was bogus and the method (go.mod edit + tidy under MVS coupling) was unsound. I caught the contradiction only by diffing the tags. A session that shipped the report one step earlier would have filed a wrong upstream issue.
2. **The `$?`-capture shell bug — gotcha 3, which I knew and still hit.** `echo "rc=$?"` came after an intermediate assignment, so 5 failing runs printed `rc=0`; only the inconsistent `fails=3` exposed it. Exactly the trap the repo's own AGENTS.md documents.
3. **The first "baseline" experiment was invalid by construction.** I ran a workspace-mode worktree test while `go.work` carries 35 local-path replaces — it tested the live go-cqrs-lite tree, not the commit under test, and every later worktree conclusion built on that flawed design. Checking go.work replaces took one command and should have been step zero.
4. **Early experiment design changed two variables at once** (workspace→off AND single-test→suite), making the first isolation results uninterpretable and forcing a redo.
5. **Isolation round 1 covered 5 of 43 bumps and missed the actual trigger** (`signing`); the full pair list should have been extracted before any isolation run. Round 2 caught signing, but by list-composition luck, not method.

## e) WHAT WE SHOULD IMPROVE

1. **Enumerate before isolating.** Extract the complete bump-pair list FIRST, single-test EVERY member, and treat passing singles as necessary-not-sufficient (MVS floor coupling means singles exercise a different graph than the wave).
2. **Verify bisect states, not just verdicts.** Before trusting any revert-state PASS, assert the reverted module's resolved version actually moved (`go list -m`) — MVS can silently re-select what you reverted.
3. **Shell discipline is still manual.** rc must be captured immediately after the command (`cmd > /tmp/f 2>&1; rc=$?`); the repo needs a named helper or template so sessions stop re-deriving this under pressure.
4. **Log to files, never pipe away.** `nix run .#test | tail -15` destroyed the first failure list; every battery run should tee to a unique `/tmp/<repo>-<purpose>-$$-<ts>.log`.
5. **Baseline discipline:** run the full test battery BEFORE any tree work, not just `go build` — the red predates the session and a baseline would have framed the whole investigation a step earlier. And check `go.work` replaces before designing any baseline/worktree experiment.
6. **Pre-commit shortcut for docs-only commits:** BuildFlow's hook fails on the documented deterministic environment-class steps; I burned ~5 minutes letting it run before using the documented `--no-verify` + justification fallback (gotcha 8). For pure-docs phases, go straight to the fallback.
7. **Tag-label hygiene is now a proven gap:** the 2026-10-03 wave's CHANGELOG sections claimed "go-directive minor-form floor" while carrying code migrations (`signing` CloneEvent; `scheduling` submodules). Consumers cannot bisect by changelog labels. Needs an upstream mechanism (label-vs-tag-diff audit) — ROADMAP.
8. **`event.New` encoding auto-stamp is an API trap** for reconstruction/clone paths: the WithEncoding opt exists precisely for that case but nothing enforces it (upstream #52's deeper lesson; v5 API consolidation candidate: NewEvent/New duality is a split-brain surface).
9. **Battery non-short-circuit reporting:** `.#test` stops at first failing module; a per-module PASS/FAIL report (fail at end) would have shown the full local blast radius in one run.
10. **What I forgot initially:** the concurrent-session context (gotcha 4) — I almost `git add -A`-committed foreign dirty files; only the preflight analysis separated my files from theirs. Keep per-file attribution explicit in every commit during shared-tree sessions.

## f) NEXT (brainstorm — up to 50; extra items are ROADMAP fuel, not commitments)

**Now / this train:**
~~1. Triage CI run `37174506290` fully: confirm the 3 test failures are exactly the signing set (no new ones).~~ done — failures attributed precisely; superseded by later runs (37178906141)
2. Triage `loginpage` dep-budget overrun (7/5): reduce deps or justify increase in `scripts/check-dep-budgets.sh` — concurrent session's call.
~~3. Triage `loginpage` lint (golines format + gocritic offBy1 in handler_test.go) — concurrent session's call.~~ done — fixed by the episode-4 session; CI lint green
~~4. When `signing/v4.3.3` publishes: `bump-dep.sh 'larsartmann/go-cqrs-lite/signing/v4$' v4.3.3` → full battery → commit → push (train ritual).~~ done — re-pin landed at a4eec94b with the full battery green
~~5. Watch the next CI run on master after any of the above.~~ done — run 37178906141 observed (lint green, test green, one red job)
~~6. HARVEST this report's (f) into `TODO_LIST.md` (short-term) + `ROADMAP.md` (tail).~~ done — this pass
~~7. CHANGELOG entry: broadcast v0.6.2 alignment + 43-module wave + dependabot npm/e2e.~~ done — added this pass
8. Write the `docs/agents-notes.md` long-form narrative (dates, hashes, the wrong-turn story).
9. Cross-link this report in `docs/status/README.md` if the index convention requires it.
10. Verify `examples` pins resolve cleanly post-wave (gate says 0 lag — spot-check one example builds standalone).

**Upstream (go-cqrs-lite):**
11. Land #52: `WithEncoding(evt.Encoding())` in `CloneEvent` + regression test asserting `clone.Encoding() == original.Encoding()`.
12. Tag `signing/v4.3.3`; consider documenting (not retracting) v4.3.2 as poisoned-for-non-CBOR-producers (never re-tag same version at different commit — gotcha 5).
13. Changelog-accuracy audit for the 2026-10-03 wave: correct entries that claimed directive-only while carrying code (signing, scheduling submodules).
14. Fleet grep: which consumers pin `signing/v4.3.2` — they all silently drop non-CBOR signed events today.
15. Upstream API note: `event.New` []byte passthrough still stamps codec encoding — decide whether that's intended (doc says encoding "auto-stamped from the codec used") or should require explicit WithEncoding for reconstruction.
16. NewEvent/New duality: v5 consolidation note (split-brain surface).
17. Consumer-impact note in #52: the failure mode is SILENT (events dropped, no error) — severity justification.

**Tooling (this repo):**
18. `bump-dep.sh --isolate` mode: per-member single-test battery after any wave sweep (mechanizes gotcha 24a).
19. Wave-audit gate: after family bumps, diff each new tag's module dir; non-empty diff ⇒ changelog entry must not claim directive-only (upstream-able).
20. `.#test` battery: non-short-circuit per-module report mode (PASS/FAIL table, exit at end).
21. Shell rc-discipline helper in `scripts/lib/` (run_and_capture) + reference from gotcha 3.
22. Check-status convention: add this report to the index; ensure `check-status-rows.py` passes (PARTIAL rows self-consistent).
23. Reconcile Go-floor documentation: AGENTS.md quick-reference says 1.27.1 floor; wave normalized module directives to `go 1.27`; go.work floor + `.golangci run.go` still 1.27.1 — document the intended end-state (policy question, see g3).
24. CI `go-version-file: go.mod` now floats to latest 1.27.x post-normalization — pin-vs-float decision + doc.
25. `go-cache-env.sh`: when `GOWORK=off`, the go.work floor is the wrong resolution source — read the module's own directive instead.
26. Integration_test module is 1 package (sequential): consider `-run` sharding or split packages so 3 failing tests don't serialize-block ~40.
27. `read_model_missing` error UX: include DLQ depth / last-projection-error hint so silent skips surface evidence (upstream + here).
28. Signing tests: assert DLQ empty after happy paths so projection skips fail loudly with data.
29. Consumer-side guard test: signed event round-trip preserves encoding — makes upstream regressions fail with a precise message instead of read_model_missing.
30. `preflight-tree-check` worked as designed on the dirty tree — no change; consider adding its output hint to point at per-file attribution during shared-tree sessions.

**Docs / memory:**
31. dependabot comment stripping is recurring: file the BuildFlow upstream issue (auto-configure should preserve curated comment blocks) or move the posture to AGENTS.md.
32. Verify the dependabot npm `/e2e` entry is real (`e2e/package.json` exists) — auto-configurator added it; a wrong path errors every Dependabot run.
33. Note in AGENTS.md: CI's `Clone go-cqrs-lite sibling` step (ci.yml:208) pins nothing — it clones live master, so the sibling-replace CI leg is a moving target mid-upstream-session.
34. Record the "scheduling v4.5.1 main module code-identical" fact as a changelog correction upstream (comment on #52 or the wave PR).
35. Wave-triage playbook: distill this session's command sequence (30-minute path to root cause) into `docs/runbooks/`.

**Roadmap fuel (unrefined):**
36. Skip-gate policy for upstream-blocked tests (mark vs tolerate red — needs g1 answer first).
37. Consider cqrs-htmx patch tag after the re-pin so fresh consumer installs don't inherit the signing drop.
38. MVS-aware bisect tool: verify resolved versions per state; abort when a revert is a no-op.
39. Add runbook link to the CI failure annotation for `read_model_missing` class errors.
40. Audit whether other family modules have clone/reconstruct paths relying on encoding passthrough (same bug class as #52).
41. `check-dep-budgets.sh`: budget justification could live as inline annotation next to the budget table (currently a manual edit).
42. Battery: add `integration_test` to `.#test-all`-style local pre-push option with a time budget.
43. Consider `t.Skip`-with-issue-link scaffolding convention for upstream-blocked tests (reversible, greppable).
44. Upstream: timer/drain timing assertions — the scheduling submodule changes touched Due-claim limits; confirm no drain-path interaction (currently believed unrelated; cheap test to add).
45. Session hygiene: when an upstream repo has an ACTIVE session (staged files, minutes-fresh daemon commits), prefer issue-filing over tree writes — worked well here; write it into the coordination gotchas.
46. `docs/drafts/`: the .body.md extraction step could be a tiny script (voice-check + gh create in one).
47. Re-verify gotcha 24's CI-step name ("Test integration_test submodule") is what the failure annotation references — done this session; keep in sync if CI job names change.
48. Confirm 831-vs-827 require-count jump (wave added direct `query/v4` in integration_test) is intended and documented in release notes.
49. Explore whether integration_test could consume setup's public composition API only (reduce direct cqrs-lite requires; ties into the dep-budget philosophy).
50. Retrospective item: preflight-tree-check + wait-tree-quiet were NOT run before my explicit commit (only bump-dep enforces cleanliness) — consider wiring the preflight into the phase-commit habit for shared-tree sessions.

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

~~1. **CI-red tolerance policy:** until `signing/v4.3.3` lands, should the 3 signing tests be skipped (reversible `t.Skip` with an #52 link — keeps master green, hides a real regression) or do we tolerate red master as the honest signal?~~ moot — the signing tests PASS now (upstream fix landed + re-pin); no skip policy needed
~~2. **Upstream ownership:** is the currently-active go-cqrs-lite session going to own the #52 fix this train, or should I draft the upstream PR (`WithEncoding` + test) for you to review? I cannot see the other session's intent, and writing to that tree while it's mid-flight risks a collision.~~ answered by reality — the active upstream session owned the fix (de2987576) and closed #52
~~3. **The ride-along commit:** my push carried `2413e551` — a daemon commit of the concurrent session's in-progress loginpage work (now failing CI on dep-budget and lint). Was that state push-ready, or do you want a rule that foreign daemon commits get held/rebased out before I push?~~ routed — ROADMAP OQ18 (push/sync policy under concurrent sessions) carries the owner question

---

**Evidence index:** commits `149ea7fc`, `3a68c32f`, `d27600c5`, `caeba8e0` (push range `606d6186..caeba8e0`); CI run `37174506290` (failed); issue [go-cqrs-lite#52](https://github.com/LarsArtmann/go-cqrs-lite/issues/52); draft `docs/drafts/2026-10-04-signing-cloneevent-encoding.md`.
