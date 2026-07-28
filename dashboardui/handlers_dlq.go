package dashboardui

import (
	"fmt"
	"net/http"
	"strings"

	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

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
	if !d.requireProjectionHost(w) {
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
	if !d.requireDeadLetterStore(w) {
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
	if !d.requireDeadLetterStore(w) {
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
