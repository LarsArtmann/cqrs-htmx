package dashboardui

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/icons"
)

// ===== Dead-Letter Queue =====

func (d *Dashboard) dlqIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Dead Letters", "/dead-letters", r)

	links := d.buildDLQProjectionLinks(r.Context())

	ctx := r.Context()

	html := d.renderLayout(
		ctx,
		p,
		func() string {
			if len(links) == 0 {
				return emptyStateIcon(
					ctx,
					icons.BugAnt,
					"Dead-Letter Queue",
					"No projections registered. Dead letters will appear here when projection errors occur.",
				)
			}

			var b strings.Builder

			b.WriteString(`<div class="page-header"><h2>Dead-Letter Queue</h2>`)
			b.WriteString(
				`<p class="page-subtitle">Select a projection to view its dead letters.</p></div>`,
			)

			// Summary table with counts.
			var rows strings.Builder

			for _, link := range links {
				fmt.Fprintf(
					&rows,
					`<tr><td class="cell-emph"><a href="%s/dead-letters/%s">%s</a></td><td>%s</td><td>%s</td></tr>`,
					p.BasePath,
					esc(link.Name),
					esc(link.Name),
					badgeHTML(ctx, strconv.Itoa(link.Count), countBadgeType(link.Count)),
					buttonLink(
						ctx,
						"View",
						p.BasePath+"/dead-letters/"+esc(link.Name),
						"",
						display.ButtonSecondary,
						false,
					),
				)
			}

			b.WriteString(tableHTMLRaw(ctx, plainHeaders("Projection", "Dead Letters", ""), rows.String()))

			return b.String()
		},
	)
	renderPage(w, r, html)
}

func (d *Dashboard) dlqDetailHandler(w http.ResponseWriter, r *http.Request) {
	proj := r.PathValue("projection")

	var entries []projectionhost.DeadLetterEntry

	if d.config.DeadLetterStore != nil {
		var err error

		entries, err = d.config.DeadLetterStore.List(r.Context(), proj)
		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to list dead letters")

			return
		}
	}

	p := d.page("Dead Letters: "+esc(proj), "/dead-letters", r)
	html := d.renderDLQ(r.Context(), p, proj, entries)
	renderPage(w, r, html)
}

func (d *Dashboard) dlqEntryDetailHandler(w http.ResponseWriter, r *http.Request) {
	proj := r.PathValue("projection")
	eventID := r.PathValue("eventID")

	var entry projectionhost.DeadLetterEntry

	if d.config.DeadLetterStore != nil {
		entries, err := d.config.DeadLetterStore.List(r.Context(), proj)
		if err != nil {
			d.renderError(w, r, http.StatusInternalServerError, "failed to load dead letter")

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
		d.renderError(w, r, http.StatusNotFound, "dead letter not found")

		return
	}

	p := d.page("Dead Letter: "+truncate(eventID, eventIDWidth), "/dead-letters", r)
	html := d.renderDLQEntryDetail(r.Context(), p, proj, entry)
	renderPage(w, r, html)
}

func (d *Dashboard) renderDLQEntryDetail(
	ctx context.Context,
	p pageData,
	proj string,
	entry projectionhost.DeadLetterEntry,
) string {
	return d.renderLayout(ctx, p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(
			&b,
			`<h2>Dead Letter: <code>%s</code></h2>`,
			esc(truncate(entry.EventID, eventIDWidth)),
		)
		fmt.Fprintf(
			&b,
			`<div class="page-subtitle">Projection: <a href="%s/dead-letters/%s">%s</a></div>`,
			p.BasePath,
			esc(proj),
			esc(proj),
		)
		b.WriteString(`</div>`)

		b.WriteString(`<div class="two-col-grid">`)
		items := []display.DefinitionItem{
			defItem("Event Type", entry.EventType),
			defItem("Event ID", entry.EventID),
		}

		if entry.StreamID != "" {
			items = append(
				items,
				defItemCopy("Stream ID", "<span class=\"mono\">"+esc(entry.StreamID)+"</span>", entry.StreamID),
			)
		}

		items = append(items, defItem("Failed At", entry.FailedAt.Format("2006-01-02 15:04:05")))

		if entry.ErrorFamily != "" {
			items = append(items, defItemRaw("Error Family", badgeHTML(ctx, entry.ErrorFamily, display.BadgeError)))
		}

		if entry.ErrorCode != "" {
			items = append(items, defItem("Error Code", entry.ErrorCode))
		}

		b.WriteString(`<div><h3>Error Details</h3>`)
		b.WriteString(definitionListHTML(ctx, items))

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
				`<form method="POST" action="%s/dead-letters/%s/%s/delete" class="inline-form" data-confirm="Delete this dead letter?"><input type="hidden" name="_csrf" value="%s"/>%s</form>`,
				p.BasePath,
				esc(proj),
				esc(entry.EventID),
				esc(p.CSRFToken),
				buttonSubmit(ctx, "Delete", "Delete dead letter", display.ButtonOutlineDanger, nil),
			)
			b.WriteString(`</div>`)
		}

		return b.String()
	})
}

