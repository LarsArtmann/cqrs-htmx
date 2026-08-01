package dashboardui

import (
	"context"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	defaultBasePath             = "/dashboard"
	defaultTitle                = "CQRS Dashboard"
	defaultAccentColor          = "#4f46e5"
	defaultPageSize             = 50
	maxPageSize                 = 200
	defaultSSEHeartbeatInterval = 15 * time.Second
)

// EventByIDLoader loads a single event by its EventID in O(1). Implemented
// by SQL event stores. If not configured, the dashboard falls back to
// scanning the event journal for event detail views.
type EventByIDLoader interface {
	LoadByEventID(ctx context.Context, eventID id.EventID) (event.Event, error)
}

// Config wires the dashboard to go-cqrs-lite introspection interfaces.
// Only EventSource or a Journal is required; everything else is optional
// and conditionally activates panels.
type Config struct {
	// EventSource provides per-aggregate event loading (aggregate detail,
	// time-travel). Can be nil if only the global event log is needed.
	EventSource event.EventSource

	// EventByIDLoader provides O(1) single-event lookup by EventID.
	// If nil, the dashboard scans the journal for event detail views.
	EventByIDLoader EventByIDLoader

	// Journal or SeekableJournal provides the global event log.
	// SeekableJournal is preferred (paginated). Journal is the fallback.
	Journal         event.Journal
	SeekableJournal event.SeekableJournal

	// StreamReader lists aggregates for the Aggregate Browser panel.
	// If nil, the dashboard auto-creates an InMemoryStreamReader from
	// Journal (if available).
	StreamReader listing.StreamReader

	// ProjectionHost enables the Projection Dashboard panel.
	ProjectionHost *projectionhost.Host

	// DeadLetterStore enables the Dead-Letter Queue panel.
	// If ProjectionHost is set, its internal DLQ is used automatically.
	DeadLetterStore projectionhost.DeadLetterStore

	// CommandJournal enables the Command Audit panel.
	CommandJournal command.CommandJournal

	// QueryJournal enables the Query Audit panel.
	QueryJournal query.QueryJournal

	// SnapshotStore enables the Snapshot Inspector panel.
	SnapshotStore snapshot.SnapshotStore

	// EventBus enables SSE live updates (event tail, projection changes).
	EventBus event.Bus

	// SSEHeartbeatInterval controls how often connected SSE clients receive
	// keep-alive comment frames. A non-positive value disables heartbeats.
	// Default: 15 seconds.
	SSEHeartbeatInterval time.Duration

	// PayloadRenderer formats event payloads for display. If nil,
	// DefaultPayloadRenderer is used (JSON/CBOR pretty-print).
	PayloadRenderer PayloadRenderer

	// Title is the brand text in the sidebar and browser tab.
	Title string

	// BasePath is the URL prefix the dashboard is mounted under.
	BasePath string

	// AccentColor is the highlight color (any CSS color value).
	AccentColor string

	// ReadOnly disables all write operations: projection reset, DLQ
	// replay/delete/purge, snapshot delete. Default: true (safe).
	ReadOnly bool

	// PageSize controls the number of rows per page in tables.
	// Default: 50. Max: 200.
	PageSize int

	// Authorizer controls access. If nil, allows all requests (the
	// consumer MUST wrap the dashboard with their own auth middleware).
	Authorizer func(*http.Request) error

	// LogoutURL, if set, renders a logout link at the bottom of the sidebar.
	// Typically "/logout" or similar. If empty, no logout link is shown.
	LogoutURL string
}

