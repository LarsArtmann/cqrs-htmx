package cqrshtmx

import "net/http"

const (
	headerRequest           = "HX-Request"
	headerBoosted           = "HX-Boosted"
	headerCurrentURL        = "HX-Current-URL"
	headerHistoryRestore    = "HX-History-Restore-Request"
	headerPrompt            = "HX-Prompt"
	headerTarget            = "HX-Target"
	headerTriggerID         = "HX-Trigger"
	headerTriggerName       = "HX-Trigger-Name"

	headerLocation          = "HX-Location"
	headerPushURL           = "HX-Push-Url"
	headerRedirect          = "HX-Redirect"
	headerRefresh           = "HX-Refresh"
	headerReplaceURL        = "HX-Replace-Url"
	headerReswap            = "HX-Reswap"
	headerRetarget          = "HX-Retarget"
	headerReselect          = "HX-Reselect"
	headerTrigger           = "HX-Trigger"
	headerTriggerAfterSettle = "HX-Trigger-After-Settle"
	headerTriggerAfterSwap   = "HX-Trigger-After-Swap"
)

// SwapStrategy defines how HTMX swaps content into the DOM.
type SwapStrategy string

const (
	SwapInnerHTML     SwapStrategy = "innerHTML"
	SwapOuterHTML     SwapStrategy = "outerHTML"
	SwapBeforeBegin   SwapStrategy = "beforebegin"
	SwapAfterBegin    SwapStrategy = "afterbegin"
	SwapBeforeEnd     SwapStrategy = "beforeend"
	SwapAfterEnd      SwapStrategy = "afterend"
	SwapDelete        SwapStrategy = "delete"
	SwapNone          SwapStrategy = "none"
)

// IsHTMXRequest returns true if the request originates from HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return r.Header.Get(headerRequest) == "true"
}

// IsBoosted returns true if the request was sent via hx-boost.
func IsBoosted(r *http.Request) bool {
	return r.Header.Get(headerBoosted) == "true"
}

// IsHistoryRestore returns true if the request is a history restoration.
func IsHistoryRestore(r *http.Request) bool {
	return r.Header.Get(headerHistoryRestore) == "true"
}

// HTMXTarget returns the ID of the target element from the request.
func HTMXTarget(r *http.Request) string {
	return r.Header.Get(headerTarget)
}

// HTMXTrigger returns the ID of the trigger element from the request.
func HTMXTrigger(r *http.Request) string {
	return r.Header.Get(headerTriggerID)
}

// HTMXPrompt returns the user's response from hx-prompt.
func HTMXPrompt(r *http.Request) string {
	return r.Header.Get(headerPrompt)
}

// HTMXCurrentURL returns the current URL from the request.
func HTMXCurrentURL(r *http.Request) string {
	return r.Header.Get(headerCurrentURL)
}
