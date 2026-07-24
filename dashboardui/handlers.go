package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func notImplemented(w http.ResponseWriter, panel string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(
		w,
		"<div style=\"padding:40px;text-align:center;color:#64748b\"><h2>%s</h2><p>This panel is coming soon.</p></div>",
		panel,
	)
}

// ===== Event Stream Browser =====

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
				return nil, fmt.Errorf("scan journal for event: %w", err)
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

		return nil, fmt.Errorf("event %s not found in journal scan", eventID)
	}

	if d.cfg.Journal != nil {
		all, err := d.cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("read journal: %w", err)
		}

		for _, evt := range all {
			if evt.ID() == eventID {
				return evt, nil
			}
		}
	}

	return nil, fmt.Errorf("no event source available to load event %s", eventID)
}

func (d *Dashboard) renderEventDetail(p pageData, evt event.Event) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		payload := renderPayload(d.cfg.PayloadRenderer, evt)
		meta := evt.Metadata()

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px"><code>%s</code></h2>`, esc(string(evt.Type())))
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.85em;font-family:monospace">%s</div>`, esc(evt.ID().String()))
		b.WriteString(`</div>`)

		b.WriteString(`<div style="display:grid;grid-template-columns:1fr 1fr;gap:24px">`)

		// Metadata panel
		b.WriteString(`<div><h4 style="margin:0 0 8px">Metadata</h4><table style="width:100%%;border-collapse:collapse;font-size:0.88em">`)
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
			b.WriteString(`<h4 style="margin:16px 0 8px">Custom Metadata</h4><table style="width:100%%;border-collapse:collapse;font-size:0.88em">`)
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

type eventRow struct {
	Time     time.Time
	Type     string
	StreamID string
	Stream   string
	Version  string
	EventID  string
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

// ===== Aggregate Browser =====

func (d *Dashboard) aggregatesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Aggregates", "/aggregates", r)

	var listings []listing.StreamListing

	if d.cfg.StreamReader != nil {
		page, err := d.cfg.StreamReader.List(r.Context(), listing.ListOptions{Limit: uint(d.cfg.PageSize)})
		if err == nil && page != nil {
			listings = page.Items
		}
	}

	html := d.renderAggregates(p, listings)
	renderPage(w, r, html)
}

func (d *Dashboard) aggregateDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType := r.PathValue("type")
	streamID := r.PathValue("id")

	ref, err := StreamRefFromID(streamType, streamID)
	if err != nil {
		http.Error(w, "invalid stream reference: "+err.Error(), http.StatusBadRequest)

		return
	}

	events, err := d.cfg.EventSource.Load(r.Context(), ref)
	if err != nil {
		http.Error(w, "failed to load aggregate: "+err.Error(), http.StatusInternalServerError)

		return
	}

	p := d.page("Aggregate: "+streamType+"/"+truncate(streamID, 12), "/aggregates", r)

	link := func(href, label string) string {
		return fmt.Sprintf(`<a href="%s" style="color:var(--accent);text-decoration:none">%s</a>`, href, label)
	}

	html := d.renderAggregateDetail(p, ref, events, link)
	renderPage(w, r, html)
}

