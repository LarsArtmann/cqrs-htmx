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
2. **Coupling, measured the right way (corrected 2026-10-25, master plan
   S24):** of the 78 `dashboardui/`-touching commits in the last-500-commits
   window, **2.6% also touched root-package files** (2/78); across the full
   history it is 10.1% (67/663), early co-evolution era included. (The
   original "0% of 300" claim was a measurement artifact — a pathspec-filtered
   `git log -- dashboardui/` cannot list root files, so co-change was
   invisible by construction; the correct command and both windows are
   recorded in
   [`../drafts/2026-10-25_docs-truth-quartet-s22-s26.md`](../drafts/2026-10-25_docs-truth-quartet-s22-s26.md).)
   The UI modules already evolve independently of the core — a split would buy no
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
