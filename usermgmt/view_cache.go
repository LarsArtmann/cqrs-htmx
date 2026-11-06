package usermgmt

import (
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
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

// newViewCache wires the opt-in FindByID write-through cache. Since
// kv v4.4.0 the upstream cache fronts ANY kv.ViewStore
// ([kv.NewCacheForViewStore]) — this used to be a 182-line hand-rolled
// otter adapter reproducing kv.Cache's contract (ADR-0032) over the
// interface; the seam is now a thin translation:
//
//   - disabled (the default): the raw store passes through untouched.
//   - enabled: kv.NewCacheForViewStore with capacity (and TTL when set).
//     Copy isolation is kv's shallow-copy default — sound for the
//     scalar-shaped view types (strings and bools only); reference-bearing
//     views must not be routed through SQL read models (unchanged posture).
//
// Secondary indexes (FindByEmail, FindByName) are deliberately NOT cached:
// their keys are mutable view columns, and keeping them fresh would require
// dual-key invalidation on every EmailChanged-style event — the exact trap
// the 2026-07-30 ROADMAP evaluation called out.
func newViewCache[V any, K fmt.Stringer](
	store kv.ViewStore[V, K], opts ...ViewCacheOption,
) (kv.ViewStore[V, K], error) {
	cfg := viewCacheConfig{} //nolint:exhaustruct // zero value is the no-cache default; options mutate
	for _, opt := range opts {
		opt(&cfg)
	}

	if !cfg.enabled {
		return store, nil
	}

	kvOpts := []kv.CacheOption[V, K]{kv.WithCacheCapacity[V, K](cfg.capacity)}
	if cfg.ttl > 0 {
		kvOpts = append(kvOpts, kv.WithCacheTTL[V, K](cfg.ttl))
	}

	return kv.NewCacheForViewStore[V, K](store, kvOpts...)
}