func (cfg Config) withDefaults() (Config, error) {
	if cfg.EventSource == nil && cfg.Journal == nil && cfg.SeekableJournal == nil {
		return cfg, errConfig(
			"at least one of Config.EventSource, Config.Journal, or Config.SeekableJournal is required",
		)
	}

	if cfg.Title == "" {
		cfg.Title = defaultTitle
	}

	if cfg.BasePath == "" {
		cfg.BasePath = defaultBasePath
	}

	cfg.BasePath = trimTrailingSlash(cfg.BasePath)
	if cfg.AccentColor == "" {
		cfg.AccentColor = defaultAccentColor
	}

	if cfg.PageSize == 0 {
		cfg.PageSize = defaultPageSize
	}

	if cfg.PageSize > maxPageSize {
		cfg.PageSize = maxPageSize
	}

	if cfg.PayloadRenderer == nil {
		cfg.PayloadRenderer = DefaultPayloadRenderer{}
	}

	if cfg.SSEHeartbeatInterval == 0 {
		cfg.SSEHeartbeatInterval = defaultSSEHeartbeatInterval
	}

	return cfg, nil
}

// Capabilities describes which panels are available based on the
// interfaces the consumer provided. The dashboard uses this to decide
// which nav items to show and which routes to register.
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

func (cfg Config) capabilities() Capabilities {
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

// hasEventRead returns true if any event reading interface is available.
func (c Capabilities) hasEventRead() bool {
	return c.Journal || c.SeekableJournal
}

// navItem represents a sidebar navigation entry.
type navItem struct {
	Href   string
	Label  string
	Icon   string
	Active bool
}

func buildNav(caps Capabilities) []navItem {
	var items []navItem

	add := func(href, label, icon string) {
		items = append(items, navItem{Href: href, Label: label, Icon: icon})
	}

	add("/", "Overview", "chart")

	if caps.hasEventRead() {
		add("/events", "Events", "queue")
	}

	if caps.StreamReader || caps.EventSource {
		add("/aggregates", "Aggregates", "cube")
	}

	if caps.ProjectionHost {
		add("/projections", "Projections", "arrow-path")
	}

	if caps.DeadLetterStore || caps.ProjectionHost {
		add("/dead-letters", "Dead Letters", "bug")
	}

	if caps.CommandJournal {
		add("/commands", "Commands", "clipboard")
	}

	if caps.QueryJournal {
		add("/queries", "Queries", "magnifying-glass")
	}

	if caps.EventSource {
		add("/time-travel", "Time Travel", "clock")
	}

	if caps.SnapshotStore {
		add("/snapshots", "Snapshots", "archive")
	}

	return items
}

// pageData is passed to every templ page renderer.
type pageData struct {
	Title     string
	BasePath  string
	Accent    string
	Brand     string
	Nav       []navItem
	LogoutURL string
	CSRFToken string
	ReadOnly  bool
	Caps      Capabilities
}

// StreamRefFromID constructs an id.StreamRef from type + ID strings.
// Used by handlers that parse path parameters.
func StreamRefFromID(streamType string, streamID string) (id.StreamRef, error) {
	parsedType, err := id.ParseStreamType(streamType)
	if err != nil {
		return id.StreamRef{}, errorfamily.WrapRejection(err,
			"dashboardui.stream_ref.invalid_type", "parse stream type")
	}

	sid, err := id.ParseStreamID(streamID)
	if err != nil {
		return id.StreamRef{}, errorfamily.WrapRejection(err,
			"dashboardui.stream_ref.invalid_id", "parse stream ID")
	}

	return id.NewStreamRef(parsedType, sid), nil
}

// journalForReplay returns the best available journal for SSE reconnect replay.
// SeekableJournal is preferred (efficient cursor-based ReadFrom); Journal is the fallback.
// Returns nil if no journal is configured.
func (cfg Config) journalForReplay() event.Journal { //nolint:ireturn // intentionally returns the Journal interface to abstract over SeekableJournal/Journal
	if cfg.SeekableJournal != nil {
		return cfg.SeekableJournal // SeekableJournal embeds Journal
	}

	return cfg.Journal
}

func trimTrailingSlash(s string) string {
	for len(s) > 1 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}

	return s
}
