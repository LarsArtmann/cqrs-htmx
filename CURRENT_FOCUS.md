# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-21 evening (the family train session: twelve tags cut, the wave published, the close-out §f executed) · **Owner:** Lars · **Module count:** 31 (`go.work`)

## The active thread

**The 2026-11-21 family train landed — root v4.19.0 is live on the proxy — and the fleet-floor question is now THE decision.** The train published the whole wave at tag level (FlashErrorHandler, the CSPBuilder knob, Retryable, NewFromJSON + the event-catalog artifact, the tenant-deletion deriver; twelve tags from identity-model/v4.18.0 through systemadapter/v4.14.0, all verify-tag-guarded, remote-peel-verified, proxy-confirmed). All 85 internal requires swept; both sibling-local replaces the train unblocked are stripped; the pre-tag lockstep reds (5 CI jobs) healed to green in the post-train hermetic battery. The 4 remaining strict-lag entries are the RECEIPTED go-appkit hold (v0.13.0 declares go 1.27.4; nixpkgs-unstable HEAD still ships 1.27.3, probed 2026-11-21 — the hold is mechanically forced until the fleet rev moves; the premature 17:05 sweep was re-healed same evening, receipt on the fleet-floor TODO row addendum 2 + the [v4.19.0] Housekeeping entry). Every push until the floor moves needs the named-justification `--no-verify` protocol. The close-out §f of the pareto plan is executed in the same session: the CSP knob (root + adminui `Config.CSPBuilder`, admin-demo adopted, 20/20 fast-slice green), the nested-retag-reminder gate (P01's rule mechanized), the `.#render-receipt` flake app + runbook, the release-train gate's demonstrated unpublished reds (self-test 7/7), the GH release for oauth2 v4.13.1 (releases had been dark since 09-30 — the release-dance-helper row is the standing gap), the Q3 freeze precedent in FORMAT.md, and the docs batch (FEATURES Worth Considering + JSON Directory API rows, README audit link + fit-test pin, setup footprint pointer, skill editing contract + Path D, libs-index presentation overlay).

## Waiting on the owner

- **§g.1 (20-47 report)** — the NixOS system templ (0.3.1123, unpublished master build) is the hook-poison's last standing cause: repin it to a published release (one-line system-flake change, owner-side), or ask for a runbook patch note first.
- **§g.3 (20-47 report)** — K2 multi-instance follow-up: a reference SQL/Redis `idempotency.Store` guide example, or the seam + docs suffice (§g.2 is answered — the anomaly was settled same-evening).
- **Per-row M2M D1–D4 / adminui D1–D3** — the packets stay parked; each decision surfaces individually through the native decision tool per the owner's 2026-11-19 verdict.
- **Fleet-cache hygiene (from the gotcha-10 poison find)** — a substituter served stale fixed-output content under new revs (`cache.home.lan/*` is the suspect class, unattributed): worth a fleet-side prune/provenance check.
- **§g.2 (03-09 report)** — gate robustness vs fail-loud for shared-worktree transients: should one-retry tolerance become the norm for the local-only gates, or does strict fail-loud + the documented re-run-before-triage rule (now gotcha 68) stand?
- **§g.3 (03-09 report) — migration-plan D1/D2, the W3/W4 gate.** The 11-08 owner decision sheet says "Defaults already ADOPTED — no tick needed" while the plan's §6 says W3/W4 "hinge only on the D1/D2 owner ticks" — those two statements contradict each other; pick one (adopt-the-defaults unblocks the read-tier cutover waves, or keep W3/W4 parked until an explicit tick). Recorded per the no-re-ask directive — not surfaced via the question tool.

Known standing reds: the strict release-train gate's 4 appkit entries (the receipted hold above) — every push rides the named-justification protocol until the fleet floor moves. Quiet-window carryovers, trued 2026-11-21 evening: erraudit TOTAL=354 (DOWN from the 364 window ceiling — the hold is satisfied, receipt in AGENTS); the archive round-48 landed same-day (tail 15); the one-green check-modules composite receipt re-attempts AFTER the floor decision (the strict train step is its only red leg).
