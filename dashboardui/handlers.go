package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

func notImplemented(w http.ResponseWriter, panel string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(
		w,
		"<div style=\"padding:40px;text-align:center;color:#64748b\"><h2>%s</h2><p>This panel is coming soon.</p></div>",
		panel,
	)
}

func streamRefFromRequest(r *http.Request) (id.StreamRef, error) {
	return StreamRefFromID(r.PathValue("type"), r.PathValue("id"))
}

func (d *Dashboard) loadStreamFromRequest(
	r *http.Request,
) (id.StreamRef, []event.Event, error) {
	ref, err := streamRefFromRequest(r)
	if err != nil {
		return id.StreamRef{}, nil, fmt.Errorf("invalid stream reference: %w", err)
	}

	events, err := d.cfg.EventSource.Load(r.Context(), ref)
	if err != nil {
		return id.StreamRef{}, nil, fmt.Errorf("failed to load aggregate: %w", err)
	}

	return ref, events, nil
}

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

	ref, events, err := d.loadStreamFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

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
		fmt.Fprintf(
			&b,
			`<h2 style="margin:0 0 4px">%s: <code>%s</code></h2>`,
			esc(string(ref.Type)),
			esc(ref.ID.String()),
		)
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">%d events · current version %s</div>`,
			len(events), latestVersion(events))
		b.WriteString(`</div>`)

		if d.caps.EventSource && len(events) > 0 {
			maxVer := events[len(events)-1].Version().Int()

			fmt.Fprintf(
				&b,
				`<div style="margin-bottom:16px"><a href="%s/time-travel/%s/%s" style="display:inline-flex;align-items:center;gap:6px;padding:6px 12px;border:1px solid var(--border);border-radius:6px;text-decoration:none;color:var(--accent);font-size:0.85em">Inspect time-travel for this aggregate</a></div>`,
				p.BasePath,
				esc(string(ref.Type)),
				esc(ref.ID.String()),
			)

			_ = maxVer
		}

		if len(events) == 0 {
			b.WriteString(
				`<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No events</h3><p>This aggregate has no recorded events.</p></div>`,
			)

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
	p := d.page("Time Travel", "/time-travel", r)

	var listings []listing.StreamListing

	if d.cfg.StreamReader != nil {
		page, err := d.cfg.StreamReader.List(r.Context(), listing.ListOptions{Limit: uint(d.cfg.PageSize)})
		if err == nil && page != nil {
			listings = page.Items
		}
	}

	html := d.renderTimeTravelIndex(p, listings)
	renderPage(w, r, html)
}

func (d *Dashboard) renderTimeTravelIndex(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div style="margin-bottom:16px">`)
		b.WriteString(
			`<p style="color:var(--muted);margin:0">Inspect an aggregate at any point in its history. Slide through versions to see the state at each step.</p>`,
		)
		b.WriteString(`</div>`)

		if len(listings) == 0 {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No aggregates found</h3><p>Configure a StreamReader to list aggregates for time-travel inspection.</p></div>`
		}

		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Type</th><th style="padding:8px">ID</th>`)
		b.WriteString(`<th style="padding:8px">Current Version</th><th style="padding:8px"></th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, l := range listings {
			fmt.Fprintf(&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px"><a href="%s/time-travel/%s/%s" style="color:var(--accent);text-decoration:none">Inspect</a></td>
			</tr>`,
				esc(string(l.Type)),
				esc(truncate(l.ID.String(), 24)),
				esc(l.Version.String()),
				p.BasePath,
				esc(string(l.Type)),
				esc(l.ID.String()),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}

func (d *Dashboard) timeTravelDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType := r.PathValue("type")
	streamID := r.PathValue("id")

	ref, allEvents, err := d.loadStreamFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if len(allEvents) == 0 {
		p := d.page("Time Travel: "+streamType, "/time-travel", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No events</h3></div>`
		}))

		return
	}

	maxVersion := allEvents[len(allEvents)-1].Version()

	// Parse requested version from query param, default to latest.
	requestedVersion := maxVersion

	if v := r.URL.Query().Get("v"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			requestedVersion = event.Version(parsed)
		}
	}

	if requestedVersion > maxVersion {
		requestedVersion = maxVersion
	}

	// Load events up to the requested version.
	eventsToVersion, err := d.cfg.EventSource.LoadToVersion(r.Context(), ref, requestedVersion)
	if err != nil {
		http.Error(w, "failed to load version: "+err.Error(), http.StatusInternalServerError)

		return
	}

	p := d.page("Time Travel: "+streamType+"/"+truncate(streamID, 12), "/time-travel", r)
	html := d.renderTimeTravelDetail(p, ref, eventsToVersion, requestedVersion, maxVersion)
	renderPage(w, r, html)
}

