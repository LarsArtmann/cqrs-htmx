# Upstream Issue Drafts (2026-09-18)

> Drafts prepared during the command-audit Pareto session (plan M17 + M18).
> NOT filed — filing goes through the verify-before-filing gate + Lars's voice.
> Each draft carries its in-repo evidence trail.

---

## Draft 1 — BuildFlow: docs-only pre-commit fast-path (M17)

**Repo:** BuildFlow (internal tooling)
**Type:** Feature request / performance
**Title:** Skip Go toolchain steps in pre-commit mode when the staged diff is docs-only

**Body draft:**

Pre-commit runs the full 18-step pipeline regardless of what is staged. On this
repository (27 Go modules under one go.work) a docs-only commit — only
`.md`/`.html` files in the staged set — still pays for:

- golangci-lint (workspace-wide load; ~1-2 min when the workspace is healthy),
- govalid-generate fan-out across all workspace modules,
- go mod tidy / gomod checks per touched module,
- template + codegen drift checks.

Two concrete failure modes observed (2026-09-17/18 sessions, cqrs-htmx):

1. **Wasted wall time:** docs-only commits took minutes of Go toolchain work
   that could not be affected by the change.
2. **Forced hook bypass:** while the workspace was mid-migration (root go.mod
   `go 1.27.1` vs go.work `1.26.7` — a concurrent-session state), the Go steps
   failed LOADED (not content) failures on every commit, so every docs commit
   had to use `--no-verify` with justification. The documented fallback exists,
   but the hook redness trains people to bypass.

**Ask:** in `--build-mode pre-commit --staged-only`, if the staged file set
matches none of the Go step inputs (no `*.go`, `go.mod`, `go.sum`, `*_templ.go`,
flake-relevant files), skip the Go toolchain steps (keep the content checks
that apply: large-file guard, formatter-relevant files if staged, etc.).
A `steps.inputs`-style declaration per step would also let future steps opt in
to input-based skipping declaratively.

**Evidence in-repo:** `docs/status/2026-09-18_07-52_pareto-plan-execution-session-status.md`
§d.1 (hook red on toolchain, daemon race absorbed commits);
AGENTS.md → "Findings gate DEMOTED" + "LIVE toolchain tug-of-war" gotchas.

---

## Draft 2 — BuildFlow: gomod-check + go-mod-ignore-check double-count (M18)

**Repo:** BuildFlow (internal tooling)
**Type:** Bug report
**Title:** gomod-check and go-mod-ignore-check report the same findings twice

**Body draft:**

In this repository's findings output, `gomod-check` and `go-mod-ignore-check`
both report the SAME 51 findings (25 vendor-consistency errors originating
from one local untracked `examples/*/vendor/` directory + 2
missing-submodule-replace errors + the go-structure-linter set), doubling the
noise without adding information. Whatever shared finder backs both steps, the
second step should either deduplicate against the first or the two should
partition their scopes.

**Evidence in-repo:** `.buildflow.yml` (the `fail_on: none` triage comment
block, 2026-09-17) — the double-count is listed there as a known-tool-bug note.
The triggering vendor directory was removed locally 2026-09-18 (trash; the dir
was gitignored and regenerable via `go mod vendor`), so the vendor-findings
half of the double-count no longer reproduces here — the double-reporting
mechanism itself is still worth fixing.

---

## Draft 3 — go-structure-linter: suppression feature + findings-gate restoration (M27 status check)

**Repo:** go-structure-linter (internal tooling)
**Type:** Status inquiry / dependency note

The cqrs-htmx findings gate was demoted to `fail_on: none` (2026-09-17,
triage in `.buildflow.yml`) pending TWO conditions: go-structure-linter
shipping a suppression feature (config already authored in
`.go-structure-linter.yml`) AND BuildFlow bumping its v0.10.0 pin. Before
restoring `fail_on: critical`, check the go-structure-linter repo for:

- a released suppression/config mechanism matching `.go-structure-linter.yml`,
- whether the root-package-files finding class (50 of 51 findings; a
  published-library-by-design false positive — all root-module files live in
  the repo root) gained a scope or exclude option.

**Verification performed this session (2026-09-18):** CHECKED. The in-config
`suppressions:` feature is implemented on go-structure-linter master (CHANGELOG
`## [Unreleased]`, checkout at `v0.10.0-98-g92af9d03`, 98 commits past the
last tag) — i.e. the feature exists but NO release tag carries it yet. The
findings-gate restoration in cqrs-htmx remains correctly blocked on: (1) a
go-structure-linter release tag shipping suppressions, (2) BuildFlow bumping
its pin. No action available in this repo; re-check at the next hygiene pass.
