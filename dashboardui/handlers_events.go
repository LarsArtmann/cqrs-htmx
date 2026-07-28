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
		http.Error(w, "failed to load events: "+err.Error(), http.StatusInternalServerError)

		return
	}

	html := d.renderEvents(p, events)
	renderPage(w, r, html)
}

func (d *Dashboard) eventDetailHandler(w http.ResponseWriter, r *http.Request) {
	eventIDStr := r.PathValue("id")

	eventID, err := id.ParseEventID(eventIDStr)
	if err != nil {
		http.Error(w, "invalid event ID: "+err.Error(), http.StatusBadRequest)

		return
	}

	evt, err := d.loadEventByID(r.Context(), eventID)
	if err != nil {
		http.Error(w, "event not found: "+err.Error(), http.StatusNotFound)

		return
	}

	p := d.page("Event: "+truncate(string(evt.Type()), 30), "/events", r)
	html := d.renderEventDetail(p, evt)
	renderPage(w, r, html)
}

// loadEventByID retrieves a single event. Uses EventByIDLoader if available
// (O(1)), otherwise scans the journal.
func (d *Dashboard) loadEventByID(ctx context.Context, eventID id.EventID) (event.Event, error) {
	if d.cfg.EventByIDLoader != nil {
		return d.cfg.EventByIDLoader.LoadByEventID(ctx, eventID)
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

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px"><code>%s</code></h2>`, esc(string(evt.Type())))
		fmt.Fprintf(
			&b,
			`<div style="color:var(--muted);font-size:0.85em;font-family:monospace">%s</div>`,
			esc(evt.ID().String()),
		)
		b.WriteString(`</div>`)

		b.WriteString(`<div style="display:grid;grid-template-columns:1fr 1fr;gap:24px">`)

		// Metadata panel
		b.WriteString(
			`<div><h4 style="margin:0 0 8px">Metadata</h4><table style="width:100%%;border-collapse:collapse;font-size:0.88em">`,
		)
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
			b.WriteString(
				`<h4 style="margin:16px 0 8px">Custom Metadata</h4><table style="width:100%%;border-collapse:collapse;font-size:0.88em">`,
			)

			for k, v := range meta.Custom {
				metaRow(&b, esc(string(k)), esc(v))
			}

			b.WriteString(`</table>`)
		}

		b.WriteString(`</div>`)

		// Payload panel
		b.WriteString(`<div><h4 style="margin:0 0 8px">Payload</h4>`)
		fmt.Fprintf(
			&b,
			`<pre style="background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:16px;overflow-x:auto;font-size:0.85em;line-height:1.5;margin:0"><code>%s</code></pre>`,
			esc(string(payload)),
		)
		b.WriteString(`</div>`)

		b.WriteString(`</div>`)

		return b.String()
	})
}

func (d *Dashboard) loadRecentEvents(ctx context.Context, limit int) ([]event.Event, error) {
	if d.cfg.SeekableJournal != nil {
		return d.cfg.SeekableJournal.ReadFrom(ctx, id.EventID{}, limit)
	}

	if d.cfg.Journal != nil {
		all, err := d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, err
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
			return `<div style="padding:40px;text-align:center;color:#64748b"><h3>No events yet</h3><p>Events will appear here as they are committed to the store.</p></div>`
		}

		var (
			rows     string
			rowsSb72 strings.Builder
		)
		for _, evt := range events {
			fmt.Fprintf(&rowsSb72, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px">%s</td>
			</tr>`,
				evt.OccurredAt().Format("2006-01-02 15:04:05"),
				evt.Type(),
				truncate(evt.StreamID().String(), 24),
				evt.StreamType(),
				evt.Version().String())
		}

		rows += rowsSb72.String()

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Event Stream</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Time</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Stream ID</th>
					<th style="padding:8px">Stream Type</th>
					<th style="padding:8px">Version</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows)
	})
}
