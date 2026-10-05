# v1.20.0 root go.mod: 4 submodule requires are zero-commit pseudo-versions — the tag is unconsumable from the proxy

## Problem

The published `v1.20.0` root go.mod requires four submodules at replace-generated zero-commit pseudo-versions instead of their real tags:

```
github.com/larsartmann/templ-components/charts/echarts v1.20.0-00010101000000-000000000000
github.com/larsartmann/templ-components/datastar v1.20.0-00010101000000-000000000000
github.com/larsartmann/templ-components/errorpage v1.20.0-00010101000000-000000000000
github.com/larsartmann/templ-components/htmx v1.20.0-00010101000000-000000000000
```

Proxy view: https://proxy.golang.org/github.com/larsartmann/templ-components/@v/v1.20.0.mod

All four real tags exist (`charts/echarts/v1.20.0`, `datastar/v1.20.0`, `errorpage/v1.20.0`, `htmx/v1.20.0` — `git ls-remote` confirms each), but a zero-commit pseudo-version never resolves, so any consumer bumping to v1.20.0 fails with `unknown revision`. Downstream in cqrs-htmx this blocked the family-alignment sweep and every workspace-mode build until we pinned back.

## Root cause

`icons` and `utils` resolved correctly (`v1.20.0`, clean requires) — those got the drop-replaces → tidy → tag treatment. The four broken ones still carried their local `replace ... => ./...` directives when the release go.mod was finalized, so `go mod tidy` kept the pseudo-version for them. `98c6777e` re-added the replaces after the release — correct for dev, but the same trap is armed for the next tag.

v1.19.4 (the previous release) required all six submodules at clean `v1.19.4` tags (proxy: https://proxy.golang.org/github.com/larsartmann/templ-components/@v/v1.19.4.mod), so this is a v1.20.0 regression in the release ritual, not in the code.

## Fix

**Cut `v1.20.1` from a tree where the four submodule replaces are dropped and `go mod tidy` has rewritten the requires to the real tags** (exactly the icons/utils pattern), then re-add the replaces. v1.20.0 itself cannot be re-pushed — the proxy has the tag cached.

Optional guard so this class dies permanently: a pre-tag check that fails on `-00010101000000` in any go.mod, or equivalently a release ritual that drops all family replaces + tidies before every tag.

💘 Generated with Crush
