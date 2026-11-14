# go-filewatcher Adoption Report (NOT adopted — correctly)

> Audit date: 2026-11-14 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-filewatcher/v2` v2.5.0 · Checkout: `/home/lars/projects/go-filewatcher`

## Verdict

| Direct usage       | Transitive | Latest                       | Verdict                   |
| ------------------ | ---------- | ---------------------------- | ------------------------- |
| **0 import sites** | no         | v2.5.0 (verified 2026-11-14) | **Correctly not adopted** |

## What it is

Production-grade fsnotify wrapper: recursive watching, 17+ composable filters, 18 middleware, global+per-path debouncing, .gitignore awareness, ENOSPC resilience, NFS/FUSE polling, self-healing, Prometheus/OTel hooks.

## Why cqrs-htmx should NOT adopt it

No file-watching concern exists anywhere in an HTTP/CQRS/HTMX library: no config hot-reload, no asset watching (bundles build via flake apps), no log tailing. Adding it would be a dependency looking for a justification. (The stale [docs/LIBRARY_INTEGRATIONS.md](../../LIBRARY_INTEGRATIONS.md) table already said "No" in 2026-05 — unchanged.)

Version currency: v2.5.0 (2026-11-06) is a minor bump inside the /v2 line that was already current at the last audit (v2.4.3) — it adds a `Healthy()` health-check API and an introspectable circuit breaker. No bearing on this verdict.

## Flip conditions

A dev-mode feature (e.g. live rebuild of templated admin assets when files change) would be the only plausible trigger — and even then it belongs in a dev tool behind a build tag, not the shipped library.

## How to verify

```bash
rg -n 'go-filewatcher' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'fsnotify' -g '*.go' -g '!scripts/**' .                 # none
git -C ~/projects/go-filewatcher for-each-ref --sort=-v:refname --format='%(refname:short)' refs/tags | head -1   # v2.5.0
```

## Prior reports

- [docs/LIBRARY_INTEGRATIONS.md](../../LIBRARY_INTEGRATIONS.md) (2026-05-13, superseded) — same verdict, then and now.
- 2026-11-05 (superseded by this refresh) — same verdict, then at v2.4.3.
