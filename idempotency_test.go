package cqrshtmx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
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
	if err := store.CheckAndRecord(ctx, "cmd-dup", time.Minute); err != nil {
		t.Fatalf("first CheckAndRecord: %v", err)
	}
	err := store.CheckAndRecord(ctx, "cmd-dup", time.Minute)
	if !errors.Is(err, ErrDuplicateCommand) {
		t.Fatalf("expected ErrDuplicateCommand, got %v", err)
	}
}

func TestCheckAndRecord_AllowsNewID(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	ctx := context.Background()
	if err := store.CheckAndRecord(ctx, "cmd-new", time.Minute); err != nil {
		t.Fatalf("CheckAndRecord: %v", err)
	}
	if err := store.CheckAndRecord(ctx, "cmd-other", time.Minute); err != nil {
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

	// After sweep, Seen must report false — the entry was removed by the
	// background goroutine, not just lazily on read.
	seen, _ := store.Seen(ctx, "will-expire")
	if seen {
		t.Fatal("expected swept entry to not be seen")
	}
}

func TestMemoryIdempotencyStore_CloseIsIdempotent(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	store.Close()
	store.Close() // must not panic
}

func TestMemoryIdempotencyStore_SeenLazyDeletesExpired(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0) // no sweeper
	defer store.Close()

	ctx := context.Background()
	_ = store.Record(ctx, "will-expire", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	// Seen returns false (expired) AND removes the entry lazily.
	seen, _ := store.Seen(ctx, "will-expire")
	if seen {
		t.Fatal("expected expired entry to not be seen")
	}
	// Seen returned false AND lazily deleted the entry: a fresh
	// CheckAndRecord on the same key must now succeed (not ErrDuplicate).
	if err := store.CheckAndRecord(ctx, "will-expire", time.Minute); err != nil {
		t.Fatalf("expected CheckAndRecord to succeed after lazy delete, got %v", err)
	}
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
			_ = store.CheckAndRecord(ctx, "concurrent-cmd", time.Minute)
		}
	}()

	for range 100 {
		_, _ = store.Seen(ctx, "concurrent-cmd")
	}
	<-done
}

// TestCheckAndRecord_ConcurrentSameIDExactlyOneSucceeds is the atomicity proof.
// N goroutines race to claim the same command ID. Because CheckAndRecord holds
// a single write lock for the check+record, exactly one goroutine must win and
// all others must receive ErrDuplicateCommand. Under the old racy free function
// (separate Seen + Record calls), multiple goroutines could slip through.
func TestCheckAndRecord_ConcurrentSameIDExactlyOneSucceeds(t *testing.T) {
	t.Parallel()
	store := NewMemoryIdempotencyStore(0)
	defer store.Close()

	const n = 200
	ctx := context.Background()
	var wg sync.WaitGroup
	var wins atomic.Int64
	var dups atomic.Int64
	start := make(chan struct{})

	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			<-start // release all goroutines at once
			err := store.CheckAndRecord(ctx, "race-cmd", time.Minute)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, ErrDuplicateCommand):
				dups.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", wins.Load())
	}
	if dups.Load() != n-1 {
		t.Fatalf("expected %d duplicates, got %d", n-1, dups.Load())
	}
}
