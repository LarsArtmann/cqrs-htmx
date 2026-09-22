package dashboardui

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ===== Dead-Letter Queue =====

func (d *Dashboard) dlqIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Dead Letters", "/dead-letters", r)

	links := d.buildDLQProjectionLinks(r.Context())

	renderPage(w, r, dlqIndexPage(p, links))
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

	p := d.page("Dead Letters: "+proj, "/dead-letters", r)
	renderPage(w, r, dlqPage(p, proj, entries))
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

	var payload []byte
	if entry.Event != nil {
		payload = payloadBytes(d.config.PayloadRenderer, entry.Event)
	}

	p := d.page("Dead Letter: "+truncate(eventID, eventIDWidth), "/dead-letters", r)
	renderPage(w, r, dlqEntryDetailPage(p, proj, entry, payload))
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