func (d *Dashboard) renderAggregateDetail(
	p pageData,
	ref id.StreamRef,
	events []event.Event,
	link func(string, string) string,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px">%s: <code>%s</code></h2>`, esc(string(ref.Type)), esc(ref.ID.String()))
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">%d events · current version %s</div>`,
			len(events), latestVersion(events))
		b.WriteString(`</div>`)

		if d.caps.EventSource && len(events) > 0 {
			maxVer := events[len(events)-1].Version().Int()
			fmt.Fprintf(&b,
				`<div style="margin-bottom:16px"><a href="%s/time-travel/%s/%s" style="display:inline-flex;align-items:center;gap:6px;padding:6px 12px;border:1px solid var(--border);border-radius:6px;text-decoration:none;color:var(--accent);font-size:0.85em">Inspect time-travel for this aggregate</a></div>`,
				p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()))
			_ = maxVer
		}

		if len(events) == 0 {
			b.WriteString(`<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No events</h3><p>This aggregate has no recorded events.</p></div>`)

			return b.String()
		}

		b.WriteString(`<h4 style="margin-bottom:8px">Event Timeline</h4>`)
		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Version</th><th style="padding:8px">Type</th>`)
		b.WriteString(`<th style="padding:8px">Occurred At</th><th style="padding:8px">Event ID</th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, evt := range events {
			fmt.Fprintf(&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-weight:600">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
			</tr>`,
				esc(evt.Version().String()),
				link(fmt.Sprintf("%s/events/%s", p.BasePath, esc(evt.ID().String())), esc(string(evt.Type()))),
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
				fmt.Sprintf(`<code style="font-size:0.8em">%s</code>`, truncate(evt.ID().String(), 20)),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}

func latestVersion(events []event.Event) string {
	if len(events) == 0 {
		return "0"
	}

	return events[len(events)-1].Version().String()
}

func (d *Dashboard) renderAggregates(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		if len(listings) == 0 {
			return `<div style="padding:40px;text-align:center;color:#64748b"><h3>No aggregates found</h3></div>`
		}

		var (
			rows      string
			rowsSb131 strings.Builder
		)
		for _, l := range listings {
			fmt.Fprintf(&rowsSb131, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px">%d</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
			</tr>`,
				truncate(l.ID.String(), 24),
				l.Type,
				l.Version.String(),
				l.EventCount,
				l.LastEventAt.Format("2006-01-02 15:04:05"))
		}

		rows += rowsSb131.String()

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Aggregates</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">ID</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Version</th>
					<th style="padding:8px">Events</th>
					<th style="padding:8px">Last Event</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows)
	})
}

// ===== Projection Dashboard =====

func (d *Dashboard) projectionsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Projections", "/projections", r)

	var projs []projectionStat

	if d.cfg.ProjectionHost != nil {
		lagPerProj := d.cfg.ProjectionHost.LagPerProjection()
		for _, ws := range d.cfg.ProjectionHost.Status() {
			lag := lagPerProj[ws.Name]
			projs = append(projs, projectionStat{
				Name:       ws.Name,
				Status:     string(ws.Status),
				Lag:        lag.String(),
				Processed:  ws.Processed,
				Errors:     ws.Errors,
				StatusKind: projectionStatusKind(string(ws.Status)),
			})
		}
	}

	html := d.renderProjections(p, projs)
	renderPage(w, r, html)
}

