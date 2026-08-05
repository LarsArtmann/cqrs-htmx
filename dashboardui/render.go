package dashboardui

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
)

const contentTypeHTML = "text/html; charset=utf-8"

// writeHTML writes pre-rendered HTML with no-store caching, logging under the
// given label.
func writeHTML(w http.ResponseWriter, r *http.Request, html, label string) {
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", "no-store")

	if _, err := w.Write([]byte(html)); err != nil {
		slog.ErrorContext(r.Context(), "dashboardui: "+label, "error", err)
	}
}

func renderPage(w http.ResponseWriter, r *http.Request, html string) {
	writeHTML(w, r, html, "write page")
}

// toastDetail is aliased to the shared cqrshtmx.ToastDetail (same wire shape as
// adminui's). dashboardui's triggerToast writes it directly as the HX-Trigger
// body, while adminui nests it under a named event — hence the function stays
// per-module even though the struct is shared.
type toastDetail = cqrshtmx.ToastDetail

func triggerToast(w http.ResponseWriter, kind, message string) {
	detail, _ := json.Marshal(toastDetail{Message: message, Kind: kind})
	w.Header().Set("Hx-Trigger", string(detail))
}

// renderError logs the full error and shows a generic message to the user.
// Safe to call with a nil request.
func renderError(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	ctx := context.Background() //nolint:contextcheck // fallback when request is nil; replaced by r.Context() below
	path := ""

	if r != nil {
		ctx = r.Context()
		path = r.URL.Path
	}

	slog.ErrorContext(ctx, "dashboardui: handler error",
		"status", statusCode, "message", message, "path", path)
	http.Error(w, message, statusCode)
}

// emptyState renders the standard empty-state panel.
func emptyState(title, message string) string {
	if message == "" {
		return fmt.Sprintf(`<div class="empty-state"><h2>%s</h2></div>`, esc(title))
	}

	return fmt.Sprintf(`<div class="empty-state"><h2>%s</h2><p>%s</p></div>`, esc(title), esc(message))
}

// isHTMXRequest returns true when the request came from an HTMX-boosted
// link or explicit hx-get/hx-post. When true, handlers render only the main
// content (no full HTML shell) for smaller payloads and faster swaps.
func isHTMXRequest(r *http.Request) bool {
	return r != nil && r.Header.Get("HX-Request") == "true"
}

func redirect(w http.ResponseWriter, r *http.Request, path string) {
	cqrshtmx.HTMXRedirect(w, r, path)
}

// renderStreamIndex renders one of the dashboard's stream-listing pages
// (aggregates, snapshots, time-travel). It binds the page title and base path,
// looks up streams via the configured reader with cursor-based pagination,
// runs the per-page renderer, and writes the result. Shared across the three
// stream-index handlers so the common prelude doesn't drift.
func (d *Dashboard) renderStreamIndex(
	w http.ResponseWriter,
	r *http.Request,
	title, basePath string,
	render func(pageData, []listing.StreamListing, paginationState) string,
) {
	p := d.page(title, basePath, r)
	listings, page := d.listStreamsPaged(r)
	page = page.WithCountInfo(len(listings))
	renderPage(w, r, render(p, listings, page))
}
