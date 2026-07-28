package dashboardui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
)

// ===== Snapshots =====

func (d *Dashboard) snapshotsIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Snapshots", "/snapshots", r)

	listings := d.listStreams(r)

	html := d.renderSnapshotsIndex(p, listings)
	renderPage(w, r, html)
}

func (d *Dashboard) renderSnapshotsIndex(p pageData, listings []listing.StreamListing) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(
			`<p style="color:var(--muted);margin-bottom:16px">Inspect snapshot state for any aggregate. Snapshots store a point-in-time cache of aggregate state to accelerate loading.</p>`,
		)

		if len(listings) == 0 {
			b.WriteString(`<div style="padding:40px;text-align:center;color:var(--muted)">`)
			b.WriteString(`<h3>No aggregates found</h3>`)
			b.WriteString(`<p>Configure a StreamReader to browse snapshots by aggregate.</p>`)
			b.WriteString(`</div>`)

			return b.String()
		}

		b.WriteString(`<table style="width:100%;border-collapse:collapse">`)
		b.WriteString(`<thead><tr style="text-align:left;border-bottom:2px solid var(--border)">`)
		b.WriteString(`<th style="padding:8px">Type</th><th style="padding:8px">ID</th>`)
		b.WriteString(`<th style="padding:8px">Version</th><th style="padding:8px"></th>`)
		b.WriteString(`</tr></thead><tbody>`)

		for _, l := range listings {
			fmt.Fprintf(&b, `<tr style="border-bottom:1px solid var(--border)">
				<td style="padding:8px">%s</td>
				<td style="padding:8px;font-family:monospace;font-size:0.85em">%s</td>
				<td style="padding:8px">%s</td>
				<td style="padding:8px"><a href="%s/snapshots/%s/%s" style="color:var(--accent);text-decoration:none">View</a></td>
			</tr>`,
				esc(string(l.Type)),
				esc(truncate(l.ID.String(), 24)),
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

func (d *Dashboard) snapshotDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType := r.PathValue("type")
	streamID := r.PathValue("id")

	ref, err := streamRefFromRequest(r)
	if err != nil {
		http.Error(w, "invalid stream reference: "+err.Error(), http.StatusBadRequest)

		return
	}

	snap, err := d.cfg.SnapshotStore.Load(r.Context(), ref)
	if err != nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, 12), "/snapshots", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return fmt.Sprintf(`<div style="padding:40px;text-align:center;color:var(--muted)">
				<h3>No snapshot found</h3>
				<p>No snapshot exists for %s/<code>%s</code>.</p>
			</div>`, esc(streamType), esc(truncate(streamID, 16)))
		}))

		return
	}

	if snap == nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, 12), "/snapshots", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return `<div style="padding:40px;text-align:center;color:var(--muted)"><h3>No snapshot</h3></div>`
		}))

		return
	}

	p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, 12), "/snapshots", r)
	html := d.renderSnapshotDetail(p, ref, snap)
	renderPage(w, r, html)
}

func (d *Dashboard) renderSnapshotDetail(p pageData, ref id.StreamRef, snap *snapshot.Snapshot) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		fmt.Fprintf(&b, `<div style="margin-bottom:24px">`)
		fmt.Fprintf(&b, `<h2 style="margin:0 0 4px">Snapshot: <code>%s</code></h2>`, esc(ref.ID.String()))
		fmt.Fprintf(&b, `<div style="color:var(--muted);font-size:0.88em">Version %s · Created %s</div>`,
			esc(snap.Version.String()), esc(snap.CreatedAt.Format(time.RFC3339)))
		b.WriteString(`</div>`)

		// Delete button (if not read-only).
		if !p.ReadOnly {
			fmt.Fprintf(&b, `<form method="POST" action="%s/snapshots/%s/%s/delete" style="margin-bottom:24px">`,
				p.BasePath, esc(string(ref.Type)), esc(ref.ID.String()))
			fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, p.CSRFToken)
			b.WriteString(
				`<button type="submit" style="padding:6px 12px;border:1px solid var(--err);border-radius:6px;background:transparent;color:var(--err);cursor:pointer;font-size:0.85em">Delete Snapshot</button>`,
			)
			b.WriteString(`</form>`)
		}

		// Metadata.
		b.WriteString(`<h4 style="margin-bottom:8px">Metadata</h4>`)
		b.WriteString(`<table style="width:100%;border-collapse:collapse;font-size:0.88em;margin-bottom:24px">`)
		metaRow(&b, "Stream Type", esc(string(snap.StreamType)))
		metaRow(&b, "Stream ID", esc(snap.StreamID.String()))
		metaRow(&b, "Version", esc(snap.Version.String()))
		metaRow(&b, "Created At", esc(snap.CreatedAt.Format(time.RFC3339)))
		metaRow(&b, "State Size", esc(fmt.Sprintf("%d bytes", len(snap.State))))
		b.WriteString(`</table>`)

		// State.
		b.WriteString(`<h4 style="margin-bottom:8px">State</h4>`)

		stateDisplay := d.renderSnapshotState(snap.State)
		fmt.Fprintf(
			&b,
			`<pre style="background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:16px;overflow-x:auto;font-size:0.85em;line-height:1.5;margin:0"><code>%s</code></pre>`,
			stateDisplay,
		)

		return b.String()
	})
}

func (d *Dashboard) renderSnapshotState(state []byte) string {
	if len(state) == 0 {
		return esc("(empty)")
	}

	// Try to pretty-print as JSON.
	out, err := d.cfg.PayloadRenderer.Render(state, codec.EncodingJSON)
	if err == nil && len(out) > 0 {
		return esc(string(out))
	}

	return esc(string(state))
}

func (d *Dashboard) snapshotDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.SnapshotStore == nil {
		http.Error(w, "snapshot store not configured", http.StatusBadRequest)

		return
	}

	ref, err := streamRefFromRequest(r)
	if err != nil {
		http.Error(w, "invalid stream reference: "+err.Error(), http.StatusBadRequest)

		return
	}

	if err := d.cfg.SnapshotStore.Delete(r.Context(), ref); err != nil {
		triggerToast(w, "err", "Delete failed: "+err.Error())
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	triggerToast(w, "ok", "Snapshot deleted")
	redirect(w, r, d.cfg.BasePath+"/snapshots")
}
