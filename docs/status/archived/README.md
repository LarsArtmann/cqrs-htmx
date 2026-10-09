# Archived status reports

Closed history — every archived report's forward-looking items carry inline resolutions (`done at <hash>` / `Won't implement` / `NOT-DO` / `DUPLICATE/ROUTED → TODO_LIST`). Do not re-open archived files during HARVEST/VERIFY; the convention lives in [`docs/status/README.md`](../README.md).

## Archive manifest (bulk passes)

One line per archived file for bulk sweeps (per the docs-health ARCHIVE rule; single-file archives are documented by their commit messages).

- **2026-10-09 (docs-health tail pass, 2 files):**
  - `2026-10-08_22-02_train-ci-repair-family-walk-fix-fixture-helper-status.md` — ANNOTATE+ARCHIVE: patch-train/httputil-release/CI-repair session. 55 items struck: fixtures `70b1289e`, checker fix receipt `23b54cc2` (CHANGELOG-verified), runbook docs `b2a12066`, R12/T8 `baeffce1`+`c5d223de`, R15/T14 `ab23e247`, A012 0-re-derived, push landed → origin green (`c5bbc693`); survivors routed to TODO rows + decision table D14/D15 + the train-trust-plan row; §f28 gate-covered, §f30 NOT-DO, §f34/35 Won't-implement.
  - `2026-10-09_00-06_fixture-consolidation-upstream-wave-branching-flow-hardening_status.md` — ANNOTATE+ARCHIVE: fixture consolidation + upstream-wave alignment + branching-flow ratchet session. 8 items struck: battery/owner lanes routed (TODO battery row + D14/D15 + GCL row), concurrent sync feature landed (`278b16c2`), CI verified green (run 37851669797 on `c5bbc693`), feedback inbox empty.
