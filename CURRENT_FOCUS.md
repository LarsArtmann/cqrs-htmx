# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-21 (master-CI ratchet heal + harvest session; active thread unchanged) · **Owner:** Lars · **Module count:** 31 (`go.work`)

## The active thread

**None — waiting on the owner.** The 2026-11-19 evening executed all four owner verdicts and settled the open anomaly: K2 GO → the default command idempotency guard is landed and verified (own-the-claim design; receipts in the [K2 packet](docs/proposals/2026-11-18_k2-idempotency-default-on-owner-decision.md) § Executed and the [20-47 report](docs/status/2026-11-19_20-47_owner-verdicts-k2-wave-templ-1118-hook-fix_status.md)); hook fix → templ aligned to the latest PUBLISHED release (v0.3.1118 — v0.3.1123 is the owner's unpublished system build) in go.mods, the CI pin, and templGen; cache attribution → unattributed, the provisioning ritual landed in gotcha 66; M2M/adminui → per-row (verdicts recorded in both packets). The §b3 hash anomaly was then settled — and it mattered: the first "aligned" templGen was still the 1100 source behind cache-satisfied fixed-outputs (stale hashes validated, the day's gen verification void); the true hashes are pinned, the true-1118 regen landed (21 stamp lines, stamp-only), and the mechanism lives in gotcha 10 with the war status in gotcha 67 (e).

## Waiting on the owner

- **§g.1 (20-47 report)** — the NixOS system templ (0.3.1123, unpublished master build) is the hook-poison's last standing cause: repin it to a published release (one-line system-flake change, owner-side), or ask for a runbook patch note first.
- **§g.3 (20-47 report)** — K2 multi-instance follow-up: a reference SQL/Redis `idempotency.Store` guide example, or the seam + docs suffice (§g.2 is answered — the anomaly was settled same-evening).
- **Per-row M2M D1–D4 / adminui D1–D3** — the packets stay parked; each decision surfaces individually through the native decision tool per the owner's 2026-11-19 verdict.
- **Fleet-cache hygiene (from the gotcha-10 poison find)** — a substituter served stale fixed-output content under new revs (`cache.home.lan/*` is the suspect class, unattributed): worth a fleet-side prune/provenance check.
- **§g.2 (03-09 report)** — gate robustness vs fail-loud for shared-worktree transients: should one-retry tolerance become the norm for the local-only gates, or does strict fail-loud + the documented re-run-before-triage rule (now gotcha 68) stand?
- **§g.3 (03-09 report) — migration-plan D1/D2, the W3/W4 gate.** The 11-08 owner decision sheet says "Defaults already ADOPTED — no tick needed" while the plan's §6 says W3/W4 "hinge only on the D1/D2 owner ticks" — those two statements contradict each other; pick one (adopt-the-defaults unblocks the read-tier cutover waves, or keep W3/W4 parked until an explicit tick). Recorded per the no-re-ask directive — not surfaced via the question tool.

Known standing reds: none local. Quiet-window carryovers, trued 2026-11-21: erraudit re-true DONE (TOTAL=364, converged ×3 with the AGENTS receipt — the blocking-promotion window RESTARTED at 364, closes 2026-12-21); the local check-modules composite bill is RESOLVED (isolation leg green ×2 — the old `root FAIL (vet)` was the sibling-lane transient, gotcha 68; cqrs-lint gate green 26/26 after a stale 4.20.1 `~/go/bin` shadow binary was retired — the profile dev build is the only instrument again; branching-flow +11 adjudicated under existing verdicts and re-pinned 212 → 220 with the triage-decisions row; corpus re-trues at every docs commit per the treadmill); the archive round is STILL OWED at tail 20 (the ledger's loudest open item — archive BEFORE any next report).
