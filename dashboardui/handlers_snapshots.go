package dashboardui

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
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
		// Not-found is an honest absence: render the missing-snapshot page.
		// Any other error is a store failure and must surface as a 500, not
		// masquerade as a missing snapshot (the pre-split behavior hid infra
		// outages behind "No snapshot found").
		if errors.Is(err, snapshot.ErrSnapshotNotFound) {
			p := d.page("Snapshot: "+streamType+"/"+truncate(streamID, titleIDWidth), "/snapshots", r)
			renderPage(w, r, snapshotMissingPage(p, streamType, streamID))

			return
		}

		d.renderError(w, r, http.StatusInternalServerError, "failed to load snapshot")

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

	// Idempotent delete: a snapshot that is already gone is success, not a
	// 500 "Delete failed" toast.
	if err := d.config.SnapshotStore.Delete(r.Context(), ref); err != nil && !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		slog.ErrorContext(
			r.Context(),
			"dashboardui.audit",
			d.audit(r,
				"op",
				"snapshot.delete",
				"stream_type",
				string(ref.Type),
				"stream_id",
				ref.ID.String(),
				"result",
				"error",
				"error",
				err,
			)...,
		)
		triggerToast(w, "err", "Delete failed")
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	slog.InfoContext(
		r.Context(),
		"dashboardui.audit",
		d.audit(r,
			"op",
			"snapshot.delete",
			"stream_type",
			string(ref.Type),
			"stream_id",
			ref.ID.String(),
			"result",
			"ok",
		)...,
	)

	triggerToast(w, "ok", "Snapshot deleted")
	redirect(w, r, d.config.BasePath+"/snapshots")
}
