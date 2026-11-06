package dashboardui

import (
	"net/http"
	"strconv"
	"time"
)

// ===== Command Fates =====

// commandFatesIndexHandler serves the Command fates panel: the newest
// commandlifecycle outcomes (rejected, failed, retried, dead-lettered,
// completed), newest first. The view is a fixed window over the journal
// tail — every fate the recorder wrote stays counted in the total.
func (d *Dashboard) commandFatesIndexHandler(w http.ResponseWriter, r *http.Request) {
	p := d.page("Command Fates", "/command-fates", r)

	pageSize := parsePageSize(r, d.config.PageSize)

	fates, total, err := d.loadCommandFates(r.Context(), pageSize)
	if err != nil {
		d.renderError(w, r, http.StatusInternalServerError, "failed to load command fates")

		return
	}

	renderPage(w, r, commandFatesPage(p, fates, total))
}

// fateTimeLabel renders a fate timestamp; a zero time (a payload this build
// could not decode) renders as a dash instead of the zero-year.
func fateTimeLabel(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	return t.Format("2006-01-02 15:04:05")
}

// fateAttemptLabel renders the attempt number; zero means the payload
// carried none (or could not be decoded).
func fateAttemptLabel(attempt int) string {
	if attempt == 0 {
		return "-"
	}

	return strconv.Itoa(attempt)
}

// familyBadgeLabel joins the errorfamily classification with its machine
// code when the rejecting chain declared one.
func familyBadgeLabel(family, errorCode string) string {
	if errorCode != "" {
		return family + "/" + errorCode
	}

	return family
}
