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
| `docs/status/*.md`                 | The most recent session reports only (unarchived tail; currently 4)          |
| `docs/status/archived/`            | 402 annotated + archived session reports (2026-05-03 → 2026-09-20)           |
| `docs/status/*.html`               | 12 generated HTML report artifacts (see "HTML corpus" below)                 |
| `docs/planning/`                   | Active plans; superseded ones move to `docs/planning/archived/`              |
| `docs/reviews/archived/`           | Archived review documents                                                    |
| `docs/modularization/archived/`    | Archived module-assessment documents                                         |
| `docs/plans/archived/`             | Archived implementation plans                                                |
| `docs/feedback/processed/`         | Processed feedback documents                                                 |
| `docs/architecture-understanding/` | Generated self-contained HTML architecture reports (see "HTML corpus" below) |

`docs/status/archived/` is the ONLY archive directory — the older duplicate `docs/status/archive/` (a 92-file split-brain) was merged into it and removed.

## Annotation convention (since the 2026-09-09 full sweep)

When a historical report is verified against the current tree, it receives:

1. A dated **`> ANNOTATED YYYY-MM-DD`** blockquote at the top of the file, summarizing the verification verdict per section (DONE-SHIPPED / OPEN-TRACKED / OBSOLETE / HARVEST-CATCH, with evidence), and routing the remaining open items to `TODO_LIST.md`/`ROADMAP.md`.
2. **Inline strikethrough on every resolved item**: `~~original line~~ done at \`hash\``, `~~original line~~ done (evidence)`, `~~original line~~ done (docs-health pass YYYY-MM-DD)`, or `~~original line~~ **Won't implement — reason.**`. **Unmarked items are the "open" signal** — never strike an item you did not verify.

Annotations are **additive only** — the original report text is never deleted. Cross-references in living docs point at the archived paths.

**Completeness gates for an annotate sweep** (both must be clean before declaring the pass done):

```bash
grep -rLn '~~' docs/status/archived/          # every file carries >=1 resolution
python3 ~/.config/crush/skills/docs-health/assets/check-rows.py docs/status/archived/*.md
                                              # table rows uniformly struck/untouched (no PARTIAL)
```

## HTML corpus policy

The `*.html` files in `docs/status/` and `docs/architecture-understanding/` (~60 files) are **generated, self-contained report artifacts** (styled snapshots produced for a specific session or review) and are **exempt from annotation sweeps and freshness gates**. They are deliverables, not claims to maintain: their content is superseded by the living docs, and no markdown-source counterpart is expected. Do not edit them in place; archive or regenerate instead.

## File counts

- `docs/status/`: **4 unarchived reports** + README + 12 HTML artifacts.
- `docs/status/archived/`: **402 annotated reports** (2026-05-03 → 2026-09-20).
- The archive tail has been swept repeatedly: 2026-09-09 (full backlog), 2026-09-20 (the 38-report tail), and 2026-09-21 (this sweep: 34 further reports). Every archived file carries at least one inline resolution marker; the two completeness gates above are the enforcement.

## Why not "update them all" ad hoc?

Bulk-editing historical files without verification would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. The 2026-09-09 sweep did the verified version of this work once (every report read and checked against CHANGELOG/git/code before annotation). New reports: write them, and route anything still-open into the living docs so the next sweep stays small.
