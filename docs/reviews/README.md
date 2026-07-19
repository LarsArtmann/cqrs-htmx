# docs/reviews/ — Historical Audit & Review Reports

**These files are point-in-time audits, not living documentation.**

Each file in this directory captures what a reviewer found at a moment in time: code-quality scans, data-model reviews, architecture reviews, full-code-review passes, brutal self-reviews. They are **snapshots** — the findings reflect the state of the codebase on the date in the filename.

## How to read these

- **Treat the date in the filename as authoritative.** A review from 2026-05-07 reflects what was true on 2026-05-07; the codebase has moved on.
- **For issues that are still open**, check `TODO_LIST.md` first — living doc.
- **For what the codebase looks like today**, run the same skill again or read the code directly. Code is the source of truth; reviews are leads.

## Why not "update them all"?

Per the `update-old-docs` skill: reviews are historical artifacts. Adding "this is old" banners to 23 files would add zero information — the date in the filename already tells the reader. Restraint is success.

## When to annotate

If a specific review makes a claim that is actively dangerous if believed today (e.g., "X is broken" when X is fixed, or "Y is secure" when Y has since been found vulnerable), add a specific end-of-file appendix to that one file citing the fixing commit.

## File count

23 files. Most concentrated in 2026-05 and 2026-06 when the codebase was actively stabilizing pre-v4 release.
