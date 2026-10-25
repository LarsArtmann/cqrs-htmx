package main

import (
	"context"
	"log"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// --- ADR-0056 sync types: form -> typed command -> journaled event ---

type syncItemForm struct {
	Name string `json:"name"`
}

type syncAddItemCmd struct {
	command.BasicCommand

	name string
}

func (c *syncAddItemCmd) ItemName() string { return c.name }

const itemStreamType = id.StreamType("Item")

// seedSyncEvents gives the pull cache deterministic content on first
// connect (the offline-reads spec reads these through
// window.cqrsSync.getEvents() without issuing any command).
func seedSyncEvents(store *memorystorage.MemoryStore) {
	ctx := context.Background()

	for _, name := range []string{"seed-one", "seed-two"} {
		aggID := id.NewStreamID()
		ref := id.StreamRef{Type: itemStreamType, ID: aggID}

		evt, err := event.New(event.Type("Item.synced"), aggID, itemStreamType, event.Version(1),
			map[string]string{"name": name}, event.WithSchemaVersion(1))
		if err != nil {
			log.Fatalf("event.New: %v", err)
		}

		if err := store.AppendBatch(ctx, ref, []event.Event{evt}); err != nil {
			log.Fatalf("AppendBatch: %v", err)
		}
	}
}
