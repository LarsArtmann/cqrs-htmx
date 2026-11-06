package dashboardui

import (
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/templ-components/icons"
)

const (
	defaultBasePath             = "/dashboard"
	defaultTitle                = "CQRS Dashboard"
	defaultAccentColor          = "#4f46e5"
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

	// Topology enables the read-only telemetry panel's topology section:
	// deployed instances, buses, and the projection host shape. Providers
	// return dashboard-owned views (core.TopologyView) — an adapter maps a
	// go-cqrs-lite *system.System onto them (see systemadapter).
	Topology core.TopologyProvider

	// EngineHealths composes per-engine health into /-/healthz and /-/readyz:
	// unhealthy engines are listed on healthz and fail readyz.
	EngineHealths core.EngineHealthProvider

	// Placements enables the query-placement table: which engine and ADT the
	// metaengine planner assigned to each query, at what volume and latency
	// estimate.
	Placements core.PlacementsProvider

	// EngineStats enables the engine stat cards: live RTT EWMA/percentiles
	// and sample counts per engine.
	EngineStats core.EngineStatsProvider

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

	// PayloadTypes routes per-event-type rendering through typed renderers
	// registered via RegisterPayloadType(In). It is consulted BEFORE
	// PayloadRenderer for registered types only. If nil, the package
	// default registry (DefaultPayloadTypes) is used — startup-time
	// RegisterPayloadType calls are honored with zero config. Set an
	// explicit empty registry (NewPayloadTypeRegistry) to ignore the
	// package registry entirely.
	PayloadTypes *PayloadTypeRegistry

	// Title is the brand text in the sidebar and browser tab.
	Title string

	// EmptyStateCopy overrides the empty-state panel text per surface, for
	// hosts that brand their observability voice. Keys are the typed
	// [EmptyStateKey] constants; values merge FIELD-WISE over the upstream
	// defaults, so an override that sets only Message keeps the default
	// title. A nil map renders byte-identical upstream copy. Unknown keys
	// are rejected by [New]/[Config.Validate] — the compiler cannot catch a
	// mistyped map key.
	EmptyStateCopy map[EmptyStateKey]EmptyStateCopy

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

	// Authorizer controls access (legacy form). If nil, allows all requests
	// (the consumer MUST wrap the dashboard with their own auth middleware);
	// audit entries then record the actor only when consumer middleware
	// injects one via [WithActor].
	Authorizer func(*http.Request) error

	// ActorAuthorizer is the actor-aware form of Authorizer: on success it
	// returns the [Actor] performing the request, which the dashboard
	// injects into the request context and records in every
	// dashboardui.audit log entry (actor_id/actor_name). When set it takes
	// precedence over Authorizer; the denial path is identical (403).
	ActorAuthorizer func(*http.Request) (Actor, error)

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

	// NonceFunc resolves the per-request CSP nonce carried by every inline
	// script the dashboard renders (shell, error pages, 404 page). Nil (the
	// default) uses httputil.NonceFromRequest, which reads the value
	// httputil.Nonce puts into the request context. Set it when your nonce
	// lives elsewhere — a custom context key, an upstream header, or a
	// generator you own — and return the exact string your CSP header
	// advertises.
	NonceFunc func(*http.Request) string

	// LogoutURL, if set, renders a logout link at the bottom of the sidebar.
	// Typically "/logout" or similar. If empty, no logout link is shown.
	// Ignored when Layout is set (the consumer shell owns navigation chrome).
	LogoutURL string

	// ExtraNav appends consumer-owned entries after the capability-derived
	// sidebar items, so surfaces the consumer mounts next door to the panel
	// stay discoverable from the dashboard navigation. Href is rendered
	// VERBATIM (no BasePath prefix) — see [NavEntry]. Empty (the default)
	// changes nothing.
	ExtraNav []NavEntry
}

// Validate checks the configuration and returns the normalized copy with
// defaults applied (Title, BasePath, AccentColor, PageSize, payload
// renderer, SSE heartbeat). [New] applies it automatically; consumers can
// call it directly to fail fast — e.g. at process startup — without
// constructing a Dashboard. Returns a Rejection-family error describing
// the first invalid field.
func (config Config) Validate() (Config, error) {
	return config.withDefaults()
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

	if err := validateAccentColor(config.AccentColor); err != nil {
		return config, err
	}

	// A negative PageSize would pass through and panic the first paginated
	// request (events[:pageSize] with a negative bound), so clamp it with the
	// zero-value default.
	config.PageSize = clampPageSize(config.PageSize)

	if config.SSEHeartbeatInterval == 0 {
		config.SSEHeartbeatInterval = defaultSSEHeartbeatInterval
	}

	if err := validateEmptyStateCopy(config.EmptyStateCopy); err != nil {
		return config, err
	}

	if err := validateExtraNav(config.ExtraNav); err != nil {
		return config, err
	}

	config = config.withPayloadRenderingDefaults()

	return config, nil
}

