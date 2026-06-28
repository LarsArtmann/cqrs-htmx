package cqrshtmx

import (
	"context"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// IdempotencyStore tracks command IDs that have been dispatched to prevent
// duplicate processing. When a command ID is seen for the first time, the
// store records it with a TTL. Subsequent dispatches with the same ID are
// rejected as duplicates.
//
// This is essential for offline-first command sync: if a client submits a
// command, loses the ACK, and retries, the store prevents the command from
// executing twice.
//
// Implementations must be safe for concurrent use.
type IdempotencyStore interface {
	// Seen returns true if the command ID has already been recorded.
	Seen(ctx context.Context, commandID string) (bool, error)
	// Record marks a command ID as seen with the given TTL.
	// If the ID is already recorded, it is a no-op.
	Record(ctx context.Context, commandID string, ttl time.Duration) error
}

// ErrDuplicateCommand is returned when a command ID has already been processed.
// Maps to HTTP 409 Conflict via MapError.
var ErrDuplicateCommand = event.NewConflict(
	"cqrshtmx.idempotency.duplicate_command",
	"command with this ID has already been processed",
)

// MemoryIdempotencyStore is an in-memory IdempotencyStore with TTL-based
// expiration. It uses a background goroutine to sweep expired entries.
// Safe for concurrent use.
type MemoryIdempotencyStore struct {
	mu      sync.RWMutex
	entries map[string]time.Time // commandID → expiresAt
	stop    chan struct{}
}

// NewMemoryIdempotencyStore creates an in-memory idempotency store and starts
// a background goroutine that sweeps expired entries every sweepInterval.
// Call Close() to stop the sweeper.
func NewMemoryIdempotencyStore(sweepInterval time.Duration) *MemoryIdempotencyStore {
	s := &MemoryIdempotencyStore{
		mu:      sync.RWMutex{},
		entries: make(map[string]time.Time),
		stop:    make(chan struct{}),
	}
	if sweepInterval > 0 {
		go s.sweep(sweepInterval)
	}
	return s
}

// Seen returns true if the command ID is currently recorded and not expired.
func (s *MemoryIdempotencyStore) Seen(_ context.Context, commandID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.entries[commandID]
	if !ok {
		return false, nil
	}
	return time.Now().Before(exp), nil
}

// Record marks a command ID as seen with the given TTL.
func (s *MemoryIdempotencyStore) Record(_ context.Context, commandID string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[commandID] = time.Now().Add(ttl)
	return nil
}

// Close stops the background sweep goroutine. Safe to call multiple times.
func (s *MemoryIdempotencyStore) Close() {
	select {
	case <-s.stop:
		// Already closed.
	default:
		close(s.stop)
	}
}

func (s *MemoryIdempotencyStore) sweep(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()
			for id, exp := range s.entries {
				if now.After(exp) {
					delete(s.entries, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// CheckAndRecord atomically checks if a command ID has been seen, and if not,
// records it. Returns ErrDuplicateCommand if the ID was already recorded.
// This is the recommended way to use the store — it avoids the TOCTOU race
// between Seen and Record.
func CheckAndRecord(ctx context.Context, store IdempotencyStore, commandID string, ttl time.Duration) error {
	seen, err := store.Seen(ctx, commandID)
	if err != nil {
		return event.Wrapf(err, event.Classify(err),
			"cqrshtmx.idempotency.seen_failed", "check command ID=%s", commandID)
	}
	if seen {
		return ErrDuplicateCommand
	}
	if err := store.Record(ctx, commandID, ttl); err != nil {
		return event.Wrapf(err, event.Classify(err),
			"cqrshtmx.idempotency.record_failed", "record command ID=%s", commandID)
	}
	return nil
}
