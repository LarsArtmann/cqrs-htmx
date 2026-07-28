package dashboardui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

// ===== Time-Travel =====

func (d *Dashboard) timeTravelIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Time Travel", "/time-travel", r)

	listings := d.listStreams(r)

	html := d.renderTimeTravelIndex(p, listings)
	renderPage(w, r, html)
}

func (d *Dashboard) renderTimeTravelIndex(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div style="margin-bottom:16px">`)
		b.WriteString(
			`<p style="color:var(--muted);margin:0">Inspect an aggregate at any point in its history. Slide through versions to see the state at each step.</p>`,
		)
		b.WriteString(`</div>`)

		if len(listings) == 0 {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No aggregates found</h3><p>Configure a StreamReader to list aggregates for time-travel inspection.</p></div>`
		}

		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Type</th><th style="padding:8px">ID</th>`)
		b.WriteString(`<th style="padding:8px">Current Version</th><th style="padding:8px"></th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, l := range listings {
			fmt.Fprintf(
				&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px"><a href="%s/time-travel/%s/%s" style="color:var(--accent);text-decoration:none">Inspect</a></td>
			</tr>`,
				esc(string(l.Type)),
				esc(truncate(l.ID.String(), listIDWidth)),
				esc(l.Version.String()),
				p.BasePath,
				esc(string(l.Type)),
				esc(l.ID.String()),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}

func (d *Dashboard) timeTravelDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType, streamID := streamPathValues(r)

	ref, allEvents, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	if len(allEvents) == 0 {
		p := d.page("Time Travel: "+streamType, "/time-travel", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No events</h3></div>`
		}))

		return
	}

	maxVersion := allEvents[len(allEvents)-1].Version()

	// Parse requested version from query param, default to latest.
	requestedVersion := maxVersion

	if v := r.URL.Query().Get("v"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			requestedVersion = event.Version(parsed)
		}
	}

	if requestedVersion > maxVersion {
		requestedVersion = maxVersion
	}

	// Load events up to the requested version.
	eventsToVersion, err := d.cfg.EventSource.LoadToVersion(r.Context(), ref, requestedVersion)
	if err != nil {
		http.Error(w, "failed to load version: "+err.Error(), http.StatusInternalServerError)

		return
	}

	p := d.page("Time Travel: "+streamType+"/"+truncate(streamID, titleIDWidth), "/time-travel", r)
	html := d.renderTimeTravelDetail(p, ref, eventsToVersion, requestedVersion, maxVersion)
	renderPage(w, r, html)
}

func (d *Dashboard) renderTimeTravelDetail( //nolint:funlen // HTML string building is inherently verbose (no templ dep, see FEATURES.md)
	p pageData,
	ref id.StreamRef,
	events []event.Event,
	currentVersion event.Version,
	maxVersion event.Version,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px">Time Travel: <code>%s</code></h2>`, esc(ref.ID.String()))
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">Viewing version %d of %d</div>`,
			currentVersion.Int(), maxVersion.Int())
		b.WriteString(`</div>`)

		// Version slider.
		fmt.Fprintf(
			&b,
			`<div style="margin-bottom:24px;padding:16px;background:var(--surface);border:1px solid var(--border);border-radius:8px">`,
		)
		fmt.Fprintf(&b, `<label style="display:block;margin-bottom:8px;font-weight:600">Version</label>`)

		// Generate version links.
		b.WriteString(`<div style="display:flex;flex-wrap:wrap;gap:4px">`)

		for v := event.Version(1); v <= maxVersion; v++ {
			style := "padding:4px 10px;border:1px solid var(--border);border-radius:4px;text-decoration:none;color:var(--muted);font-size:0.85em"

			if v == currentVersion {
				style = "padding:4px 10px;border:1px solid var(--accent);border-radius:4px;text-decoration:none;color:white;background:var(--accent);font-size:0.85em;font-weight:600"
			}

			fmt.Fprintf(&b, `<a href="%s/time-travel/%s/%s?v=%d" style="%s">%d</a>`,
				p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()), v.Int(), style, v.Int())
		}

		b.WriteString(`</div></div>`)

		// Event timeline up to selected version.
		b.WriteString(`<h4 style="margin-bottom:8px">Events Through Version `)
		fmt.Fprintf(&b, `%d`, currentVersion.Int())
		b.WriteString(`</h4>`)

		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Version</th><th style="padding:8px">Type</th>`)
		b.WriteString(`<th style="padding:8px">Occurred At</th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, evt := range events {
			fmt.Fprintf(
				&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px;font-weight:600">%s</td>
				<td style="padding:8px"><a href="%s/events/%s" style="color:var(--accent);text-decoration:none"><code>%s</code></a></td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
			</tr>`,
				esc(evt.Version().String()),
				p.BasePath, esc(evt.ID().String()), esc(string(evt.Type())),
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")),
			)
		}

		b.WriteString(`</tbody></table>`)

		return b.String()
	})
}
