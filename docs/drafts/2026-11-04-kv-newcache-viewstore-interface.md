> [!NOTE]
> This filing was drafted by GLM-4.6 via Crush from an AI-run investigation, not at my request. When the failure traces to an external report, that source is linked in the body.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

**TL;DR:** `kv.NewCache` only accepts a concrete `*TypedStore`, so every `kv.ViewStore` implementation — notably `storage/view.SQLViewStore`, the real-column read-model store — cannot be cached at all, and the documented "wrap the view store in kv.NewCache" guidance does not type-check. Asking for an interface-level cache constructor.

## Problem

`kv.NewCache` is concrete-typed (`kv/cache.go:80` at `kv/v4@v4.3.5`, identical on master @ `6e117971a`):

```go
func NewCache[T any, K fmt.Stringer](
	store *TypedStore[T, K],
	opts ...CacheOption[T, K],
) (*Cache[T, K], error)
```

`SQLViewStore` is a different concrete type (real columns per view, `Query`/`Count`/`BatchSet`), so there is no way to front it with `kv.Cache`. In cqrs-htmx this forced a hand-rolled adapter reproducing the ADR-0032 contract directly over otter (`usermgmt/view_cache.go`), because the ROADMAP's own guidance — "wrap `SQLUserReadModel`'s view store in `kv.NewCache`" (2026-07-30 evaluation) — cannot compile as written.

## Why the constructor is concrete

The copy isolation is the coupling: `Cache.copy` (`kv/cache.go:222` at v4.3.5) deep-copies via `cs.store.codec.Encode`/`Decode`, and the codec lives on `TypedStore` — unavailable through the interface. Also `Cache.Has` (`cache.go:144`) calls `store.Has`, which `ViewStore` (`kv/view_store.go:18-23`) does not declare (Get/Set/Delete/Scan only).

## Proposal

`NewCacheForViewStore[V, K](store kv.ViewStore[V, K], opts ...)` with one of:

- a documented shallow-copy contract (safe for the scalar-shaped views SQL read models actually use — strings/bools), or
- a `WithCopyFunc(func(*V) *V)` option for reference-bearing views.

`Has` over the interface can fall back to Get (or the method set is trimmed for the view variant). Everything else — write-through Set/Delete, no negative caching, TinyLFU capacity, TTL — is interface-compatible as-is. cqrs-htmx would delete its adapter and consume upstream on the next train.

💘 Generated with Crush
