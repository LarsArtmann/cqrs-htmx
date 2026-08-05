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

	links := d.buildDLQProjectionLinks(r.Context())

	html := d.renderLayout(p, func() string {
		if len(links) == 0 {
			return emptyState(
				"Dead-Letter Queue",
				"No projections registered. Dead letters will appear here when projection errors occur.",
			)
		}

		var b strings.Builder

		b.WriteString(`<div class="page-header"><h2>Dead-Letter Queue</h2>`)
		b.WriteString(`<p class="page-subtitle">Select a projection to view its dead letters.</p></div>`)

		// Summary table with counts.
		var rows strings.Builder

		for _, link := range links {
			badgeClass := badgeNeutral
			if link.Count > 0 {
				badgeClass = badgeErr
			}

			fmt.Fprintf(
				&rows,
				`<tr><td class="cell-emph"><a href="%s/dead-letters/%s">%s</a></td><td><span class="%s">%d</span></td><td><a href="%s/dead-letters/%s" class="btn">View</a></td></tr>`,
				p.BasePath,
				esc(link.Name),
				esc(link.Name),
				badgeClass,
				link.Count,
				p.BasePath,
				esc(link.Name),
			)
		}

		fmt.Fprintf(
			&b,
			`<div class="table-scroll"><table class="data-table"><thead><tr><th scope="col">Projection</th><th scope="col">Dead Letters</th><th scope="col"></th></tr></thead><tbody>%s</tbody></table></div>`,
			rows.String(),
		)

		return b.String()
	})
	renderPage(w, r, html)
}

func (d *Dashboard) dlqDetailHandler(w http.ResponseWriter, r *http.Request) {
	proj := r.PathValue("projection")

	var entries []projectionhost.DeadLetterEntry

	if d.config.DeadLetterStore != nil {
		var err error

		entries, err = d.config.DeadLetterStore.List(r.Context(), proj)
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "failed to list dead letters")

			return
		}
	}

	p := d.page("Dead Letters: "+esc(proj), "/dead-letters", r)
	html := d.renderDLQ(p, proj, entries)
	renderPage(w, r, html)
}

func (d *Dashboard) dlqEntryDetailHandler(w http.ResponseWriter, r *http.Request) {
	proj := r.PathValue("projection")
	eventID := r.PathValue("eventID")

	var entry projectionhost.DeadLetterEntry

	if d.config.DeadLetterStore != nil {
		entries, err := d.config.DeadLetterStore.List(r.Context(), proj)
		if err != nil {
			renderError(w, r, http.StatusInternalServerError, "failed to load dead letter")

			return
		}

		for _, e := range entries {
			if e.EventID == eventID {
				entry = e

				break
			}
		}
	}

	if entry.EventID == "" {
		renderError(w, r, http.StatusNotFound, "dead letter not found")

		return
	}

	p := d.page("Dead Letter: "+truncate(eventID, eventIDWidth), "/dead-letters", r)
	html := d.renderDLQEntryDetail(p, proj, entry)
	renderPage(w, r, html)
}

