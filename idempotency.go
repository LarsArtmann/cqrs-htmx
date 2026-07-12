package cqrshtmx

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/idempotency/v4"
)

// Delegated to go-cqrs-lite/idempotency/v3 (tagged v3.3.0).
// These aliases preserve backward compatibility with existing cqrs-htmx consumers.
// New code should import the upstream module directly.

// IdempotencyStore tracks command IDs that have been dispatched to prevent
// duplicate processing.
type IdempotencyStore = idempotency.Store

// MemoryIdempotencyStore is an in-memory IdempotencyStore with TTL-based
// expiration and a background sweep goroutine.
type MemoryIdempotencyStore = idempotency.MemoryStore

// ErrDuplicateCommand is returned when a command ID has already been processed.
// Maps to HTTP 409 Conflict via MapError.
var ErrDuplicateCommand = idempotency.ErrDuplicate

// NewMemoryIdempotencyStore creates an in-memory idempotency store and starts
// a background goroutine that sweeps expired entries every sweepInterval.
// Call Close() to stop the sweeper.
func NewMemoryIdempotencyStore(sweepInterval time.Duration) *MemoryIdempotencyStore {
	return idempotency.NewMemoryStore(sweepInterval)
}
