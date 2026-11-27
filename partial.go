package cqrshtmx

import (
	"bytes"
	"context"
	stdhtml "html"
	"io"
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// RenderTemplComponent renders the partial component for HTMX partial requests
// and the full component for all other requests. It sets Content-Type to
// text/html; charset=utf-8.
//
// This is the standalone helper for non-CQRS handlers — routes that don't go
// through [App.Query] or [App.Command] but still need the partial-vs-full-page
// branching. For CQRS handlers, use [RenderPartialOrFull] or [Views] instead.
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
	if htmx.RenderPartial(r) {
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
// Works for HTTP responses and SSE event data. Prefer [OOBTempl] when the
// fragment is a templ component: same wire bytes, no string concatenation.
func OOBHTML(id, fragment string, swapStrategy ...SwapStrategy) string {
	if strings.Contains(fragment, "hx-swap-oob") {
		return fragment
	}

	var out strings.Builder
	// The writer cannot fail: strings.Builder never returns an error.
	_ = writeOOBEnvelope(&out, id, oobSwapAttribute(swapStrategy), func(w io.Writer) error {
		_, err := io.WriteString(w, fragment)

		return err //nolint:wrapcheck // writer error passthrough
	})

	return out.String()
}

// OOBTempl wraps a templ component as an out-of-band swap fragment. It is the
// component form of [OOBHTML] and emits byte-identical wire output, so a
// component never has to be rendered to a string first.
//
// A component whose rendered output already carries an hx-swap-oob attribute
// is passed through untouched, exactly like the string form.
//
// Usage:
//
//	@cqrshtmx.OOBTempl("item-count", itemCountBadge(items))
func OOBTempl( //nolint:ireturn // the structural component is the return contract by design
	id string,
	component TemplComponent,
	swapStrategy ...SwapStrategy,
) TemplComponent {
	return oobComponent{id: id, swap: oobSwapAttribute(swapStrategy), inner: component}
}

type oobComponent struct {
	id    string
	swap  string
	inner TemplComponent
}

func (c oobComponent) Render(ctx context.Context, w io.Writer) error {
	// Buffer only to detect a fragment that already carries the attribute:
	// double-wrapping would nest the element inside a second swap target.
	var body bytes.Buffer
	if err := c.inner.Render(ctx, &body); err != nil {
		return err //nolint:wrapcheck // component error passthrough, matching RenderTempl
	}

	if strings.Contains(body.String(), "hx-swap-oob") {
		_, err := w.Write(body.Bytes())

		return err //nolint:wrapcheck // writer error passthrough
	}

	return writeOOBEnvelope(w, c.id, c.swap, func(w io.Writer) error {
		_, err := w.Write(body.Bytes())

		return err //nolint:wrapcheck // writer error passthrough
	})
}

// writeOOBEnvelope is the single source of the out-of-band wire bytes: both
// [OOBHTML] and [OOBTempl] (and therefore [OOB] views) write through it, so
// the string and component forms cannot drift.
func writeOOBEnvelope(
	w io.Writer,
	id, swap string,
	body func(io.Writer) error,
) error {
	if _, err := io.WriteString(w, `<div id="`); err != nil {
		return err //nolint:wrapcheck // writer error passthrough
	}

	if _, err := io.WriteString(w, stdhtml.EscapeString(id)); err != nil {
		return err //nolint:wrapcheck // writer error passthrough
	}

	if _, err := io.WriteString(w, `" hx-swap-oob="`); err != nil {
		return err //nolint:wrapcheck // writer error passthrough
	}

	if _, err := io.WriteString(w, swap); err != nil {
		return err //nolint:wrapcheck // writer error passthrough
	}

	if _, err := io.WriteString(w, `">`); err != nil {
		return err //nolint:wrapcheck // writer error passthrough
	}

	if err := body(w); err != nil {
		return err
	}

	_, err := io.WriteString(w, `</div>`)

	return err //nolint:wrapcheck // writer error passthrough
}

// oobSwapAttribute resolves the hx-swap-oob attribute value: "true" by
// default (innerHTML), or the explicit strategy.
func oobSwapAttribute(swapStrategy []SwapStrategy) string {
	if len(swapStrategy) > 0 && swapStrategy[0] != "" {
		return string(swapStrategy[0])
	}

	return "true"
}
