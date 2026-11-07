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

// The StateShape persistence round-trip audit (TODO row 90): the in-memory
// store deep-copies the stamp (0d473d99), but the SQL snapshot store at
// storage/v4 v4.11.1 has NO state_shape column, no INSERT field, and no scan
// target — a stamped snapshot silently loses its stamp across a SQL
// round-trip. The decider treats an empty stamp as legacy and ACCEPTS it
// (decider/v4 snapshot_load.go: an empty snapshot stamp never mismatches), so
// the drop only bites when a shape version bumps: the old-shaped snapshot
// should be discarded and rebuilt from the journal, but is accepted instead.
// Upstream ask drafted in docs/drafts/; this test pins the CURRENT behavior
// and must FLIP (assert preservation) the moment upstream persists the stamp.
func TestSnapshotShape_SQLStoreDropsStampToday(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	defer db.Close()

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

	if loaded.StateShape != "" {
		t.Fatalf("SQL store now round-trips StateShape (%q) — upstream shipped the fix; "+
			"flip this test to assert preservation and close TODO row 90", loaded.StateShape)
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
