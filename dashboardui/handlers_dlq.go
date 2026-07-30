package dashboardui

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Dead-Letter Queue =====

func (d *Dashboard) dlqIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Dead Letters", "/dead-letters", r)
	html := d.renderLayout(p, func() string {
		return emptyState("Dead-Letter Queue", "Select a projection to view its dead letters.")
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
			triggerToast(w, "err", "Replay failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

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
			triggerToast(w, "err", "Delete failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		triggerToast(w, "ok", "Dead letter deleted")
		redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) dlqPurgeHandler(w http.ResponseWriter, r *http.Request) {
	d.withDeadLetterStore(w, func(store projectionhost.DeadLetterStore) { //nolint:contextcheck // handler closure
		proj := r.PathValue("projection")
		if err := store.Purge(r.Context(), proj); err != nil {
			triggerToast(w, "err", "Purge failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		triggerToast(w, "ok", "Dead letters purged")
		redirect(w, r, d.cfg.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) renderDLQ(p pageData, proj string, entries []projectionhost.DeadLetterEntry) string {
	return d.renderLayout(p, func() string {
		if len(entries) == 0 {
			return emptyState("No dead letters for "+esc(proj), "")
		}

		var rows strings.Builder

		for _, e := range entries {
			fmt.Fprintf(&rows, `<tr><td class="mono">%s</td><td><code>%s</code></td><td><span class="badge badge-err">%s</span></td><td>%s</td></tr>`,
				esc(e.FailedAt.Format("2006-01-02 15:04:05")),
				esc(e.EventType),
				esc(truncate(e.Error, errorDisplayWidth)),
				esc(e.ErrorFamily))
		}

		return fmt.Sprintf(`<h3>Dead Letters: %s</h3><table class="data-table"><thead><tr><th scope="col">Failed At</th><th scope="col">Event Type</th><th scope="col">Error</th><th scope="col">Family</th></tr></thead><tbody>%s</tbody></table>`, esc(proj), rows.String())
	})
}
