package usermgmt

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func TestPlaceholderFor_AllDialects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		dialect string
		want    string
	}{
		{dialectPostgres, "$1"},
		{dialectPgx, "$1"},
		{dialectSQLite, "?"},
		{dialectSQLite3, "?"},
		{dialectMySQL, "?"},
	}
	for _, tc := range cases {
		t.Run(tc.dialect, func(t *testing.T) {
			t.Parallel()

			pf, err := placeholderFor(tc.dialect)
			if err != nil {
				t.Fatalf("placeholderFor(%q): %v", tc.dialect, err)
			}
			if got := pf(1); got != tc.want {
				t.Errorf("placeholder(%q) = %q, want %q", tc.dialect, got, tc.want)
			}
		})
	}

	if _, err := placeholderFor("oracle"); err == nil {
		t.Error("expected error for unsupported dialect")
	}
}

// TestSQLEventStore_OperationsAfterClose exercises the error-return branches of
// every read/write method when the underlying DB is no longer usable.
func TestSQLEventStore_OperationsAfterClose(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	store, err := NewSQLEventStore(context.Background(), db, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}

	// Close the DB so all subsequent operations fail at the driver level.
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	ctx := context.Background()
	ref := event.AggregateRef{ID: id.NewAggregateID(), Type: aggregateTypeUser}

	// Upstream storage.SQLEventStore checks closed-state before the nil-input
	// short-circuit, so even nil inputs return an error on a closed store.
	// This is more defensive than the old hand-rolled store.
	if err := store.Save(ctx, ref, nil, 0); err == nil {
		t.Error("Save(nil) after close: expected error (closed store)")
	}
	if err := store.AppendBatch(ctx, ref, nil); err == nil {
		t.Error("AppendBatch(nil) after close: expected error (closed store)")
	}

	// Real operations must surface errors, not panic.
	if _, err := store.Load(ctx, ref); err == nil {
		t.Error("Load after close: expected error")
	}
	if _, err := store.LoadFromVersion(ctx, ref, 0); err == nil {
		t.Error("LoadFromVersion after close: expected error")
	}
	if _, err := store.LoadToVersion(ctx, ref, 1); err == nil {
		t.Error("LoadToVersion after close: expected error")
	}
	if _, err := store.LoadToTimestamp(ctx, ref, time.Now()); err == nil {
		t.Error("LoadToTimestamp after close: expected error")
	}
	if _, err := store.ReadAll(ctx); err == nil {
		t.Error("ReadAll after close: expected error")
	}
}
