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
	d.renderStreamIndex(w, r, "Time Travel", "/time-travel", d.renderTimeTravelIndex)
}

func (d *Dashboard) renderTimeTravelIndex(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<p class="page-subtitle" style="margin-bottom:16px">Inspect an aggregate at any point in its history. Slide through versions to see the state at each step.</p>`)

		if len(listings) == 0 {
			return emptyState("No aggregates found", "Configure a StreamReader to list aggregates for time-travel inspection.")
		}

		var rows strings.Builder

		for _, l := range listings {
			fmt.Fprintf(&rows, `<tr><td>%s</td><td class="mono">%s</td><td>%s</td><td><a href="%s/time-travel/%s/%s" class="btn">Inspect</a></td></tr>`,
				esc(string(l.Type)),
				esc(truncate(l.ID.String(), listIDWidth)),
				esc(l.Version.String()),
				p.BasePath,
				esc(string(l.Type)),
				esc(l.ID.String()))
		}

		fmt.Fprintf(&b, `<table class="data-table"><thead><tr><th scope="col">Type</th><th scope="col">ID</th><th scope="col">Current Version</th><th scope="col"></th></tr></thead><tbody>%s</tbody></table>`, rows.String())

		return b.String()
	})
}

func (d *Dashboard) timeTravelDetailHandler(w http.ResponseWriter, r *http.Request) {
	ref, allEvents, ok := d.loadStreamFromRequest(w, r)
	if !ok {
		return
	}

	if len(allEvents) == 0 {
		p := d.page("Time Travel: "+string(ref.Type), "/time-travel", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return emptyState("No events", "")
		}))
		return
	}

	maxVersion := allEvents[len(allEvents)-1].Version()

	requestedVersion := maxVersion

	if v := r.URL.Query().Get("v"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 64); err == nil {
			requestedVersion = event.Version(parsed)
		}
	}

	if requestedVersion > maxVersion {
		requestedVersion = maxVersion
	}

	eventsToVersion, err := d.cfg.EventSource.LoadToVersion(r.Context(), ref, requestedVersion)
	if err != nil {
		renderError(w, r, http.StatusInternalServerError, "failed to load version")
		return
	}

	p := d.page("Time Travel: "+streamTitlePath(ref), "/time-travel", r)
	html := d.renderTimeTravelDetail(p, ref, eventsToVersion, requestedVersion, maxVersion)
	renderPage(w, r, html)
}

func (d *Dashboard) renderTimeTravelDetail(
	p pageData,
	ref id.StreamRef,
	events []event.Event,
	currentVersion event.Version,
	maxVersion event.Version,
) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(&b, `<h2>Time Travel: <code>%s</code></h2>`, esc(ref.ID.String()))
		fmt.Fprintf(&b, `<div class="page-subtitle">Viewing version %d of %d</div>`, currentVersion.Int(), maxVersion.Int())
		b.WriteString(`</div>`)

		b.WriteString(`<div class="panel">`)
		b.WriteString(`<label class="panel-title">Version</label>`)
		b.WriteString(`<div class="version-links">`)

		for v := event.Version(1); v <= maxVersion; v++ {
			if v == currentVersion {
				fmt.Fprintf(&b, `<a href="%s/time-travel/%s/%s?v=%d" class="pagination"><span class="current">%d</span></a>`,
					p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()), v.Int(), v.Int())
			} else {
				fmt.Fprintf(&b, `<a href="%s/time-travel/%s/%s?v=%d">%d</a>`,
					p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()), v.Int(), v.Int())
			}
		}

		b.WriteString(`</div></div>`)

		fmt.Fprintf(&b, `<h4>Events Through Version %d</h4>`, currentVersion.Int())

		var rows strings.Builder

		for _, evt := range events {
			fmt.Fprintf(&rows, `<tr><td style="font-weight:600">%s</td><td><a href="%s/events/%s"><code>%s</code></a></td><td class="mono">%s</td></tr>`,
				esc(evt.Version().String()),
				p.BasePath, esc(evt.ID().String()), esc(string(evt.Type())),
				esc(evt.OccurredAt().Format("2006-01-02 15:04:05")))
		}

		fmt.Fprintf(&b, `<table class="data-table"><thead><tr><th scope="col">Version</th><th scope="col">Type</th><th scope="col">Occurred At</th></tr></thead><tbody>%s</tbody></table>`, rows.String())

		return b.String()
	})
}
