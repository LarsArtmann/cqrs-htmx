package dashboardui

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Projection Dashboard =====

func (d *Dashboard) projectionsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Projections", "/projections", r)
	projs := buildProjectionStats(d.config.ProjectionHost)

	html := d.renderProjections(p, projs)
	renderPage(w, r, html)
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
		renderError(w, r, http.StatusNotFound, "projection not found")

		return
	}

	p := d.page("Projection: "+truncate(name, eventTypeWidth), "/projections", r)
	html := d.renderProjectionDetail(p, *found)
	renderPage(w, r, html)
}

func (d *Dashboard) withProjectionHost(w http.ResponseWriter, fn func(host *projectionhost.Host)) {
	if d.config.ProjectionHost == nil {
		renderError(w, nil, http.StatusBadRequest, "projection host not configured")

		return
	}

	fn(d.config.ProjectionHost)
}

func (d *Dashboard) withDeadLetterStore(w http.ResponseWriter, fn func(store projectionhost.DeadLetterStore)) {
	if d.config.DeadLetterStore == nil {
		renderError(w, nil, http.StatusBadRequest, "dead letter store not configured")

		return
	}

	fn(d.config.DeadLetterStore)
}

func (d *Dashboard) projectionResetHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(w, func(host *projectionhost.Host) { //nolint:contextcheck // handler closure
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

		slog.InfoContext(r.Context(), "dashboardui.audit", "op", "projection.reset", "projection", name, "result", "ok")
		triggerToast(w, "ok", "Projection reset")
		redirect(w, r, d.config.BasePath+"/projections")
	})
}

func (d *Dashboard) renderProjections(p pageData, projs []projectionStat) string {
	return d.renderLayout(p, func() string {
		if len(projs) == 0 {
			return emptyState("No projections registered", "")
		}

		var b strings.Builder
		b.WriteString(`<h2>Projections</h2>`)

		var rows strings.Builder

		for _, proj := range projs {
			badgeClass := badgeNeutral

			switch proj.StatusKind {
			case statusGood:
				badgeClass = badgeOK
			case statusWarn:
				badgeClass = badgeWarn
			case statusBad:
				badgeClass = badgeErr
			}

			var actions string
			if !p.ReadOnly {
				actions = fmt.Sprintf(
					`<form method="POST" action="%s/projections/%s/reset" class="inline-form" data-confirm="Reset projection %s? This will re-process all events from the beginning." aria-label="Reset projection %s"><input type="hidden" name="_csrf" value="%s"/><button type="submit" class="btn btn-danger" aria-label="Reset projection %s">Reset</button></form>`,
					p.BasePath,
					esc(proj.Name),
					esc(proj.Name),
					esc(proj.Name),
					esc(p.CSRFToken),
					esc(proj.Name),
				)
			}

			dlqLink := ""
			if proj.Errors > 0 || d.caps.DeadLetterStore || d.caps.ProjectionHost {
				dlqLink = fmt.Sprintf(
					`<a href="%s/dead-letters/%s" class="btn" aria-label="View dead letters for %s">DLQ (%d)</a>`,
					p.BasePath,
					esc(proj.Name),
					esc(proj.Name),
					proj.Errors,
				)
			}

			lastErr := "—"
			if proj.LastError != "" {
				lastErr = esc(truncate(proj.LastError, errorDisplayWidth))
			}

			fmt.Fprintf(
				&rows,
				`<tr><td class="cell-emph"><a href="%s/projections/%s">%s</a></td><td><span class="%s">%s</span></td><td class="mono">%s</td><td>%d</td><td>%d</td><td>%d</td><td class="mono" title="%s">%s</td><td>%s</td><td>%s %s</td></tr>`,
				p.BasePath,
				esc(proj.Name),
				esc(proj.Name),
				badgeClass,
				esc(proj.Status),
				esc(proj.Lag),
				proj.Processed,
				proj.Errors,
				proj.Restarts,
				esc(proj.Checkpoint),
				esc(truncate(proj.Checkpoint, listIDWidth)),
				lastErr,
				dlqLink,
				actions,
			)
		}

		fmt.Fprintf(
			&b,
			`<div class="table-scroll"><table class="data-table"><thead><tr><th scope="col">Name</th><th scope="col">Status</th><th scope="col">Lag</th><th scope="col">Processed</th><th scope="col">Errors</th><th scope="col">Restarts</th><th scope="col">Checkpoint</th><th scope="col">Last Error</th><th scope="col">Actions</th></tr></thead><tbody>%s</tbody></table></div>`,
			rows.String(),
		)

		return b.String()
	})
}

func (d *Dashboard) renderProjectionDetail(p pageData, proj projectionStat) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2>%s</h2>`, esc(proj.Name))

		badgeClass := badgeNeutral

		switch proj.StatusKind {
		case statusGood:
			badgeClass = badgeOK
		case statusWarn:
			badgeClass = badgeWarn
		case statusBad:
			badgeClass = badgeErr
		}

		fmt.Fprintf(&b, `<span class="%s">%s</span>`, badgeClass, esc(proj.Status))
		b.WriteString(`</div>`)

		b.WriteString(`<div class="stat-grid">`)
		statCard(&b, strconv.FormatInt(proj.Processed, 10), "Processed", "")
		statCard(&b, strconv.FormatInt(proj.Errors, 10), "Errors", "err")
		statCard(&b, strconv.Itoa(proj.Restarts), "Restarts", "warn")
		statCard(&b, proj.Lag, "Lag", "")
		b.WriteString(`</div>`)

		b.WriteString(`<h3>Details</h3><table class="meta-table">`)
		metaRowCopyable(&b, "Checkpoint", esc(truncate(proj.Checkpoint, listIDWidth)), proj.Checkpoint)
		metaRow(&b, "Status", esc(proj.Status))

		if proj.LastError != "" {
			metaRow(&b, "Last Error", esc(proj.LastError))
		} else {
			metaRow(&b, "Last Error", "<span class=\"muted\">none</span>")
		}

		b.WriteString(`</table>`)

		b.WriteString(`<div class="filter-bar">`)
		fmt.Fprintf(
			&b,
			`<a href="%s/dead-letters/%s" class="btn">View Dead Letters (%d)</a>`,
			p.BasePath,
			esc(proj.Name),
			proj.Errors,
		)

		if !p.ReadOnly {
			fmt.Fprintf(
				&b,
				`<form method="POST" action="%s/projections/%s/reset" class="inline-form" data-confirm="Reset projection %s? This will re-process all events from the beginning."><input type="hidden" name="_csrf" value="%s"/><button type="submit" class="btn btn-danger">Reset Projection</button></form>`,
				p.BasePath,
				esc(proj.Name),
				esc(proj.Name),
				esc(p.CSRFToken),
			)
		}

		fmt.Fprintf(&b, `<a href="%s/projections" class="btn">Back to Projections</a>`, p.BasePath)
		b.WriteString(`</div>`)

		return b.String()
	})
}
