package adminui

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"encoding/json/jsontext"

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
	triggers := map[string]jsontext.Value{}
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
// redirect issues an HTMX-aware redirect: HX-Redirect for HTMX requests, a
// standard 303 for normal navigation. The target must be a same-origin path:
// safeRedirect rejects anything that is not root-relative, preventing open-
// redirect attacks even when the path is derived from user-controlled input
// (e.g. a tenant id taken from the URL).
func redirect(w http.ResponseWriter, r *http.Request, path string) {
	path = safeRedirectPath(path)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", path)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, path, http.StatusSeeOther) //nolint:gosec // G710: safeRedirectPath guarantees same-origin
}

// safeRedirectPath returns path if it is a safe same-origin redirect target
// (starts with "/" but not "//"), otherwise "/". Protocol-relative URLs ("//")
// and scheme-bearing URLs ("https://...") are rejected because browsers would
// follow them off-site.
func safeRedirectPath(path string) string {
	if path == "" || path[0] != '/' || len(path) > 1 && path[1] == '/' {
		return "/"
	}
	return path
}
