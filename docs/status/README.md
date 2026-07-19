# docs/status/ — Historical Session Snapshots

**These files are point-in-time snapshots, not living documents.**

Each file in this directory captures what someone knew at the end of a work session. They are **append-only history** — never rewritten in place. A file from 2026-05-07 reflects what was true on 2026-05-07, not what is true today.

## How to read these

- **A reader who opens one of these** should treat it as a historical artifact, like a git commit message or a lab notebook page.
- **For current state**, see the living docs at the repository root:
  - `FEATURES.md` — what works today, by status
  - `TODO_LIST.md` — actionable work, current
  - `AGENTS.md` — non-obvious project context for AI sessions
  - `README.md` — what this project is, today

## Why not "update them all"?

Bulk-editing 91 historical files would be a [Verschlimmbesserung](https://en.wikipedia.org/wiki/Verschlimmbessern) — a well-intentioned change that makes things worse. Most snapshots are already clear in context (they carry a date, a session goal, and a "what's done" list). Adding identical "this is historical" banners to each would add noise without information.

If a specific file is actively misleading (e.g., claims something is broken when it's fixed, or claims something is TODO when it's shipped), add a **specific, end-of-file appendix note** to that one file — citing the commit or release that resolves it. The `update-old-docs` skill governs this work.

## File count

91 files spanning 2026-05-03 through 2026-07-17. The pace of session reports slowed as the project stabilized post-v4 release.
