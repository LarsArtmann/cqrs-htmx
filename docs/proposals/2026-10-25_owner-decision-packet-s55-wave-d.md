# Owner decision packet — Wave D close-out (S55 + S56, 2026-10-25)

One read for every owner-gated decision in the Wave D arc: the TODO D-table (D1–D17), the 10-42 §6 defaults now past their execution, the 10-43 addendum §3 surfacing, and the two open questions from the 14-01 report. Verdict-first throughout; every row names what its verdict unblocks. Fresh measurements dated 2026-10-25 ~14:30 CET.

## 0. One-pass verdict table

`Tick = agree with the recommendation.` Rows already resolved by execution are marked and need no action.

| # | Decision | Recommendation | Unblocks |
| --- | --- | --- | --- |
| D1 | OQ23 setup/core split (PapDashboard) | GO, conditional on their flip trigger | PapDashboard reply send |
| D2 | OQ24 pluggable SSE envelope | NO-GO today; revisit on 2nd frozen-wire consumer | — |
| D3 | rawIDToken LIVE-SECRET confirm | tick-to-close (executed; D10 companion also resolved `d05585e8`) | — |
| D4 | T08 fleet cqrs-lint binary swap | approve — after D6 (pin `b06ac8add5e8`+; binary 4.16.1 is out) | gotcha-13 retirement |
| D5 | /mnt/buildcache reclaim | urgency DOWN (69%, 66G free, 2026-10-25) — watch suffices | — |
| D6 | cqrs-lint distribution tag (nested module) | **tag it** — the single highest-leverage tick below | D4 + strict CI gate |
| D7 | feedback-inbox epoch + SEC move | ratify as recorded (epoch 2026-08-01) | checker build |
| D8 | Toolchain policy (OQ26) | close as decided (fleet floor since moved to 1.27.2 — row predates) | — |
| D9 | datastar-demo / loginpage | KEEP / closed — loginpage ADOPTED templ-components 2026-10-04 | — |
| D10 | pkce_verifier echo | RESOLVED 2026-10-18 (`d05585e8`) | — |
| D11 | Scorecard evidence policy | keep module-root-min (deterministic); folded into §7 watch | — |
| D12 | Gotcha-25 grep-guard | RESOLVED 2026-10-07 (blocking stage live) | — |
| D13 | `ParseUserID` prefix policy | KEEP generic (memo done: `docs/drafts/2026-10-07-d13-parseuserid-prefix-policy-memo.md`) | — |
| D14 | Cut `httputil/server_timing/v1.0.2` | cut — mechanical verify-tag release, no code delta | license-scanner truth |
| D15 | BuildFlow pre-commit tuning | approve remaining half — see §5; today's incident is the evidence | green pre-commits under load |
| D16 | GitHub Releases per root tag | DEFER (default stands) | — |
| D17 | gosec CI pin `@master` | pin to a tagged release at next CI touch | supply-chain hygiene |
| R1–R3 | 10-42 §6 defaults (parallel-continue · fixed-expiry lockout · S36/S39 in scope) | ratify — all three executed and shipped | books close clean |
| Q2 | branching-flow +12 re-pin permission | whoever hits the shared checkpoint first MAY re-pin, dated ledger row, attribution to W01 surface | green local tree |
| Q3 | treadmill CI posture | keep PR-time advisory + add nightly strict-lag 0 workflow (early warning, zero PR tax) | next CI touch |
| U1 | Upstream ask drafts batch (P15, event.New, cqrs-lint forms, go-output, BuildFlow ×2, journal reverse-read) | approve drafting under `docs/proposals/upstream/` + filing | W15.6 (sibling lane) |
| U2 | ReadLastN upstream ask (go-cqrs-lite) | approve filing/track; SQL impl belongs upstream | O(n)→O(maxReplay) tails |

## 1. The D6 chain is the highest-leverage tick (cqrs-lint)

D6 (tag the nested `cmd/cqrs-lint` module in go-cqrs-lite) unblocks D4 (fleet binary swap — all local verification done; expect B024 re-appearances at `setup/setup.go:139` / `usermgmt/es_setup.go:221`, documented FPs) AND the strict `check-cqrs-lint` CI gate (script is CI-ready, blocked only on a proxy-resolvable tag). Binary 4.16.1 (2026-10-12) carries the C040 source fix plus `doctor`/`scorecard`; the fleet pin still builds the pre-fix binary. One tag, three rows move.

## 2. PapDashboard (D1 + D2 + ready reply)

Recommendations unchanged from the round-14 packet: OQ23 **GO conditional** on their SaaS/multi-user flip trigger (acceptance criteria recorded verbatim; ~1–2 days when triggered); OQ24 **NO-GO** (no consumer blocked; the golden-pinned envelope is the feature). The reply draft is verified against the proxy and ready to send ([round-14 packet §9](2026-10-01_owner-decision-packet-round14-tail.md)); sending remains yours.

## 3. Upstream watch states (measured 2026-10-25)

