# Decision Docs — Gated Work, Prepared 2026-08-30

> **ANNOTATED 2026-09-09 (docs-health):** the appkit row is no longer "gated work" — the ADR-001 fold-in started (RunWithAppkit stable 2026-09-07); its revalidation doc was archived to `archived/` with per-item verdicts. The remaining five rows are still awaiting the user, exactly as listed.

These are the PREPARED plans for the user-gated long tail. Nothing here has
been executed; each document contains the exact commands, risks, and the
decision it awaits.

| Doc                                                                                            | Decision awaited                                                                                                               | Risk class          |
| ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ------------------- |
| [2026-08-30_v4-branch-purge-plan.md](2026-08-30_v4-branch-purge-plan.md)                       | approve history rewrite of `origin/v4`                                                                                         | force-push          |
| [2026-08-30_setup-demo-blob-purge-plan.md](2026-08-30_setup-demo-blob-purge-plan.md)           | approve master history rewrite                                                                                                 | force-push          |
| [2026-08-30_buildcache-hardware-decision.md](2026-08-30_buildcache-hardware-decision.md)       | replace/retire the dead sda1 mount                                                                                             | hardware spend      |
| [2026-08-30_cqrs-lint-go-distribution-draft.md](2026-08-30_cqrs-lint-go-distribution-draft.md) | approve shipping a Go-installable cqrs-lint                                                                                    | new artifact        |
| [2026-08-30_upstream-issue-drafts.md](2026-08-30_upstream-issue-drafts.md)                     | file upstream asks (go-cqrs-lite)                                                                                              | external comms      |
| [2026-08-30_appkit-foldin-revalidation.md](archived/2026-08-30_appkit-foldin-revalidation.md)  | ~~execute the ADR-001 fold-in (now unblocked)~~ IN PROGRESS — `RunWithAppkit` stable 2026-09-07; items (b)-(f) on TODO_LIST P3 | architecture change |
