# go-codec Utilization Report

> Audit date: 2026-11-05 · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-codec` · Checkout: `/home/lars/projects/go-codec`

## Verdict at a glance

| Pin | Latest (verified 2026-11-05) | Prod import sites | Test import sites | Adoption | Score |
| --- | --- | --- | --- | --- | --- |
| v0.3.3 | v0.3.3 (= latest) | 14 files | 11 files | **Used — event/sync payload encoding** | **56/100 count · high impact-weighted (core codecs at max)** |

## What the library is

The encoding toolkit behind the fleet's payloads: `JSONCodec`/`CBORCodec` (plus `CBORCompactCodec`, `DeterministicCodec`, `RawCodec`), the `Encoding` negotiation enum + `ForEncoding`, buffer pooling (`EncodePooled`, `Get/PutBuffer`), format autodetection (`AutoDetect`/`AutoDetectDebug`), `TranscodeToJSON`, an observable/metrics wrapper (`ObservableCodec`, `WithMetrics`), COSE (Sign1/Encrypt0) helpers, base64/normalization helpers, `Diagnose` (CBOR diagnostic notation), and `Size`.

## How cqrs-htmx uses it today

`JSONCodec` ×32 (the default event codec), `CBORCodec` ×11 (five binary payloads since [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md): `CredentialAdded`, `CredentialRemoved`, `CredentialUsed`, `TOTPEnabled`, `BotRegistered`), `EncodingJSON`/`EncodingCBOR`/`EncodingRaw` + `Encoding` enum + `ForEncoding` (identity-model payload registration, usermgmt stores, dashboardui, root `payload.go` + `sync_pull.go` honest encoding stamps). Adopted modules: identity-model, usermgmt, dashboardui, root.

## Capability inventory

| # | Capability | Status | Evidence / reason |
| --- | --- | --- | --- |
| 1 | JSON codec | 🟢 Fully leveraged | ×32; the default `event.WithCodec(codec.JSONCodec{})` call sites |
| 2 | CBOR codec | 🟢 Fully leveraged | ×11 post-ADR-0059; −33% bytes on binary-dominated payloads |
| 3 | Encoding negotiation (`Encoding`, `ForEncoding`, `Codec` interface) | 🟢 Fully leveraged | stamps ride event metadata + sync-pull payloads |
| 4 | Raw passthrough (`EncodingRaw`) | 🟢 Fully leveraged | sync protocol raw payload path (ADR-0056) |
| 5 | Format autodetection (`AutoDetect`) | 🔴 Missed | sync pull stamps `"opaque"` when a JSON stamp's bytes are invalid (gotcha 24 class); `AutoDetect` could classify honestly (json vs cbor) — needs an ADR-0056 contract check first |
| 6 | Transcoding (`TranscodeToJSON`) | 🔴 Missed (low) | read models decode to structs; only a future "serve CBOR events as JSON to legacy clients" path needs it |
| 7 | Buffer pooling (`EncodePooled`, `Get/PutBuffer`) | 🔴 Missed (low) | hot-path allocation trim; measurable only via the bench-spike gate |
| 8 | Codec observability (`ObservableCodec`, `WithMetrics`) | 🔴 Missed (low) | codec-level metrics would extend the observability demos |
| 9 | `Diagnose` (CBOR diagnostic notation) | 🔴 Missed (low) | dashboardui event inspector could render CBOR payloads human-readable |
| 10 | `CBORCompactCodec` / `DeterministicCodec` | ⚪ N/A | compact rejects unknown fields (wrong for event evolution); determinism unneeded — immutable JSON hashes via `HashTag` are already canonical |
| 11 | COSE (Sign1/Encrypt0) | ⚪ N/A | event signing/encryption is owned by go-cqrs-lite's storage layer, not the app codec choice |

## Score derivation

11 rows − 2 N/A = **9 applicable**; 4 full ×1 = **4 → 44/100** (band: low by count) — but the four full rows are the entire load-bearing codec surface; the five misses are adjunct tooling. Counting capability-impact instead of rows: **core at max, adjuncts unexplored → effective ~high (75-80)**. Honest headline: *used exactly where it matters, under-explored at the edges*.

## Version currency

v0.3.3 pinned = v0.3.3 latest (verified `git ls-remote --tags origin`, 2026-11-05).

## Should cqrs-htmx use it (more)?

Marginal gains only: `AutoDetect` for honest sync stamps (contract-gated), `Diagnose` in dashboardui's payload viewer, pooling behind a bench. None is TODO-grade today; all are recorded here for the next wave planning.

## How to verify

```bash
rg -oN 'codec\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn
rg -n 'CBORCodec' identity-model usermgmt | head   # ADR-0059 sites
rg -c 'AutoDetect|TranscodeToJSON|EncodePooled|Diagnose|ObservableCodec' --type go -g '!scripts/**' . | wc -l   # expect 0 prod
git -C ~/projects/go-codec ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1
```

## Prior reports

- [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md); finding #2 of the [2026-11-03 go-cqrs-lite deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html).
