# go-codec Utilization Report

> Audit date: 2026-11-22 (refreshes the 2026-11-14 audit) · Method: library-deep-dive (fleet series, [index](index.html))
> Library: `github.com/larsartmann/go-codec` · Checkout: `/home/lars/projects/go-codec`

## Verdict at a glance

| Pin    | Latest (verified 2026-11-22) | Prod import sites | Test import sites | Adoption                                   | Score                                 |
| ------ | ---------------------------- | ----------------- | ----------------- | ------------------------------------------ | ------------------------------------- |
| v0.3.3 | v0.3.3 (= latest)            | 17 files          | 14 files          | **Used — dual-codec (JSON+CBOR) payloads** | **71/100 count · high (core at max)** |

## What the library is

The encoding toolkit behind the fleet's payloads: `JSONCodec`/`CBORCodec` (plus `CBORCompactCodec`, `DeterministicCodec`, `RawCodec`), the `Encoding` negotiation enum + `ForEncoding`, buffer pooling (`EncodePooled`, `Get/PutBuffer`), format autodetection (`AutoDetect`/`AutoDetectDebug`), `TranscodeToJSON`, an observable/metrics wrapper (`ObservableCodec`, `WithMetrics`), COSE (Sign1/Encrypt0) helpers, base64/normalization helpers, `Diagnose` (CBOR diagnostic notation), and `Size`.

## How cqrs-htmx uses it today

ADR-0059 made this a **dual-codec** story, and since go-cqrs-lite event/v4 v4.15.2 `event.New` defaults to CBOR — so the decode side had to grow encoding-aware switches everywhere (gotcha 55, `docs/gotchas.md:225`):

- **JSON** (`JSONCodec` ×37): still the codec for non-binary payloads (`es_bot_decide.go:99`, `es_decide_security.go:64`, snapshot default) — and **adoption grew this pass**: the new `legalparty-model` module stamps its events with it (`legalparty-model/decider.go:52,100,144`), the driver of ×32→×37.
- **CBOR encode**: one shared stateless instance `cborEventCodec` (`usermgmt/es_decide_credentials.go:15`) drives **all five** ADR-0059 payloads — CredentialAdded/Removed/Used (`es_decide_credentials.go:50,93,155`), TOTPEnabled (`es_decide_security.go:32`), BotRegistered (`es_bot_decide.go:63`) — the latter two sites now cite the shared instance too (they spelled the literal type at the 2026-11-14 audit).
- **CBOR/JSON decode — switch on `evt.Encoding()`**: canonical switch in `dashboardui/core/payload.go` (`typedRender` :100-118, `DefaultPayloadRenderer` :131-168); mirrored in `dashboardui/core/command_fates.go:316` (`decodeLifecyclePayload`); root `payload.go:19-38` (`DecodePayloadAuto`/`DecodePayload`/`DecodePayloads` re-exports); per-event resolution via `codec.ForEncoding(evt.Encoding())` at `identity-model/payload.go:21` **and — new this pass — `legalparty-model/events.go:41`**; `usermgmt/snapshot.go:45` pluggable snapshot `Codec`.
- **Sync stamps**: `sync_pull.go:30-34` reports honest delivery framings (`SyncPayloadEncodingJSON`/`SyncPayloadEncodingOpaque`) per ADR-0056 Amendment 1.
- **Adjunct adopted**: `codec.Diagnose` EDN fallback for stamped-encoding decode failures (`dashboardui/core/payload.go:195`, landed 2026-11-06) — the UI never surfaces raw binary bytes.

Symbol histogram (2026-11-22): `JSONCodec` ×37, `EncodingJSON` ×18, `Encoding` ×13, `CBORCodec` ×11, `EncodingCBOR` ×7, `EncodingRaw` ×5, `ForEncoding` ×3, `Codec` ×3, `Diagnose` ×2 (both prod), `EncodePooled` ×2 (bench only).

## Capability inventory

