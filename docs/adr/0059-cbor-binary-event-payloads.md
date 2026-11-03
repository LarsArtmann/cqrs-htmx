# ADR 0059: CBOR for Binary Event Payloads

**Status:** Accepted
**Date:** 2026-11-03
**Related:** [ADR 0056](0056-frontend-sync-protocol.md) (sync wire contract, honest encoding stamps), [2026-11-03 utilization audit](../research/2026-11-03_go-cqrs-lite-deep-dive.html) (missed opportunity #12)

## Context

Every `usermgmt` event is created with an explicit `event.WithCodec(codec.JSONCodec{})`
(adjudicated under cqrs-lint A027: one codec per call site, no global `DefaultCodec`
coupling). JSON marshals `[]byte` fields as base64: +33% bytes plus a base64
encode/decode pass on every write and read. Five payloads are dominated by such
fields:

| Payload | Binary fields | Typical bytes |
|---|---|---|
| `CredentialAdded` | `ID`, `PublicKey`, `AAGUID` | 32 + ~77 + 16 |
| `CredentialRemoved` | `ID` | 32–1024 |
| `CredentialUsed` | `CredentialID` | 32–1024 |
| `TOTPEnabled` | `Secret` | 20–64 |
| `BotRegistered` | `TokenHash` | 32 |

(The audit originally named four payloads — a line-scan of `identity-model/events.go`
missed `CredentialAdded`, whose `[]byte` fields arrive via the embedded
`CredentialCore` from `credential.go`. Corrected at planning time; see the plan's
errata.)

Everything the flip touches is already encoding-aware:

- **Decode**: every fold, read model, and projection decodes via
  `identitymodel.UnmarshalPayload`, which resolves the codec from the event's
  own encoding stamp (`codec.ForEncoding(evt.Encoding())`).
- **Storage**: the SQL event store persists `payload_encoding` per row
  (upstream `storage/v4` schema); events are self-describing, so mixed
  JSON+CBOR journals are valid and need no migration.
- **SSE**: `transport.DomainEventToSSE` streams a metadata-only envelope —
  payload bytes never cross it.
- **Sync pull (ADR-0056)**: non-JSON stamps are reported as-is and delivered
  base64 (`payloadEncoding: "cbor"` + `payloadB64`); the OpenAPI description
  and the frontend-sync guide already document the codec-name case.

## Decision

The five binary payloads above are emitted **CBOR** (`event.WithCodec(codec.CBORCodec{})`);
every other event keeps its explicit JSON codec. The choice is per event type and
stamped per event — no global `DefaultCodec` mutation, no migration for existing
journals, and flipping back is a one-line change per site.

JSON stays the codec for all text-dominated payloads because JSON journals stay
human-debuggable and sync-deliverable inline (`payloadEncoding: "json"`), which
CBOR events cannot promise.

## Consequences

- Storage and wire size for the five heaviest payloads drops ~35–45% (asserted in
  `TestCBORPayloadSizeWin_TOTPEnabled` as the durable, measurable claim).
- Mixed journals are now real: readers must keep decoding via encoding stamps
  (`UnmarshalPayload`/`DecodePayloadAuto`), never a hard-coded codec. All shipped
  readers already do.
- Sync clients that pull these five events receive `payloadEncoding: "cbor"` with
  `payloadB64` instead of inline JSON. That is the honest degradation the wire
  contract documents; these payloads are server-side secret material (credential
  IDs, TOTP secrets, token hashes), so browser-side JSON decode was never a
  supported scenario.
- Future CBOR candidates must carry the time.Time caveat: JSON encodes time as
  RFC3339 strings, CBOR as tagged numbers — consumers decoding with the wrong
  codec assumptions would corrupt timestamps. None of the five payloads here
  carries a time field (envelope `occurred_at` is metadata, not payload).
- The five events remain on schema version 1: encoding is not schema.