func (d *Dashboard) projectionResetHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.ProjectionHost == nil {
		http.Error(w, "projection host not configured", http.StatusBadRequest)

		return
	}

	name := r.PathValue("name")
	if err := d.cfg.ProjectionHost.Reset(r.Context(), name); err != nil {
		triggerToast(w, "err", "Reset failed: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	triggerToast(w, "ok", "Projection reset")
	redirect(w, r, d.cfg.BasePath+"/projections")
}

func (d *Dashboard) renderProjections(p pageData, projs []projectionStat) string {
	return d.renderLayout(p, func() string {
		if len(projs) == 0 {
			return `<div style="padding:40px;text-align:center;color:#64748b"><h3>No projections registered</h3></div>`
		}

		var rows string

		var rowsSb209 strings.Builder

		for _, proj := range projs {
			color := "#64748b"

			switch proj.StatusKind {
			case "good":
				color = "#16a34a"
			case "warn":
				color = "#d97706"
			case "bad":
				color = "#dc2626"
			}

			fmt.Fprintf(&rowsSb209, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-weight:500">%s</td>
				<td style="padding:8px"><span style="color:%s;font-weight:600">%s</span></td>
				<td style="padding:8px;font-family:monospace">%s</td>
				<td style="padding:8px">%d</td>
				<td style="padding:8px">%d</td>
			</tr>`, proj.Name, color, proj.Status, proj.Lag, proj.Processed, proj.Errors)
		}

		rows += rowsSb209.String()

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Projections</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Name</th>
					<th style="padding:8px">Status</th>
					<th style="padding:8px">Lag</th>
					<th style="padding:8px">Processed</th>
					<th style="padding:8px">Errors</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows)
	})
}

// ===== Dead-Letter Queue =====

func (d *Dashboard) dlqIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Dead Letters", "/dead-letters", r)
	html := d.renderLayout(p, func() string {
		return `<div style="padding:40px;text-align:center;color:#64748b"><h3>Dead-Letter Queue</h3><p>Select a projection to view its dead letters.</p></div>`
	})
	renderPage(w, r, html)
}

func (d *Dashboard) dlqDetailHandler(w http.ResponseWriter, r *http.Request) {
	proj := r.PathValue("projection")

	var entries []projectionhost.DeadLetterEntry

	if d.cfg.DeadLetterStore != nil {
		var err error

		entries, err = d.cfg.DeadLetterStore.List(r.Context(), proj)
		if err != nil {
			http.Error(w, "failed to list dead letters: "+err.Error(), http.StatusInternalServerError)

			return
		}
	}

	p := d.page("Dead Letters: "+proj, "/dead-letters", r)
	html := d.renderDLQ(p, proj, entries)
	renderPage(w, r, html)
}

func (d *Dashboard) dlqReplayHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.ProjectionHost == nil {
		http.Error(w, "projection host not configured", http.StatusBadRequest)

		return
	}

	proj := r.PathValue("projection")

	result, err := d.cfg.ProjectionHost.ReplayDeadLetters(r.Context(), proj)
	if err != nil {
		triggerToast(w, "err", "Replay failed: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	msg := fmt.Sprintf("Replayed %d, %d still failing", len(result.Replayed), len(result.StillFailing))
	triggerToast(w, "ok", msg)
	redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
}

func (d *Dashboard) dlqDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.DeadLetterStore == nil {
		http.Error(w, "dead letter store not configured", http.StatusBadRequest)

		return
	}

	proj := r.PathValue("projection")

	eventID := r.PathValue("eventID")
	if err := d.cfg.DeadLetterStore.Delete(r.Context(), proj, eventID); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	triggerToast(w, "ok", "Dead letter deleted")
	redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
}

func (d *Dashboard) dlqPurgeHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.DeadLetterStore == nil {
		http.Error(w, "dead letter store not configured", http.StatusBadRequest)

		return
	}

	proj := r.PathValue("projection")
	if err := d.cfg.DeadLetterStore.Purge(r.Context(), proj); err != nil {
		triggerToast(w, "err", "Purge failed: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	triggerToast(w, "ok", "Dead letters purged")
	redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
}

func (d *Dashboard) renderDLQ(p pageData, proj string, entries []projectionhost.DeadLetterEntry) string {
	return d.renderLayout(p, func() string {
		if len(entries) == 0 {
			return fmt.Sprintf(
				`<div style="padding:40px;text-align:center;color:#64748b"><h3>No dead letters for %s</h3></div>`,
				proj,
			)
		}

		var (
			rows      string
			rowsSb326 strings.Builder
		)
		for _, e := range entries {
			fmt.Fprintf(&rowsSb326, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px;color:#dc2626">%s</td>
				<td style="padding:8px">%s</td>
			</tr>`,
				e.FailedAt.Format("2006-01-02 15:04:05"),
				e.EventType,
				truncate(e.Error, 60),
				e.ErrorFamily)
		}

		rows += rowsSb326.String()

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Dead Letters: %s</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Failed At</th>
					<th style="padding:8px">Event Type</th>
					<th style="padding:8px">Error</th>
					<th style="padding:8px">Family</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, proj, rows)
	})
}

// ===== Command/Query Audit =====

