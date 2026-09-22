package dashboardui

import (
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Projection Dashboard =====

func (d *Dashboard) projectionsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Projections", "/projections", r)
	projs := buildProjectionStats(d.config.ProjectionHost)

	renderPage(w, r, projectionsPage(p, projs))
}

func (d *Dashboard) projectionDetailHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	projs := buildProjectionStats(d.config.ProjectionHost)

	var found *projectionStat

	for i := range projs {
		if projs[i].Name == name {
			found = &projs[i]

			break
		}
	}

	if found == nil {
		d.renderError(w, r, http.StatusNotFound, "projection not found")

		return
	}

	p := d.page("Projection: "+truncate(name, eventTypeWidth), "/projections", r)
	renderPage(w, r, projectionDetailPage(p, *found))
}

func (d *Dashboard) withProjectionHost(w http.ResponseWriter, fn func(host *projectionhost.Host)) {
	if d.config.ProjectionHost == nil {
		d.renderError(w, nil, http.StatusBadRequest, "projection host not configured")

		return
	}

	fn(d.config.ProjectionHost)
}

func (d *Dashboard) withDeadLetterStore(
	w http.ResponseWriter,
	fn func(store projectionhost.DeadLetterStore),
) {
	if d.config.DeadLetterStore == nil {
		d.renderError(w, nil, http.StatusBadRequest, "dead letter store not configured")

		return
	}

	fn(d.config.DeadLetterStore)
}

func (d *Dashboard) projectionResetHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(
		w,
		func(host *projectionhost.Host) { //nolint:contextcheck // handler closure
			name := r.PathValue("name")
			if err := host.Reset(r.Context(), name); err != nil {
				slog.InfoContext(
					r.Context(),
					"dashboardui.audit",
					"op",
					"projection.reset",
					"projection",
					name,
					"result",
					"error",
				)
				triggerToast(w, "err", "Reset failed")
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			slog.InfoContext(
				r.Context(),
				"dashboardui.audit",
				"op",
				"projection.reset",
				"projection",
				name,
				"result",
				"ok",
			)
			triggerToast(w, "ok", "Projection reset")
			redirect(w, r, d.config.BasePath+"/projections")
		},
	)
}
