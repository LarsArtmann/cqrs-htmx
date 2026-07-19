# docs/planning/ — Historical Plans & Execution Graphs

**These files are point-in-time plans, not the current roadmap.**

Each file in this directory captures a plan, execution graph, or proposal as it existed at a moment in time. They are **historical records** — many describe work that has since been completed, superseded, or abandoned.

## How to read these

- **A reader who opens one of these** should check the date first and then verify each claim against the current codebase. Plans are hypotheses, not reality.
- **For current direction**, see:
  - `ROADMAP.md` (repo root) — long-term vision, current
  - `TODO_LIST.md` (repo root) — short-term actionable work, current
  - `docs/adr/` — Architecture Decision Records (the "why" behind current design)

## Why not "update them all"?

Most planning files already include their own "what shipped" or "status" sections. Bulk-editing 39 files would add noise without information. Per the `update-old-docs` skill philosophy of restraint: annotate specific files only when a reader would be actively misled.

## Notable subdirectories

- `docs/planning/archive/` — older plans moved out of active view
- Many files reference D2 execution graphs (`.d2` + `.svg` pairs) that visualize the Pareto-prioritized work at the time of writing.
