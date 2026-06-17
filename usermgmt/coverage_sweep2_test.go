package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/decider/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
)

func TestRegisterCommands_DuplicateReturnsError(t *testing.T) {
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	defer func() { _ = bus.Close() }()
	repo, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	disp := command.NewDispatcher()

	if err := RegisterCommands(disp, repo); err != nil {
		t.Fatalf("first RegisterCommands: %v", err)
	}
	// Registering the same commands twice must surface an error (duplicate type).
	if err := RegisterCommands(disp, repo); err == nil {
		t.Fatal("expected error when registering duplicate commands")
	}
}

func TestReadModel_FindByUserID_InvalidID(t *testing.T) {
	rm := NewUserReadModel()
	// A non-ULID string cannot be parsed as an AggregateID.
	if u, ok := rm.FindByUserID(NewUserID("not-a-valid-ulid")); ok || u != nil {
		t.Errorf("expected nil/false for invalid UserID, got %v ok=%v", u, ok)
	}
}
