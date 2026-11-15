# Fleet Library Utilization Series — Report Format

> The conventions every `docs/review/libs/<LIB>.md` report follows, so the next
> refresh session inherits them instead of re-deriving them. The 2026-11-14
> 22-library refresh and the 2026-11-05 original both converged on this shape.

## File naming and scope

- One `<LIB>.md` per library (or per deliberately-split family member, e.g.
  `httputil.md` vs `httputil-endpoints.md` vs `httputil-server-timing.md`).
- `index.html` is the fleet table (uncounted by the corpus ratchet — md-only
  metric); `index.render.png` is the human-eyeballed render receipt.
- Reports are LIVING documents: refresh in place, never fork a
  `<LIB>-2026-11-xx.md` copy. History lives in git and in the "Prior reports"
  section.

## Header block (first three lines)

```markdown
# <LIB> Utilization Report

> Audit date: YYYY-MM-DD · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/<LIB>` (submodules…) · Checkout: `/home/lars/projects/<LIB>`
```

- The audit-date stamp is mandatory and MUST move on every refresh — the
  docs-freshness gate and the index "audited" column read it.
- Link depth from this directory is `../../` (guides: `../../guides/…`,
  ADRs: `../../adr/…`, research: `../../research/…`). Never deeper — the
  2026-11-14 go-datastar first draft shipped five broken `../../../research/`
  links caught only by `check-docs-links`.

## Verdict at a glance (table)

| Pin | Latest (verified DATE) | Prod import sites | Adoption | Score |
| --- | ---------------------- | ----------------- | -------- | ----- |

- "Latest" must come from `git ls-remote --tags` run THAT SESSION (or carry
  the older verified date honestly). Never quote a tag from memory.
- "Score" format: `NN/100 · at latest` (or `· NN behind`, or `· N/A`).

## Body sections, in order

1. **What the library is** — 1-3 sentences, module map for multi-module libs.
2. **How cqrs-htmx uses it today** — bullet list, every claim with a
   `path/to/file.go:NN` receipt (symbol:line, not file-level hand-waving).
3. **Capability inventory** — the numbered table:

   | # | Capability | Status | Evidence / reason |
   | - | ---------- | ------ | ----------------- |

   Status vocabulary is a CLOSED set: 🟢 Fully leveraged · 🟡 Partially used ·
   🔴 Not used · ⚪ N/A. "N/A" needs a one-clause reason in the same row.
4. **Score derivation** — show the arithmetic: applicable rows, full ×1,
   partial ×0.5, the resulting score, and the one-line band rationale.
5. **Issues / open items** (or folded into the inventory when small) — each
   with severity (Important/Minor), the trigger, and the owning plan or task
   ID when one exists.
6. **How to verify** — a fenced bash block of commands that re-derive the
   report's numbers. **Every command must have been run during the audit**
   (the verify-block-must-run rule) and the block carries the expected
   outputs as `#` comments. A future session replays the block literally; a
   replay that diverges means the report has rotted (the 2026-11-15 B10
   replay caught the metaengine tag re-lag within a day of the refresh).
7. **Prior reports** — bullet links to predecessor documents with their
   scores; mark supersessions inline.

## Score bands (informal)

90+ fully leveraged and current · 80s used with named gaps · 70s partially
leveraged · <60 underused or stale · N/A-class rows are excluded from the
numerator AND denominator. A score change without a capability-row flip is a
smell — the derivation must move WITH the inventory.

## Refresh ritual (the checklist the next session follows)

1. Version truth: `git ls-remote --tags` for every lib, diff against the
   workspace `go.mod` pins (the `$`-anchored `bump-dep` recipe for sweeps).
2. Drift detection: `git log --since=<last-audit>` over this repo; re-read
   every cited `path:line` receipt — lines move, re-verify each symbol.
3. Refresh reports in place (audit-date stamp, inventory deltas, score
   re-derivation) and regenerate `index.html` LAST — compile the final table
   from the report files as a checksum against dropped work (the 2026-11-14
   refresh dropped go-etag exactly this way and caught it at the index pass).
4. Replay every "How to verify" block after editing.
5. Wire the receipts: TODO fleet row, research README series row, corpus
   ledger sentence (measure AFTER the last edit), and the index footer link
   to this file.
