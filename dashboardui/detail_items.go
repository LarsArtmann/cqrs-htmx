package dashboardui

import (
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/templ-components/display"
)

// Data-building helpers shared by the templ page components. Definition-item
// builders return plain data (the library escapes terms/details); the markup
// passed to defItemCopy/defItemRaw is trusted display HTML.

// defItemComponent builds a definition item whose detail is a templ
// component (library badges, copy buttons).
func defItemComponent(term string, component templ.Component) display.DefinitionItem {
	return display.DefinitionItem{Term: term, Detail: "", DetailComponent: component}
}

// monoSpan renders s as an inline mono span for definition list details.
func monoSpan(s string) string {
	return "<span class=\"mono\">" + s + "</span>"
}

// payloadBytes renders an event's payload through the configured renderer.
func payloadBytes(renderer PayloadRenderer, evt event.Event) []byte {
	return renderPayload(renderer, evt)
}

// combinedFilterSortParams joins the filter and sort query params preserved
// across pagination links.
func combinedFilterSortParams(filter eventFilter, sortBy sortState) string {
	combined := filter.ExtraParams()
	if sp := sortBy.extraParams(); sp != "" {
		if combined != "" {
			combined += "&" + sp
		} else {
			combined = sp
		}
	}

	return combined
}

// timelinePageState derives the pagination state for the aggregate timeline
// slice: HasNext/NextCursor from the slice, PageLen/PageStart/TotalCount for
// the "Showing X–Y of Z" label.
func timelinePageState(
	events []event.Event,
	pagedEvents []event.Event,
	page paginationState,
	hasNext bool,
) paginationState {
	timeline := page

	timeline.HasNext = hasNext
	if hasNext && len(pagedEvents) > 0 {
		timeline.NextCursor = pagedEvents[len(pagedEvents)-1].Version().String()
	}

	timeline.PageLen = len(pagedEvents)
	if len(pagedEvents) > 0 {
		timeline.PageStart = int(pagedEvents[0].Version().UInt64())
	}

	timeline.TotalCount = strconv.Itoa(len(events))

	return timeline
}

// eventCustomMetaItems builds definition items for custom event metadata.
func eventCustomMetaItems(custom map[event.MetadataKey]string) []display.DefinitionItem {
	items := make([]display.DefinitionItem, 0, len(custom))

	for k, v := range custom {
		items = append(items, defItem(string(k), v))
	}

	return items
}

// projectionDetailItems builds the projection detail definition list:
// checkpoint with copy button, status, and last error (or explicit "none").
func projectionDetailItems(proj projectionStat) []display.DefinitionItem {
	items := []display.DefinitionItem{
		defItemCopy(
			"Checkpoint",
			monoSpan(esc(truncate(proj.Checkpoint, listIDWidth))),
			proj.Checkpoint,
		),
		defItem("Status", proj.Status),
	}

	if proj.LastError != "" {
		items = append(items, defItem("Last Error", proj.LastError))
	} else {
		items = append(items, defItemRaw("Last Error", `<span class="muted">none</span>`))
	}

	return items
}

// prefixedActorID is the structural surface the dashboard needs from the
// distinct command, query, and event metadata actor-ID types.
type prefixedActorID interface {
	IsZero() bool
	PrefixedString() string
}

// metadataItems builds the correlation, causation, actor, and request
// definition items shared by the command, query, and event detail pages.
// Blank IDs and zero actors are omitted.
func metadataItems(corrID, causID, reqID string, actorID prefixedActorID) []display.DefinitionItem {
	items := make([]display.DefinitionItem, 0, 4)

	if corrID != "" {
		items = append(items, defItemCopy("Correlation ID", monoSpan(esc(corrID)), corrID))
	}

	if causID != "" {
		items = append(items, defItemCopy("Causation ID", monoSpan(esc(causID)), causID))
	}

	if !actorID.IsZero() {
		actorPrefixed := actorID.PrefixedString()
		items = append(items, defItemCopy("Actor ID", monoSpan(esc(actorPrefixed)), actorPrefixed))
	}

	if reqID != "" {
		items = append(items, defItemCopy("Request ID", monoSpan(esc(reqID)), reqID))
	}

	return items
}

