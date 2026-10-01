# docs/status/ — Historical Session Snapshots

**These files are point-in-time snapshots, not living documents.**

Each report in this tree captures what someone knew at the end of a work session. Reports are **append-only history** — never rewritten in place. A file from 2026-06-16 reflects what was true on 2026-06-16, not what is true today.

## How to read these

- **A reader who opens one of these** should treat it as a historical artifact, like a git commit message or a lab notebook page.
- **For current state**, see the living docs at the repository root:
  - `FEATURES.md` — what works today, by status
  - `TODO_LIST.md` — actionable work, current
  - `AGENTS.md` — non-obvious project context for AI sessions
  - `README.md` — what this project is, today

## Layout (reorganized 2026-09-09; archive consolidated 2026-09-20)

| Path                               | Contents                                                                     |
| ---------------------------------- | ---------------------------------------------------------------------------- |
| `docs/status/*.md`                 | The most recent session reports only (unarchived tail; currently 2)          |
| `docs/status/archived/`            | 437 archived session reports (2026-05-03 → 2026-10-01)                       |
| `docs/status/*.html`               | 12 generated HTML report artifacts (see "HTML corpus" below)                 |
| `docs/planning/`                   | Active plans; superseded ones move to `docs/planning/archived/`              |
| `docs/reviews/archived/`           | Archived review documents                                                    |
| `docs/modularization/archived/`    | Archived module-assessment documents                                         |
| `docs/plans/archived/`             | Archived implementation plans                                                |
| `docs/feedback/processed/`         | Processed feedback documents                                                 |
| `docs/architecture-understanding/` | Generated self-contained HTML architecture reports (see "HTML corpus" below) |

`docs/status/archived/` is the ONLY archive directory — the older duplicate `docs/status/archive/` (a 92-file split-brain) was merged into it and removed.

## Annotation convention

### Current convention (reports dated 2026-09-09 onward)

When a historical report is verified against the current tree, it receives:

1. A dated **`> ANNOTATED YYYY-MM-DD`** blockquote at the top of the file, summarizing the verification verdict per section (DONE-SHIPPED / OPEN-TRACKED / OBSOLETE / HARVEST-CATCH, with evidence), and routing the remaining open items to `TODO_LIST.md`/`ROADMAP.md`.
2. **Inline strikethrough on every resolved item**: `~~original line~~ done at \`hash\``,` ~~original line~~ done (evidence)`,` ~~original line~~ done (docs-health pass YYYY-MM-DD)`, or` ~~original line~~ **Won't implement — reason.**`. **Unmarked items are the "open" signal** — never strike an item you did not verify.

Annotations are **additive only** — the original report text is never deleted. Cross-references in living docs point at the archived paths.

### Older dialects (reports dated before 2026-09-09) — LEGACY-EXEMPT

The inline-strikethrough convention was established on **2026-09-09**. Everything archived before that date predates it and is **exempt from the presence gate**:

| Dialect                        | Dates                   | Files | Marker                                         |
| ------------------------------ | ----------------------- | ----- | ---------------------------------------------- |
| Inline strikethrough (current) | 2026-09-09 onward       | 37    | `~~…~~ done (evidence)` + dated blockquote     |
| Prose blockquote               | 2026-06-17 → 2026-09-07 | 72    | `> ANNOTATED …` verdict blockquote, no strikes |
| Unannotated                    | 2026-05-03 → 2026-08-09 | 268   | none                                           |

The prose-blockquote and unannotated eras are **historically complete as written** — their still-open items were harvested by later docs-health sweeps (2026-09-09, 2026-09-20, 2026-09-21). Retro-annotating 340 legacy files would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) with near-zero information gain, so the gate is deliberately scoped to the current convention era. New reports always use the current convention.

### Mixed tables are first-class (not a defect)

A table may mix **struck rows (done)** with **unstruck rows (open)**. That is exactly what "absence of a marker IS the open signal" means at row granularity, and it is how a report records partial completion honestly. The repo gate `scripts/check-status-rows.py` counts such tables and reports them as `deliberately-mixed` — informational, **never** a reason to un-strike a verified-done row. The only row-level failure that matters is a `PARTIAL` row — a single row whose cells disagree (some struck, some not).

### Completeness gates

Both gates are repo-owned, CI-wired (the `checks` job), and run inside `nix run .#check-modules`:

