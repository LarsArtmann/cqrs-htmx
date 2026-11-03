package usermgmt

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// R4: the projection DLQ default is durable when the event store is a
// SQLite-dialect SQL store, memory otherwise. These tests pin the selection
// and the headline claim — poison entries survive a restart.

func TestDurableDeadLetterStore_SQLiteJournal(t *testing.T) {
	ctx := context.Background()
	store, err := NewSQLEventStore(ctx, newSQLTestDB(t), "sqlite")
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	dlq, err := durableDeadLetterStore(ctx, store)
	if err != nil {
		t.Fatalf("durableDeadLetterStore: %v", err)
	}
	if _, ok := dlq.(*projectionhost.SQLiteDeadLetterStore); !ok {
		t.Fatalf("sqlite journal: got %T, want *projectionhost.SQLiteDeadLetterStore", dlq)
	}
}

func TestDurableDeadLetterStore_PostgresDialectKeepsMemory(t *testing.T) {
	// A postgres-dialect store over any DB: the constructor runs no DDL, so
	// building it over a sqlite handle is fine — only the dialect matters.
	store, err := storage.NewSQLEventStoreWithDialect(newSQLTestDB(t), sqlpkg.PostgresDialect{})
	if err != nil {
		t.Fatalf("postgres-dialect store: %v", err)
	}

	dlq, err := durableDeadLetterStore(context.Background(), store)
	if err != nil {
		t.Fatalf("durableDeadLetterStore: %v", err)
	}
	if _, ok := dlq.(*projectionhost.MemoryDeadLetterStore); !ok {
		t.Fatalf("postgres dialect: got %T, want memory store (no dialect-correct SQL DLQ at this pin)", dlq)
	}
}

func TestDurableDeadLetterStore_NonSQLJournalKeepsMemory(t *testing.T) {
	dlq, err := durableDeadLetterStore(context.Background(), memory.NewMemoryStore())
	if err != nil {
		t.Fatalf("durableDeadLetterStore: %v", err)
	}
	if _, ok := dlq.(*projectionhost.MemoryDeadLetterStore); !ok {
		t.Fatalf("non-SQL journal: got %T, want memory store", dlq)
	}
}

// TestSQLiteDeadLetterStore_SurvivesRestart is the R4 headline: an entry
// stored before a "restart" is still listed by a store opened after it.
func TestSQLiteDeadLetterStore_SurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "events.db")

	db1, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db1.SetMaxOpenConns(1)
	if err := OptimizeSQLiteDB(ctx, db1); err != nil {
		t.Fatalf("OptimizeSQLiteDB: %v", err)
	}
	store1, err := NewSQLEventStore(ctx, db1, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLEventStore: %v", err)
	}

	dlq1, err := durableDeadLetterStore(ctx, store1)
	if err != nil {
		t.Fatalf("durableDeadLetterStore: %v", err)
	}

	// A real poison event, as projectionhost delivers to Store in production.
	poison, err := event.New(
		eventUserRegistered, id.NewStreamID(), aggregateTypeUser, 1,
		UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "poison@example.com",
			DisplayName:   "Poison",
			Roles:         []Role{RoleUser},
		},
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}
	entry := projectionhost.DeadLetterEntry{
		ProjectionName: "user-read-model",
		EventID:        poison.ID().String(),
		EventType:      string(poison.Type()),
		StreamID:       poison.StreamID().Get(),
		Event:          poison,
		Error:          "decode failed: boom",
		FailedAt:       time.Now().UTC(),
	}
	if err := dlq1.Store(ctx, entry); err != nil {
		t.Fatalf("Store: %v", err)
	}

	// "Restart": close everything, reopen the same database file.
	if err := store1.Close(); err != nil {
		t.Fatalf("close store1: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close db1: %v", err)
	}

	db2, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	db2.SetMaxOpenConns(1)
	if err := OptimizeSQLiteDB(ctx, db2); err != nil {
		t.Fatalf("OptimizeSQLiteDB: %v", err)
	}
	t.Cleanup(func() { _ = db2.Close() })
	store2, err := NewSQLEventStore(ctx, db2, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLEventStore (restart): %v", err)
	}
	t.Cleanup(func() { _ = store2.Close() })

	dlq2, err := durableDeadLetterStore(ctx, store2)
	if err != nil {
		t.Fatalf("durableDeadLetterStore (restart): %v", err)
	}

	entries, err := dlq2.List(ctx, "user-read-model")
	if err != nil {
		t.Fatalf("List after restart: %v", err)
	}
	if len(entries) != 1 || entries[0].EventID != entry.EventID || entries[0].Error != entry.Error {
		t.Fatalf("after restart: got %+v, want the pre-restart entry", entries)
	}
}
