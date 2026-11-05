# go-atomic-write Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-atomic-write` v0.6.2 · Checkout: `/home/lars/projects/go-atomic-write`

## Verdict

| Direct usage | Transitive | Latest | Verdict |
| --- | --- | --- | --- |
| **0 import sites** | yes — indirect via `samber-do-auditlog` → go-output chain (`auditlog/go.mod`) | v0.6.2 (verified 2026-11-05) | **Correctly not adopted** |

## What it is

Crash-safe, race-free file writes (fingerprint TOCTOU check, single-rename atomicity, fsync durability) — the disciplined replacement for every `os.WriteFile` that matters.

## Why cqrs-htmx should NOT adopt it

cqrs-htmx is an HTTP/CQRS **library**: its stores write through SQL drivers and in-memory journals, not plain files. Verified: no `os.WriteFile` in non-test library code (assets are embedded via `//go:embed`; SQL scripts are fixtures). Where the fleet does write files durably (auditlog's viewer persistence upstream), the dependency already arrives transitively and correctly.

## Flip conditions

If any module ever gains a file-backed store or cache (e.g. a default SQLite file with sidecar metadata written directly), adopt this instead of a hand-rolled temp+rename.

## How to verify

```bash
rg -n 'larsartmann/go-atomic-write' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'os\.WriteFile' -g '*.go' -g '!*_test.go' -g '!scripts/**' .        # none in library code
rg -n 'go-atomic-write' auditlog/go.mod
git -C ~/projects/go-atomic-write ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- None.