| #   | Capability                                                          | Status             | Evidence / reason                                                                                                                                                                                                                                                                                                                                                 |
| --- | ------------------------------------------------------------------- | ------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | JSON codec                                                          | 🟢 Fully leveraged | ×37 `event.WithCodec(codec.JSONCodec{})` call sites (grew ×32→×37 with legalparty-model, 2026-11-14→22)                                                                                                                                                                                                                                                           |
| 2   | CBOR codec                                                          | 🟢 Fully leveraged | 5 ADR-0059 payloads via the one shared `cborEventCodec`; −33% bytes on binary-dominated payloads; decode-side switches everywhere gotcha 55 demands — re-verified this pass                                                                                                                                                                                       |
| 3   | Encoding negotiation (`Encoding`, `ForEncoding`, `Codec` interface) | 🟢 Fully leveraged | `evt.Encoding()` rule at every decode site; `ForEncoding` per-event resolution now at TWO sites (identity-model `payload.go:21`, legalparty-model `events.go:41`); sync-pull delivery stamps (ADR-0056 Am. 1)                                                                                                                                                       |
| 4   | Raw passthrough (`EncodingRaw`)                                     | 🟢 Fully leveraged | sync protocol raw payload path (ADR-0056); `EncodingRaw` cases pinned by exhaustive linter (gotcha 55)                                                                                                                                                                                                                                                            |
| 5   | Format autodetection (`AutoDetect`)                                 | ⚪ N/A (closed)    | **Stretch row retired** — ADR-0056 Amendment 2 (`docs/adr/0056-frontend-sync-protocol.md:79`, 2026-11-06, fleet-plan A18): upstream's own contract ("NOT a security boundary… never use it to skip encoding validation", upstream `autodetect.go:58-60` — wording re-verified 2026-11-22) forbids wire use; `opaque` is the honest shape; upstream API exists (`autodetect.go:63`), zero prod uses |
| 6   | Transcoding (`TranscodeToJSON`)                                     | 🔴 Missed (low)    | still zero uses; only a future "serve CBOR events as JSON to legacy clients" path needs it                                                                                                                                                                                                                                                                        |
| 7   | Buffer pooling (`EncodePooled`, `Get/PutBuffer`)                    | ⚪ N/A (closed)    | A20 bench closed: pooled median _wins_ (245.8 vs 259.3 ns/op) but **DROP** verdict — no cqrs-htmx-owned call site; ask went upstream; bench stays as receipt (`usermgmt/codec_pool_bench_test.go:62` — line re-verified this pass)                                                                                                                                 |
| 8   | Codec observability (`ObservableCodec`, `WithMetrics`)              | 🔴 Missed (low)    | codec-level metrics would extend the observability demos; no open gate                                                                                                                                                                                                                                                                                            |
| 9   | `Diagnose` (CBOR diagnostic notation)                               | 🟢 Fully leveraged | EDN fallback in the payload inspector (`dashboardui/core/payload.go:195`, fleet-plan A19); test receipt `payload_diagnose_test.go:15` — both lines re-verified this pass                                                                                                                                                                                           |
| 10  | `CBORCompactCodec` / `DeterministicCodec`                           | ⚪ N/A             | compact rejects unknown fields (wrong for event evolution); determinism unneeded — immutable hashes via `HashTag` are already canonical                                                                                                                                                                                                                           |
| 11  | COSE (Sign1/Encrypt0)                                               | ⚪ N/A             | event signing/encryption owned by go-cqrs-lite's storage layer, not the app codec choice                                                                                                                                                                                                                                                                          |

## Score derivation

11 rows − 4 N/A (rows 5, 7, 10, 11 — two closed with receipts, two structurally inapplicable) = **7 applicable**; 5 full ×1 = **5 → 71/100** (band: high by count). Unchanged from 2026-11-14 (44 at 2026-11-05 → 71 at 2026-11-14): this pass's growth (legalparty-model) deepens existing 🟢 rows 1 and 3 — no row flips, so the score carries. The four core rows remain the entire load-bearing codec surface; open misses (Transcode, Observable) are low-value.

## Version currency

v0.3.3 pinned (root, usermgmt, dashboardui, identity-model go.mod) = v0.3.3 latest (verified `git ls-remote --tags origin`, 2026-11-22). **Nothing functional past v0.3.3**: library HEAD `98bb1bc` (2026-11-15) is chore commits; the CHANGELOG's [Unreleased] carries only a repo-level CI lint-stack raise (golangci-lint v2.15.0). v0.3.2..v0.3.3 = 19 commits, headline `d90d20f` "fix(codec): CBOR decode accepts text strings for TextUnmarshaler types" + CI (scheduled self-relative bench gate, per-gate job summaries) and docs chores.

## Should cqrs-htmx use it (more)?

No open stretch rows remain — both 2026-11-05 stretch items closed 2026-11-06 (A18 N/A by contract, A19 adopted, A20 dropped with bench receipt), re-verified this pass. Remaining misses (TranscodeToJSON, ObservableCodec) stay low-value until a concrete gate appears.

## How to verify

```bash
rg -oN 'codec\.[A-Z][A-Za-z0-9]*' --type go -g '!scripts/**' . | sed 's/.*://' | sort | uniq -c | sort -rn
# 37 JSONCodec · 18 EncodingJSON · 13 Encoding · 11 CBORCodec · 7 EncodingCBOR · 5 EncodingRaw · 3 ForEncoding · 3 Codec · 2 EncodePooled · 2 Diagnose
rg -n 'cborEventCodec' --type go usermgmt | grep -v _test   # shared instance (:15) + 5 ADR-0059 encode sites (:50,93,155 credentials · :32 security · :63 bot)
rg -n 'AutoDetect|TranscodeToJSON|ObservableCodec' --type go -g '!scripts/**' -g '!*_test.go' . | wc -l   # expect 0 prod
rg -n 'EncodePooled|Diagnose' --type go -g '!scripts/**' -g '!*_test.go' .   # EncodePooled: none prod (bench-only); Diagnose: dashboardui/core/payload.go (:143 doc, :195 call)
git -C ~/projects/go-codec ls-remote --tags --sort=-v:refname origin | grep -v '\^{}' | head -1   # <sha>	refs/tags/v0.3.3
```

## Prior reports

- 2026-11-14 (prior state of this file): 71/100 · this pass: legalparty-model joins the consumers (JSONCodec ×32→37, ForEncoding ×2→3, 15→17 prod files), every receipt re-trued to the current tree, all closed-verdict rows (AutoDetect/pooling/Diagnose) re-verified — score unchanged, no row flips.
- [ADR-0059](../../adr/0059-cbor-binary-event-payloads.md); [ADR-0056](../../adr/0056-frontend-sync-protocol.md) Amendments 1-2; gotcha 55 (`docs/gotchas.md:225`); finding #2 of the [2026-11-03 go-cqrs-lite deep dive](../../research/2026-11-03_go-cqrs-lite-deep-dive.html); A18/A19/A20 receipts in the [2026-11-06 SUPERB fleet-libs follow-through plan](../../planning/2026-11-06_00-37_SUPERB-fleet-libs-audit-follow-through-pareto-plan.md).
