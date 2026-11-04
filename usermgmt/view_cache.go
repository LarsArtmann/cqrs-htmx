package usermgmt

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/maypok86/otter/v2"
)

// ViewCacheConfig is the ServiceConfig-level shape of the opt-in FindByID
// write-through cache for SQL-backed read models. Nil (the default) keeps
// every read on the SQL store — zero behavior change.
type ViewCacheConfig struct {
	// Capacity is the maximum number of cached views before TinyLFU
	// eviction. Must be positive when the cache is enabled.
	Capacity int

	// TTL bounds staleness for multi-writer topologies: another process
	// writing the same database bypasses this cache (no cross-instance
	// invalidation exists), so entries older than TTL stop being served.
	// Zero means entries live until evicted by capacity — only sound when
	// a single process owns the database.
	TTL time.Duration
}

// ViewCacheOption configures the opt-in FindByID write-through cache on the
// SQL read-model constructors. The default (no options) is a disabled cache.
type ViewCacheOption func(*viewCacheConfig)

type viewCacheConfig struct {
	enabled  bool
	capacity int
	ttl      time.Duration
}

// WithViewCacheCapacity enables the FindByID write-through cache with the
// given maximum number of entries (TinyLFU admission, lock-free reads).
func WithViewCacheCapacity(n int) ViewCacheOption {
	return func(c *viewCacheConfig) { c.enabled, c.capacity = true, n }
}

// WithViewCacheTTL bounds how long a cached view keeps being served. Only
// meaningful together with [WithViewCacheCapacity]. See [ViewCacheConfig.TTL]
// for the multi-writer topology this protects against.
func WithViewCacheTTL(d time.Duration) ViewCacheOption {
	return func(c *viewCacheConfig) { c.ttl = d }
}

// viewCacheOptions translates the ServiceConfig-level struct into constructor
// options; a nil config means "disabled" (nil options).
func viewCacheOptions(cfg *ViewCacheConfig) []ViewCacheOption {
	if cfg == nil {
		return nil
	}

	opts := []ViewCacheOption{WithViewCacheCapacity(cfg.Capacity)}
	if cfg.TTL > 0 {
		opts = append(opts, WithViewCacheTTL(cfg.TTL))
	}

	return opts
}

// viewCache is an opt-in, FindByID-only write-through cache in front of a
// [kv.ViewStore] — the interface *storage.SQLViewStore satisfies, so the seam
// is v5-neutral: it survives the ADR-0123 metaengine migration unchanged.
//
// The upstream [kv.Cache] cannot serve this seam: it wraps the concrete
// *kv.TypedStore (opaque KV blobs), not the real-column view stores the SQL
// read models are built on. This adapter reproduces kv.Cache's contract
// (ADR-0032) over the interface, on the same backing library (otter):
//
//   - Get serves cache hits as shallow copies; misses delegate to the store
//     and cache the result. Negative results (kv.ErrNotFound) are NOT cached.
//   - Set and Delete are write-through — store first, then cache — so the
//     projection sync path keeps the cache exact under the single-writer
//     assumption (all writes flow through the owning read model).
//   - A nil inner cache means "disabled": every method is a pure passthrough.
//
// Values are shallow-copied on every Get because the cached view types
// (UserView and friends) are scalar-shaped — strings and bools only, no
// slices or maps — so a shallow copy IS a deep copy. Do not route
// reference-bearing view types through this adapter.
//
// Secondary indexes (FindByEmail, FindByName) are deliberately NOT cached:
// their keys are mutable view columns, and keeping them fresh would require
// dual-key invalidation on every EmailChanged-style event — the exact trap
// the 2026-07-30 ROADMAP evaluation called out.
type viewCache[V any, K fmt.Stringer] struct {
	store kv.ViewStore[V, K]
	cache *otter.Cache[string, *V]
}

// newViewCache wraps store with the configured cache. A nil cache (no
// enabling option) keeps every operation on the store itself.
func newViewCache[V any, K fmt.Stringer](
	store kv.ViewStore[V, K], opts ...ViewCacheOption,
) (*viewCache[V, K], error) {
	cfg := viewCacheConfig{} //nolint:exhaustruct // zero value is the no-cache default; options mutate
	for _, opt := range opts {
		opt(&cfg)
	}

	adapter := &viewCache[V, K]{store: store} //nolint:exhaustruct // disabled until an option enables it
	if !cfg.enabled {
		return adapter, nil
	}

	if cfg.capacity <= 0 {
		return nil, errorfamily.NewRejection(
			"usermgmt.view_cache.invalid_capacity",
			fmt.Sprintf("view cache capacity must be positive when enabled, got %d", cfg.capacity),
		)
	}

	otterOpts := &otter.Options[string, *V]{ //nolint:exhaustruct // MaximumSize is the only required knob; match the linter name enabled in .golangci.yml (pre-exhaustruct_v5 migration)
		MaximumSize: cfg.capacity,
	}

	if cfg.ttl > 0 {
		otterOpts.ExpiryCalculator = otter.ExpiryWriting[string, *V](cfg.ttl)
	}

	adapter.cache = otter.Must(otterOpts)

	return adapter, nil
}

// Get returns the view for id. Cache hits return a shallow copy the caller
// owns; misses delegate to the store and cache the result — except misses
// that error (kv.ErrNotFound and friends), which never enter the cache.
func (c *viewCache[V, K]) Get(ctx context.Context, id K) (*V, error) {
	if c.cache == nil {
		return c.store.Get(ctx, id) //nolint:wrapcheck // call sites attach the read-model error codes
	}

	if cached, ok := c.cache.GetIfPresent(id.String()); ok {
		view := *cached

		return &view, nil
	}

	view, err := c.store.Get(ctx, id)
	if err != nil {
		return nil, err //nolint:wrapcheck // call sites attach the read-model error codes
	}

	c.cache.Set(id.String(), view)

	return view, nil
}

// Set writes through: the store first, then the cache. A store error leaves
// the cache untouched (never cache a write that did not land).
func (c *viewCache[V, K]) Set(ctx context.Context, id K, view *V) error {
	if err := c.store.Set(ctx, id, view); err != nil {
		return err //nolint:wrapcheck // call sites attach the read-model error codes
	}

	if c.cache != nil {
		owned := *view
		c.cache.Set(id.String(), &owned)
	}

	return nil
}

// Delete writes through: the store first, then the cache eviction.
func (c *viewCache[V, K]) Delete(ctx context.Context, id K) error {
	if err := c.store.Delete(ctx, id); err != nil {
		return err //nolint:wrapcheck // call sites attach the read-model error codes
	}

	if c.cache != nil {
		c.cache.Invalidate(id.String())
	}

	return nil
}