// commandMetaItems builds the command detail metadata items: stream facts
// plus correlation, causation, and actor IDs when present.
func commandMetaItems(cmd *command.PersistedCommand) []display.DefinitionItem {
	meta := cmd.Metadata()

	items := []display.DefinitionItem{
		defItem("Command Type", string(cmd.Type())),
		defItem("Stream Type", string(cmd.StreamType())),
		defItemCopy("Stream ID", monoSpan(esc(cmd.StreamID().String())), cmd.StreamID().String()),
		defItem("Received At", cmd.ReceivedAt().Format("2006-01-02 15:04:05")),
		defItemCopy("Command ID", monoSpan(esc(cmd.ID().String())), cmd.ID().String()),
	}

	items = append(items, metadataItems(
		meta.CorrelationID.String(), meta.CausationID.String(),
		meta.RequestID.String(), meta.ActorID,
	)...)

	return items
}

// queryMetaItems builds the query detail metadata items.
func queryMetaItems(q *query.PersistedQuery) []display.DefinitionItem {
	meta := q.Metadata()

	items := []display.DefinitionItem{
		defItem("Query Type", string(q.Type())),
		defItem("Received At", q.ReceivedAt().Format("2006-01-02 15:04:05")),
		defItemCopy("Request ID", monoSpan(esc(q.ID().String())), q.ID().String()),
	}

	items = append(items, metadataItems(
		meta.CorrelationID.String(), meta.CausationID.String(),
		meta.RequestID.String(), meta.ActorID,
	)...)

	return items
}

// dlqEntryErrorItems builds the DLQ entry detail error definition items.
func dlqEntryErrorItems(entry projectionhost.DeadLetterEntry) []display.DefinitionItem {
	items := []display.DefinitionItem{
		defItem("Event Type", entry.EventType),
		defItem("Event ID", entry.EventID),
	}

	if entry.StreamID != "" {
		items = append(
			items,
			defItemCopy("Stream ID", monoSpan(esc(entry.StreamID)), entry.StreamID),
		)
	}

	items = append(items, defItem("Failed At", entry.FailedAt.Format("2006-01-02 15:04:05")))

	if entry.ErrorFamily != "" {
		items = append(items, defItemComponent("Error Family", badge(entry.ErrorFamily, display.BadgeError)))
	}

	if entry.ErrorCode != "" {
		items = append(items, defItem("Error Code", entry.ErrorCode))
	}

	return items
}

// snapshotMetaItems builds the snapshot detail metadata items.
func snapshotMetaItems(snap *snapshot.Snapshot) []display.DefinitionItem {
	return []display.DefinitionItem{
		defItem("Stream Type", string(snap.StreamType)),
		defItemCopy("Stream ID", monoSpan(esc(snap.StreamID.String())), snap.StreamID.String()),
		defItem("Version", snap.Version.String()),
		defItem("Created At", snap.CreatedAt.Format(time.RFC3339)),
		defItem("State Size", humanByteSize(len(snap.State))),
	}
}

// snapshotStateText renders snapshot state bytes as display text (pretty
// JSON when possible, falling back to the raw bytes). The result is NOT
// escaped — the templ template escapes it.
func (d *Dashboard) snapshotStateText(state []byte) string {
	if len(state) == 0 {
		return "(empty)"
	}

	out, err := d.config.PayloadRenderer.Render(state, codec.EncodingJSON)
	if err == nil && len(out) > 0 {
		return string(out)
	}

	return string(state)
}

// eventMetaItems builds the metadata definition items for an event detail
// page: stream/version/encoding facts plus correlation, causation, actor,
// and request IDs when present.
func eventMetaItems(evt event.Event, meta event.Metadata) []display.DefinitionItem {
	items := []display.DefinitionItem{
		defItem("Stream Type", string(evt.StreamType())),
		defItemCopy("Stream ID", monoSpan(esc(evt.StreamID().String())), evt.StreamID().String()),
		defItem("Version", evt.Version().String()),
		defItem("Schema Version", strconv.FormatInt(int64(evt.SchemaVersion()), 10)),
		defItem("Encoding", string(evt.Encoding())),
		defItem("Occurred At", evt.OccurredAt().Format(time.RFC3339)),
	}

	items = append(items, metadataItems(
		meta.CorrelationID.String(), meta.CausationID.String(),
		meta.RequestID.String(), meta.ActorID,
	)...)

	if deadline, ok := evt.Deadline(); ok {
		items = append(items, defItem("Deadline", deadline.Format(time.RFC3339)))
	}

	return items
}

// recentEventTime picks the overview recent-events row time display: the
// relative time when the timestamp is known, the raw display otherwise.
func recentEventTime(e RecentEvent) string {
	if !e.OccurredAt.IsZero() {
		return relativeTime(e.OccurredAt)
	}

	return e.Time
}
