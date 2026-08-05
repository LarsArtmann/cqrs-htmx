package core

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// EventByIDLoader loads a single event by its EventID in O(1). Implemented
// by SQL event stores. If not configured, the dashboard falls back to
// scanning the event journal for event detail views.
type EventByIDLoader interface {
	LoadByEventID(ctx context.Context, eventID id.EventID) (event.Event, error)
}

// Config holds the data-source interfaces that core functions need.
// It is a subset of the parent package's [Config] — only the fields
// relevant to data fetching, not rendering configuration (Title, BasePath,
// AccentColor, etc.).
type Config struct {
	EventSource     event.EventSource
	EventByIDLoader EventByIDLoader
	Journal         event.Journal
	SeekableJournal event.SeekableJournal
	StreamReader    listing.StreamReader
	ProjectionHost  *projectionhost.Host
	DeadLetterStore projectionhost.DeadLetterStore
	CommandJournal  command.CommandJournal
	QueryJournal    query.QueryJournal
	SnapshotStore   snapshot.SnapshotStore
	EventBus        event.Bus
	PageSize        int
	PayloadRenderer PayloadRenderer
}

// Capabilities describes which panels are available based on the
// interfaces the consumer provided.
type Capabilities struct {
	EventSource     bool
	EventByIDLoader bool
	Journal         bool
	SeekableJournal bool
	StreamReader    bool
	ProjectionHost  bool
	DeadLetterStore bool
	CommandJournal  bool
	QueryJournal    bool
	SnapshotStore   bool
	EventBus        bool
}

// DetectCapabilities inspects the Config and returns which panels can
// be activated based on which interfaces the consumer provided.
func DetectCapabilities(cfg Config) Capabilities {
	return Capabilities{
		EventSource:     cfg.EventSource != nil,
		EventByIDLoader: cfg.EventByIDLoader != nil,
		Journal:         cfg.Journal != nil,
		SeekableJournal: cfg.SeekableJournal != nil,
		StreamReader:    cfg.StreamReader != nil,
		ProjectionHost:  cfg.ProjectionHost != nil,
		DeadLetterStore: cfg.DeadLetterStore != nil,
		CommandJournal:  cfg.CommandJournal != nil,
		QueryJournal:    cfg.QueryJournal != nil,
		SnapshotStore:   cfg.SnapshotStore != nil,
		EventBus:        cfg.EventBus != nil,
	}
}

// HasEventRead returns true if any event reading interface is available.
func (c Capabilities) HasEventRead() bool {
	return c.Journal || c.SeekableJournal
}
