package dashboardui

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

const contentTypeHTML = "text/html; charset=utf-8"

func renderPage(w http.ResponseWriter, r *http.Request, html string) {
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", "no-store")

	if _, err := w.Write([]byte(html)); err != nil {
		slog.ErrorContext(r.Context(), "dashboardui: write page", "error", err)
	}
}

func renderPartial(w http.ResponseWriter, r *http.Request, html string) {
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", "no-store")

	if _, err := w.Write([]byte(html)); err != nil {
		slog.ErrorContext(r.Context(), "dashboardui: write partial", "error", err)
	}
}

func isPartial(r *http.Request) bool {
	return cqrshtmx.RenderPartial(r)
}

// toastDetail is aliased to the shared cqrshtmx.ToastDetail (same wire shape as
// adminui's). dashboardui's triggerToast writes it directly as the HX-Trigger
// body, while adminui nests it under a named event — hence the function stays
// per-module even though the struct is shared.
type toastDetail = cqrshtmx.ToastDetail

func triggerToast(w http.ResponseWriter, kind, message string) {
	detail, _ := json.Marshal(toastDetail{Message: message, Kind: kind})
	w.Header().Set("HX-Trigger", string(detail))
}

func redirect(w http.ResponseWriter, r *http.Request, path string) {
	if cqrshtmx.IsHTMXRequest(r) {
		w.Header().Set("HX-Redirect", path)
		w.WriteHeader(http.StatusOK)

		return
	}

	http.Redirect(w, r, path, http.StatusSeeOther)
}
