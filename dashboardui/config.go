package dashboardui

import (
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

	// SSEMaxReplay caps the number of events replayed to a first-time SSE
	// subscriber (no Last-Event-ID). A value of 0 means "use the transport
	// default" (1000). Set to a positive int to bound the initial backfill
	// window and prevent sending the entire journal history on first connect.
	SSEMaxReplay int

	// PayloadRenderer formats event payloads for display. If nil,
	// DefaultPayloadRenderer is used (JSON/CBOR pretty-print).
	PayloadRenderer PayloadRenderer

	// Title is the brand text in the sidebar and browser tab.
	Title string

	// BasePath is the URL prefix the dashboard is mounted under.
	BasePath string

	// AccentColor is the highlight color. Must be a CSS color literal — #hex
	// (3/4/6/8 digits), rgb()/rgba()/hsl()/hsla() with numeric components, or
	// a named CSS color (or "transparent"). New rejects anything else: the
	// value is interpolated into an inline <style> block that HTML escaping
	// cannot sanitize, so only this closed grammar is accepted.
	// Default: #4f46e5.
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

	// VersionzRequireAuth routes the /-/versionz endpoint through Authorizer
	// (403 on denial). Default false: versionz follows the public-probe
	// convention (like healthz/readyz) so load balancers can read build
	// stamps without credentials — but it reveals Title, BasePath,
	// capabilities, module version, and the binary's VCS revision. Set true
	// to keep that surface behind the Authorizer; without an Authorizer the
	// flag has no effect.
	VersionzRequireAuth bool

	// Layout, when set, replaces the dashboard's built-in page shell (sidebar,
	// header, theme, stylesheet links) with a consumer-owned document: every
	// full-page render calls it with the page metadata and the ready-rendered
	// content, and its component becomes the response. This is the embed seam
	// for apps that want the panels INSIDE their own app chrome — see
	// [LayoutFunc] for the contract. When nil (default), the dashboard renders
	// its own complete standalone document (the mount-and-go destination).
	Layout LayoutFunc

	// LogoutURL, if set, renders a logout link at the bottom of the sidebar.
	// Typically "/logout" or similar. If empty, no logout link is shown.
	// Ignored when Layout is set (the consumer shell owns navigation chrome).
	LogoutURL string
}

func (config Config) withDefaults() (Config, error) {
	if config.EventSource == nil && config.Journal == nil && config.SeekableJournal == nil {
		return config, errConfig(
			"at least one of Config.EventSource, Config.Journal, or Config.SeekableJournal is required",
		)
	}

	if config.Title == "" {
		config.Title = defaultTitle
	}

	if config.BasePath == "" {
		config.BasePath = defaultBasePath
	}

	config.BasePath = trimTrailingSlash(config.BasePath)
	if config.AccentColor == "" {
		config.AccentColor = defaultAccentColor
	}

	if !isValidAccentColor(config.AccentColor) {
		return config, errConfig(
			"Config.AccentColor must be a CSS color literal (#hex, rgb()/rgba()/hsl()/hsla() with numeric components, or a named color); got " + config.AccentColor,
		)
	}

	if config.PageSize == 0 {
		config.PageSize = defaultPageSize
	}

	if config.PageSize > maxPageSize {
		config.PageSize = maxPageSize
	}

	if config.PayloadRenderer == nil {
		config.PayloadRenderer = DefaultPayloadRenderer{}
	}

	if config.SSEHeartbeatInterval == 0 {
		config.SSEHeartbeatInterval = defaultSSEHeartbeatInterval
	}

	return config, nil
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

	if caps.HasEventRead() {
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

// pageData is passed to every page renderer.
type pageData struct {
	Title     string
	BasePath  string
	Accent    string
	Brand     string
	Nav       []navItem
	LogoutURL string
	CSRFToken string
	// Nonce is the per-request CSP nonce (httputil.NonceFromRequest). Inline
	// scripts emitted by adopted templ-components components carry it; empty
	// when the middleware stack does not issue nonces.
	Nonce    string
	ReadOnly bool
	Caps     Capabilities
	// HTMX is true when the request carries the HX-Request header (boosted
	// link or explicit hx-get). When true, renderLayout returns only the
	// <main> content, skipping the full HTML shell for faster swaps.
	HTMX bool

	// shell is the consumer-provided layout ([Config.Layout]); nil when the
	// built-in document shell should render.
	shell LayoutFunc
}

// StreamRefFromID constructs an id.StreamRef from type + ID strings.
// Used by handlers that parse path parameters.
func StreamRefFromID(streamType string, streamID string) (id.StreamRef, error) {
	parsedType, err := id.ParseStreamType(streamType)
	if err != nil {
		return id.StreamRef{}, errorfamily.WrapRejection(err,
			"dashboardui.stream_ref.invalid_type", "parse stream type").
			WithContext("stream_type", streamType)
	}

	sid, err := id.ParseStreamID(streamID)
	if err != nil {
		return id.StreamRef{}, errorfamily.WrapRejection(err,
			"dashboardui.stream_ref.invalid_id", "parse stream ID").
			WithContext("stream_id", streamID)
	}

	return id.NewStreamRef(parsedType, sid), nil //nolint:erraudit // FP: success-path return, nothing lost
}

// journalForReplay returns the best available journal for SSE reconnect replay.
// SeekableJournal is preferred (efficient cursor-based ReadFrom); Journal is the fallback.
// Returns nil if no journal is configured.
func (config Config) journalForReplay() event.Journal { //nolint:ireturn // intentionally returns the Journal interface to abstract over SeekableJournal/Journal
	if config.SeekableJournal != nil {
		return config.SeekableJournal // SeekableJournal embeds Journal
	}

	return config.Journal
}

func trimTrailingSlash(s string) string {
	for len(s) > 1 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}

	return s
}