// clampPageSize bounds PageSize to [core.DefaultPageSize, core.MaxPageSize]:
// a non-positive value would panic the first paginated request
// (events[:pageSize] with a negative bound), and an oversized one would
// ignore the consumer-facing page ceiling.
func clampPageSize(size int) int {
	if size <= 0 {
		return core.DefaultPageSize()
	}

	return min(size, core.MaxPageSize())
}

// validateAccentColor rejects AccentColor values outside the closed CSS
// grammar — the value is interpolated into an inline <style> block where HTML
// escaping cannot sanitize it.
func validateAccentColor(color string) error {
	if isValidAccentColor(color) {
		return nil
	}

	return errConfig(
		"Config.AccentColor must be a CSS color literal (#hex, rgb()/rgba()/hsl()/hsla() with numeric components, or a named color); got " + color,
	)
}

// validateExtraNav rejects entries that could not render usefully: an empty
// Label shows as a blank link, an empty Href points at the current page.
func validateExtraNav(entries []NavEntry) error {
	for i, entry := range entries {
		if entry.Label == "" {
			return errConfig("Config.ExtraNav[" + strconv.Itoa(i) + "].Label must not be empty")
		}

		if entry.Href == "" {
			return errConfig("Config.ExtraNav[" + strconv.Itoa(i) + "].Href must not be empty")
		}
	}

	return nil
}

// withPayloadRenderingDefaults fills the payload display seam: the untyped
// renderer falls back to pretty-print, the typed registry to the package
// registry (startup-time RegisterPayloadType calls then work with zero
// config).
func (config Config) withPayloadRenderingDefaults() Config {
	if config.PayloadRenderer == nil {
		config.PayloadRenderer = DefaultPayloadRenderer{}
	}

	if config.PayloadTypes == nil {
		config.PayloadTypes = DefaultPayloadTypes()
	}

	return config
}

// NavEntry is one consumer-owned sidebar navigation entry, appended after
// the capability-derived items via [Config.ExtraNav].
type NavEntry struct {
	// Label is the display text ("Event Catalog").
	Label string

	// Href is the link target, rendered VERBATIM: unlike built-in nav
	// entries it is NOT prefixed with BasePath. Point it at a sibling
	// surface outside the mounted subtree ("/event-catalog"), an absolute
	// URL ("https://docs.example.com"), or build a dashboard-internal path
	// yourself with Config.BasePath.
	Href string

	// Icon is the optional templ-components icon name for the entry
	// (e.g. icons.Link). The zero value renders the link icon.
	Icon icons.Name
}

// extraNavItems converts validated consumer entries to the rendered nav
// shape ([NavLink]), defaulting the icon. Entries are appended AFTER the
// capability items; buildNav stays the source of truth for those.
func extraNavItems(entries []NavEntry) []NavLink {
	items := make([]NavLink, 0, len(entries))

	for _, entry := range entries {
		icon := entry.Icon
		if icon == "" {
			icon = icons.Link
		}

		items = append(items, NavLink{Href: entry.Href, Verbatim: true, Label: entry.Label, Icon: icon})
	}

	return items
}

func buildNav(caps Capabilities) []NavLink {
	var items []NavLink

	add := func(href, label string, icon icons.Name) {
		items = append(items, NavLink{Href: href, Label: label, Icon: icon})
	}

	add("/", "Overview", icons.Chart)

	if caps.HasEventRead() {
		add("/events", "Events", icons.QueueList)
	}

	if caps.StreamReader || caps.EventSource {
		add("/aggregates", "Aggregates", icons.Cube)
	}

	if caps.ProjectionHost {
		add("/projections", "Projections", icons.ArrowPath)
	}

	if caps.DeadLetterStore || caps.ProjectionHost {
		add("/dead-letters", "Dead Letters", icons.BugAnt)
	}

	if caps.CommandJournal {
		add("/commands", "Commands", icons.Clipboard)
	}

	if caps.QueryJournal {
		add("/queries", "Queries", icons.Search)
	}

	if caps.EventSource {
		add("/time-travel", "Time Travel", icons.Clock)
	}

	if caps.SnapshotStore {
		add("/snapshots", "Snapshots", icons.ArchiveBox)
	}

	if caps.Telemetry {
		add("/telemetry", "Telemetry", icons.Signal)
	}

	return items
}

// pageData is passed to every page renderer.
type pageData struct {
	Title     string
	BasePath  string
	Accent    string
	Brand     string
	Nav       []NavLink
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

	// empty carries the resolved empty-state copy (upstream defaults merged
	// with Config.EmptyStateCopy overrides); templ panels read it per key.
	empty map[EmptyStateKey]EmptyStateCopy
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
