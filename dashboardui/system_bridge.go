package dashboardui

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// System is the duck-typed capability surface [FromSystem] maps from. It is
// satisfied structurally by go-cqrs-lite's *system.System — and by any wrapper
// that forwards these accessors — WITHOUT dashboardui importing the system
// package (idea 308: capability seams stay leaf-typed; a wrapper that
// forwards the accessors stays probeable, unlike a concrete assertion on the
// store value).
type System interface {
	EventStore() event.Store
	Bus() event.Bus
	ProjectionHost() *projectionhost.Host
	SnapshotStore() snapshot.SnapshotStore
	CommandStore() command.Store
	QueryStore() query.QueryStore
}

// FromSystem builds a dashboard Config from a go-cqrs-lite system (or any
// [System] implementation): the event store feeds the event panels (whatever
// journal/seek/event-by-ID interfaces its engine implements, plus a derived
// stream reader when the engine has none), the bus feeds live SSE updates,
// and the projection host, snapshot store, and command/query stores light up
// their panels. Fields the system cannot provide keep their zero values; the
// caller layers Title/BasePath/Authorizer on top as usual:
//
//	cfg := dashboardui.FromSystem(sys)
//	cfg.Title = "MyApp CQRS"
//	dash := dashboardui.MustNew(cfg)
//
// Autodetect recognizes System values too, so a source list may mix stores
// and systems.
func FromSystem(sys System) Config {
	var config Config

	applySystem(&config, sys)

	return config
}

// applySystem merges a System's capabilities into config without overwriting
// fields an earlier source already claimed — the same first-wins rule
// [Autodetect] applies between sources.
func applySystem(config *Config, sys System) {
	applySystemEventStore(config, sys.EventStore())
	applySystemBus(config, sys.Bus())
	applySystemProjectionHost(config, sys.ProjectionHost())
	applySystemSnapshotStore(config, sys.SnapshotStore())
	applySystemCommandStore(config, sys.CommandStore())
	applySystemQueryStore(config, sys.QueryStore())
}

// systemEventStoreProbes wire every event-reading interface an engine's store
// may implement. They carry the first-wins nil guard from [fieldProbe].
//
//nolint:gochecknoglobals // immutable capability-probe table
var systemEventStoreProbes = []capabilityProbe{
	fieldProbe[event.EventSource]{field: func(c *Config) *event.EventSource { return &c.EventSource }},
	fieldProbe[event.Journal]{field: func(c *Config) *event.Journal { return &c.Journal }},
	fieldProbe[event.SeekableJournal]{field: func(c *Config) *event.SeekableJournal { return &c.SeekableJournal }},
	fieldProbe[EventByIDLoader]{field: func(c *Config) *EventByIDLoader { return &c.EventByIDLoader }},
	fieldProbe[listing.StreamReader]{field: func(c *Config) *listing.StreamReader { return &c.StreamReader }},
}

func applySystemEventStore(config *Config, store event.Store) {
	if store == nil {
		return
	}

	for _, p := range systemEventStoreProbes {
		p.probe(config, store)
	}

	// Engines without a native stream reader still expose the journal —
	// derive one so the aggregates panel lights up everywhere.
	if config.StreamReader == nil && config.Journal != nil {
		config.StreamReader = listing.NewInMemoryStreamReader(config.Journal)
	}
}

func applySystemBus(config *Config, bus event.Bus) {
	if bus != nil && config.EventBus == nil {
		config.EventBus = bus
	}
}

func applySystemProjectionHost(config *Config, host *projectionhost.Host) {
	if host != nil && config.ProjectionHost == nil {
		config.ProjectionHost = host
	}
}

func applySystemSnapshotStore(config *Config, store snapshot.SnapshotStore) {
	if store != nil && config.SnapshotStore == nil {
		config.SnapshotStore = store
	}
}

func applySystemCommandStore(config *Config, store command.Store) {
	if store == nil || config.CommandJournal != nil {
		return
	}

	if v, ok := store.(command.CommandJournal); ok {
		config.CommandJournal = v
	}
}

func applySystemQueryStore(config *Config, store query.QueryStore) {
	if store == nil || config.QueryJournal != nil {
		return
	}

	if v, ok := store.(query.QueryJournal); ok {
		config.QueryJournal = v
	}
}
