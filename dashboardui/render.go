package dashboardui

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/httputil"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/errorpage"
	"github.com/larsartmann/templ-components/icons"
	"github.com/larsartmann/templ-components/utils"
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

// renderError logs the full error and renders a styled, family-aware error
// page (templ-components errorpage): a bare card for HTMX swaps, a minimal
// document loading the dashboard stylesheets otherwise. Safe to call with a
// nil request (renders the shell-less plain fallback path).
func (d *Dashboard) renderError(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	ctx := context.Background() //nolint:contextcheck // fallback when request is nil; replaced by r.Context() below
	nonce := ""
	path := ""

	if r != nil {
		ctx = r.Context()
		nonce = httputil.NonceFromRequest(r)
		path = r.URL.Path
	}

	slog.ErrorContext(ctx, "dashboardui: handler error",
		"status", statusCode, "message", message, "path", path)

	props := errorpage.DefaultErrorPageProps()
	props.Family = statusToFamily(statusCode)
	props.StatusCode = statusCode
	props.Title = http.StatusText(statusCode)
	props.Message = message
	props.Nonce = nonce
	props.Timestamp = ""

	var b strings.Builder
	if err := errorpage.ErrorPage(props).Render(ctx, &b); err != nil {
		// Library render failed; keep the plain-text fallback so the user
		// still sees status + message.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(statusCode)
		_, _ = fmt.Fprintf(w, "%s\n", message)

		return
	}

	w.Header().Set("Content-Type", contentTypeHTML)
	w.WriteHeader(statusCode)

	if isHTMXRequest(r) {
		_, _ = w.Write([]byte(b.String()))

		return
	}

	_, _ = w.Write([]byte(d.renderErrorShell(props.Title, b.String())))
}

// renderErrorShell wraps pre-rendered error markup in a minimal HTML document
// that loads the dashboard stylesheets. It bridges the library's bare
// component output and the full renderLayout shell (which needs pageData);
// the CSS keeps the card centered without duplicating the layout.
func (d *Dashboard) renderErrorShell(title, inner string) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString(
		"<meta charset=\"utf-8\"/>\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"/>\n",
	)
	fmt.Fprintf(&b, "<title>%s</title>\n", esc(title))
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%s/-/dashboard.css\"/>\n", d.config.BasePath)
	fmt.Fprintf(&b, "<link rel=\"stylesheet\" href=\"%s/-/dashboard-tw.css\"/>\n", d.config.BasePath)
	b.WriteString("</head>\n<body>\n<div class=\"error-shell\">\n")
	b.WriteString(inner)
	b.WriteString("\n</div>\n</body>\n</html>")

	return b.String()
}

// statusToFamily maps an HTTP status code to the error-page family the
// library styles with. Client errors read as Rejection (helpful tone),
// conflicts as Conflict, availability blips as Transient, everything else
// (500s) as Infrastructure.
func statusToFamily(statusCode int) errorpage.Family {
	switch {
	case statusCode == http.StatusConflict:
		return errorpage.FamilyConflict
	case statusCode == http.StatusServiceUnavailable,
		statusCode == http.StatusBadGateway,
		statusCode == http.StatusGatewayTimeout:
		return errorpage.FamilyTransient
	case statusCode >= 400 && statusCode < 500:
		return errorpage.FamilyRejection
	default:
		return errorpage.FamilyInfrastructure
	}
}

// emptyState renders the standard empty-state panel (library component,
// default inbox icon).
func emptyState(title, message string) string {
	return emptyStateIcon(icons.Inbox, title, message)
}

// emptyStateIcon renders the library's EmptyState with a page-specific icon
// (callers pass their nav icon for visual continuity). The ctx is a plain
// context.Background because empty states render inside layout closures that
// carry no request scope: the render is a synchronous, pure string build (a
// strings.Writer cannot fail), so cancellation is not a concern here.
func emptyStateIcon(icon icons.Name, title, message string) string {
	var b strings.Builder

	props := display.EmptyStateProps{
		BaseProps:   utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Title:       title,
		TitleTag:    "h2",
		Description: message,
		Icon:        icon,
		ActionText:  "",
		ActionHref:  "",
		ActionAttrs: nil,
	}
	_ = display.EmptyState(props).Render(context.Background(), &b)

	return b.String()
}

// isHTMXRequest returns true when the request came from an HTMX-boosted
// link or explicit hx-get/hx-post. When true, handlers render only the main
// content (no full HTML shell) for smaller payloads and faster swaps.
func isHTMXRequest(r *http.Request) bool {
	return r != nil && r.Header.Get("Hx-Request") == "true"
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
