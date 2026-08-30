package usermgmt

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	_ "modernc.org/sqlite"
)

// TestNewService_SQLReadModels_SurviveRestart is the service-level persistence
// contract for the documented production config (CheckpointStore +
// ReadModelDB + ReadModelDialect): a user registered on one service instance
// is readable from a fresh instance built on the same database. The
// checkpointed restart relies on the SQL hydrate path — the checkpoint says
// "journal fully applied", so nothing replays and only hydration can
// repopulate the read models.
func TestNewService_SQLReadModels_SurviveRestart(t *testing.T) {
	ctx := t.Context()
	dbFile := filepath.Join(t.TempDir(), "readmodels.sqlite")

	openDB := func(t *testing.T) *sql.DB {
		t.Helper()
		db, err := sql.Open("sqlite", "file:"+dbFile)
		if err != nil {
			t.Fatalf("open sqlite: %v", err)
		}
		db.SetMaxOpenConns(1)
		// The checkpoint store does not self-migrate; apply its schema the
		// way a deployment init script would.
		if _, err := db.Exec(storage.SQLiteCheckpointSchema()); err != nil {
			t.Fatalf("migrate checkpoints table: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })
		return db
	}

	// Service #1: register a user and close. Synchronous projection drain
	// (the default) guarantees the read model row + checkpoint are committed
	// before Close.
	db1 := openDB(t)
	cp1, err := storage.NewSQLiteCheckpointStore(db1)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore #1: %v", err)
	}
	svc1, err := NewService(ServiceConfig{
		CheckpointStore:  cp1,
		ReadModelDB:      db1,
		ReadModelDialect: "sqlite",
	})
	if err != nil {
		t.Fatalf("NewService #1: %v", err)
	}

	reg, err := svc1.Register(ctx, RegisterRequest{
		Email: "persist@example.com", DisplayName: "Persistence",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc1.Close(); err != nil {
		t.Fatalf("Close #1: %v", err)
	}
	_ = db1.Close()

	// Service #2 on the SAME database: the user must already be there via
	// hydration (each service has its own in-memory journal; only the SQL
	// state carries over).
	db2 := openDB(t)
	cp2, err := storage.NewSQLiteCheckpointStore(db2)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore #2: %v", err)
	}
	svc2, err := NewService(ServiceConfig{
		CheckpointStore:  cp2,
		ReadModelDB:      db2,
		ReadModelDialect: "sqlite",
	})
	if err != nil {
		t.Fatalf("NewService #2: %v", err)
	}
	t.Cleanup(func() { _ = svc2.Close() })

	user, err := svc2.GetUser(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("GetUser after restart: %v", err)
	}
	if user.Email != "persist@example.com" {
		t.Errorf("email after restart = %q, want persist@example.com", user.Email)
	}
}
