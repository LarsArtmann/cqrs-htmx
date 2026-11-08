# go-credentials-store Adoption Report (NOT adopted — correctly; it is a sibling application)

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-credentials-store` v0.3.1 · Checkout: `/home/lars/projects/go-credentials-store`

## Verdict

| Direct usage       | Transitive | Latest                       | Verdict                                               |
| ------------------ | ---------- | ---------------------------- | ----------------------------------------------------- |
| **0 import sites** | no         | v0.3.1 (verified 2026-11-05) | **Correctly not adopted — category error, not a gap** |

## What it is

An event-sourced, encrypted-at-rest credentials vault for **personal infrastructure** (API keys, Playwright session blobs, passwords) with lifecycle audit and a loopback admin UI — built ON go-cqrs-lite's `system` composition root + `metaengine` read models, sealed XChaCha20-Poly1305 before events.

## Why cqrs-htmx should NOT adopt it

1. **It is a consumer, not a dependency** — a sibling application composed from the same foundation cqrs-htmx sits on (go-cqrs-lite system/metaengine). Depending on it would invert the stack: a web/CQRS library importing an end-user vault application.
2. **Different domain entirely** — cqrs-htmx's usermgmt manages human identity (WebAuthn/TOTP/OAuth2, tenants, roles); credentials-store manages machine/personal secrets. Zero shared aggregates.
3. **Its value to this repo is as a reference**: it is a proof that the system+metaengine composition (which `systemadapter` bridges) sustains a real event-sourced application — worth citing in systemadapter's docs, nothing more.

## Flip conditions

None realistic. If cqrs-htmx ever needed a "bot credentials" feature, the honest shape would be an event-sourced aggregate inside identity-model, not an import of a vault app.

## How to verify

```bash
rg -n 'go-credentials-store' --type go -g '!scripts/**' . | wc -l   # 0
rg -n 'system.New|metaengine' ~/projects/go-credentials-store/README.md | head -3
git -C ~/projects/go-credentials-store ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- None.
