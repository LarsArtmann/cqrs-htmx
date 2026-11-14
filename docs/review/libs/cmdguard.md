# cmdguard Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/cmdguard/v5` v5.1.0 · Checkout: `/home/lars/projects/cmdguard`

## Verdict

| Direct usage       | Transitive                  | Latest                       | Verdict                                                       |
| ------------------ | --------------------------- | ---------------------------- | ------------------------------------------------------------- |
| **0 import sites** | no (no go.mod pin anywhere) | v5.1.0 (verified 2026-11-14) | **Correctly not adopted — a CLI framework in a library repo** |

## What it is

A production CLI framework wrapping Cobra: type-safe struct-tag flags, with command structure (missing handlers, duplicate commands, invalid flags) validated at construction instead of at 2am in production — plus lazy-service dependency injection with lifecycle, reverse-order graceful shutdown on SIGINT, `DoctorCommand` health checks, a zero-panic error contract, and styled output. Aimed at applications you run from a terminal.

## Why cqrs-htmx should NOT adopt it

1. **Category error** — cmdguard builds CLI applications; cqrs-htmx is a Go **library** consumed by import. It has no CLI surface: zero cobra usage anywhere, and the one internal binary, `cmd/cqrs-htmx-upgrade` (a one-shot v3→v4 codemod), needs only two stdlib `flag` bools — a validated-flag framework would be pure overhead there.
2. **Task automation already has a home** — the flake provides nix apps (`nix run .#test`, `.#build`, `.#lint`, `.#fmt`, `.#bump-dep`, …), standardized in the AGENTS.md command table; no Makefile/justfile exists to replace. cmdguard solves a problem this repo does not have.
3. **Examples are servers, not CLIs** — every `examples/*` main is a `setup.Run`/HTTP server, so consumers inherit no command surface to upgrade either.

## Flip conditions

A future `cqrs-htmx` scaffolding CLI (generate a consumer module: routes, store, projections wired up) is the one plausible trigger — and even then cmdguard/v5 belongs pinned in **that CLI's own go.mod**, never in the library modules.

## How to verify

```bash
rg -l 'cmdguard' --type go . | wc -l                                     # 0
rg -l 'spf13/cobra' --type go . | wc -l                                  # 0
rg -n 'flag\.' cmd/cqrs-htmx-upgrade/main.go | head -2                   # stdlib flag is all the codemod needs
rg -n 'apps = \{' flake.nix                                              # task automation home (nix apps)
git -C ~/projects/cmdguard for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1
```

## Prior reports

- None.
