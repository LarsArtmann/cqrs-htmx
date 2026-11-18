## Problem

`MarshalSignals` marshals map-backed signal values through `encoding/json/v2`, and jsonv2 does NOT sort map keys — the emitted bytes are non-deterministic across runs of the same map. Every byte-exact consumer of a signals patch (golden tests, replay caches, `Last-Event-ID`-style diffing) flakes on multi-key patches.

Verified at master `f909928`:

- `signals.go:82-90` (`MarshalSignals`): `json.Marshal(v)` on `encoding/json/v2`, no ordering control.
- `response.go:194` (`sendSignalsMap`): builds the patch from `map[string]any`, so the error-path senders (`response.go:208`, `response.go:244`) inherit the non-determinism.

Empirical probe (Go `encoding/json/v2`, 200 iterations, same 2-key map):

```
179 x {"count":1,"name":"x"}
 21 x {"name":"x","count":1}
```

~10% of runs emit a different byte string. Downstream this produced intermittent test failures in a consumer's SSE assertions (multi-key `RequireSignals` byte pins) — the failure moves with map iteration order, so it reproduces under load and disappears in filtered reruns.

## Proposal

Sort map keys at marshal time inside `MarshalSignals` (jsonv2 accepts ordered input via `slices.Sorted`/an ordered map implementation, or a small `MarshalOrdered` helper for the map path). Struct-backed signals are unaffected either way — field order is struct order today and stays deterministic.

Consumer-side byte pins stay honest: the wire bytes become a stable contract instead of one of N random orderings.

💘 Generated with Crush
