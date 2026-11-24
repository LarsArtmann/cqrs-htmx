package usermgmt

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	_ "modernc.org/sqlite"
)

// The StateShape persistence round-trip contract (TODO row 90, CLOSED):
// upstream go-cqrs-lite#117 shipped in storage/v4 v4.11.2 — snapshot.go now
// carries a state_shape column, INSERTs the stamp, and scans it back on load
// (source-verified at the tag 2026-11-24: lines INSERT 73/87, SELECT 141/155,
// scan 168, return 198). A stamped snapshot survives a SQL round-trip, so the
// decider's shape guard (decider/v4 snapshot_load.go) actually discards a
// stale-shaped snapshot and rebuilds from the journal. This test pins the
// preservation contract permanently: a regression that drops the stamp again
// fails here. (Historical shape: at v4.11.1 the store had no state_shape
// column at all and the sentinel test asserted the drop — it flipped in the
// same change that swept the pin onto v4.11.2.)
func TestSnapshotShape_SQLStorePersistsStamp(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(sqlpkg.SQLiteDialect{}.SnapshotSchema()); err != nil {
		t.Fatalf("create snapshots schema: %v", err)
	}

	store, err := storage.NewSQLSnapshotStoreWithDialect(db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new sql snapshot store: %v", err)
	}

	ref := id.NewStreamRef("user", id.NewStreamID())
	stamped := &snapshot.Snapshot{
		StreamID:   ref.ID,
		StreamType: ref.Type,
		Version:    event.Version(7),
		State:      []byte(`{"email":"a@b.c"}`),
		Encoding:   record.EncodingJSON,
		CreatedAt:  time.Now().UTC(),
		StateShape: snapshotStateShapeVersion,
	}

	ctx := context.Background()
	if err := store.Save(ctx, *stamped); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded == nil {
		t.Fatal("load: snapshot missing after save")
	}

	if loaded.StateShape != snapshotStateShapeVersion {
		t.Fatalf("SQL store dropped StateShape (loaded %q, want %q) — a stamp regression: "+
			"a stale-shaped snapshot would be accepted as legacy instead of discarded", loaded.StateShape, snapshotStateShapeVersion)
	}

	// Control: the in-memory store preserves the stamp today.
	mem := NewMemorySnapshotStore()
	if err := mem.Save(ctx, *stamped); err != nil {
		t.Fatalf("memory save: %v", err)
	}

	memLoaded, err := mem.Load(ctx, ref)
	if err != nil {
		t.Fatalf("memory load: %v", err)
	}

	if memLoaded == nil || memLoaded.StateShape != snapshotStateShapeVersion {
		t.Fatalf("memory store control failed: %+v", memLoaded)
	}
}
