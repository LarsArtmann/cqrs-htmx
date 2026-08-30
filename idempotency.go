package cqrshtmx

import (
	"time"

	"github.com/larsartmann/go-idempotency"
)

// Delegated to go-idempotency.
// These aliases preserve backward compatibility with existing cqrs-htmx consumers.
// New code should import the upstream module directly.

// IdempotencyStore tracks command IDs that have been dispatched to prevent
// duplicate processing.
type IdempotencyStore = idempotency.Store

// MemoryIdempotencyStore is an in-memory IdempotencyStore with TTL-based
// expiration and a background sweep goroutine.
//
//nolint:staticcheck // SA1019: deprecated upstream but kept as the documented in-memory default (library principle: never force persistence); removal rides the v5 re-export cleanup
type MemoryIdempotencyStore = idempotency.MemoryStore

// ErrDuplicateCommand is returned when a command ID has already been processed.
// Maps to HTTP 409 Conflict via MapError.
var ErrDuplicateCommand = idempotency.ErrDuplicate

// NewMemoryIdempotencyStore creates an in-memory idempotency store and starts
// a background goroutine that sweeps expired entries every sweepInterval.
// Call Close() to stop the sweeper.
func NewMemoryIdempotencyStore(sweepInterval time.Duration) *MemoryIdempotencyStore {
	//cqrs-lint:ignore(C026) sweepInterval is a sweeper cadence, not an entry TTL; the idempotency store manages its own TTL internally
	//nolint:staticcheck // SA1019: the in-memory store is the documented library default (library principle: never force persistence); production consumers inject their own store
	return idempotency.NewMemoryStore(sweepInterval)
}
