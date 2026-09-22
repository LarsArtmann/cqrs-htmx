package dashboardui

import (
	"log/slog"
	"net/http"
)

// ===== Snapshots =====

func (d *Dashboard) snapshotsIndexHandler(w http.ResponseWriter, r *http.Request) {
	d.renderStreamIndex(w, r, "Snapshots", "/snapshots", snapshotsIndexPage)
}

func (d *Dashboard) snapshotDetailHandler(w http.ResponseWriter, r *http.Request) {
	streamType := r.PathValue("type")
	streamID := r.PathValue("id")

	ref, err := streamRefFromRequest(r)
	if err != nil {
		d.renderError(w, r, http.StatusBadRequest, "invalid stream reference")

		return
	}

	snap, err := d.config.SnapshotStore.Load(r.Context(), ref)
	if err != nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
		renderPage(w, r, snapshotMissingPage(p, streamType, streamID))

		return
	}

	if snap == nil {
		p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
		renderPage(w, r, snapshotEmptyPage(p))

		return
	}

	p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
	renderPage(w, r, snapshotDetailPage(p, ref, snap, d.snapshotStateText(snap.State)))
}

func (d *Dashboard) snapshotDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if d.config.SnapshotStore == nil {
		d.renderError(w, r, http.StatusBadRequest, "snapshot store not configured")

		return
	}

	ref, err := streamRefFromRequest(r)
	if err != nil {
		d.renderError(w, r, http.StatusBadRequest, "invalid stream reference")

		return
	}

	if err := d.config.SnapshotStore.Delete(r.Context(), ref); err != nil {
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
	redirect(w, r, d.config.BasePath+"/snapshots")
}
