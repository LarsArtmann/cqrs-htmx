# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-17 (07:2x, settled-tree gate heal + K3 sent + CI triage) · **Owner:** Lars · **Module count:** 30 (`go.work`)

## The active thread

**The handoff's 10-step list is executed; the tree is verified, pushed, and CI-green; the session waits on the owner's three questions (the 07-20 report §g).**

Landed 2026-11-16→17 (receipts in [docs/status/2026-11-17_07-20_settled-tree-gate-heal-k3-pr101-ci-triage_status.md](docs/status/2026-11-17_07-20_settled-tree-gate-heal-k3-pr101-ci-triage_status.md)):

- **The T04 production bug caught and fixed**: the tenant-deletion cascade queried the display-form stream ID — for real tenants it would silently strand every member; fixtures had masked it by mirroring the bug's shape. Fixed at all three layers (deriver, systemadapter fixture, gfl fixture).
- **K3 sent**: templ-components PR [#101](https://github.com/LarsArtmann/templ-components/pull/101) (clean branch, browser e2e verified pre-send, CSS Freshness rebuilt same-change `1aef12bc`); its `TestCrossbrowserLockstep` red is pre-existing on master (spike 1.65.0 vs pin 1.64.1).
- **All 5 settled-tree composite reds healed** (dep budget raise, streamid gate + the bug behind it, branching-flow re-pin 212, usermgmt lint nolint) + the 4 CI reds triaged and fixed (errorfamily, wrapcheck, corpus, e2e — the last not reproducible in the exact runner layout and green on re-run).
- erraudit TOTAL=352 (window restarted, closes 2026-12-16); chromium battery 127/127 ×2 + ran-count gate; archive round tail 7 → 3 (now 4 with this report); FEATURES rows; gate triage improvements; harness runbook; §f.12/20/22/23/29/30/35/36/47 landed.
- Pushed through `84b406f4` + the report commit; master CI GREEN (44010050578).

## Waiting on the owner

- **§g.1 K2** — idempotency default-on for token dispatch (no recorded recommendation found; design call).
- **§g.2 K4 scope** — do the pre-staged decision packets tick under the D18 blanket-directive precedent (M2M/migration/adminui waves), or walk per-row first?
- **§g.3** — the e2e CI flake (failed then green on the same sha): was the re-run yours; annotate known-flaky or keep fail-loud?

Known standing reds: templ-components master's `TestCrossbrowserLockstep` (pre-existing, surfaced by PR #101), PR #101's CSS-freshness re-run verdict (fix pushed, unobserved), docs-gates audit-all (drain-or-re-pin, TODO P2), B18 trigger-gated.
