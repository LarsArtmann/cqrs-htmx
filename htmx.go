package cqrshtmx

import (
	"context"
	"net/http"
)

const (
	headerRequest        = "HX-Request"
	headerBoosted        = "HX-Boosted"
	headerCurrentURL     = "HX-Current-URL"
	headerHistoryRestore = "HX-History-Restore-Request"
	headerPrompt         = "HX-Prompt"
	headerTarget         = "HX-Target"
	headerTriggerName    = "HX-Trigger-Name"

	headerLocation           = "HX-Location"
	headerPushURL            = "HX-Push-Url"
	headerRedirect           = "HX-Redirect"
	headerRefresh            = "HX-Refresh"
	headerReplaceURL         = "HX-Replace-Url"
	headerReswap             = "HX-Reswap"
	headerRetarget           = "HX-Retarget"
	headerReselect           = "HX-Reselect"
	headerTrigger            = "HX-Trigger"
	headerTriggerAfterSettle = "HX-Trigger-After-Settle"
	headerTriggerAfterSwap   = "HX-Trigger-After-Swap"
)

type htmxKey struct{}

// HeaderTrue is the HTMX header value for boolean true headers.
// Use this instead of hardcoding "true" in tests or middleware.
const HeaderTrue = "true"

// SwapStrategy defines how HTMX swaps content into the DOM.
type SwapStrategy string

// SwapStrategy constants define HTMX content swap methods.
const (
	SwapInnerHTML   SwapStrategy = "innerHTML"
	SwapOuterHTML   SwapStrategy = "outerHTML"
	SwapBeforeBegin SwapStrategy = "beforebegin"
	SwapAfterBegin  SwapStrategy = "afterbegin"
	SwapBeforeEnd   SwapStrategy = "beforeend"
	SwapAfterEnd    SwapStrategy = "afterend"
	SwapDelete      SwapStrategy = "delete"
	SwapNone        SwapStrategy = "none"
)

// Valid reports whether s is a known HTMX swap strategy.
func (s SwapStrategy) Valid() bool {
	switch s {
	case SwapInnerHTML, SwapOuterHTML, SwapBeforeBegin, SwapAfterBegin,
		SwapBeforeEnd, SwapAfterEnd, SwapDelete, SwapNone:
		return true
	}

	return false
}

// HTMXRequest holds parsed HTMX request headers, stored in context
// by HTMXMiddleware. Use HTMXFromContext to retrieve it.
type HTMXRequest struct {
	IsHTMX           bool
	IsBoosted        bool
	IsHistoryRestore bool
	Target           string
	// TriggerID is the id attribute of the element that triggered the request.
	// Maps to the HX-Trigger header.
	TriggerID string
	// TriggerName is the name attribute of the element that triggered the request.
	// Maps to the HX-Trigger-Name header.
	TriggerName string
	Prompt      string
	CurrentURL  string
}

// RenderPartial returns true when a partial HTML response should be rendered.
// True for HTMX requests that are not history restorations.
func (h *HTMXRequest) RenderPartial() bool {
	return h.IsHTMX && !h.IsHistoryRestore
}

// parseHTMXRequest extracts all HTMX headers from the request.
func parseHTMXRequest(r *http.Request) *HTMXRequest {
	return &HTMXRequest{
		IsHTMX:           r.Header.Get(headerRequest) == HeaderTrue,
		IsBoosted:        r.Header.Get(headerBoosted) == HeaderTrue,
		IsHistoryRestore: r.Header.Get(headerHistoryRestore) == HeaderTrue,
		Target:           r.Header.Get(headerTarget),
		TriggerID:        r.Header.Get(headerTrigger),
		TriggerName:      r.Header.Get(headerTriggerName),
		Prompt:           r.Header.Get(headerPrompt),
		CurrentURL:       r.Header.Get(headerCurrentURL),
	}
}

// WithHTMX stores a parsed HTMXRequest in the context.
func WithHTMX(ctx context.Context, h *HTMXRequest) context.Context {
	return context.WithValue(ctx, htmxKey{}, h)
}

// HTMXFromContext retrieves the parsed HTMXRequest from context.
// Returns nil if HTMXMiddleware was not applied.
func HTMXFromContext(ctx context.Context) *HTMXRequest {
	h, _ := ctx.Value(htmxKey{}).(*HTMXRequest)

	return h
}

// htmxBoolField returns a boolean HTMX header value, preferring the parsed
// context from HTMXMiddleware and falling back to the raw header.
func htmxBoolField(r *http.Request, extract func(*HTMXRequest) bool, header string) bool {
	if h := HTMXFromContext(r.Context()); h != nil {
		return extract(h)
	}

	return r.Header.Get(header) == HeaderTrue
}

// htmxStringField returns a string HTMX header value, preferring the parsed
// context from HTMXMiddleware and falling back to the raw header.
func htmxStringField(r *http.Request, extract func(*HTMXRequest) string, header string) string {
	if h := HTMXFromContext(r.Context()); h != nil {
		return extract(h)
	}

	return r.Header.Get(header)
}

// IsHTMXRequest returns true if the request originates from HTMX.
// Checks the HX-Request header directly — works with or without HTMXMiddleware.
func IsHTMXRequest(r *http.Request) bool {
	return htmxBoolField(r, func(h *HTMXRequest) bool { return h.IsHTMX }, headerRequest)
}

// IsBoosted returns true if the request was sent via hx-boost.
func IsBoosted(r *http.Request) bool {
	return htmxBoolField(r, func(h *HTMXRequest) bool { return h.IsBoosted }, headerBoosted)
}

// IsHistoryRestore returns true if the request is a history restoration.
func IsHistoryRestore(r *http.Request) bool {
	return htmxBoolField(
		r,
		func(h *HTMXRequest) bool { return h.IsHistoryRestore },
		headerHistoryRestore,
	)
}

// RenderPartial returns true when the request is from HTMX and is not
// a history restore. Use this to decide whether to render a partial
// HTML fragment or a full page.
func RenderPartial(r *http.Request) bool {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.RenderPartial()
	}

	return r.Header.Get(headerRequest) == HeaderTrue &&
		r.Header.Get(headerHistoryRestore) != HeaderTrue
}

// HTMXTarget returns the ID of the target element from the request.
func HTMXTarget(r *http.Request) string {
	return htmxStringField(r, func(h *HTMXRequest) string { return h.Target }, headerTarget)
}

// HTMXTrigger returns the ID of the trigger element from the request.
func HTMXTrigger(r *http.Request) string {
	return htmxStringField(r, func(h *HTMXRequest) string { return h.TriggerID }, headerTrigger)
}

// HTMXTriggerName returns the name of the trigger element from the request.
func HTMXTriggerName(r *http.Request) string {
	return htmxStringField(
		r,
		func(h *HTMXRequest) string { return h.TriggerName },
		headerTriggerName,
	)
}

// HTMXPrompt returns the user's response from hx-prompt.
func HTMXPrompt(r *http.Request) string {
	return htmxStringField(r, func(h *HTMXRequest) string { return h.Prompt }, headerPrompt)
}

// HTMXCurrentURL returns the current URL from the request.
func HTMXCurrentURL(r *http.Request) string {
	return htmxStringField(r, func(h *HTMXRequest) string { return h.CurrentURL }, headerCurrentURL)
}
