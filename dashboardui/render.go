package dashboardui

import (
	"context"
	"encoding/json/jsontext"
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

	"github.com/a-h/templ"
)

const contentTypeHTML = "text/html; charset=utf-8"

// writeHTML renders a component with no-store caching, logging under the
// given label. Used for HTMX partial responses.
func writeHTML(w http.ResponseWriter, r *http.Request, c templ.Component, label string) {
	w.Header().Set("Content-Type", contentTypeHTML)
	w.Header().Set("Cache-Control", "no-store")

	if err := c.Render(r.Context(), w); err != nil {
		slog.ErrorContext(r.Context(), "dashboardui: "+label, "error", err)
	}
}

func renderPage(w http.ResponseWriter, r *http.Request, c templ.Component) {
	writeHTML(w, r, c, "write page")
}

// toastDetail is aliased to the shared cqrshtmx.ToastDetail (same wire shape as
// adminui's).
type toastDetail = cqrshtmx.ToastDetail

// triggerToast emits a "dashboardui:toast" HX-Trigger event; the dashboardJS
// listener bridges it to the library's tcShowToast (feedback.ToastContainer).
// Mirrors adminui's named-event contract, including the best-effort merge with
// any HX-Trigger events already set on the response.
func triggerToast(w http.ResponseWriter, kind, message string) {
	detail, _ := json.Marshal(toastDetail{Message: message, Kind: kind})

	var triggers map[string]jsontext.Value

	if h := w.Header().Get("Hx-Trigger"); h != "" {
		_ = json.Unmarshal([]byte(h), &triggers) // best-effort merge
	}

	if triggers == nil {
		triggers = map[string]jsontext.Value{}
	}

	triggers["dashboardui:toast"] = detail
	merged, _ := json.Marshal(triggers)
	w.Header().Set("Hx-Trigger", string(merged))
}

// renderError logs the full error and renders a styled, family-aware error
// page (templ-components errorpage): a bare card for HTMX swaps, a minimal
// document loading the dashboard stylesheets otherwise. Safe to call with a
// nil request (renders the shell-less plain fallback path).
func (d *Dashboard) renderError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message string,
) {
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

	if r != nil && cqrshtmx.IsHTMXRequest(r) {
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
	fmt.Fprintf(
		&b,
		"<link rel=\"stylesheet\" href=\"%s/-/dashboard-tw.css\"/>\n",
		d.config.BasePath,
	)
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
func emptyState(ctx context.Context, title, message string) string {
	return emptyStateIcon(ctx, icons.Inbox, title, message)
}

// emptyStateIcon renders the library's EmptyState with a page-specific icon
// (callers pass their nav icon for visual continuity). The ctx threads the
// request scope through to the templ render, matching the other library
// component helpers (badgeHTML, statCardHTML).
func emptyStateIcon(ctx context.Context, icon icons.Name, title, message string) string {
	var b strings.Builder

	//nolint:modernize // nested BaseProps is deliberate: promoted keys crash exhaustruct_v5 v5.0.3 (makeslice panic)
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
	_ = display.EmptyState(props).Render(ctx, &b)

	return b.String()
}

// listNoteCountHTML renders the count-only ListNote variant ("Showing N
// items.") for complete lists whose whole message is the count — the DLQ
// table being the canonical case, where the count is the Replay All /
// Purge All blast radius. AriaLabel carries the domain noun the generic
// visible text lacks ("Dead letter count").
func listNoteCountHTML(ctx context.Context, shown int, ariaLabel string) string {
	var b strings.Builder

	//nolint:modernize // nested BaseProps is deliberate: promoted keys crash exhaustruct_v5 v5.0.3 (makeslice panic)
	props := display.ListNoteProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: ariaLabel, Nonce: ""},
		Shown:     shown,
		Total:     0,
		Variant:   display.ListNoteCount,
	}
	_ = display.ListNote(props).Render(ctx, &b)

	return b.String()
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
	render func(context.Context, pageData, []listing.StreamListing, paginationState) string,
) {
	p := d.page(title, basePath, r)
	listings, page := d.listStreamsPaged(r)
	page = page.WithCountInfo(len(listings))
	renderPage(w, r, render(r.Context(), p, listings, page))
}
