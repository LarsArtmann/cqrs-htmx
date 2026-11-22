# go-atomic-write Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-22 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-atomic-write` v0.6.2 · Checkout: `/home/lars/projects/go-atomic-write`

## Verdict

| Direct usage       | Transitive                                                                                                              | Latest                       | Verdict                   |
| ------------------ | ----------------------------------------------------------------------------------------------------------------------- | ---------------------------- | ------------------------- |
| **0 import sites** | yes — indirect via `samber-do-auditlog` in 4 go.mods: auditlog, integration_test, samber-do-demo, e2e/server (all `// indirect`) | v0.6.2 (verified 2026-11-22) | **Correctly not adopted** |

## What it is

Crash-safe, race-free file writes (fingerprint TOCTOU check, single-rename atomicity, fsync durability) — the disciplined replacement for every `os.WriteFile` that matters.

## Why cqrs-htmx should NOT adopt it

cqrs-htmx is an HTTP/CQRS **library**: its stores write through SQL drivers and in-memory journals, not plain files — verified: zero `os.WriteFile` in non-test **library** code (assets embed via `//go:embed`; SQL scripts are fixtures). Where the fleet does write files durably (auditlog's viewer persistence upstream), the dependency already arrives transitively and correctly — the direct requirer is `samber-do-auditlog` (`~/projects/samber-do-auditlog/go.mod:12`), reaching this workspace indirectly in four go.mods: `auditlog/go.mod:28`, `integration_test/go.mod:96`, `examples/samber-do-demo/go.mod:55`, and `e2e/server/go.mod:72` (the e2e server pulls it via the auditlog live viewer it mounts, `e2e/server/bridges.go:120-149`). The 2026-11-14 report cited three modules — the fourth was already present in the 2026-11-14 tree, under-counted then; no new transitive consumer has appeared since. Same edge, honest count. No bearing on the verdict.

The one exception found at the 2026-11-05 audit — `cmd/cqrs-htmx-upgrade/rewrite.go` (the one-shot v3→v4 codemod CLI) rewriting consumer source trees with a plain `os.WriteFile` — was closed without adoption and stays closed: the codemod goes through a hand-rolled `writeFileAtomic` (write-temp-then-rename, permission-preserving, same-directory temp, cleanup on failure; `rewrite.go:102`, receipt re-verified exact 2026-11-22). Verified: zero `os.WriteFile` remains in non-test, non-script code. A 30-line local helper in a one-shot CLI beats a new edge dependency.

Upstream drift since the tag (HEAD `4f10347`, 2026-11-15): the CHANGELOG `[Unreleased]` section adds `WriteFuncWithPerm(path, fn, perm)` for streamed writes with explicit permissions, plus a parent-directory `MkdirAll` fix (first write into a fresh cache dir no longer ENOENTs). Both are wins for the transitive consumers (auditlog's viewer persistence) at their next pin bump — and neither creates a file-writing concern inside this repo. Flip conditions unchanged.

## Flip conditions

If any module ever gains a file-backed store or cache (e.g. a default SQLite file with sidecar metadata written directly), adopt this instead of a hand-rolled temp+rename.

## How to verify

```bash
rg -n 'larsartmann/go-atomic-write' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'os\.WriteFile' -g '*.go' -g '!*_test.go' -g '!scripts/**' .        # no matches — codemod writes via writeFileAtomic (rewrite.go:102)
rg -l 'go-atomic-write' -g 'go.mod' .                                    # 4: auditlog, e2e/server, examples/samber-do-demo, integration_test (all // indirect)
rg -n 'go-atomic-write' ~/projects/samber-do-auditlog/go.mod             # :12 — the one direct requirer
git -C ~/projects/go-atomic-write for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1   # v0.6.2
```

## Prior reports

- 2026-11-14 (prior state of this file): correctly-not-adopted, no score (this report's convention) · flip triggers re-hold — `writeFileAtomic` still at `rewrite.go:102`, still zero `os.WriteFile`; transitive set corrected 3→4 modules (`e2e/server` was already in the 2026-11-14 tree, under-counted); upstream `[Unreleased]` at HEAD `4f10347` (2026-11-15) adds `WriteFuncWithPerm` + MkdirAll fix.
- 2026-11-05 (superseded by this refresh) — same verdict; the codemod `os.WriteFile` exception has since been fixed with a local `writeFileAtomic`.
