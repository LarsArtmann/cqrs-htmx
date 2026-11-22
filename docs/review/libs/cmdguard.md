# cmdguard Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-22 (prior: 2026-11-14) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/cmdguard/v5` v5.1.0 · Checkout: `/home/lars/projects/cmdguard`

## Verdict

| Direct usage       | Transitive                  | Latest                       | Verdict                                                       |
| ------------------ | --------------------------- | ---------------------------- | ------------------------------------------------------------- |
| **0 import sites** | no (no go.mod pin anywhere) | v5.1.0 (verified 2026-11-22) | **Correctly not adopted — a CLI framework in a library repo** |

## What it is

A production CLI framework wrapping Cobra: type-safe struct-tag flags, with command structure (missing handlers, duplicate commands, invalid flags) validated at construction instead of at 2am in production — plus lazy-service dependency injection with lifecycle, reverse-order graceful shutdown on SIGINT, `DoctorCommand` health checks, a zero-panic error contract, and styled output. Aimed at applications you run from a terminal.

## Why cqrs-htmx should NOT adopt it

1. **Category error** — cmdguard builds CLI applications; cqrs-htmx is a Go **library** consumed by import. It has no CLI surface: zero cobra usage anywhere, and the one internal binary, `cmd/cqrs-htmx-upgrade` (a one-shot v3→v4 codemod), needs only two stdlib `flag` bools (main.go:42-43) — a validated-flag framework would be pure overhead there.
2. **Task automation already has a home** — the flake provides nix apps (`nix run .#test`, `.#build`, `.#lint`, `.#fmt`, `.#bump-dep`, …; `apps = {` at flake.nix:381), standardized in the AGENTS.md command table; no Makefile/justfile exists to replace (re-verified). cmdguard solves a problem this repo does not have.
3. **Examples are servers, not CLIs** — every `examples/*` main (13 of them) is a `setup.Run`/HTTP server shape (e.g. examples/basic/main.go serves via `httputil.NewServer`), so consumers inherit no command surface to upgrade either.

Upstream check (2026-11-22): cmdguard's `[Unreleased]` is docs/website/CI tooling only (doc-table drift tripwires, a pre-tag release battery, website truth guards, fleet-bump/fleet-sweep scripts) — no API or capability that changes the category verdict; HEAD 0bf6678 sits past v5.1.0 on that bookkeeping.

## Flip conditions

A future `cqrs-htmx` scaffolding CLI (generate a consumer module: routes, store, projections wired up) is the one plausible trigger — and even then cmdguard/v5 belongs pinned in **that CLI's own go.mod**, never in the library modules.

## How to verify

```bash
rg -l 'cmdguard' --type go . | wc -l                                     # 0
rg -l 'spf13/cobra' --type go . | wc -l                                  # 0
rg -n 'flag\.' cmd/cqrs-htmx-upgrade/main.go | head -2                   # :42,:43 — stdlib flag is all the codemod needs
rg -n 'apps = \{' flake.nix                                              # :381 — task automation home (nix apps)
git -C ~/projects/cmdguard ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # v5.1.0
```

## Prior reports

- 2026-11-14 (prior state of this file): not-adopted verdict, no score (not-adopted convention) · delta: verdict unchanged against the 2026-11-22 tree — receipts re-derived (flag → main.go:42-43, nix apps → flake.nix:381), upstream `[Unreleased]` checked (docs/CI tooling only, nothing that weakens the category objection).
