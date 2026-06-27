package adminui

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

// renderPage writes a full templ component (a page) as HTML.
func renderPage(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := c.Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "adminui: render page", "error", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// renderPartial writes a templ fragment for an HTMX swap (no full document).
func renderPartial(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := c.Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "adminui: render partial", "error", err)
	}
}

// toastDetail is the payload dispatched as the adminui:toast HTMX trigger event.
type toastDetail struct {
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// triggerToast queues a toast on the client by setting the HX-Trigger header.
// kind is "" (default), "ok", or "err".
func triggerToast(w http.ResponseWriter, kind, message string) {
	detail, err := json.Marshal(toastDetail{Message: message, Kind: kind})
	if err != nil {
		return // unreachable: marshaling a static struct never fails
	}
	// Preserve any existing trigger events (other HX-Trigger entries).
	triggers := map[string]json.RawMessage{}
	if h := w.Header().Get("HX-Trigger"); h != "" {
		_ = json.Unmarshal([]byte(h), &triggers) // best-effort merge
	}
	triggers["adminui:toast"] = detail
	merged, err := json.Marshal(triggers)
	if err != nil {
		return
	}
	w.Header().Set("HX-Trigger", string(merged))
}

// redirect issues an HTMX-aware redirect: HX-Redirect for HTMX requests, a
// standard 303 for normal navigation.
func redirect(w http.ResponseWriter, r *http.Request, path string) {
	// HTMX clients follow HX-Redirect; browsers follow the 303. The path is always
	// built from Config.BasePath + a fixed suffix, so it cannot be user-controlled.
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", path)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, path, http.StatusSeeOther) //nolint:gosec // G710: path is config-derived, not user input
}
