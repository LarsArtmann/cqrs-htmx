# go-error-family Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-error-family` · Checkout: `/home/lars/projects/go-error-family`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| v0.12.0 | v0.12.0 (= latest) | 131 files | 54 files | **Used — the error substrate of the whole fleet** | **65/100 count · ~82 impact-weighted** |

## What the library is

Typed error families (Rejection / Conflict / Transient / Corruption / Infrastructure / Orchestration) with machine-readable codes, context, classification/registry, retryability predicates, audience-aware messages, CLI exit-code handling, and HTTP status mapping. In this repo `errors.New`/`fmt.Errorf` are **banned in non-test code** (branching-flow enforced); every error flows through this library or its `go-cqrs-lite/event` re-exports.

## How cqrs-htmx uses it today

Symbol census (prod references): `NewRejection` ×200, `WrapInfrastructure` ×116, `Wrapf` ×109, `WrapTransient` ×97, `Classify` ×77, `NewTransient` ×70, `Newf` ×44, `WrapRejection` ×35, `NewInfrastructure` ×38, `NewConflict` ×29, `WrapCorruption` ×25, `Code` ×12, `IsRetryable` ×4, `Orchestration` ×5 (auditlog, adminui, usermgmt). Families map to HTTP status via `httperr.MapError` (Rejection→400, Conflict→409, Transient→503, Corruption→500, Infrastructure→503); denials share codes `forbidden`/`unauthorized` walked by `IsForbidden`/`IsUnauthorized`.

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | Family constructors (`New{Rejection,Transient,Conflict,Infrastructure,Corruption}` + `Newf`) | 🟢 Fully leveraged | 400+ ctor refs fleet-wide |
| 2 | Wrap constructors (`Wrap*`/`Wrapf` variants) | 🟢 Fully leveraged | 382 refs across wrap variants |
| 3 | Classification (`Classify`, `RegisterClassification` via event re-export) | 🟢 Fully leveraged | `Classify` ×77; root registers httputil classifications via `sync.Once` |
| 4 | Error type, codes, context (`Error`, `Code`, `Summary`) | 🟢 Fully leveraged | `code` field rides JSON/ProblemDetails error payloads |
| 5 | Family predicates (`Family`, `IsRetryable`) | 🟢 Fully leveraged | retry predicates drive middleware policy |
| 6 | Orchestration family (multi-step composition errors) | 🟢 Fully leveraged | auditlog/adminui/usermgmt sites |
| 7 | `WrapOnce` idempotent wrapping | 🔴 Missed | 0 prod uses; double-wrap through middleware chains re-wraps causes — low-stakes dedup nicety |
| 8 | Audience-aware errors (`Audience`, `ParseAudience`) | 🔴 Missed | 0 uses; `httperr.SafeDetail` could split user-facing vs operator detail — candidate, needs design |
| 9 | Structured classified logging (`LogError`, `LogErrorContext`) | 🔴 Missed | 0 uses; repo logs via its own slog wiring — family-aware log levels would follow severity |
| 10 | CLI handling (`HandleError*`, `ExitCode`, `ExitCoder`) | ⚪ N/A | Library, not a CLI; consumer apps (and `go-appkit` examples) own exit codes |
| 11 | HTTP mapping (`HTTPStatus`, `HTTPHandler`) | ⚪ N/A | Deliberate: `httperr` owns the error→HTTP surface; a second mapping layer would conflict |
| 12 | Registries & message templates (`NewRegistry`, `RegisterClassifier(s)`, `RegisterTemplate`, `TemplateForCode`) | 🟡 Partially used | Only the global classification registry (via event re-export) is used; per-`Registry` isolation and code→message templates unexplored (templates could power consistent SafeDetail strings) |

## Score derivation

12 rows − 2 N/A = **10 applicable**; 6 full ×1 + 1 partial ×0.5 = **6.5 → 65/100** (band: moderate). Impact-weighted reading: the missed rows are adjunct conveniences — the load-bearing surface (creation, wrapping, classification, predicates) is 100% adopted and enforced by gates, so the effective posture is **high (~82)**.

## Version currency

v0.12.0 pinned = v0.12.0 latest (verified `git ls-remote --tags origin`). Local checkout at v0.12.0, 0 commits behind origin.

## Should cqrs-htmx use it (more)?

Yes — and maximally on the core. Two genuine stretch items worth a TODO each: (1) `Audience`-filtered `SafeDetail` in `httperr`, (2) `LogError`-style classified logging in the middleware seam. Neither is load-bearing.

## How to verify

```bash
rg -oN 'errorfamily\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn | head -25
for s in WrapOnce Audience LogError NewRegistry RegisterTemplate; do rg -c "errorfamily\.$s" --type go . | wc -l; done   # expect 0s
git -C ~/projects/go-error-family ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- [docs/LIBRARY_INTEGRATIONS.md](../../LIBRARY_INTEGRATIONS.md) (2026-05-13) graded go-error-family "transitive, should consider" — superseded by direct, pervasive adoption (see its superseded banner).