func (d *Dashboard) commandsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Commands", "/commands", r)

	var cmds []*command.PersistedCommand

	if d.cfg.CommandJournal != nil {
		var err error

		if seekable, ok := d.cfg.CommandJournal.(command.SeekableCommandJournal); ok {
			cmds, err = seekable.ReadFrom(r.Context(), id.CommandID{}, d.cfg.PageSize)
		} else {
			cmds, err = d.cfg.CommandJournal.ReadAll(r.Context())
			if err == nil && len(cmds) > d.cfg.PageSize {
				cmds = cmds[:d.cfg.PageSize]
			}
		}

		if err != nil {
			http.Error(w, "failed to load commands: "+err.Error(), http.StatusInternalServerError)

			return
		}
	}

	html := d.renderCommands(p, cmds)
	renderPage(w, r, html)
}

func (d *Dashboard) renderCommands(p pageData, cmds []*command.PersistedCommand) string {
	return d.renderLayout(p, func() string {
		if len(cmds) == 0 {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No commands recorded</h3><p>Commands will appear here as they are dispatched.</p></div>`
		}

		var rows strings.Builder

		for _, cmd := range cmds {
			fmt.Fprintf(&rows, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.8em;color:var(--muted)">%s</td>
			</tr>`,
				esc(cmd.ReceivedAt().Format("2006-01-02 15:04:05")),
				esc(string(cmd.Type())),
				esc(string(cmd.StreamType())),
				esc(cmd.StreamID().String()),
				truncate(cmd.ID().String(), 20),
			)
		}

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Command Audit</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Received At</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Stream Type</th>
					<th style="padding:8px">Stream ID</th>
					<th style="padding:8px">Command ID</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows.String())
	})
}

func (d *Dashboard) queriesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Queries", "/queries", r)

	var queries []*query.PersistedQuery

	if d.cfg.QueryJournal != nil {
		var err error

		if seekable, ok := d.cfg.QueryJournal.(query.SeekableQueryJournal); ok {
			queries, err = seekable.ReadQueriesFrom(r.Context(), id.RequestID{}, d.cfg.PageSize)
		} else {
			queries, err = d.cfg.QueryJournal.ReadAllQueries(r.Context())
			if err == nil && len(queries) > d.cfg.PageSize {
				queries = queries[:d.cfg.PageSize]
			}
		}

		if err != nil {
			http.Error(w, "failed to load queries: "+err.Error(), http.StatusInternalServerError)

			return
		}
	}

	html := d.renderQueries(p, queries)
	renderPage(w, r, html)
}

func (d *Dashboard) renderQueries(p pageData, queries []*query.PersistedQuery) string {
	return d.renderLayout(p, func() string {
		if len(queries) == 0 {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No queries recorded</h3><p>Queries will appear here as they are executed.</p></div>`
		}

		var rows strings.Builder

		for _, q := range queries {
			fmt.Fprintf(&rows, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px;font-family:monospace;font-size:0.8em;color:var(--muted)">%s</td>
			</tr>`,
				esc(q.ReceivedAt().Format("2006-01-02 15:04:05")),
				esc(string(q.Type())),
				truncate(q.ID().String(), 20),
			)
		}

		return fmt.Sprintf(`
			<h3 style="margin-bottom:12px">Query Audit</h3>
			<table style="width:100%%;border-collapse:collapse">
				<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">
					<th style="padding:8px">Received At</th>
					<th style="padding:8px">Type</th>
					<th style="padding:8px">Request ID</th>
				</tr></thead>
				<tbody>%s</tbody>
			</table>`, rows.String())
	})
}

// ===== Time-Travel =====

func (d *Dashboard) timeTravelIndexHandler(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, "Time Travel")
}

func (d *Dashboard) timeTravelDetailHandler(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, "Time Travel Detail")
}

// ===== Snapshots =====

func (d *Dashboard) snapshotsIndexHandler(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, "Snapshots")
}

func (d *Dashboard) snapshotDetailHandler(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, "Snapshot Detail")
}

func (d *Dashboard) snapshotDeleteHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// Ensure we use the imports.
var (
	_ = id.NewAggregateID
	_ = event.Type("")
)
