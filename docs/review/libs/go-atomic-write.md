# go-atomic-write Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-atomic-write` v0.6.2 · Checkout: `/home/lars/projects/go-atomic-write`

## Verdict

| Direct usage       | Transitive                                                                                   | Latest                       | Verdict                   |
| ------------------ | -------------------------------------------------------------------------------------------- | ---------------------------- | ------------------------- |
| **0 import sites** | yes — indirect via `samber-do-auditlog` (auditlog, integration_test, samber-do-demo go.mods) | v0.6.2 (verified 2026-11-14) | **Correctly not adopted** |

## What it is

Crash-safe, race-free file writes (fingerprint TOCTOU check, single-rename atomicity, fsync durability) — the disciplined replacement for every `os.WriteFile` that matters.

## Why cqrs-htmx should NOT adopt it

cqrs-htmx is an HTTP/CQRS **library**: its stores write through SQL drivers and in-memory journals, not plain files — verified: zero `os.WriteFile` in non-test **library** code (assets embed via `//go:embed`; SQL scripts are fixtures). Where the fleet does write files durably (auditlog's viewer persistence upstream), the dependency already arrives transitively and correctly.

The one exception found at the last audit — `cmd/cqrs-htmx-upgrade/rewrite.go` (the one-shot v3→v4 codemod CLI) rewriting consumer source trees with a plain `os.WriteFile` — has since been **closed without adoption**: the codemod now goes through a hand-rolled `writeFileAtomic` (write-temp-then-rename, permission-preserving, same-directory temp, cleanup on failure; `rewrite.go:102`). Verified: zero `os.WriteFile` remains in non-test, non-script code. A 30-line local helper in a one-shot CLI beats a new edge dependency.

## Flip conditions

If any module ever gains a file-backed store or cache (e.g. a default SQLite file with sidecar metadata written directly), adopt this instead of a hand-rolled temp+rename.

## How to verify

```bash
rg -n 'larsartmann/go-atomic-write' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'os\.WriteFile' -g '*.go' -g '!*_test.go' -g '!scripts/**' .        # none — codemod writes via writeFileAtomic (rewrite.go:102)
rg -l 'go-atomic-write' */go.mod examples/*/go.mod                       # auditlog, integration_test, samber-do-demo (indirect)
git -C ~/projects/go-atomic-write for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1
```

## Prior reports

- 2026-11-05 (superseded by this refresh) — same verdict; the codemod `os.WriteFile` exception has since been fixed with a local `writeFileAtomic`.
