package usermgmt

import (
	"context"
	"database/sql"
	"testing"
)

// newSQLTestDB creates an in-memory SQLite database suitable for SQL read model tests.
// Uses shared cache + single connection so all operations see the same data.
func newSQLTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := OptimizeSQLiteDB(context.Background(), db); err != nil {
		t.Fatalf("OptimizeSQLiteDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// --- SQLUserReadModel ---

func TestSQLUserReadModel_RegisterAndQuery(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteUserReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel: %v", err)
	}

	// makeEvent uses the package-level testAggID for all events.
	evt := makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "alice@example.com",
		DisplayName:   "Alice",
		Roles:         []Role{RoleUser},
	})

	// Feed through the SQL read model's Handle.
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	// Verify in-memory read model has the user.
	user, ok := rm.FindByID(testAggID)
	if !ok {
		t.Fatal("FindByID: user not found in in-memory model")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", user.Email)
	}

	// Verify SQL store has the user via FindByIDSQL.
	userID := NewUserID(testAggID.String())
	view, err := rm.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("FindByIDSQL: %v", err)
	}
	if view.Email != "alice@example.com" {
		t.Errorf("SQL email = %q, want alice@example.com", view.Email)
	}
	if view.DisplayName != "Alice" {
		t.Errorf("SQL display_name = %q, want Alice", view.DisplayName)
	}
	if view.Tombstoned {
		t.Error("SQL tombstoned = true, want false")
	}

	// FindByEmailSQL.
	byEmail, err := rm.FindByEmailSQL(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("FindByEmailSQL: %v", err)
	}
	if len(byEmail) != 1 {
		t.Fatalf("FindByEmailSQL: got %d results, want 1", len(byEmail))
	}
	if byEmail[0].Email != "alice@example.com" {
		t.Errorf("FindByEmailSQL email = %q", byEmail[0].Email)
	}

	// CountSQL — one non-tombstoned user.
	count, err := rm.CountSQL(ctx)
	if err != nil {
		t.Fatalf("CountSQL: %v", err)
	}
	if count != 1 {
		t.Errorf("CountSQL = %d, want 1", count)
	}
}

func TestSQLUserReadModel_UpdateAndDelete(t *testing.T) {
	ctx := t.Context()
	db := newSQLTestDB(t)
	rm, err := NewSQLiteUserReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteUserReadModel: %v", err)
	}

	// Register.
	registered := makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "bob@example.com",
		DisplayName:   "Bob",
		Roles:         []Role{RoleUser},
	})
	if err := rm.Handle(ctx, registered); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}

	// Change email.
	emailChanged := makeEvent(t, eventEmailChanged, 2, EmailChangedPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "bob.new@example.com",
	})
	if err := rm.Handle(ctx, emailChanged); err != nil {
		t.Fatalf("Handle EmailChanged: %v", err)
	}

	// Verify old email is gone from SQL, new email is present.
	userID := NewUserID(testAggID.String())
	view, err := rm.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("FindByIDSQL after email change: %v", err)
	}
	if view.Email != "bob.new@example.com" {
		t.Errorf("email = %q, want bob.new@example.com", view.Email)
	}

	oldEmail, err := rm.FindByEmailSQL(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("FindByEmailSQL old: %v", err)
	}
	if len(oldEmail) != 0 {
		t.Errorf("old email still in SQL: got %d results", len(oldEmail))
	}

	newEmail, err := rm.FindByEmailSQL(ctx, "bob.new@example.com")
	if err != nil {
		t.Fatalf("FindByEmailSQL new: %v", err)
	}
	if len(newEmail) != 1 {
		t.Errorf("new email: got %d results, want 1", len(newEmail))
	}

	// Verify email.
	verified := makeEvent(t, eventEmailVerified, 3, EmailVerifiedPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "bob.new@example.com",
	})
	if err := rm.Handle(ctx, verified); err != nil {
		t.Fatalf("Handle EmailVerified: %v", err)
	}
	view, err = rm.FindByIDSQL(ctx, userID)
	if err != nil {
		t.Fatalf("FindByIDSQL after verify: %v", err)
	}
	if !view.EmailVerified {
		t.Error("email_verified = false, want true")
	}

	// Delete user.
	deleted := makeEvent(t, eventUserDeleted, 4, UserDeletedPayload{
		SchemaVersion: currentSchemaVersion,
		Reason:        "test",
	})
	if err := rm.Handle(ctx, deleted); err != nil {
		t.Fatalf("Handle UserDeleted: %v", err)
	}

	// SQL store should no longer have the row.
	_, err = rm.FindByIDSQL(ctx, userID)
	if err == nil {
		t.Error("FindByIDSQL after delete: expected error, got nil")
	}

	// Count should be 0.
	count, err := rm.CountSQL(ctx)
	if err != nil {
		t.Fatalf("CountSQL after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("CountSQL after delete = %d, want 0", count)
	}
}

func TestNewSQLUserReadModel_PostgresConstructor(t *testing.T) {
	// The Postgres constructor uses a different dialect but we can't test
	// without a live Postgres. Verify it at least accepts a DB and fails
	// gracefully on schema creation with SQLite (or succeeds — depends on
	// whether the DDL is portable). We just verify no panic.
	db := newSQLTestDB(t)
	_, _ = NewSQLUserReadModel(db)
}
