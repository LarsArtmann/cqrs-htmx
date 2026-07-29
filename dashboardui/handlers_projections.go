package dashboardui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

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

// withProjectionHost checks that a projection host is configured and calls fn
// with it. If not configured, writes a 400 error and returns without calling fn.
func (d *Dashboard) withProjectionHost(w http.ResponseWriter, fn func(host *projectionhost.Host)) {
	if d.cfg.ProjectionHost == nil {
		http.Error(w, "projection host not configured", http.StatusBadRequest)

		return
	}

	fn(d.cfg.ProjectionHost)
}

// withDeadLetterStore checks that a dead-letter store is configured and calls
// fn with it. If not configured, writes a 400 error and returns without calling fn.
func (d *Dashboard) withDeadLetterStore(w http.ResponseWriter, fn func(store projectionhost.DeadLetterStore)) {
	if d.cfg.DeadLetterStore == nil {
		http.Error(w, "dead letter store not configured", http.StatusBadRequest)

		return
	}

	fn(d.cfg.DeadLetterStore)
}

func (d *Dashboard) projectionResetHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(w, func(host *projectionhost.Host) {
		name := r.PathValue("name")
		if err := host.Reset(r.Context(), name); err != nil {
			triggerToast(w, "err", "Reset failed: "+err.Error())
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		triggerToast(w, "ok", "Projection reset")
		redirect(w, r, d.cfg.BasePath+"/projections")
	})
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
			case statusGood:
				color = "#16a34a"
			case statusWarn:
				color = "#d97706"
			case statusBad:
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
