# Module-extraction decision memo (one repo vs two repos by audience)

**Written:** 2026-10-24 · **Answers:** master plan §7 Q1 + the 2026-10-23 corpus-audit's module-extraction question · **Status:** recorded verdict, owner may override

## The question

Should `cqrs-htmx` split into two repositories by audience — a "library" repo
(consumers) and a "product" repo (dashboards, examples, e2e) — or stay one
29-module workspace?

## The evidence

1. **The direction-neutral re-assessment (2026-10-18)** —
   [`2026-10-18_PROPOSAL.html`](2026-10-18_PROPOSAL.html) — re-measured every
   boundary: **"boundaries are correct — zero merges, zero splits, zero
   reorganizes."** The 29-module DAG is the assessed optimum, not accretion.
2. **Coupling, measured the right way (2026-10-24, master plan S24):** of the
   300 most recent commits touching `dashboardui/` (spanning 2026-09-19 →
   2026-10-25), **0% also touched root-package files** in the same commit. The
   UI modules already evolve independently of the core — a split would buy no
   decoupling that the module boundaries don't already enforce.
3. **The cost side is structural:** 29 modules publish as a family train with
   per-module tags, cross-module replaces, and a strict release-train gate
   (AGENTS gotchas 5/6/30/38). Two repos multiply exactly that machinery; the
   go-cqrs-lite sibling already demonstrates the coordination tax.
4. **The audience split is already achieved in-tree:** consumers import
   `usermgmt`/`root` without ever seeing adminui/dashboardui (nothing depends on
   the UI modules); the docs corpus eviction (2026-10-24) removed the
   attention-cost argument (docs/ went 809 → 203 files, −118k lines).

## Verdict

**KEEP ONE REPO.** The 2026-10-18 KEEP-ALL assessment stands, now with coupling
evidence. Revisit triggers (either is sufficient): (a) a consumer asks for a
smaller import surface than modules provide (module pruning would then precede
any repo split), or (b) UI-module release cadence diverges from the core's by
more than one train.
