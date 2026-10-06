package dashboardui

import (
	"log/slog"
	"net/http"
)

// parseAfterCursor parses a non-empty after cursor with the given ID-parse
// function. A malformed cursor renders a 400 error page (and reports
// ok=false) instead of silently resetting pagination to the first page,
// which used to strand users on a "Next" loop that never advanced.
func parseAfterCursor[T any](
	d *Dashboard,
	w http.ResponseWriter,
	r *http.Request,
	raw string,
	parse func(string) (T, error),
) (T, bool) {
	var zero T

	if raw == "" {
		return zero, true
	}

	parsed, err := parse(raw)
	if err != nil {
		slog.WarnContext(r.Context(), "dashboardui: malformed after cursor",
			"after", raw, "error", err)
		d.renderError(w, r, http.StatusBadRequest, "invalid after cursor")

		return zero, false
	}

	return parsed, true
}
