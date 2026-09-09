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

## Layout (reorganized 2026-09-09)

| Path                        | Contents                                                                                                  |
| --------------------------- | --------------------------------------------------------------------------------------------------------- |
| `docs/status/*.md`          | The 3 most recent session reports only (unarchived tail)                                                   |
| `docs/status/archived/`     | 273 annotated + archived session reports (2026-06-16 → 2026-09-07)                                        |
| `docs/status/*.html`        | Generated HTML report artifacts (see "HTML corpus" below)                                                 |
| `docs/planning/`            | Active plans; superseded ones move to `docs/planning/archived/`                                            |
| `docs/reviews/archived/`    | Archived review documents                                                                                 |
| `docs/modularization/archived/` | Archived module-assessment documents                                                                  |
| `docs/plans/archived/`      | Archived implementation plans                                                                             |
| `docs/feedback/processed/`  | Processed feedback documents                                                                              |
| `docs/architecture-understanding/` | Generated self-contained HTML architecture reports (see "HTML corpus" below)                        |

## Annotation convention (since the 2026-09-09 full sweep)

When a historical report is verified against the current tree, it receives:

1. A dated **`> ANNOTATED YYYY-MM-DD`** blockquote at the top of the file, summarizing the verification verdict per section (DONE-SHIPPED / OPEN-TRACKED / OBSOLETE / HARVEST-CATCH, with evidence).
2. Targeted **inline suffixes** where a specific line needs it: `✅ done` (shipped, evidence), `→ routed` (moved to TODO_LIST/ROADMAP with destination), or `STALE` (factually wrong today).

Annotations are **additive only** — the original report text is never deleted. Cross-references in living docs point at the archived paths.

## HTML corpus policy

The `*.html` files in `docs/status/` and `docs/architecture-understanding/` (~60 files) are **generated, self-contained report artifacts** (styled snapshots produced for a specific session or review) and are **exempt from annotation sweeps and freshness gates**. They are deliverables, not claims to maintain: their content is superseded by the living docs, and no markdown-source counterpart is expected. Do not edit them in place; archive or regenerate instead.

## File counts

- `docs/status/`: 3 unarchived reports + 273 archived reports + 12 HTML artifacts.
- Archived reports span 2026-06-16 through 2026-09-07. The pace of session reports grew through the v4.x trains; the 2026-09-09 docs-health sweep verified and archived the full backlog in one pass.

## Why not "update them all" ad hoc?

Bulk-editing historical files without verification would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. The 2026-09-09 sweep did the verified version of this work once (every report read and checked against CHANGELOG/git/code before annotation). New reports: write them, and route anything still-open into the living docs so the next sweep stays small.
