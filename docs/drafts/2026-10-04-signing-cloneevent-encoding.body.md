> [!NOTE]
> This filing was drafted by GLM via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

TL;DR: `signing/v4.3.2`'s `CloneEvent` migrated from `event.NewEvent` to `event.New` without carrying `WithEncoding` over, so every signed clone gets auto-stamped with the default CBOR encoding while its payload bytes stay whatever the producer wrote — consumers decoding JSON-labeled-as-CBOR fail, and signed events silently never reach read models. Ask: preserve `evt.Encoding()` in `CloneEvent`, add a regression test, tag `signing/v4.3.3`; cqrs-htmx re-pins after that.

## Symptom

cqrs-htmx `integration_test` battery (published pins, `GOWORK=off`, Go 1.27.1, `GOEXPERIMENT=jsonv2`), 6/6 deterministic failures — exactly the 3 signing-enabled tests:

```
--- FAIL: TestSigningEncryption_StoreEncryptionAndBusSigning (0.05s)
--- FAIL: TestSigningEncryption_BusLevelCrypto (0.05s)
--- FAIL: TestSigningEncryption_Ed25519AsymmetricSigning (0.04s)
    Register: [transient:usermgmt.user.read_model_missing] user not in read model after register
```

Failing instantly (not a 30s drain timeout): the projection consumes nothing. With cqrs-htmx's consumer wave reverted to `signing/v4.3.1` (isolated by bump-bisection over the 43-module wave), the same battery is green.

## Root cause (source)

The chain, every link read at tag level:

1. Producers stamp real encodings — cqrs-htmx usermgmt builds all domain events with `event.WithCodec(codec.JSONCodec{})` (`usermgmt/es_decide.go:48`): JSON bytes, `encoding=JSON`.
2. `signing/v4.3.2` `CloneEvent` (`signing/event.go:26`) migrated `event.NewEvent` → `event.New` and passes `PayloadReadOnly` + `WithEventID/WithOccurredAt/WithSchemaVersion/WithMetadata/WithCustom` — but **no `WithEncoding`/`WithCodec`** (`git diff signing/v4.3.1 signing/v4.3.2 -- signing/event.go`).
3. `event.New` (`event/event_new.go:63-70` at `event/v4.13.0`) probes the opts for a codec, finds none, falls back to `DefaultCodec` (CBOR), and stamps `evt.encoding = c.Encoding()` whenever the clone carries no explicit encoding.
4. Result: the clone's payload is still JSON bytes but now labeled `encoding=CBOR`. The store/projection side trusts the stamp, CBOR-decodes JSON, the fold never applies the event, and the read model misses the user.

`signing/v4.3.1` used `event.NewEvent(... []byte)`, which never touches encoding (clone carried `encoding=""` and consumers decoded with their own codec) — so this is new in v4.3.2, riding a wave whose CHANGELOG section for these tags says "go-directive minor-form floor" only. The API migration rode along unlabeled, which is also why consumer-side bisection initially chased `scheduling/v4.5.1` (its main module is code-identical to v4.5.0; only submodules changed).

## Fix

- [ ] `CloneEvent` (`signing/event.go`): add `event.WithEncoding(evt.Encoding())` to the `event.New` call — the `event.New` doc comment names exactly this reconstruction case ("essential when reconstructing events from a wire format or storage where the payload bytes and encoding are already known").
- [ ] Regression test: clone a `WithCodec(JSONCodec{})` event and assert `clone.Encoding() == original.Encoding()` (fails today: JSON in, CBOR out).
- [ ] Tag `signing/v4.3.3`; cqrs-htmx re-pins and its battery goes back to green.

## To verify

`cd integration_test && GOWORK=off go test ./... -count=1 -run 'TestSigningEncryption'` green with `signing@v4.3.3`; full battery green; `cqrs-htmx` CI `Test integration_test submodule` step green on the re-pin push.

💘 Generated with Crush

Assisted-by: Crush:glm-5.3-flash
