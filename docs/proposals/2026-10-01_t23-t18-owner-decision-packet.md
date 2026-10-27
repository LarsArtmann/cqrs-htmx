# Owner decision packet — cqrs-lint CI parity (T23) + feedback-inbox checker (T18g)

**Prepared:** 2026-10-01 · Status: DECISIONS REQUESTED (both items are owner-gated: one tags a module in a foreign repo, one sets a repo convention).

## 1. cqrs-lint Go-installable distribution (T23)

**New fact since the 2026-08-30 draft:** `cmd/cqrs-lint` is a NESTED MODULE inside go-cqrs-lite (it has its own `go.mod`). The draft's "new repo or move the analyzer" options are both dead — the publish shape is now just a TAG:

- **Recommended:** tag the nested submodule (tag name `cmd/cqrs-lint/v0.1.0` or `.../v4.x` to match the family train) → install path `go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint@<that-tag>`. Tag parity with the flake pin = tag from the commit the flake's goPkg/source pin resolves (T08's pin bump is the natural vehicle).
- Consequence: `scripts/check-cqrs-lint.sh` gains a CI story (install + run per module), the gate stops being local-only, and the "9 tools unavailable"-class cqrs-lint gap in non-devShell environments disappears.
- Effort after approval: ~1 h (tag via verify-tag.sh choreography in go-cqrs-lite, flip the CI step, wire the strict gate item).
- **Decision requested:** approve tagging `cmd/cqrs-lint` in go-cqrs-lite + which version scheme (standalone `v0.x` vs family `v4.x`).

## 2. Feedback-inbox checker (T18 second half)

**State found 2026-10-01:** `new/` empty; `processed/` has 8 files — 7 predate the `> **PROCESSED**` marker convention (legacy, resolved in-file by their own "Resolution Status" sections), 1 modern file carries the marker; the stray root file `sec-consumer-feedback.md` (2026-07-05, SEC) was verified (its DecodeJSONWithRequest claims exist in `options_decode.go` + tests) and MOVED into `processed/` with a marker today — the stray-file half of the decision is already executed and needs only ratification.

**Recommended convention** (build the checker on it once confirmed):

1. Checker: `docs/feedback/new/` must be empty at TRAIN time (warning otherwise); every `processed/` file carries a top-of-file `PROCESSED|RESOLVED|ANNOTATED` marker, epoch-exempt for files dated before 2026-08-01 (the 7 legacy files stay as-is — their in-file resolution sections are the record).
2. New feedback lands in `new/` only; processing = verify claims → annotate → `git mv` (gotcha 20, unchanged).

- **Decision requested:** ratify the epoch exemption + the moved SEC file; then the checker is unambiguous to build (~45 min, atomic-gate checklist).
