package usermgmt

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// newSQLiteTestDispatcher wires all four aggregate command sets to a dispatcher
// using the SQLiteEventSourcedSetup's repositories.
func newSQLiteTestDispatcher(t *testing.T, s *SQLiteEventSourcedSetup) *command.Dispatcher {
	t.Helper()
	disp := command.NewDispatcher()
	if err := RegisterCommands(disp, s.UserRepository); err != nil {
		t.Fatalf("RegisterCommands: %v", err)
	}
	if err := RegisterMembershipCommands(disp, s.MembershipRepository); err != nil {
		t.Fatalf("RegisterMembershipCommands: %v", err)
	}
	if err := RegisterTenantCommands(disp, s.TenantRepository); err != nil {
		t.Fatalf("RegisterTenantCommands: %v", err)
	}
	if err := RegisterBotCommands(disp, s.BotRepository); err != nil {
		t.Fatalf("RegisterBotCommands: %v", err)
	}
	return disp
}

func TestSQLiteSetup_CreateAndClose(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") + "?cache=shared"
	setup, err := NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: dsn})
	if err != nil {
		t.Fatalf("NewSQLiteEventSourcedSetup: %v", err)
	}
	if setup.DB == nil {
		t.Error("DB is nil")
	}
	if setup.Authz() == nil {
		t.Error("Authz() is nil")
	}
	if setup.ReadModel == nil {
		t.Error("ReadModel is nil")
	}
	if setup.UserRepository == nil {
		t.Error("UserRepository is nil")
	}

	// Close should succeed.
	if err := setup.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Close is idempotent.
	if err := setup.Close(); err != nil {
		t.Fatalf("Close (2nd): %v", err)
	}
}

func TestSQLiteSetup_GracefulClose(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") + "?cache=shared"
	setup, err := NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: dsn})
	if err != nil {
		t.Fatalf("NewSQLiteEventSourcedSetup: %v", err)
	}
	ctx := context.Background()
	if err := setup.GracefulClose(ctx); err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}
}

func TestCreatePostgresReadModels_NilDB(t *testing.T) {
	// When db is nil, should return in-memory read models (fallback path).
	userRm, memRm, tenRm, botRm, err := createPostgresReadModels(nil)
	if err != nil {
		t.Fatalf("createPostgresReadModels(nil): %v", err)
	}
	if userRm == nil || memRm == nil || tenRm == nil || botRm == nil {
		t.Error("expected non-nil read models for nil db fallback")
	}
}

func TestSQLiteSetup_GracefulClose_CancelledContext(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") + "?cache=shared"
	setup, err := NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: dsn})
	if err != nil {
		t.Fatalf("NewSQLiteEventSourcedSetup: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// GracefulClose with a cancelled context should still close (or return ctx.Err).
	_ = setup.GracefulClose(ctx)
}

func TestSQLiteSetup_RegisterUserThroughStack(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "test.db") + "?cache=shared"
	setup, err := NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: dsn})
	if err != nil {
		t.Fatalf("NewSQLiteEventSourcedSetup: %v", err)
	}
	defer func() { _ = setup.Close() }()

	disp := newSQLiteTestDispatcher(t, setup)
	ctx := context.Background()
	aggID := id.NewAggregateID()
	dispatchRegisterUser(t, disp, ctx, aggID, "alice@example.com", "Alice")

	// Verify in-memory read model (rebuilt via projection from events).
	sqlRM, ok := setup.ReadModel.(*SQLUserReadModel)
	if !ok {
		t.Fatalf("ReadModel is %T, want *SQLUserReadModel", setup.ReadModel)
	}
	user, found := sqlRM.FindByID(aggID)
	if !found {
		t.Fatal("user not found in read model after register")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", user.Email)
	}

	// Verify SQL store has the row.
	view, err := sqlRM.FindByIDSQL(ctx, NewUserID(aggID.String()))
	if err != nil {
		t.Fatalf("FindByIDSQL: %v", err)
	}
	if view.Email != "alice@example.com" {
		t.Errorf("SQL email = %q", view.Email)
	}
}

func TestSQLiteSetup_RestartSurvival(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "restart.db")
	dsn := "file:" + dbPath + "?cache=shared"
	ctx := context.Background()
	aggID := id.NewAggregateID()
	email := "bob@example.com"

	// Phase 1: create setup, register user, close.
	{
		setup, err := NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: dsn})
		if err != nil {
			t.Fatalf("setup1: %v", err)
		}
		disp := newSQLiteTestDispatcher(t, setup)
		dispatchRegisterUser(t, disp, ctx, aggID, email, "Bob")
		if err := setup.Close(); err != nil {
			t.Fatalf("setup1 close: %v", err)
		}
	}

	// Phase 2: reopen the same DB, verify user survived.
	{
		setup, err := NewSQLiteEventSourcedSetup(SQLiteSetupConfig{DSN: dsn})
		if err != nil {
			t.Fatalf("setup2: %v", err)
		}
		defer func() { _ = setup.Close() }()

		sqlRM, ok := setup.ReadModel.(*SQLUserReadModel)
		if !ok {
			t.Fatalf("ReadModel is %T, want *SQLUserReadModel", setup.ReadModel)
		}
		// Journal replay should have rebuilt the read model.
		user, found := sqlRM.FindByID(aggID)
		if !found {
			t.Fatal("user not found after restart — persistence failed")
		}
		if user.Email != email {
			t.Errorf("email after restart = %q, want %q", user.Email, email)
		}

		// SQL store should also have the row.
		view, err := sqlRM.FindByIDSQL(ctx, NewUserID(aggID.String()))
		if err != nil {
			t.Fatalf("FindByIDSQL after restart: %v", err)
		}
		if view.Email != email {
			t.Errorf("SQL email after restart = %q, want %q", view.Email, email)
		}
	}
}
