package dashboardui

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Dead-Letter Queue =====

func (d *Dashboard) dlqIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Dead Letters", "/dead-letters", r)

	var projLinks string
	if d.cfg.ProjectionHost != nil {
		var links strings.Builder
		for _, ws := range d.cfg.ProjectionHost.Status() {
			fmt.Fprintf(&links, `<a href="%s/dead-letters/%s" class="btn btn-accent">%s</a>`,
				p.BasePath, esc(ws.Name), esc(ws.Name))
		}
		projLinks = links.String()
	}

	html := d.renderLayout(p, func() string {
		if projLinks == "" {
			return emptyState("Dead-Letter Queue", "No projections registered. Dead letters will appear here when projection errors occur.")
		}

		return fmt.Sprintf(`<div class="page-header"><h3>Dead-Letter Queue</h3><p class="page-subtitle">Select a projection to view its dead letters.</p></div><div class="filter-bar">%s</div>`, projLinks)
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
			renderError(w, r, http.StatusInternalServerError, "failed to list dead letters")
			return
		}
	}

	p := d.page("Dead Letters: "+esc(proj), "/dead-letters", r)
	html := d.renderDLQ(p, proj, entries)
	renderPage(w, r, html)
}

func (d *Dashboard) dlqReplayHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(w, func(host *projectionhost.Host) { //nolint:contextcheck // handler closure
		proj := r.PathValue("projection")

		result, err := host.ReplayDeadLetters(r.Context(), proj)
		if err != nil {
			slog.InfoContext(r.Context(), "dashboardui.audit", "op", "dlq.replay", "projection", proj, "result", "error")
			triggerToast(w, "err", "Replay failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		slog.InfoContext(r.Context(), "dashboardui.audit", "op", "dlq.replay", "projection", proj, "replayed", len(result.Replayed), "still_failing", len(result.StillFailing), "result", "ok")

		msg := fmt.Sprintf("Replayed %d, %d still failing", len(result.Replayed), len(result.StillFailing))
		triggerToast(w, "ok", msg)
		redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) dlqDeleteHandler(w http.ResponseWriter, r *http.Request) {
	d.withDeadLetterStore(w, func(store projectionhost.DeadLetterStore) { //nolint:contextcheck // handler closure
		proj := r.PathValue("projection")

		eventID := r.PathValue("eventID")
		if err := store.Delete(r.Context(), proj, eventID); err != nil {
			slog.InfoContext(r.Context(), "dashboardui.audit", "op", "dlq.delete", "projection", proj, "event_id", eventID, "result", "error")
			triggerToast(w, "err", "Delete failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		slog.InfoContext(r.Context(), "dashboardui.audit", "op", "dlq.delete", "projection", proj, "event_id", eventID, "result", "ok")
		triggerToast(w, "ok", "Dead letter deleted")
		redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) dlqPurgeHandler(w http.ResponseWriter, r *http.Request) {
	d.withDeadLetterStore(w, func(store projectionhost.DeadLetterStore) { //nolint:contextcheck // handler closure
		proj := r.PathValue("projection")
		if err := store.Purge(r.Context(), proj); err != nil {
			slog.InfoContext(r.Context(), "dashboardui.audit", "op", "dlq.purge", "projection", proj, "result", "error")
			triggerToast(w, "err", "Purge failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		slog.InfoContext(r.Context(), "dashboardui.audit", "op", "dlq.purge", "projection", proj, "result", "ok")
		triggerToast(w, "ok", "Dead letters purged")
		redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) renderDLQ(p pageData, proj string, entries []projectionhost.DeadLetterEntry) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h3>Dead Letters: %s</h3>`, esc(proj))
		b.WriteString(`</div>`)

		if !p.ReadOnly {
			b.WriteString(`<div class="filter-bar">`)
			if d.caps.ProjectionHost {
				fmt.Fprintf(&b, `<form method="POST" action="%s/dead-letters/%s/replay" class="inline-form" onsubmit="return confirm('Replay all dead letters for %s?')">`,
					p.BasePath, esc(proj), esc(proj))
				fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
				b.WriteString(`<button type="submit" class="btn btn-accent">Replay All</button>`)
				b.WriteString(`</form>`)
			}
			if d.caps.DeadLetterStore {
				fmt.Fprintf(&b, `<form method="POST" action="%s/dead-letters/%s/purge" class="inline-form" onsubmit="return confirm('Purge ALL dead letters for %s? This cannot be undone.')">`,
					p.BasePath, esc(proj), esc(proj))
				fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
				b.WriteString(`<button type="submit" class="btn btn-danger">Purge All</button>`)
				b.WriteString(`</form>`)
			}
			b.WriteString(`</div>`)
		}

		if len(entries) == 0 {
			return emptyState("No dead letters for "+esc(proj), "")
		}

		var rows strings.Builder
		for _, e := range entries {
			var actions string
			if !p.ReadOnly && d.caps.DeadLetterStore {
				actions = fmt.Sprintf(`<form method="POST" action="%s/dead-letters/%s/%s/delete" class="inline-form" onsubmit="return confirm('Delete this dead letter?')"><input type="hidden" name="_csrf" value="%s"/><button type="submit" class="btn btn-danger">Delete</button></form>`,
					p.BasePath, esc(proj), esc(e.EventID), esc(p.CSRFToken))
			}

			fmt.Fprintf(&rows, `<tr><td class="mono" title="%s">%s</td><td><code>%s</code></td><td><span class="badge badge-err">%s</span></td><td>%s</td><td>%s</td></tr>`,
				esc(e.FailedAt.Format("2006-01-02 15:04:05")),
				esc(relativeTime(e.FailedAt)),
				esc(e.EventType),
				esc(truncate(e.Error, errorDisplayWidth)),
				esc(e.ErrorFamily),
				actions)
		}

		fmt.Fprintf(&b, `<table class="data-table"><thead><tr><th scope="col">Failed At</th><th scope="col">Event Type</th><th scope="col">Error</th><th scope="col">Family</th><th scope="col">Actions</th></tr></thead><tbody>%s</tbody></table>`, rows.String())

		return b.String()
	})
}
