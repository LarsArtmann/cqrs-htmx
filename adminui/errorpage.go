package adminui

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/errorpage"
)

// errorFamilyFor maps an HTTP status to the errorpage visual family.
func errorFamilyFor(status int) errorpage.Family {
	if status >= 500 {
		return errorpage.FamilyInfrastructure
	}
	return errorpage.FamilyRejection
}

// isHTMXRequest reports whether the request comes from an HTMX swap (in which
// case error responses render as bare cards fitting the swap target, not as
// full documents).
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("Hx-Request") == "true"
}

// writeErrorPage writes a templ-components error page instead of a bare
// text/plain http.Error body: the ErrorPage card for HTMX swaps, the card
// inside the panel Layout for navigations. The HTTP status is preserved.
func (h *Handler) writeErrorPage(w http.ResponseWriter, r *http.Request, status int, title, message string) {
	props := errorpage.ErrorPageProps{
		Family:     errorFamilyFor(status),
		StatusCode: status,
		Title:      title,
		Message:    message,
		WayOut:     "Back to dashboard",
		WayOutHref: h.config.BasePath + "/",
	}
	var b strings.Builder
	if err := errorpage.ErrorPage(props).Render(r.Context(), &b); err != nil {
		http.Error(w, title, status)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if isHTMXRequest(r) {
		_, _ = w.Write([]byte(b.String()))
		return
	}
	page := h.page(title, "", nil, r)
	if err := Layout(page, templ.Raw(b.String())).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "adminui: render error page", "error", err)
	}
}

// notFoundHandler serves the styled 404 for any unmatched GET route under the
// panel: the bare NotFound404 view for HTMX swaps, inside the panel Layout
// otherwise.
func (h *Handler) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	props := errorpage.DefaultNotFound404Props()
	props.GoHomeHref = h.config.BasePath + "/"
	props.GoHomeText = "Back to dashboard"
	var b strings.Builder
	if err := errorpage.NotFound404(props).Render(r.Context(), &b); err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	if isHTMXRequest(r) {
		_, _ = w.Write([]byte(b.String()))
		return
	}
	page := h.page("Page not found", "", nil, r)
	if err := Layout(page, templ.Raw(b.String())).Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "adminui: render 404 page", "error", err)
	}
}
