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
	headerTriggerID      = "HX-Trigger"
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

type htmxContextKey string

const htmxKey htmxContextKey = "cqrshtmx_htmx_request"

// HeaderTrue is the HTMX header value for boolean true headers.
// Use this instead of hardcoding "true" in tests or middleware.
const HeaderTrue = "true"

// headerTrue is the unexported alias for internal consistency.
// Internal use only; new code should use HeaderTrue.
const headerTrue = HeaderTrue

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

// HTMXRequest holds parsed HTMX request headers, stored in context
// by HTMXMiddleware. Use HTMXFromContext to retrieve it.
type HTMXRequest struct {
	IsHTMX           bool
	IsBoosted        bool
	IsHistoryRestore bool
	Target           string
	TriggerID        string
	TriggerName      string
	Prompt           string
	CurrentURL       string
}

// RenderPartial returns true when a partial HTML response should be rendered.
// True for HTMX requests that are not history restorations.
func (h *HTMXRequest) RenderPartial() bool {
	return h.IsHTMX && !h.IsHistoryRestore
}

// parseHTMXRequest extracts all HTMX headers from the request.
func parseHTMXRequest(r *http.Request) *HTMXRequest {
	return &HTMXRequest{
		IsHTMX:           r.Header.Get(headerRequest) == headerTrue,
		IsBoosted:        r.Header.Get(headerBoosted) == headerTrue,
		IsHistoryRestore: r.Header.Get(headerHistoryRestore) == headerTrue,
		Target:           r.Header.Get(headerTarget),
		TriggerID:        r.Header.Get(headerTriggerID),
		TriggerName:      r.Header.Get(headerTriggerName),
		Prompt:           r.Header.Get(headerPrompt),
		CurrentURL:       r.Header.Get(headerCurrentURL),
	}
}

// WithHTMX stores a parsed HTMXRequest in the context.
func WithHTMX(ctx context.Context, h *HTMXRequest) context.Context {
	return context.WithValue(ctx, htmxKey, h)
}

// HTMXFromContext retrieves the parsed HTMXRequest from context.
// Returns nil if HTMXMiddleware was not applied.
func HTMXFromContext(ctx context.Context) *HTMXRequest {
	h, _ := ctx.Value(htmxKey).(*HTMXRequest)
	return h
}

// IsHTMXRequest returns true if the request originates from HTMX.
// Checks the HX-Request header directly — works with or without HTMXMiddleware.
func IsHTMXRequest(r *http.Request) bool {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.IsHTMX
	}
	return r.Header.Get(headerRequest) == headerTrue
}

// IsBoosted returns true if the request was sent via hx-boost.
func IsBoosted(r *http.Request) bool {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.IsBoosted
	}
	return r.Header.Get(headerBoosted) == headerTrue
}

// IsHistoryRestore returns true if the request is a history restoration.
func IsHistoryRestore(r *http.Request) bool {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.IsHistoryRestore
	}
	return r.Header.Get(headerHistoryRestore) == headerTrue
}

// RenderPartial returns true when the request is from HTMX and is not
// a history restore. Use this to decide whether to render a partial
// HTML fragment or a full page.
func RenderPartial(r *http.Request) bool {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.RenderPartial()
	}
	return r.Header.Get(headerRequest) == headerTrue &&
		r.Header.Get(headerHistoryRestore) != headerTrue
}

// HTMXTarget returns the ID of the target element from the request.
func HTMXTarget(r *http.Request) string {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.Target
	}
	return r.Header.Get(headerTarget)
}

// HTMXTrigger returns the ID of the trigger element from the request.
func HTMXTrigger(r *http.Request) string {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.TriggerID
	}
	return r.Header.Get(headerTriggerID)
}

// HTMXTriggerName returns the name of the trigger element from the request.
func HTMXTriggerName(r *http.Request) string {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.TriggerName
	}
	return r.Header.Get(headerTriggerName)
}

// HTMXPrompt returns the user's response from hx-prompt.
func HTMXPrompt(r *http.Request) string {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.Prompt
	}
	return r.Header.Get(headerPrompt)
}

// HTMXCurrentURL returns the current URL from the request.
func HTMXCurrentURL(r *http.Request) string {
	if h := HTMXFromContext(r.Context()); h != nil {
		return h.CurrentURL
	}
	return r.Header.Get(headerCurrentURL)
}
