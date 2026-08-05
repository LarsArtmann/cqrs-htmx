package dashboardui

import (
	"context"
	"net/http"
	"time"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Type aliases (transparent re-exports of core data types) =====
//
// These aliases ensure the main package's API is unchanged. Existing
// consumers who reference dashboardui.Capabilities, dashboardui.PayloadRenderer,
// etc. continue to work. Internally, all logic lives in core.

// Capabilities describes which panels are available based on the
// configured data sources.
type Capabilities = core.Capabilities

// EventByIDLoader is the interface for loading a single event by ID.
type EventByIDLoader = core.EventByIDLoader

// PayloadRenderer formats event payloads for display.
type PayloadRenderer = core.PayloadRenderer

// DefaultPayloadRenderer is the default payload renderer.
type DefaultPayloadRenderer = core.DefaultPayloadRenderer

// Unexported aliases used internally by the rendering layer.
type projectionStat = core.ProjectionStat
type overviewStats = core.Overview
type paginationState = core.PageState
type eventFilter = core.EventFilter
type dlqProjectionLink = core.DLQProjectionLink

// ===== Status constant aliases =====

const (
	statusGood    = core.StatusGood
	statusWarn    = core.StatusWarn
	statusBad     = core.StatusBad
	statusNeutral = core.StatusNeutral
)

const filterScanLimit = core.FilterScanLimit

const overviewCountLimit = core.OverviewCountLimit

// ===== Thin wrappers (delegate to core, preserve main package API) =====

func buildProjectionStats(host *projectionhost.Host) []projectionStat {
	return core.ProjectionStats(host)
}

func projectionStatusKind(status string) string {
	return core.ProjectionStatusKind(status)
}

func relativeTime(t time.Time) string {
	return core.RelativeTime(t)
}

func humanByteSize(bytes int) string {
	return core.HumanByteSize(bytes)
}

func renderPayload(r PayloadRenderer, evt event.Event) []byte {
	return core.RenderPayload(r, evt)
}

func prettyJSON(raw []byte) string {
	return core.PrettyJSON(raw)
}

func pushCursor(history, cursor string) string {
	return core.PushCursor(history, cursor)
}

func popCursor(history string) (string, string) {
	return core.PopCursor(history)
}

func paginationQuery(after, prevHistory string, pageSize int, extraParams string) string {
	return core.PaginationQuery(after, prevHistory, pageSize, extraParams)
}

func parsePageSize(r *http.Request, defaultSize int) int {
	return core.ParsePageSize(r, defaultSize)
}

func parseCursorParams(r *http.Request) (string, string, bool) {
	return core.ParseCursorParams(r)
}

func parseEventFilter(r *http.Request) eventFilter {
	return core.ParseEventFilter(r)
}

func (config Config) capabilities() Capabilities {
	return core.DetectCapabilities(config.coreConfig())
}

// hasEventRead is retained for the main package's internal use (buildNav).
// It delegates to the exported core.Capabilities.HasEventRead().

// coreConfig extracts the data-source subset of Config for core functions.
func (config Config) coreConfig() core.Config {
	return core.Config{
		EventSource:     config.EventSource,
		EventByIDLoader: config.EventByIDLoader,
		Journal:         config.Journal,
		SeekableJournal: config.SeekableJournal,
		StreamReader:    config.StreamReader,
		ProjectionHost:  config.ProjectionHost,
		DeadLetterStore: config.DeadLetterStore,
		CommandJournal:  config.CommandJournal,
		QueryJournal:    config.QueryJournal,
		SnapshotStore:   config.SnapshotStore,
		EventBus:        config.EventBus,
		PageSize:        config.PageSize,
		PayloadRenderer: config.PayloadRenderer,
	}
}

func (d *Dashboard) overviewStats(ctx context.Context) overviewStats {
	return core.FetchOverview(ctx, d.config.coreConfig())
}

func (d *Dashboard) buildDLQProjectionLinks(ctx context.Context) []dlqProjectionLink {
	return core.DLQProjectionLinks(ctx, d.config.coreConfig())
}

func (d *Dashboard) loadRecentEvents(ctx context.Context, after id.EventID, limit int) ([]event.Event, error) {
	return core.LoadRecentEvents(ctx, d.config.coreConfig(), after, limit)
}

func (d *Dashboard) loadFilteredEvents(
	ctx context.Context,
	after id.EventID,
	filter eventFilter,
	pageSize int,
) ([]event.Event, error) {
	return core.LoadFilteredEvents(ctx, d.config.coreConfig(), after, filter, pageSize)
}

func (d *Dashboard) loadEventByID(ctx context.Context, eventID id.EventID) (event.Event, error) {
	return core.LoadEventByID(ctx, d.config.coreConfig(), eventID)
}

func (d *Dashboard) findEventNeighbors(ctx context.Context, eventID id.EventID) (string, string) {
	return core.FindEventNeighbors(ctx, d.config.coreConfig(), eventID)
}

func (d *Dashboard) listStreamsPaged(r *http.Request) ([]listing.StreamListing, paginationState) {
	return core.ListStreamsPaged(r, d.config.coreConfig())
}
