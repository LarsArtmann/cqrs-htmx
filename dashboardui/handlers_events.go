package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func (d *Dashboard) eventsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Events", "/events", r)

	events, err := d.loadRecentEvents(r.Context(), d.cfg.PageSize)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "failed to load events")

		return
	}

	html := d.renderEvents(p, events)
	renderPage(w, r, html)
}

func (d *Dashboard) eventDetailHandler(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.PathValue("id")

	eventID, err := id.ParseEventID(eventIDStr)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "invalid event ID")

		return
	}

	evt, err := d.loadEventByID(r.Context(), eventID)
	if err != nil {
		renderError(w, r, http.StatusNotFound, "event not found")

		return
	}

	p := d.page("Event: "+truncate(string(evt.Type()), eventTypeWidth), "/events", r)
	html := d.renderEventDetail(p, evt)
	renderPage(w, r, html)
}

//nolint:cyclop // journal fallback
func (d *Dashboard) loadEventByID(ctx context.Context, eventID id.EventID) (event.Event, error) {
	if d.cfg.EventByIDLoader != nil {
		evt, err := d.cfg.EventByIDLoader.LoadByEventID(ctx, eventID)
		if err != nil {
			var zero event.Event

			return zero, errorfamily.WrapInfrastructure(err,
				"dashboardui.event_detail.load_failed", "load event by ID")
		}

		return evt, nil
	}

	if d.cfg.SeekableJournal != nil {
		const scanLimit = 5000

		var after id.EventID

		for {
			batch, err := d.cfg.SeekableJournal.ReadFrom(ctx, after, scanLimit)
			if err != nil {
				return nil, errorfamily.WrapInfrastructure(err,
					"dashboardui.event_detail.scan_failed", "scan journal for event")
			}

			for _, evt := range batch {
				if evt.ID() == eventID {
					return evt, nil
				}
			}

			if len(batch) < scanLimit {
				break
			}

			after = batch[len(batch)-1].ID()
		}

		return nil, errorfamily.Newf(event.Rejection,
			"dashboardui.event_detail.not_found", "event %s not found in journal scan", eventID)
	}

	if d.cfg.Journal != nil {
		all, err := d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.event_detail.read_failed", "read journal")
		}

		for _, evt := range all {
			if evt.ID() == eventID {
				return evt, nil
			}
		}
	}

	return nil, errorfamily.Newf(event.Infrastructure,
		"dashboardui.event_detail.no_source", "no event source available to load event %s", eventID)
}

func (d *Dashboard) renderEventDetail(p pageData, evt event.Event) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		payload := renderPayload(d.cfg.PayloadRenderer, evt)
		meta := evt.Metadata()

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2><code>%s</code></h2>`, esc(string(evt.Type())))
		fmt.Fprintf(&b, `<div class="page-subtitle mono">%s</div>`, esc(evt.ID().String()))
		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)

		b.WriteString(`<div><h4>Metadata</h4><table class="meta-table">`)
		metaRow(&b, "Stream Type", esc(string(evt.StreamType())))
		metaRow(&b, "Stream ID", esc(evt.StreamID().String()))
		metaRow(&b, "Version", esc(evt.Version().String()))
		metaRow(&b, "Schema Version", esc(fmt.Sprintf("%d", evt.SchemaVersion())))
		metaRow(&b, "Encoding", esc(string(evt.Encoding())))
		metaRow(&b, "Occurred At", esc(evt.OccurredAt().Format(time.RFC3339)))

		if corrID := meta.CorrelationID.String(); corrID != "" {
			metaRow(&b, "Correlation ID", esc(corrID))
		}

		if causID := meta.CausationID.String(); causID != "" {
			metaRow(&b, "Causation ID", esc(causID))
		}

		if userID := meta.UserID.String(); userID != "" {
			metaRow(&b, "User ID", esc(userID))
		}

		if reqID := meta.RequestID.String(); reqID != "" {
			metaRow(&b, "Request ID", esc(reqID))
		}

		if deadline, ok := evt.Deadline(); ok {
			metaRow(&b, "Deadline", esc(deadline.Format(time.RFC3339)))
		}

		b.WriteString(`</table>`)

		if len(meta.Custom) > 0 {
			b.WriteString(`<h4>Custom Metadata</h4><table class="meta-table">`)

			for k, v := range meta.Custom {
				metaRow(&b, esc(string(k)), esc(v))
			}

			b.WriteString(`</table>`)
		}

		b.WriteString(`</div>`)

		b.WriteString(`<div><h4>Payload</h4>`)
		fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, esc(string(payload)))
		b.WriteString(`</div>`)

		b.WriteString(`</div>`)

		return b.String()
	})
}

func (d *Dashboard) loadRecentEvents(ctx context.Context, limit int) ([]event.Event, error) {
	if d.cfg.SeekableJournal != nil {
		events, err := d.cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, limit)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.recent_events.read_failed", "read recent events")
		}

		return events, nil
	}

	if d.cfg.Journal != nil {
		all, err := d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(err,
				"dashboardui.recent_events.read_all_failed", "read all events")
		}

		if len(all) > limit {
			all = all[:limit]
		}

		return all, nil
	}

	return nil, nil
}

func (d *Dashboard) renderEvents(p pageData, events []event.Event) string {
	return d.renderLayout(p, func() string {
		if len(events) == 0 {
			return emptyState("No events yet", "Events will appear here as they are committed to the store.")
		}

		var rows strings.Builder

		for _, evt := range events {
			fmt.Fprintf(
				&rows,
				`<tr><td class="mono">%s</td><td><a href="%s/events/%s"><code>%s</code></a></td><td class="mono">%s</td><td>%s</td><td>%s</td></tr>`,
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
				p.BasePath,
				esc(evt.ID().String()),
				esc(string(evt.Type())),
				esc(truncate(evt.StreamID().String(), listIDWidth)),
				esc(string(evt.StreamType())),
				esc(evt.Version().String()),
			)
		}

		return fmt.Sprintf(
			`<h3>Event Stream</h3><table class="data-table"><thead><tr><th scope="col">Time</th><th scope="col">Type</th><th scope="col">Stream ID</th><th scope="col">Stream Type</th><th scope="col">Version</th></tr></thead><tbody>%s</tbody></table>`,
			rows.String(),
		)
	})
}
