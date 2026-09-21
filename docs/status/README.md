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
| `docs/status/*.md`                 | The most recent session reports only (unarchived tail; currently 3)          |
| `docs/status/archived/`            | 402 archived session reports (2026-05-03 → 2026-09-20)                       |
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
2. **Inline strikethrough on every resolved item**: `~~original line~~ done at \`hash\``, `~~original line~~ done (evidence)`, `~~original line~~ done (docs-health pass YYYY-MM-DD)`, or `~~original line~~ **Won't implement — reason.**`. **Unmarked items are the "open" signal** — never strike an item you did not verify.

Annotations are **additive only** — the original report text is never deleted. Cross-references in living docs point at the archived paths.

### Older dialects (reports dated before 2026-09-09) — LEGACY-EXEMPT

The inline-strikethrough convention was established on **2026-09-09**. Everything archived before that date predates it and is **exempt from the presence gate**:

| Dialect                          | Dates                     | Files | Marker                                        |
| -------------------------------- | ------------------------- | ----- | --------------------------------------------- |
| Inline strikethrough (current)   | 2026-09-09 onward         | 37    | `~~…~~ done (evidence)` + dated blockquote    |
| Prose blockquote                 | 2026-06-17 → 2026-09-07   | 72    | `> ANNOTATED …` verdict blockquote, no strikes |
| Unannotated                      | 2026-05-03 → 2026-08-09   | 268   | none                                          |

The prose-blockquote and unannotated eras are **historically complete as written** — their still-open items were harvested by later docs-health sweeps (2026-09-09, 2026-09-20, 2026-09-21). Retro-annotating 340 legacy files would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) with near-zero information gain, so the gate is deliberately scoped to the current convention era. New reports always use the current convention.

### Mixed tables are first-class (not a defect)

A table may mix **struck rows (done)** with **unstruck rows (open)**. That is exactly what "absence of a marker IS the open signal" means at row granularity, and it is how a report records partial completion honestly. The row tool `check-rows.py` reports such tables as `INCOMPLETE`; that finding is informational, **never** a reason to un-strike a verified-done row. The only row-level failure that matters is a `PARTIAL` row — a single row whose cells disagree (some struck, some not).

### Completeness gates

```bash
# Gate 1 (CI-wired): presence + dated blockquote for every gated-era report.
bash scripts/check-status-annotations.sh

# Gate 2 (manual, per audit): no PARTIAL rows in any struck table.
#   Mixed tables in the 8 adjudicated reports (2026-08-05_11-46, 2026-09-09_06-06,
#   2026-09-09_20-09, 2026-09-14_13-56_otel, 2026-09-17_13-11_templ-components,
#   2026-09-17_13-23_library-deep-dive, 2026-09-17_18-31_stability,
#   2026-09-17_21-04_adminui-migration) are intentional; ONLY `PARTIAL` lines fail.
python3 ~/.config/crush/skills/docs-health/assets/check-rows.py $(grep -rLl '~~' docs/status/archived/*.md)
```

**Adjudication log:** the 8 files above were individually inspected on 2026-09-21; every mixed table pairs verified-done rows with genuinely-open ones, and the two `PARTIAL` rows in `2026-08-05_11-46` (hand-annotated before the tooling existed) were normalized to full-row strikethrough.

## HTML corpus policy

The `*.html` files in `docs/status/` and `docs/architecture-understanding/` (~60 files) are **generated, self-contained report artifacts** (styled snapshots produced for a specific session or review) and are **exempt from annotation sweeps and freshness gates**. They are deliverables, not claims to maintain: their content is superseded by the living docs, and no markdown-source counterpart is expected. Do not edit them in place; archive or regenerate instead.

## File counts

- `docs/status/`: **3 unarchived reports** + README + 12 HTML artifacts.
- `docs/status/archived/`: **402 archived reports** (2026-05-03 → 2026-09-20).
- The archive tail has been swept repeatedly: 2026-09-09 (full backlog), 2026-09-20 (the 38-report tail), and 2026-09-21 (the 34-report tail). Of the 402 archived reports, **37 carry the current inline-strikethrough convention** (gated by `check-status-annotations.sh`), 72 carry the older prose blockquote, and 268 predate annotation entirely (legacy-exempt). Gate 1 enforces the current-convention era; Gate 2 keeps the row shapes honest.

## Why not "update them all" ad hoc?

Bulk-editing historical files without verification would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. The 2026-09-09 sweep did the verified version of this work once (every report read and checked against CHANGELOG/git/code before annotation). New reports: write them, and route anything still-open into the living docs so the next sweep stays small.
