package htmx

import (
	"context"
	"net/http"
)

// HTMX request and response header names, spelled exactly as they travel on
// the wire.
const (
	HeaderRequest        = "Hx-Request"
	HeaderBoosted        = "Hx-Boosted"
	HeaderCurrentURL     = "Hx-Current-Url"
	HeaderHistoryRestore = "Hx-History-Restore-Request"
	HeaderPrompt         = "Hx-Prompt"
	HeaderTarget         = "Hx-Target"
	HeaderTriggerName    = "Hx-Trigger-Name"

	HeaderLocation = "Hx-Location"
	HeaderPushURL  = "Hx-Push-Url"
	// HeaderRedirect is the HTMX response header that makes the browser perform
	// a client-side redirect. Exported because the HTTP error handlers live in a
	// sibling package and write it.
	HeaderRedirect           = "Hx-Redirect"
	HeaderRefresh            = "Hx-Refresh"
	HeaderReplaceURL         = "Hx-Replace-Url"
	HeaderReswap             = "Hx-Reswap"
	HeaderRetarget           = "Hx-Retarget"
	HeaderReselect           = "Hx-Reselect"
	HeaderTrigger            = "Hx-Trigger"
	HeaderTriggerAfterSettle = "Hx-Trigger-After-Settle"
	HeaderTriggerAfterSwap   = "Hx-Trigger-After-Swap"
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

// Request holds parsed HTMX request headers, stored in context
// by HTMXMiddleware. Use FromContext to retrieve it.
type Request struct {
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
func (h *Request) RenderPartial() bool {
	return h.IsHTMX && !h.IsHistoryRestore
}

// ParseRequest extracts all HTMX headers from the request.
func ParseRequest(r *http.Request) *Request {
	return &Request{
		IsHTMX:           r.Header.Get(HeaderRequest) == HeaderTrue,
		IsBoosted:        r.Header.Get(HeaderBoosted) == HeaderTrue,
		IsHistoryRestore: r.Header.Get(HeaderHistoryRestore) == HeaderTrue,
		Target:           r.Header.Get(HeaderTarget),
		TriggerID:        r.Header.Get(HeaderTrigger),
		TriggerName:      r.Header.Get(HeaderTriggerName),
		Prompt:           r.Header.Get(HeaderPrompt),
		CurrentURL:       r.Header.Get(HeaderCurrentURL),
	}
}

// WithRequest stores a parsed Request in the context.
func WithRequest(ctx context.Context, h *Request) context.Context {
	return context.WithValue(ctx, htmxKey{}, h)
}

// FromContext retrieves the parsed Request from context.
// Returns nil if HTMXMiddleware was not applied.
func FromContext(ctx context.Context) *Request {
	h, _ := ctx.Value(htmxKey{}).(*Request)

	return h
}

// htmxBoolField returns a boolean HTMX header value, preferring the parsed
// context from HTMXMiddleware and falling back to the raw header. A nil
// request is never an HTMX request.
func htmxBoolField(r *http.Request, extract func(*Request) bool, header string) bool {
	if r == nil {
		return false
	}

	if h := FromContext(r.Context()); h != nil {
		return extract(h)
	}

	return r.Header.Get(header) == HeaderTrue
}

// htmxStringField returns a string HTMX header value, preferring the parsed
// context from HTMXMiddleware and falling back to the raw header. A nil
// request yields the empty string.
func htmxStringField(r *http.Request, extract func(*Request) string, header string) string {
	if r == nil {
		return ""
	}

	if h := FromContext(r.Context()); h != nil {
		return extract(h)
	}

	return r.Header.Get(header)
}

// IsRequest returns true if the request originates from HTMX.
// Checks the HX-Request header directly — works with or without HTMXMiddleware.
func IsRequest(r *http.Request) bool {
	return htmxBoolField(r, func(h *Request) bool { return h.IsHTMX }, HeaderRequest)
}

// IsBoosted returns true if the request was sent via hx-boost.
func IsBoosted(r *http.Request) bool {
	return htmxBoolField(r, func(h *Request) bool { return h.IsBoosted }, HeaderBoosted)
}

// IsHistoryRestore returns true if the request is a history restoration.
func IsHistoryRestore(r *http.Request) bool {
	return htmxBoolField(
		r,
		func(h *Request) bool { return h.IsHistoryRestore },
		HeaderHistoryRestore,
	)
}

// RenderPartial returns true when the request is from HTMX and is not
// a history restore. Use this to decide whether to render a partial
// HTML fragment or a full page.
func RenderPartial(r *http.Request) bool {
	if h := FromContext(r.Context()); h != nil {
		return h.RenderPartial()
	}

	return r.Header.Get(HeaderRequest) == HeaderTrue &&
		r.Header.Get(HeaderHistoryRestore) != HeaderTrue
}

// Target returns the ID of the target element from the request.
func Target(r *http.Request) string {
	return htmxStringField(r, func(h *Request) string { return h.Target }, HeaderTarget)
}

// Trigger returns the ID of the trigger element from the request.
func Trigger(r *http.Request) string {
	return htmxStringField(r, func(h *Request) string { return h.TriggerID }, HeaderTrigger)
}

// TriggerName returns the name of the trigger element from the request.
func TriggerName(r *http.Request) string {
	return htmxStringField(
		r,
		func(h *Request) string { return h.TriggerName },
		HeaderTriggerName,
	)
}

// Prompt returns the user's response from hx-prompt.
func Prompt(r *http.Request) string {
	return htmxStringField(r, func(h *Request) string { return h.Prompt }, HeaderPrompt)
}

// CurrentURL returns the current URL from the request.
func CurrentURL(r *http.Request) string {
	return htmxStringField(r, func(h *Request) string { return h.CurrentURL }, HeaderCurrentURL)
}