func (d *Dashboard) dlqReplayHandler(w http.ResponseWriter, r *http.Request) {
	d.withProjectionHost(
		w,
		func(host *projectionhost.Host) { //nolint:contextcheck // handler closure
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

			msg := fmt.Sprintf(
				"Replayed %d, %d still failing",
				len(result.Replayed),
				len(result.StillFailing),
			)
			triggerToast(w, "ok", msg)
			redirect(w, r, d.config.BasePath+"/dead-letters/"+proj)
		},
	)
}

func (d *Dashboard) dlqDeleteHandler(w http.ResponseWriter, r *http.Request) {
	d.withDeadLetterStore(
		w,
		func(store projectionhost.DeadLetterStore) { //nolint:contextcheck // handler closure
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
		},
	)
}

func (d *Dashboard) dlqPurgeHandler(w http.ResponseWriter, r *http.Request) {
	d.withDeadLetterStore(
		w,
		func(store projectionhost.DeadLetterStore) { //nolint:contextcheck // handler closure
			proj := r.PathValue("projection")
			if err := store.Purge(r.Context(), proj); err != nil {
				slog.InfoContext(
					r.Context(),
					"dashboardui.audit",
					"op",
					"dlq.purge",
					"projection",
					proj,
					"result",
					"error",
				)
				triggerToast(w, "err", "Purge failed")
				w.WriteHeader(http.StatusInternalServerError)

				return
			}

			slog.InfoContext(
				r.Context(),
				"dashboardui.audit",
				"op",
				"dlq.purge",
				"projection",
				proj,
				"result",
				"ok",
			)
			triggerToast(w, "ok", "Dead letters purged")
			redirect(w, r, d.config.BasePath+"/dead-letters/"+proj)
		},
	)
}

func (d *Dashboard) renderDLQ(
	ctx context.Context,
	p pageData,
	proj string,
	entries []projectionhost.DeadLetterEntry,
) string {
	return d.renderLayout(ctx, p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2>Dead Letters: %s</h2>`, esc(proj))
		b.WriteString(`</div>`)

		if !p.ReadOnly {
			b.WriteString(`<div class="filter-bar">`)

			if d.caps.ProjectionHost {
				fmt.Fprintf(
					&b,
					`<form method="POST" action="%s/dead-letters/%s/replay" class="inline-form" data-confirm="Replay all dead letters for %s?" aria-label="Replay all dead letters for %s">`,
					p.BasePath,
					esc(proj),
					esc(proj),
					esc(proj),
				)
				fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
				b.WriteString(
					buttonSubmit(
						ctx,
						"Replay All",
						"Replay all dead letters",
						display.ButtonOutlineInfo,
						nil,
					),
				)
				b.WriteString(`</form>`)
			}

			if d.caps.DeadLetterStore {
				fmt.Fprintf(
					&b,
					`<form method="POST" action="%s/dead-letters/%s/purge" class="inline-form" data-confirm="Purge ALL dead letters for %s? This cannot be undone." aria-label="Purge all dead letters for %s">`,
					p.BasePath,
					esc(proj),
					esc(proj),
					esc(proj),
				)
				fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
				b.WriteString(
					buttonSubmit(
						ctx,
						"Purge All",
						"Purge all dead letters",
						display.ButtonOutlineDanger,
						nil,
					),
				)
				b.WriteString(`</form>`)
			}

			b.WriteString(`</div>`)
		}

		if len(entries) == 0 {
			return emptyStateIcon(ctx, icons.BugAnt, "No dead letters for "+esc(proj), "")
		}

		var rows strings.Builder

		for _, e := range entries {
			var actions string
			if !p.ReadOnly && d.caps.DeadLetterStore {
				actions = fmt.Sprintf(
					`<form method="POST" action="%s/dead-letters/%s/%s/delete" class="inline-form" data-confirm="Delete this dead letter?" aria-label="Delete dead letter %s"><input type="hidden" name="_csrf" value="%s"/>%s</form>`,
					p.BasePath,
					esc(proj),
					esc(e.EventID),
					esc(e.EventID),
					esc(p.CSRFToken),
					buttonSubmit(
						ctx,
						"Delete",
						"Delete dead letter "+esc(e.EventID),
						display.ButtonOutlineDanger,
						nil,
					),
				)
			}

			fmt.Fprintf(
				&rows,
				`<tr><td class="mono" title="%s">%s</td><td><a href="%s/dead-letters/%s/%s"><code>%s</code></a></td><td>%s</td><td>%s</td><td>%s %s</td></tr>`,
				esc(e.FailedAt.Format("2006-01-02 15:04:05")),
				esc(relativeTime(e.FailedAt)),
				p.BasePath,
				esc(proj),
				esc(e.EventID),
				esc(e.EventType),
				esc(truncate(e.Error, errorDisplayWidth)),
				badgeHTML(ctx, e.ErrorFamily, display.BadgeError),
				buttonLink(
					ctx,
					"View",
					p.BasePath+"/dead-letters/"+esc(proj)+"/"+esc(e.EventID),
					"",
					display.ButtonSecondary,
					false,
				),
				actions,
			)
		}

		b.WriteString(tableHTMLRaw(
			ctx,
			plainHeaders("Failed At", "Event Type", "Error", "Family", "Actions"),
			rows.String()))

		return b.String()
	})
}
