package dashboardui

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

func notImplemented(w http.ResponseWriter, panel string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<div style=\"padding:40px;text-align:center;color:#64748b\"><h2>%s</h2><p>This panel is coming soon.</p></div>", panel)
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
	notImplemented(w, "Event Detail")
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

		var rows string
		for _, evt := range events {
			rows += fmt.Sprintf(`<tr style="border-bottom:1px solid var(--border)">
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
				evt.Version().String(),
			)
		}

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
	notImplemented(w, "Aggregate Detail")
}

func (d *Dashboard) renderAggregates(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		if len(listings) == 0 {
			return `<div style="padding:40px;text-align:center;color:#64748b"><h3>No aggregates found</h3></div>`
		}

		var rows string
		for _, l := range listings {
			rows += fmt.Sprintf(`<tr style="border-bottom:1px solid var(--border)">
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
				l.LastEventAt.Format("2006-01-02 15:04:05"),
			)
		}

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
			rows += fmt.Sprintf(`<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-weight:500">%s</td>
				<td style="padding:8px"><span style="color:%s;font-weight:600">%s</span></td>
				<td style="padding:8px;font-family:monospace">%s</td>
				<td style="padding:8px">%d</td>
				<td style="padding:8px">%d</td>
			</tr>`, proj.Name, color, proj.Status, proj.Lag, proj.Processed, proj.Errors)
		}

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
			return fmt.Sprintf(`<div style="padding:40px;text-align:center;color:#64748b"><h3>No dead letters for %s</h3></div>`, proj)
		}

		var rows string
		for _, e := range entries {
			rows += fmt.Sprintf(`<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px"><code>%s</code></td>
				<td style="padding:8px;color:#dc2626">%s</td>
				<td style="padding:8px">%s</td>
			</tr>`,
				e.FailedAt.Format("2006-01-02 15:04:05"),
				e.EventType,
				truncate(e.Error, 60),
				e.ErrorFamily,
			)
		}

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
	notImplemented(w, "Commands")
}

func (d *Dashboard) queriesIndexHandler(w http.ResponseWriter, r *http.Request) {
	notImplemented(w, "Queries")
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

// Ensure we use the imports
var (
	_ = id.NewAggregateID
	_ = event.Type("")
)
