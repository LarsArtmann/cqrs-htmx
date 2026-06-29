package usermgmt

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	memoryv3 "github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
)

// TestCheckpointStore_OptInReplay demonstrates the checkpoint-based replay
// pattern. When a CheckpointStore is provided to EventSourcedConfig,
// StartProjections resumes from the last checkpoint instead of replaying the
// full journal on every restart.
//
// This test verifies the CheckpointStore round-trip that StartProjections
// relies on internally (see loadReplayEvents in es_projection_setup.go):
//  1. A fresh store returns a zero checkpoint (full replay).
//  2. After processing events, the checkpoint is saved.
//  3. On restart, the stored checkpoint is loaded — replay resumes from there.
//
// In production, use storage.NewSQLCheckpointStore / NewSQLiteCheckpointStore
// for durable persistence across process restarts. MemoryCheckpointStore is
// used here for hermetic testing.
func TestCheckpointStore_OptInReplay(t *testing.T) {
	t.Parallel()

	const cpName = "usermgmt:start_projections"
	cpStore := memoryv3.NewMemoryCheckpointStore()

	// First boot: no checkpoint yet → full replay.
	cp, err := cpStore.Load(t.Context(), cpName)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if !cp.IsZero() {
		t.Fatal("fresh checkpoint store should return zero checkpoint")
	}

	// Simulate StartProjections processing an event and saving the checkpoint.
	evtID := id.NewEventID()
	if err := cpStore.Save(t.Context(), cpName, event.Checkpoint{
		EventID:     evtID,
		ProcessedAt: time.Now(),
	}); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	// Restart: checkpoint exists → replay resumes from evtID.
	cp2, err := cpStore.Load(t.Context(), cpName)
	if err != nil {
		t.Fatalf("load checkpoint after restart: %v", err)
	}
	if cp2.IsZero() {
		t.Fatal("checkpoint should be non-zero after save")
	}
	if cp2.EventID != evtID {
		t.Fatalf("checkpoint event ID mismatch: got %s, want %s", cp2.EventID, evtID)
	}

	// Per-projection isolation: a different name gets its own checkpoint.
	cp3, err := cpStore.Load(t.Context(), "other:projection")
	if err != nil {
		t.Fatalf("load other projection checkpoint: %v", err)
	}
	if !cp3.IsZero() {
		t.Fatal("different projection name should have zero checkpoint")
	}
}