- **treefmt-nix#545** — still OPEN (templ wrapper hardcodes `pkgs.go`).
- **a-h/templ** — latest release still `v0.3.1070` (2026-10-04, BEFORE the 10-05 fix); the workaround retirement still waits for the first post-10-05 release.
- **BuildFlow#29** (go-work-sync union-graph proof) — CLOSED completed.
- **BF1** (fixture-tree pruning in `surface.Discover`) — still the one live BuildFlow ask; BF2/BF3 shipped.
- **catalog cross-repo consumers — CLOSED**: go-appkit `docs` at catalog **v4.8.0**, `integration` at **v4.9.0** (both were "still v4.7.1" when the watch opened). The TODO row can close.
- **Upstream treadmill**: 20 repos · 77 prefixes · worst lag **0** (fresh run, cache age 689s). The workspace is fully upstream-aligned.

## 4. datastar-demo · E005 · GCL · ReadLastN · lessons

- **datastar-demo**: KEEP as-is (third confirmation; the only runnable datastar-adapter proof).
- **E005** (cqrs-lint cross-module-registration proposal): triage done, evidence at [`2026-10-01_cqrs-lint-e005-cross-module-registration.md`](2026-10-01_cqrs-lint-e005-cross-module-registration.md) — approve filing.
- **GCL follow-through**: remaining = cqrs-lint/coverage legs on the M04–M10 surface (golangci verdict captured: 11 findings pre-train), the `fail()`/`Close()` teardown twin, and the cross-repo docs-debt approval (2026-09-30 session, its g1). All owner-gated on that repo's own "done" bar.
- **ReadLastN**: `transport.TailReader` is structural; the SQL implementation belongs upstream (go-cqrs-lite storage v4.11.1 lacks it). Approve filing/tracking; this repo inherits via the family train.
- **crush-config lessons** (F12 InvokeAs ambiguity, F11 method-less mux mounts): edits belong in the crush-config repo, not here.

## 5. D15 — BuildFlow pre-commit tuning (incident-fresh evidence)

The T01 budgets closed the deadline class (half-resolved). Today (2026-10-25 14:22, load avg 211) the pre-commit battery failed a **docs-only** commit on 16 steps — govulncheck killed at spawn, golangci-lint loop-detected in untouched modules — the exact env-class tax the row names. Remaining ask, now with a second live incident: (a) gomod-freshness exclusion for the deliberate `scripts/testdata/` fixtures (the preflight also re-flags the 13 known `use`-path "mismatches" every run), (b) load- or mode-conditional skip for network/CPU-heavy steps (govulncheck, license-check) in pre-commit mode. (Correction, verified 14:39: the 10-24 `nix-fmt` tmpdir-roundtrip loop NO LONGER warrants a filing — BuildFlow `58115f7` (2026-10-25 06:11) added root-marker copies for exactly this class, the installed binary carries it, and a fresh `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s nix-fmt` run passes; the W16 row closed as fixed-upstream.)

## 6. New questions (from the 14-01 report §g)

1. **Healing push scope — RESOLVED BY ACTION**: the healing push went out (14:26, `1921e57c..2e64862b`, strict pre-push gates green; the 5 commits reconciled `git log --stat` beforehand — W09 fixes + formatting rides + status docs, zero sibling code fragments). CI watch in flight; the loginpage G124 leg is the sibling's W02 half.
2. **Branching-flow +12** (all in the sibling's W01 surface, set-diff-proven): recommend "first checkpoint re-pin allowed, WITH a dated ledger row attributing the findings" so a shared green tree doesn't wait on lane order.
3. **Treadmill posture**: recommend advisory at PR time (upstreams moving is normal) + a nightly strict-lag 0 workflow (the release-train pattern) — worst-lag-0 is measured-green today, so the nightly is pure early warning at zero PR cost.

## 7. S56 watch table (2026-10-25 readings)

| Watch | Today | Trend | Next check |
| --- | --- | --- | --- |
| `/mnt/buildcache` | 69% used, 66G free | stable-vs-10-11 (59%→69%, creeping) | `df -h` before gate batteries; reclaim still owner's call (D5) |
| Fleet upstream asks | #545 OPEN · templ pre-fix-release · BF1 live · #29 closed | unchanged | monthly, or on any upstream release |
| Scorecard timing variance | unmeasured (low; D11 keeps root-min) | unchanged | next scorecard run |
| Catalog cross-repo consumers | both ≥ v4.8.0 — **watch CLOSED** | resolved | none |
| cqrs-lint binary freshness | fleet pin pre-C040-fix; 4.16.1 out | drifting until D6 | D6 verdict |
| Upstream treadmill | worst lag 0 / 77 prefixes | green | nightly if Q3 approved |

---

*Sources: [TODO_LIST.md](../../TODO_LIST.md) D-table + routed rows · [round-14 packet](2026-10-01_owner-decision-packet-round14-tail.md) · [T23/T18 packet](2026-10-01_t23-t18-owner-decision-packet.md) · [10-42 plan](../planning/2026-10-25_10-42_SUPERB-wave-d-completion-code-fidelity-to-closure-pareto-plan.md) §6/§7 · [10-43 addendum](../planning/2026-10-25_10-43_SUPERB-lane-addendum-two-session-split-of-the-10-42-plan.md) §3 · [14-01 status](../status/archived/2026-10-25_14-01_lane-addendum-w09-built-stale-first-cut-red_status.md) §g.*
