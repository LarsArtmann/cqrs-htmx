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

func (d *Dashboard) withProjectionHost(w http.ResponseWriter, fn func(host *projectionhost.Host)) {
	if d.cfg.ProjectionHost == nil {
		http.Error(w, "projection host not configured", http.StatusBadRequest)

		return
	}

	fn(d.cfg.ProjectionHost)
}

func (d *Dashboard) withDeadLetterStore(w http.ResponseWriter, fn func(store projectionhost.DeadLetterStore)) {
	if d.cfg.DeadLetterStore == nil {
		http.Error(w, "dead letter store not configured", http.StatusBadRequest)

		return
	}

	fn(d.cfg.DeadLetterStore)
}

func (d *Dashboard) projectionResetHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(w, func(host *projectionhost.Host) { //nolint:contextcheck // handler closure
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
			return emptyState("No projections registered", "")
		}

		var rows strings.Builder

		for _, proj := range projs {
			badgeClass := "badge badge-neutral"

			switch proj.StatusKind {
			case statusGood:
				badgeClass = "badge badge-ok"
			case statusWarn:
				badgeClass = "badge badge-warn"
			case statusBad:
				badgeClass = "badge badge-err"
			}

			fmt.Fprintf(
				&rows,
				`<tr><td style="font-weight:500">%s</td><td><span class="%s">%s</span></td><td class="mono">%s</td><td>%d</td><td>%d</td></tr>`,
				esc(proj.Name),
				badgeClass,
				esc(proj.Status),
				esc(proj.Lag),
				proj.Processed,
				proj.Errors,
			)
		}

		return fmt.Sprintf(
			`<h3>Projections</h3><table class="data-table"><thead><tr><th scope="col">Name</th><th scope="col">Status</th><th scope="col">Lag</th><th scope="col">Processed</th><th scope="col">Errors</th></tr></thead><tbody>%s</tbody></table>`,
			rows.String(),
		)
	})
}
