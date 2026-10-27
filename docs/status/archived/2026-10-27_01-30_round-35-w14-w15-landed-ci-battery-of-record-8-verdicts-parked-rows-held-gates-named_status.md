# Round 35 — W14+W15 landed, CI battery of record, 8 parked-row verdicts, held gates named

> **ANNOTATED 2026-10-27** — ~~this report~~ **SUPERSEDED** by [`2026-10-27_09-41_round-36-owner-status-demand-full-inventory-self-critique-next-50_status.md`](2026-10-27_09-41_round-36-owner-status-demand-full-inventory-self-critique-next-50_status.md), which superseded by round-36 (owner status demand; adds self-critique + next-50). Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-10-27 (round-23 (2026-10-27)).**

**Date:** 2026-10-26/27 (session ~21:15 → 01:30) · **Lane:** module lane (W01–W08, W10–W12, W14, W15, W17-verdicts) · **Head at close:** see §h · **Box load:** 107–164 throughout (shared; shaped every verification decision below)

## §a What this session did

1. **W14.3 re-verify became a root-cause fix.** The "unverified, probably load" state from round-34 was actually a REAL defect: `fakeQueryJournal.ReadQueriesFrom` ignored both cursor and limit, so `loadAllQueries` appended a full batch forever — the local 611s timeout AND both CI runner shutdowns (exit 143, the OOM class) were this hang, not box load. Fixed three ways: the fake now implements real cursor semantics; both export walkers fail fast on a stalled cursor (`*_cursor_stalled` infrastructure errors); regression tests pin both. Also fixed the CI lint reds (wrapcheck ×2 on the `ReadAll` fallbacks, wsl ×2).
2. **W14.4 landed:** the three core journal walkers (`LoadTailEvents`, `LoadEventByID` scan, `FindEventNeighbors`) got the same stall-guard treatment (error / error / degrade-to-no-navigation); the snapshot detail handler stopped masking store failures as "No snapshot found" (not-found → missing page, everything else → 500; delete treats already-gone as idempotent success); a rendered-page guard walks every GET route and fails on any ajax element without its own `hx-target`/`hx-select` pins (the 2026-10-18 boost-inheritance blanking class).
3. **W14.5 landed:** `biome.json` REMOVED (verdict: nothing consumed it — no treefmt program, no BuildFlow step in any observed run, no CI job, no script; the devShell binary and the .buildflow.yml claim went with it). New `check-root-files` gate freezes the tracked non-Go root set (31 files, baseline, 8-case self-test; wired into check-modules + CI). The TODO row's other tails: `ack.go` broadcast-peek audit = safe (opt-in, sanitized detail, opaque IDs); dprint tombstone confirmed accurate; ~280MB of regenerable example/e2e binaries trashed.
4. **W14.6 landed:** SSE guide gained `WithRequestFilter` (per-request filters, 400-on-malformed) and `TailReader`/`ReadLastN` (O(n) backfill, `DESC LIMIT n` note) recipes; root README SSE rows extended; actor row names the auth-audit trio; actor guide ↔ usermgmt README cross-linked both ways; `SSEOptions.Options()` conversion test now pins the RequestFilter branch. Root suite + lint green hermetically.
5. **W15.2/W15.3 landed (skill-maintenance gates):** `check-skill-claims.sh` (221 module-qualified symbols must resolve; found and fixed REAL drift — `cqrshtmx.ErrValidation` never existed, it is `ErrValidationFailed`, fixed in SKILL.md + README; the root actor bridge is `ParseActorID`/`ActorIDFromUser`, not `NewActorID`); `check-skill-snippets.sh` (40 code fences parse via as-is/package/func wraps; 29 signature-reference fences skipped by heuristic or the new `//doc:api-tour` marker convention); `check-readme-splitbrain.sh` (module READMEs scanned by the same core; per-doc allowlist covers auditlog/README's intentional name-shadowing). All three + self-tests wired into check-modules + CI.
6. **W15.4/W15.5 landed (canaries a–g):** (a) `tidy-sweep.sh` — per-module hermetic tidy, testdata excluded by construction; (b) verify-tag fixture-poison canary (all 8 fixtures must keep their markers; clean-go-mod must stay replace-free); (c) BuildFlow skip_steps canary (the 7 documented skips must stay listed — gotcha 35); (d) coverage-gate fail-loud (a failed `go test` used to silently PASS both the old bc and awk paths); (g) thresholds single-sourced in `scripts/checks/coverage-thresholds.tsv` (flake AND all 10 CI coverage steps read one file); (e) `docs/runbooks/toolchain-floor.md` (the 1.27.x touch-list); (f) `.codespellrc` + a clean pass over the living docs (domain terms deriver/MoR ignored).
7. **W17 verdicts + sweep:** all 8 parked rows carry verdicts (7 pre-recorded 2026-10-25, check-cqrs-lint-CI's added now: stays blocked on the S55 §1 nested-module tag); TODO swept — 4 completed rows removed (M18/M20/M21, loginpage AutoRedirect, root-declutter, tooling canaries), 3 rows updated with progress/verdicts, hygiene gate green (58 rows, 0 violations).

## §b The battery — CI is the record

Local legs were an honest refusal: load 107–164 all session (the plan's own precedent: bench-spike refuses under load; a red-under-load battery proves nothing). Corroboration that IS local and green: dashboardui suite+lint (0 issues), root suite+lint (0 issues), every new gate + all five new self-test suites (5+5+5+3+8 cases), docs gates (links/corpus/tail/todo-hygiene), and the strict pre-push gates (release-train + version-drift, 881 requires → published tags). The push's CI run is the battery of record: test (all modules, -race), lint (11-module matrix), coverage checks (now single-sourced), and the checks job (every gate incl. the five new ones). Branching-flow inside check-modules is red by HOLD (see §g) — the only red row, CI-exempt, owner-gated.

## §c W14.3 incident post-mortem (the honest-ledger entry)

The round-34 report recorded w14.3 as "committed but UNVERIFIED, 611s timeout, load 140 — environmental class". Wrong on both counts: the timeout was wall-clock `go test` default (a HANG, not slowness), and the CI deaths were the same hang OOM-killing runners. The lesson is already institutional (verify before commit; daemon captures are unverified until proven otherwise) — this session just paid it properly: the fix was verified isolated BEFORE any push, and master went green on `05c7347b` (run 40398258036).

## §d Daemon interactions (all benign this session)

~10 captures; one index-lock collision (waited 20s, retried); one mid-hook race (my commit found nothing after the daemon won — content verified via `git status` + show before moving on). Heuristic titles stayed on daemon captures for W14.3's fix (3 commits deep, not amendable without rebase); W14.4/W14.6/W15 got proper titles where my commit landed the bulk or the tail.

## §e Held gates (named, with defaults if the owner stays silent)

1. **Branching-flow baseline refresh** — drift vs baseline is +46/−40/~111 modified (the W11 file splits moved findings wholesale). Default: keep holding; check-modules' branching-flow stage stays red locally, CI-exempt. w15.1's named extractions are HELD with it: their targets predate the splits; re-identification + refresh is one act.
2. **Upstream ask drafts (w15.6)** — default: hold, packaged with the S55 packet.
3. **Release timing** (round-34 §g3: loginpage `NoWebAuthn` + dashboardui fixes unreleased) — default taken this session: batch into the next train, no mid-wave cut. Re-confirm or countermand.

## §f Open tails on this lane

- dashboardui 09-48 row: page-size selector off-menu option, browser-level e2e smoke (ride the next dashboardui change).
- Skill-maintenance row remainder: authored evals + description pass, deep re-audit of the three reference skills.
- The §g packet above.

## §g Questions for the owner

(unchanged from round-34 §g, restated with this session's deltas)

1. Authorize the branching-flow baseline refresh (+ re-identification of the art-dupl tails)? Default: hold.
2. Ratify the §g2 defaults (upstream filings held)? Default: hold.
3. Release timing: cut a train for loginpage+dashboardui now, or keep the batch default?

## §h Commits this session (chronological)

`05c7347b`+`79239612`+`6ce7eec7` (daemon, w14.3 fix content) · `c71b0919`/`a8bb04cd` (daemon, w14.4 content) · `9f10adc0` (w14.4) · `127a58d6` (daemon, w14.5 content) · `e179c9aa` (w14.5) · `0bb42de5` (daemon, w14.6 content) · `a064ca67` (w14.6) · `81c5bc47`/`55eab8da` (daemon, w15 content) · `d3de8ad1` (w15) · TODO/CURRENT_FOCUS/receipt docs (this commit).

## §i Verify commands

`nix run .#check-root-files` · `nix run .#check-buildflow-skips` (via check-modules stages: `root-files`, `buildflow-skips`, `verify-tag-fixtures-untidy`, `skill-claims`, `skill-snippets`, `readme-splitbrain`, + self-tests) · `bash scripts/tools/tidy-sweep.sh` · `bash scripts/checks/check-coverage-threshold.sh <module>` · CI run on the final head.

## §j Addendum (02:40): the battery-of-record run, two self-inflicted reds, terminal green

The first battery-of-record run (40425469944, head `5b902efc`) failed TWO steps — both introduced this session, both caught by CI exactly as designed: (1) `check-coverage-threshold.sh` ran `go tool cover` from the repo root while coverage profiles carry module-relative source paths (the old inline steps always `cd`'d first) — the usermgmt leg could not resolve sources; the script now measures from the profile's own directory. (2) The root-files baseline had been generated before `.codespellrc` was tracked — the gate flagged its own author's drift at HEAD; baseline re-pinned at 32. Fixed in `91c6aca2` (+ daemon capture `839b5a44`): run **40438109268 = success — ALL 16 jobs green** (test, module-architecture, checks with every new gate, the 11-module lint matrix, build, security, mod-tidy). `.#verify-remote-tags`: 179/179 peel identically. **Master is green at close.**