func (d *Dashboard) renderTimeTravelDetail(
	p pageData,
	ref id.StreamRef,
	events []event.Event,
	currentVersion event.Version,
	maxVersion event.Version,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px">Time Travel: <code>%s</code></h2>`, esc(ref.ID.String()))
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">Viewing version %d of %d</div>`,
			currentVersion.Int(), maxVersion.Int())
		b.WriteString(`</div>`)

		// Version slider.
		fmt.Fprintf(
			&b,
			`<div style="margin-bottom:24px;padding:16px;background:var(--surface);border:1px solid var(--border);border-radius:8px">`,
		)
		fmt.Fprintf(&b, `<label style="display:block;margin-bottom:8px;font-weight:600">Version</label>`)

		// Generate version links.
		b.WriteString(`<div style="display:flex;flex-wrap:wrap;gap:4px">`)

		for v := event.Version(1); v <= maxVersion; v++ {
			style := "padding:4px 10px;border:1px solid var(--border);border-radius:4px;text-decoration:none;color:var(--muted);font-size:0.85em"

			if v == currentVersion {
				style = "padding:4px 10px;border:1px solid var(--accent);border-radius:4px;text-decoration:none;color:white;background:var(--accent);font-size:0.85em;font-weight:600"
			}

			fmt.Fprintf(&b, `<a href="%s/time-travel/%s/%s?v=%d" style="%s">%d</a>`,
				p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()), v.Int(), style, v.Int())
		}

		b.WriteString(`</div></div>`)

		// Event timeline up to selected version.
		b.WriteString(`<h4 style="margin-bottom:8px">Events Through Version `)
		fmt.Fprintf(&b, `%d`, currentVersion.Int())
		b.WriteString(`</h4>`)

		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Version</th><th style="padding:8px">Type</th>`)
		b.WriteString(`<th style="padding:8px">Occurred At</th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, evt := range events {
			fmt.Fprintf(&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-weight:600">%s</td>
				<td style="padding:8px"><a href="%s/events/%s" style="color:var(--accent);text-decoration:none"><code>%s</code></a></td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
			</tr>`,
				esc(evt.Version().String()),
				p.BasePath, esc(evt.ID().String()), esc(string(evt.Type())),
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}

// ===== Snapshots =====

func (d *Dashboard) snapshotsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Snapshots", "/snapshots", r)

	var listings []listing.StreamListing

	if d.cfg.StreamReader != nil {
		page, err := d.cfg.StreamReader.List(r.Context(), listing.ListOptions{Limit: uint(d.cfg.PageSize)})
		if err == nil && page != nil {
			listings = page.Items
		}
	}

	html := d.renderSnapshotsIndex(p, listings)
	renderPage(w, r, html)
}

func (d *Dashboard) renderSnapshotsIndex(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(
			`<p style="color:var(--muted);margin-bottom:16px">Inspect snapshot state for any aggregate. Snapshots store a point-in-time cache of aggregate state to accelerate loading.</p>`,
		)

		if len(listings) == 0 {
			b.WriteString(`<div style="padding:40px;text-align:center;color:var(--muted)">`)
			b.WriteString(`<h3>No aggregates found</h3>`)
			b.WriteString(`<p>Configure a StreamReader to browse snapshots by aggregate.</p>`)
			b.WriteString(`</div>`)

			return b.String()
		}

		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Type</th><th style="padding:8px">ID</th>`)
		b.WriteString(`<th style="padding:8px">Version</th><th style="padding:8px"></th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, l := range listings {
			fmt.Fprintf(&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px"><a href="%s/snapshots/%s/%s" style="color:var(--accent);text-decoration:none">View</a></td>
			</tr>`,
				esc(string(l.Type)),
				esc(truncate(l.ID.String(), 24)),
				esc(l.Version.String()),
				p.BasePath,
				esc(string(l.Type)),
				esc(l.ID.String()),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}

func (d *Dashboard) snapshotDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType := r.PathValue("type")
	streamID := r.PathValue("id")

	ref, err := streamRefFromRequest(r)
	if err != nil {
		http.Error(w, "invalid stream reference: "+err.Error(), http.StatusBadRequest)

		return
	}

	snap, err := d.cfg.SnapshotStore.Load(r.Context(), ref)
	if err != nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, 12), "/snapshots", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return fmt.Sprintf(`<div style="padding:40px;text-align:center;color:var(--muted)">
				<h3>No snapshot found</h3>
				<p>No snapshot exists for %s/<code>%s</code>.</p>
			</div>`, esc(streamType), esc(truncate(streamID, 16)))
		}))

		return
	}

	if snap == nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, 12), "/snapshots", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No snapshot</h3></div>`
		}))

		return
	}

	p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, 12), "/snapshots", r)
	html := d.renderSnapshotDetail(p, ref, snap)
	renderPage(w, r, html)
}

func (d *Dashboard) renderSnapshotDetail(p pageData, ref id.StreamRef, snap *snapshot.Snapshot) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px">Snapshot: <code>%s</code></h2>`, esc(ref.ID.String()))
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">Version %s · Created %s</div>`,
			esc(snap.Version.String()), esc(snap.CreatedAt.Format(time.RFC3339)))
		b.WriteString(`</div>`)

		// Delete button (if not read-only).
		if !p.ReadOnly {
			fmt.Fprintf(&b, `<form method="POST" action="%s/snapshots/%s/%s/delete" style="margin-bottom:24px">`,
				p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()))
			fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, p.CSRFToken)
			b.WriteString(
				`<button type="submit" style="padding:6px 12px;border:1px solid var(--err);border-radius:6px;background:transparent;color:var(--err);cursor:pointer;font-size:0.85em">Delete Snapshot</button>`,
			)
			b.WriteString(`</form>`)
		}

		// Metadata.
		b.WriteString(`<h4 style="margin-bottom:8px">Metadata</h4>`)
		b.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:0.88em;margin-bottom:24px">`)
		metaRow(&b, "Stream Type", esc(string(snap.StreamType)))
		metaRow(&b, "Stream ID", esc(snap.StreamID.String()))
		metaRow(&b, "Version", esc(snap.Version.String()))
		metaRow(&b, "Created At", esc(snap.CreatedAt.Format(time.RFC3339)))
		metaRow(&b, "State Size", esc(fmt.Sprintf("%d bytes", len(snap.State))))
		b.WriteString(`</table>`)

		// State.
		b.WriteString(`<h4 style="margin-bottom:8px">State</h4>`)

		stateDisplay := d.renderSnapshotState(snap.State)
		fmt.Fprintf(
			&b,
			`<pre style="background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:16px;overflow-x:auto;font-size:0.85em;line-height:1.5;margin:0"><code>%s</code></pre>`,
			stateDisplay,
		)

		return b.String()
	})
}

func (d *Dashboard) renderSnapshotState(state []byte) string {
	if len(state) == 0 {
		return esc("(empty)")
	}

	// Try to pretty-print as JSON.
	out, err := d.cfg.PayloadRenderer.Render(state, codec.EncodingJSON)
	if err == nil && len(out) > 0 {
		return esc(string(out))
	}

	return esc(string(state))
}

func (d *Dashboard) snapshotDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.SnapshotStore == nil {
		http.Error(w, "snapshot store not configured", http.StatusBadRequest)

		return
	}

	ref, err := streamRefFromRequest(r)
	if err != nil {
		http.Error(w, "invalid stream reference: "+err.Error(), http.StatusBadRequest)

		return
	}

	if err := d.cfg.SnapshotStore.Delete(r.Context(), ref); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	triggerToast(w, "ok", "Snapshot deleted")
	redirect(w, r, d.cfg.BasePath+"/snapshots")
}

// Ensure we use the imports.
var (
	_ = id.NewAggregateID
	_ = event.Type("")
)
