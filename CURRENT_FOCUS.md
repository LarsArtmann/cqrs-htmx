# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-19 (evening — verdict wave executed, hash-poison found and fixed) · **Owner:** Lars · **Module count:** 30 (`go.work`)

## The active thread

**None — waiting on the owner.** The 2026-11-19 evening executed all four owner verdicts and settled the open anomaly: K2 GO → the default command idempotency guard is landed and verified (own-the-claim design; receipts in the [K2 packet](docs/proposals/2026-11-18_k2-idempotency-default-on-owner-decision.md) § Executed and the [20-47 report](docs/status/2026-11-19_20-47_owner-verdicts-k2-wave-templ-1118-hook-fix_status.md)); hook fix → templ aligned to the latest PUBLISHED release (v0.3.1118 — v0.3.1123 is the owner's unpublished system build) in go.mods, the CI pin, and templGen; cache attribution → unattributed, the provisioning ritual landed in gotcha 66; M2M/adminui → per-row (verdicts recorded in both packets). The §b3 hash anomaly was then settled — and it mattered: the first "aligned" templGen was still the 1100 source behind cache-satisfied fixed-outputs (stale hashes validated, the day's gen verification void); the true hashes are pinned, the true-1118 regen landed (21 stamp lines, stamp-only), and the mechanism lives in gotcha 10 with the war status in gotcha 67 (e).

## Waiting on the owner

- **§g.1 (20-47 report)** — the NixOS system templ (0.3.1123, unpublished master build) is the hook-poison's last standing cause: repin it to a published release (one-line system-flake change, owner-side), or ask for a runbook patch note first.
- **§g.3 (20-47 report)** — K2 multi-instance follow-up: a reference SQL/Redis `idempotency.Store` guide example, or the seam + docs suffice (§g.2 is answered — the anomaly was settled same-evening).
- **Per-row M2M D1–D4 / adminui D1–D3** — the packets stay parked; each decision surfaces individually through the native decision tool per the owner's 2026-11-19 verdict.
- **Fleet-cache hygiene (from the gotcha-10 poison find)** — a substituter served stale fixed-output content under new revs (`cache.home.lan/*` is the suspect class, unattributed): worth a fleet-side prune/provenance check.

Known standing reds: none local. Quiet-window carryovers: the erraudit advisory TOTAL re-true (blocking window closes 2026-12-16), one local check-modules composite, archive pass at tail 15.
