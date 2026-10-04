package dashboardui

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Autodetect builds a Config by probing store for every go-cqrs-lite
// introspection interface the dashboard renders from (EventSource, Journal,
// SeekableJournal, StreamReader, ProjectionHost, DeadLetterStore, command and
// query journals, SnapshotStore, EventBus, EventByIDLoader). Whatever the
// value implements is wired; the rest stays nil and its panel stays dark.
//
// It removes the hand-written type-assertion dance from consumer wiring —
// the only required call chain becomes:
//
//	cfg, err := dashboardui.Autodetect(store)
//	cfg.Title = "MyApp CQRS"
//	cfg.ReadOnly = true
//	dash := dashboardui.MustNew(cfg)
//	dash.Mount(mux, "/cqrs/")
//
// Fields the store cannot provide (Title, BasePath, ReadOnly, Authorizer,
// Layout, PageSize, SSE tuning) keep their zero values and are filled by
// [Config.withDefaults] or the consumer after Autodetect.
//
// The error is a rejection when the store implements none of the read
// interfaces — the same contract [New] enforces, surfaced before the
// consumer customizes anything.
//
//nolint:cyclop // capability probing is inherently one type assertion per probed interface
func Autodetect(store any) (Config, error) {
	var config Config

	if v, ok := store.(event.EventSource); ok {
		config.EventSource = v
	}

	if v, ok := store.(EventByIDLoader); ok {
		config.EventByIDLoader = v
	}

	if v, ok := store.(event.SeekableJournal); ok {
		config.SeekableJournal = v
	}

	if v, ok := store.(event.Journal); ok {
		config.Journal = v
	}

	if v, ok := store.(listing.StreamReader); ok {
		config.StreamReader = v
	}

	if v, ok := store.(*projectionhost.Host); ok {
		config.ProjectionHost = v
	}

	if v, ok := store.(projectionhost.DeadLetterStore); ok {
		config.DeadLetterStore = v
	}

	if v, ok := store.(command.CommandJournal); ok {
		config.CommandJournal = v
	}

	if v, ok := store.(query.QueryJournal); ok {
		config.QueryJournal = v
	}

	if v, ok := store.(snapshot.SnapshotStore); ok {
		config.SnapshotStore = v
	}

	if v, ok := store.(event.Bus); ok {
		config.EventBus = v
	}

	if config.EventSource == nil && config.Journal == nil && config.SeekableJournal == nil {
		return Config{}, errorfamily.NewRejection(
			"dashboardui.autodetect.no_read_interface",
			"store implements none of the interfaces the dashboard reads from "+
				"(event.EventSource, event.Journal, event.SeekableJournal)",
		).WithContext("store_type", fmt.Sprintf("%T", store))
	}

	return config, nil
}
