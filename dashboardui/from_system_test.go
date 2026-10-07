package dashboardui

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// fakeSystem implements the duck-typed [System] interface with in-memory
// stores — the same structural shape as *system.System, no system import.
type fakeSystem struct {
	events    event.Store
	bus       event.Bus
	host      *projectionhost.Host
	snapshots snapshot.SnapshotStore
	commands  command.Store
	queries   query.QueryStore
}

func (f fakeSystem) EventStore() event.Store               { return f.events }
func (f fakeSystem) Bus() event.Bus                        { return f.bus }
func (f fakeSystem) ProjectionHost() *projectionhost.Host  { return f.host }
func (f fakeSystem) SnapshotStore() snapshot.SnapshotStore { return f.snapshots }
func (f fakeSystem) CommandStore() command.Store           { return f.commands }
func (f fakeSystem) QueryStore() query.QueryStore          { return f.queries }

func fullFakeSystem() fakeSystem {
	return fakeSystem{
		events:    memorystorage.NewMemoryStore(),
		bus:       eventtest.NewFakeBus(),
		host:      &projectionhost.Host{},
		snapshots: &fakeSnapshotStore{},
		commands:  memorystorage.NewMemoryCommandStore(),
		queries:   memorystorage.NewMemoryQueryStore(),
	}
}

func TestFromSystem_MapsEveryCapability(t *testing.T) {
	config := FromSystem(fullFakeSystem())

	checks := []struct {
		name string
		ok   bool
	}{
		{"EventSource", config.EventSource != nil},
		{"Journal", config.Journal != nil},
		{"SeekableJournal", config.SeekableJournal != nil},
		{"StreamReader", config.StreamReader != nil},
		{"EventBus", config.EventBus != nil},
		{"ProjectionHost", config.ProjectionHost != nil},
		{"SnapshotStore", config.SnapshotStore != nil},
		{"CommandJournal", config.CommandJournal != nil},
		{"QueryJournal", config.QueryJournal != nil},
	}

	for _, check := range checks {
		if !check.ok {
			t.Errorf("%s should be wired from a full system", check.name)
		}
	}

	dash, err := New(config)
	if err != nil {
		t.Fatalf("New(FromSystem(...)): %v", err)
	}

	if !dash.Capabilities().HasEventRead() {
		t.Error("event read capability missing after FromSystem")
	}
}

func TestFromSystem_PartialSystemWiresOnlyWhatExists(t *testing.T) {
	config := FromSystem(fakeSystem{
		events:   memorystorage.NewMemoryStore(),
		commands: memorystorage.NewMemoryCommandStore(),
	})

	if config.EventBus != nil {
		t.Error("EventBus must stay nil when the system has none")
	}

	if config.ProjectionHost != nil {
		t.Error("ProjectionHost must stay nil when the system has none")
	}

	if config.CommandJournal == nil {
		t.Error("CommandJournal should be wired from the command store")
	}
}

func TestAutodetect_RecognizesSystemSource(t *testing.T) {
	config, err := Autodetect(fullFakeSystem())
	if err != nil {
		t.Fatalf("Autodetect(system): %v", err)
	}

	if config.EventSource == nil || config.ProjectionHost == nil {
		t.Error("a System source must map through the bridge, not the leaf probes")
	}
}

func TestAutodetect_SystemMergesFirstWins(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	config, err := Autodetect(store, fullFakeSystem())
	if err != nil {
		t.Fatalf("Autodetect(store, system): %v", err)
	}

	if _, ok := config.Journal.(*memorystorage.MemoryStore); !ok {
		t.Errorf("Journal should keep the first source's claim, got %T", config.Journal)
	}

	if config.SnapshotStore == nil {
		t.Error("the system source should still contribute what the store lacks")
	}
}
