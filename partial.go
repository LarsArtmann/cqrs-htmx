package cqrshtmx

import (
	"fmt"
	stdhtml "html"
	"net/http"
	"strings"
)

// RenderTemplComponent renders the partial component for HTMX partial requests
// and the full component for all other requests. It sets Content-Type to
// text/html; charset=utf-8.
//
// This is the standalone helper for non-CQRS handlers — routes that don't go
// through [App.Query] or [App.Command] but still need the partial-vs-full-page
// branching. For CQRS handlers, use [RenderPartialOrFull] instead.
//
// Usage:
//
//	func(w http.ResponseWriter, r *http.Request) {
//	    data := loadData()
//	    cqrshtmx.RenderTemplComponent(w, r, itemPartial(data), itemsPage(data))
//	}
func RenderTemplComponent(w http.ResponseWriter, r *http.Request, partial, full TemplComponent) error {
	w.Header().Set("Content-Type", ContentTypeHTML)

	c := full
	if RenderPartial(r) {
		c = partial
	}

	return c.Render(r.Context(), w) //nolint:wrapcheck // passthrough, matching RenderTempl pattern
}

// OOBHTML wraps an HTML fragment with HTMX out-of-band swap attributes so it
// can be returned alongside a primary response. HTMX processes elements with
// hx-swap-oob alongside the main swap target, updating them independently.
//
// The fragment is interpolated verbatim (it is HTML by contract); the id is
// escaped for a double-quoted attribute context, so an id containing quotes
// or angle brackets cannot break out of the generated attribute.
//
// By default, elements are replaced using innerHTML. Pass a [SwapStrategy] to
// use a different swap method:
//
//	html := cqrshtmx.OOBHTML("notifications", "<div>new item</div>",
//	    cqrshtmx.SwapBeforeEnd)
//
// Works for HTTP responses and SSE event data.
func OOBHTML(id, fragment string, swapStrategy ...SwapStrategy) string {
	swap := "true"
	if len(swapStrategy) > 0 && swapStrategy[0] != "" {
		swap = string(swapStrategy[0])
	}

	if strings.Contains(fragment, "hx-swap-oob") {
		return fragment
	}

	return fmt.Sprintf(
		`<div id="%s" hx-swap-oob="%s">%s</div>`,
		stdhtml.EscapeString(id), swap, fragment,
	)
}