func (d *Dashboard) renderDLQEntryDetail(p pageData, proj string, entry projectionhost.DeadLetterEntry) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2>Dead Letter: <code>%s</code></h2>`, esc(truncate(entry.EventID, eventIDWidth)))
		fmt.Fprintf(&b, `<div class="page-subtitle">Projection: <a href="%s/dead-letters/%s">%s</a></div>`,
			p.BasePath, esc(proj), esc(proj))
		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)
		b.WriteString(`<div><h3>Error Details</h3><table class="meta-table">`)
		metaRow(&b, "Event Type", esc(entry.EventType))
		metaRow(&b, "Event ID", esc(entry.EventID))

		if entry.StreamID != "" {
			metaRowCopyable(&b, "Stream ID", esc(entry.StreamID), entry.StreamID)
		}

		metaRow(&b, "Failed At", esc(entry.FailedAt.Format("2006-01-02 15:04:05")))

		if entry.ErrorFamily != "" {
			fmt.Fprintf(
				&b,
				`<tr><td class="meta-key">Error Family</td><td class="meta-val"><span class="badge badge-err">%s</span></td></tr>`,
				esc(entry.ErrorFamily),
			)
		}

		if entry.ErrorCode != "" {
			metaRow(&b, "Error Code", esc(entry.ErrorCode))
		}

		b.WriteString(`</table>`)

		b.WriteString(`<h3>Error Message</h3>`)
		fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, esc(entry.Error))

		b.WriteString(`</div>`)

		b.WriteString(`<div><h3>Event Payload</h3>`)

		if entry.Event != nil {
			payload := renderPayload(d.config.PayloadRenderer, entry.Event)
			fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, esc(string(payload)))
		} else {
			b.WriteString(`<p class="muted">Original event not available</p>`)
		}

		b.WriteString(`</div>`)

		b.WriteString(`</div>`)

		if !p.ReadOnly && d.config.DeadLetterStore != nil {
			b.WriteString(`<div class="filter-bar section-gap">`)
			fmt.Fprintf(
				&b,
				`<form method="POST" action="%s/dead-letters/%s/%s/delete" class="inline-form" onsubmit="return confirm('Delete this dead letter?')"><input type="hidden" name="_csrf" value="%s"/><button type="submit" class="btn btn-danger">Delete</button></form>`,
				p.BasePath,
				esc(proj),
				esc(entry.EventID),
				esc(p.CSRFToken),
			)
			b.WriteString(`</div>`)
		}

		return b.String()
	})
}

func (d *Dashboard) dlqReplayHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(w, func(host *projectionhost.Host) { //nolint:contextcheck // handler closure
		proj := r.PathValue("projection")

		result, err := host.ReplayDeadLetters(r.Context(), proj)
		if err != nil {
			slog.InfoContext(
				r.Context(),
				"dashboardui.audit",
				"op",
				"dlq.replay",
				"projection",
				proj,
				"result",
				"error",
			)
			triggerToast(w, "err", "Replay failed")
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		slog.InfoContext(
			r.Context(),
			"dashboardui.audit",
			"op",
			"dlq.replay",
			"projection",
			proj,
			"replayed",
			len(result.Replayed),
			"still_failing",
			len(result.StillFailing),
			"result",
			"ok",
		)

		msg := fmt.Sprintf("Replayed %d, %d still failing", len(result.Replayed), len(result.StillFailing))
		triggerToast(w, "ok", msg)
		redirect(w, r, d.config.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) dlqDeleteHandler(w http.ResponseWriter, r *http.Request) {
	d.withDeadLetterStore(w, func(store projectionhost.DeadLetterStore) { //nolint:contextcheck // handler closure
		proj := r.PathValue("projection")

		eventID := r.PathValue("eventID")
		if err := store.Delete(r.Context(), proj, eventID); err != nil {
			slog.InfoContext(
				r.Context(),
				"dashboardui.audit",
				"op",
				"dlq.delete",
				"projection",
				proj,
				"event_id",
				eventID,
				"result",
				"error",
			)
			triggerToast(w, "err", "Delete failed")
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		slog.InfoContext(
			r.Context(),
			"dashboardui.audit",
			"op",
			"dlq.delete",
			"projection",
			proj,
			"event_id",
			eventID,
			"result",
			"ok",
		)
		triggerToast(w, "ok", "Dead letter deleted")
		redirect(w, r, d.config.BasePath+"/dead-letters/"+proj)
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
		redirect(w, r, d.config.BasePath+"/dead-letters/"+proj)
	})
}

func (d *Dashboard) renderDLQ(p pageData, proj string, entries []projectionhost.DeadLetterEntry) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2>Dead Letters: %s</h2>`, esc(proj))
		b.WriteString(`</div>`)

		if !p.ReadOnly {
			b.WriteString(`<div class="filter-bar">`)

			if d.caps.ProjectionHost {
				fmt.Fprintf(
					&b,
					`<form method="POST" action="%s/dead-letters/%s/replay" class="inline-form" onsubmit="return confirm('Replay all dead letters for %s?')" aria-label="Replay all dead letters for %s">`,
					p.BasePath,
					esc(proj),
					esc(proj),
					esc(proj),
				)
				fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
				b.WriteString(
					`<button type="submit" class="btn btn-accent" aria-label="Replay all dead letters">Replay All</button>`,
				)
				b.WriteString(`</form>`)
			}

			if d.caps.DeadLetterStore {
				fmt.Fprintf(
					&b,
					`<form method="POST" action="%s/dead-letters/%s/purge" class="inline-form" onsubmit="return confirm('Purge ALL dead letters for %s? This cannot be undone.')" aria-label="Purge all dead letters for %s">`,
					p.BasePath,
					esc(proj),
					esc(proj),
					esc(proj),
				)
				fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
				b.WriteString(
					`<button type="submit" class="btn btn-danger" aria-label="Purge all dead letters">Purge All</button>`,
				)
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
				actions = fmt.Sprintf(
					`<form method="POST" action="%s/dead-letters/%s/%s/delete" class="inline-form" onsubmit="return confirm('Delete this dead letter?')" aria-label="Delete dead letter %s"><input type="hidden" name="_csrf" value="%s"/><button type="submit" class="btn btn-danger" aria-label="Delete dead letter %s">Delete</button></form>`,
					p.BasePath,
					esc(proj),
					esc(e.EventID),
					esc(e.EventID),
					esc(p.CSRFToken),
					esc(e.EventID),
				)
			}

			fmt.Fprintf(
				&rows,
				`<tr><td class="mono" title="%s">%s</td><td><a href="%s/dead-letters/%s/%s"><code>%s</code></a></td><td><span class="badge badge-err">%s</span></td><td>%s</td><td><a href="%s/dead-letters/%s/%s" class="btn">View</a> %s</td></tr>`,
				esc(e.FailedAt.Format("2006-01-02 15:04:05")),
				esc(relativeTime(e.FailedAt)),
				p.BasePath,
				esc(proj),
				esc(e.EventID),
				esc(e.EventType),
				esc(truncate(e.Error, errorDisplayWidth)),
				esc(e.ErrorFamily),
				p.BasePath,
				esc(proj),
				esc(e.EventID),
				actions,
			)
		}

		fmt.Fprintf(
			&b,
			`<div class="table-scroll"><table class="data-table"><thead><tr><th scope="col">Failed At</th><th scope="col">Event Type</th><th scope="col">Error</th><th scope="col">Family</th><th scope="col">Actions</th></tr></thead><tbody>%s</tbody></table></div>`,
			rows.String(),
		)

		return b.String()
	})
}