```bash
# Gate 1: presence + dated blockquote for every gated-era report.
bash scripts/check-status-annotations.sh

# Gate 2: no PARTIAL rows in any struck table; deliberately-mixed tables
#   are counted and reported (first-class by convention), never failed.
python3 scripts/check-status-rows.py

# Fixture self-tests for both gates (also CI-wired).
bash scripts/test-check-status-annotations.sh
bash scripts/test-check-status-rows.sh

# Fixer for the PARTIAL-row class Gate 2 catches: completes a row by striking
#   every remaining cell (the whole-row-strike policy above). Idempotent; the
#   checker and fixer are proven to agree by the self-test. `--dry-run` first.
python3 scripts/normalize-status-rows.py --dry-run
bash scripts/test-normalize-status-rows.sh
```

A third, **advisory** gate keeps the live tail honest: `scripts/check-docs-tail-budget.sh`
warns (exit 0) when `docs/status/*.md` (excluding this README) exceeds 3 reports,
so a growing tail surfaces at sweep boundary instead of being re-discovered.
It is deliberately **not** a blocking `check-modules` stage — a mid-session tail
of two or three reports is legitimate. `--strict` exits 1 for a manual/train-time
check. Fixture self-test: `bash scripts/test-check-docs-tail-budget.sh`.

Ambient `python3` is the contract for Gate 2 (CI runners ship it); the flake apps
(`nix run .#check-status-rows` / `.#test-status-rows` / `.#normalize-status-rows`
/ `.#test-normalize-status-rows` / `.#check-docs-tail-budget` /
`.#test-check-docs-tail-budget`) additionally pin `pkgs.python3` for hermetic use.
The docs-health skill's authoring-time `check-rows.py` asset remains the
annotator's aid — it reports mixed tables as `INCOMPLETE` for a human to
adjudicate; scope-identity with the repo gate was verified per-file and per-line
(same 19 mixed tables) on 2026-09-22.

**Adjudication log:** the 8 files above were individually inspected on 2026-09-21; every mixed table pairs verified-done rows with genuinely-open ones, and the two `PARTIAL` rows in `2026-08-05_11-46` (hand-annotated before the tooling existed) were normalized to full-row strikethrough.

## HTML corpus policy

The `*.html` files in `docs/status/` and `docs/architecture-understanding/` (~60 files) are **generated, self-contained report artifacts** (styled snapshots produced for a specific session or review) and are **exempt from annotation sweeps and freshness gates**. They are deliverables, not claims to maintain: their content is superseded by the living docs, and no markdown-source counterpart is expected. Do not edit them in place; archive or regenerate instead. **Standing decision (docs-health round 14, 2026-10-01): the 12-file `docs/status/*.html` corpus is KEEP-IN-PLACE** — archiving would be churn without information gain (they are already exempt from every gate and indexed by this README), and the TODO_LIST item that asked for the deliberate decision is closed with this line as the record.

## File counts

- `docs/status/`: **2 unarchived reports** (the 2026-10-01 12:30 round-14 report + the 10:30 W0–W3 Pareto-execution report — the live tail) + README + 12 HTML artifacts. The round-13 pass archived the 02:59 BuildFlow-recovery report and its own round-13 audit report (both annotated with same-day round-14 top-up blockquotes + evidence strikes); the round-6 stub anomaly was removed 2026-10-01.
- `docs/status/archived/`: **437 archived reports** (2026-05-03 → 2026-10-01).
- The archive tail has been swept repeatedly: 2026-09-09 (full backlog), 2026-09-20 (the 38-report tail), 2026-09-21 (the 34-report tail), 2026-09-22 (the 8-report superb-plan tail + the plan itself), 2026-10-01 round-13 (the 22-report 09-22→09-30 tail — every report annotated with a dated blockquote + evidence-backed inline strikes, open items routed to TODO_LIST/ROADMAP, then archived), and **2026-10-01 round-14 (the 10-01 pair — the 02:59 BuildFlow-recovery and 06:36 round-13 reports got W0–W3-receipt top-up blockquotes + inline strikes for everything the train/gate/erraudit sessions then resolved, then archived; the same pass triaged the six 2026-08-30 planning files — upstream-asks draft archived, gated-work index + cqrs-lint distribution draft annotated, three live owner-gated decisions confirmed leave-in-place — and confirmed the HTML corpus keep-in-place decision)**. Of the 437 archived reports, **72 carry the current inline-strikethrough convention** (gated by `check-status-annotations.sh`), 72 carry the older prose blockquote, and the rest predate annotation entirely (legacy-exempt). Gate 1 enforces the current-convention era; Gate 2 keeps the row shapes honest.

## Why not "update them all" ad hoc?

Bulk-editing historical files without verification would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. The 2026-09-09 sweep did the verified version of this work once (every report read and checked against CHANGELOG/git/code before annotation). New reports: write them, and route anything still-open into the living docs so the next sweep stays small.
