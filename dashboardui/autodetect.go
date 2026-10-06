package dashboardui

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// capabilityProbe wires one introspection interface from a source value into
// the Config field it feeds. probe reports whether the source implements the
// capability AND the field was still unset — an earlier source claiming the
// same capability wins.
type capabilityProbe interface {
	probe(config *Config, source any) bool
}

// fieldProbe adapts a nilable Config field of type T into a [capabilityProbe].
type fieldProbe[T any] struct {
	field func(*Config) *T
}

func (p fieldProbe[T]) probe(config *Config, source any) bool {
	if any(*p.field(config)) != nil {
		return false
	}

	v, ok := source.(T)
	if !ok {
		return false
	}

	*p.field(config) = v

	return true
}

// readProbes cover the minimum-viable read interfaces: at least one must be
// wired from any source or Autodetect rejects, mirroring the contract New
// enforces on hand-written Configs.
//
//nolint:gochecknoglobals // immutable capability-probe table
var readProbes = []capabilityProbe{
	fieldProbe[event.EventSource]{field: func(c *Config) *event.EventSource { return &c.EventSource }},
	fieldProbe[event.Journal]{field: func(c *Config) *event.Journal { return &c.Journal }},
	fieldProbe[event.SeekableJournal]{field: func(c *Config) *event.SeekableJournal { return &c.SeekableJournal }},
}

// optionalProbes light up the extra panels whenever a source implements them.
//
//nolint:gochecknoglobals // immutable capability-probe table
var optionalProbes = []capabilityProbe{
	fieldProbe[EventByIDLoader]{field: func(c *Config) *EventByIDLoader { return &c.EventByIDLoader }},
	fieldProbe[listing.StreamReader]{field: func(c *Config) *listing.StreamReader { return &c.StreamReader }},
	fieldProbe[*projectionhost.Host]{field: func(c *Config) **projectionhost.Host { return &c.ProjectionHost }},
	fieldProbe[projectionhost.DeadLetterStore]{field: func(c *Config) *projectionhost.DeadLetterStore {
		return &c.DeadLetterStore
	}},
	fieldProbe[command.CommandJournal]{field: func(c *Config) *command.CommandJournal { return &c.CommandJournal }},
	fieldProbe[query.QueryJournal]{field: func(c *Config) *query.QueryJournal { return &c.QueryJournal }},
	fieldProbe[snapshot.SnapshotStore]{field: func(c *Config) *snapshot.SnapshotStore { return &c.SnapshotStore }},
	fieldProbe[event.Bus]{field: func(c *Config) *event.Bus { return &c.EventBus }},
}

// Autodetect builds a Config by probing every source for the go-cqrs-lite
// introspection interfaces the dashboard renders from (EventSource, Journal,
// SeekableJournal, StreamReader, ProjectionHost, DeadLetterStore, command and
// query journals, SnapshotStore, EventBus, EventByIDLoader). Whatever a source
// implements is wired; the rest stays nil and its panel stays dark.
//
// Sources merge first-wins: they are consulted in order and each capability is
// claimed by the first source that implements it — pass the primary store
// first, auxiliary stores (a dedicated snapshot store, a projection host)
// after it. Conflicts never error; the earliest source simply wins.
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
// Fields the sources cannot provide (Title, BasePath, ReadOnly, Authorizer,
// Layout, PageSize, SSE tuning) keep their zero values and are filled by
// [Config.withDefaults] or the consumer after Autodetect.
//
// The error is a rejection when no source implements any of the read
// interfaces — the same contract [New] enforces, surfaced before the
// consumer customizes anything.
func Autodetect(sources ...any) (Config, error) {
	var config Config

	for _, source := range sources {
		for _, p := range readProbes {
			p.probe(&config, source)
		}

		for _, p := range optionalProbes {
			p.probe(&config, source)
		}
	}

	if config.EventSource == nil && config.Journal == nil && config.SeekableJournal == nil {
		return Config{}, errorfamily.NewRejection(
			"dashboardui.autodetect.no_read_interface",
			"no source implements any of the interfaces the dashboard reads from "+
				"(event.EventSource, event.Journal, event.SeekableJournal)",
		).WithContext("source_types", sourceTypes(sources))
	}

	return config, nil
}

func sourceTypes(sources []any) string {
	types := make([]string, len(sources))

	for i, source := range sources {
		types[i] = fmt.Sprintf("%T", source)
	}

	return strings.Join(types, ", ")
}
