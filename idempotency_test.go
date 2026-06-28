package cqrshtmx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

func TestMemoryIdempotencyStore_SeenReturnsFalseForNewID(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	seen, err := store.Seen(context.Background(), "cmd-1")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if seen {
		t.Fatal("expected new ID to not be seen")
	}
}

func TestMemoryIdempotencyStore_SeenReturnsTrueAfterRecord(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := store.Record(ctx, "cmd-1", time.Minute); err != nil {
		t.Fatalf("Record: %v", err)
	}
	seen, err := store.Seen(ctx, "cmd-1")
	if err != nil {
		t.Fatalf("Seen: %v", err)
	}
	if !seen {
		t.Fatal("expected recorded ID to be seen")
	}
}

func TestMemoryIdempotencyStore_ExpiredEntriesAreNotSeen(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := store.Record(ctx, "cmd-expired", 1*time.Millisecond); err != nil {
		t.Fatalf("Record: %v", err)
	}
	time.Sleep(5 * time.Millisecond)

	seen, _ := store.Seen(ctx, "cmd-expired")
	if seen {
		t.Fatal("expected expired entry to not be seen")
	}
}

func TestCheckAndRecord_RejectsDuplicate(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := CheckAndRecord(ctx, store, "cmd-dup", time.Minute); err != nil {
		t.Fatalf("first CheckAndRecord: %v", err)
	}
	err := CheckAndRecord(ctx, store, "cmd-dup", time.Minute)
	if !errors.Is(err, ErrDuplicateCommand) {
		t.Fatalf("expected ErrDuplicateCommand, got %v", err)
	}
}

func TestCheckAndRecord_AllowsNewID(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := CheckAndRecord(ctx, store, "cmd-new", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord: %v", err)
	}
	if err := CheckAndRecord(ctx, store, "cmd-other", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord for different ID: %v", err)
	}
}

func TestErrDuplicateCommand_HasConflictFamily(t *testing.T) {
	t.Parallel()
	if event.Classify(ErrDuplicateCommand) != event.Conflict {
		t.Fatalf("expected Conflict family, got %v", event.Classify(ErrDuplicateCommand))
	}
}

func TestMemoryIdempotencyStore_SweepRemovesExpiredEntries(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(10 * time.Millisecond)
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "will-expire", 1*time.Millisecond)
	time.Sleep(50 * time.Millisecond) // sweeper runs at 10ms intervals

	store.mu.RLock()
	count := len(store.entries)
	store.mu.RUnlock()
	if count > 0 {
		t.Fatalf("expected 0 entries after sweep, got %d", count)
	}
}

func TestMemoryIdempotencyStore_CloseIsIdempotent(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	store.Close()
	store.Close() // must not panic
}

func TestMemoryIdempotencyStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for range 100 {
			_ = CheckAndRecord(ctx, store, "concurrent-cmd", time.Minute)
		}
	}()

	for range 100 {
		_, _ = store.Seen(ctx, "concurrent-cmd")
	}
	<-done
}
