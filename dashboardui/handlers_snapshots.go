package dashboardui

import (
	"fmt"
	"log/slog"
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
	d.renderStreamIndex(w, r, "Snapshots", "/snapshots", d.renderSnapshotsIndex)
}

func (d *Dashboard) renderSnapshotsIndex(p pageData, listings []listing.StreamListing, page paginationState) string {
	return d.renderStreamListingPage(p, listings, page, streamListPageConfig{
		subtitle:   "Inspect snapshot state for any aggregate. Snapshots store a point-in-time cache of aggregate state to accelerate loading.",
		emptyTitle: "No aggregates found",
		emptyMsg:   "Configure a StreamReader to browse snapshots by aggregate.",
		pagePath:   "/snapshots",
		linkPath:   "/snapshots",
		linkText:   "View",
	})
}

func (d *Dashboard) snapshotDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType := r.PathValue("type")
	streamID := r.PathValue("id")

	ref, err := streamRefFromRequest(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "invalid stream reference")

		return
	}

	snap, err := d.cfg.SnapshotStore.Load(r.Context(), ref)
	if err != nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return fmt.Sprintf(
				`<div class="empty-state"><h2>No snapshot found</h2><p>No snapshot exists for %s/<code>%s</code>.</p></div>`,
				esc(streamType),
				esc(truncate(streamID, snapshotIDWidth)),
			)
		}))

		return
	}

	if snap == nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
		renderPage(w, r, d.renderLayout(p, func() string {
			return emptyState("No snapshot", "")
		}))

		return
	}

	p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
	html := d.renderSnapshotDetail(p, ref, snap)
	renderPage(w, r, html)
}

func (d *Dashboard) renderSnapshotDetail(p pageData, ref id.StreamRef, snap *snapshot.Snapshot) string {
	return d.renderLayout(p, func() string {
		var b strings.Builder

		b.WriteString(`<div class="page-header">`)
		fmt.Fprintf(
			&b,
			`<h2>Snapshot: <code class="copyable" data-copyable="%s" title="Click to copy">%s</code></h2>`,
			esc(ref.ID.String()),
			esc(ref.ID.String()),
		)
		fmt.Fprintf(
			&b,
			`<div class="page-subtitle">Version %s · Created %s (%s)</div>`,
			esc(snap.Version.String()),
			esc(snap.CreatedAt.Format(time.RFC3339)),
			esc(relativeTime(snap.CreatedAt)),
		)
		b.WriteString(`</div>`)

		if !p.ReadOnly {
			fmt.Fprintf(
				&b,
				`<form method="POST" action="%s/snapshots/%s/%s/delete" class="section-gap-lg" onsubmit="return confirm('Delete this snapshot? This cannot be undone.')">`,
				p.BasePath,
				esc(string(ref.Type)),
				esc(ref.ID.String()),
			)
			fmt.Fprintf(&b, `<input type="hidden" name="_csrf" value="%s"/>`, esc(p.CSRFToken))
			fmt.Fprintf(
				&b,
				`<button type="submit" class="btn btn-danger" aria-label="Delete snapshot for %s">Delete Snapshot</button>`,
				esc(ref.ID.String()),
			)
			b.WriteString(`</form>`)
		}

		b.WriteString(`<h3>Metadata</h3>`)
		b.WriteString(`<table class="meta-table section-gap-lg">`)
		metaRow(&b, "Stream Type", esc(string(snap.StreamType)))
		metaRowCopyable(&b, "Stream ID", esc(snap.StreamID.String()), snap.StreamID.String())
		metaRow(&b, "Version", esc(snap.Version.String()))
		metaRow(&b, "Created At", esc(snap.CreatedAt.Format(time.RFC3339)))
		metaRow(&b, "State Size", esc(humanByteSize(len(snap.State))))
		b.WriteString(`</table></div>`)

		b.WriteString(`<h3>State</h3>`)

		stateDisplay := d.renderSnapshotState(snap.State)
		fmt.Fprintf(&b, `<pre class="code-block"><code>%s</code></pre>`, stateDisplay)

		return b.String()
	})
}

func (d *Dashboard) renderSnapshotState(state []byte) string {
	if len(state) == 0 {
		return esc("(empty)")
	}

	out, err := d.cfg.PayloadRenderer.Render(state, codec.EncodingJSON)
	if err == nil && len(out) > 0 {
		return esc(string(out))
	}

	return esc(string(state))
}

func (d *Dashboard) snapshotDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if d.cfg.SnapshotStore == nil {
		renderError(w, r, http.StatusBadRequest, "snapshot store not configured")

		return
	}

	ref, err := streamRefFromRequest(r)
	if err != nil {
		renderError(w, r, http.StatusBadRequest, "invalid stream reference")

		return
	}

	if err := d.cfg.SnapshotStore.Delete(r.Context(), ref); err != nil {
		slog.InfoContext(
			r.Context(),
			"dashboardui.audit",
			"op",
			"snapshot.delete",
			"stream_type",
			string(ref.Type),
			"stream_id",
			ref.ID.String(),
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
		"snapshot.delete",
		"stream_type",
		string(ref.Type),
		"stream_id",
		ref.ID.String(),
		"result",
		"ok",
	)

	triggerToast(w, "ok", "Snapshot deleted")
	redirect(w, r, d.cfg.BasePath+"/snapshots")
}
