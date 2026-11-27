package cqrshtmx

import (
	_ "embed"
	"net/http"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
)

// trackingVersion is the version of the embedded client tracker. It must
// equal the VERSION constant inside tracking/tracker.js; a test pins the two.
const trackingVersion = "1.0.0"

//go:embed tracking/tracker.js
var trackerJS []byte

// TrackerScriptHandler serves the embedded client tracker (tracker.js), the
// browser side of [TrackHandler]. Mount it at any path, e.g.:
//
//	mux.Handle("/-/tracker.js", cqrshtmx.TrackerScriptHandler())
//
// The response is immutable for one year and revalidated by a content-derived
// ETag, so a rebuilt tracker is always refetched. The tracker collects
// nothing until a page includes the script tag, and stops for a visitor who
// opted out (Global Privacy Control, data-tracking="off", or
// window.cqrsTracker.disable()).
func TrackerScriptHandler() http.Handler {
	return hx.ServeJS(trackerJS, AssetETag("cqrs-tracker", trackerJS))
}

// TrackerScriptTag returns an HTML <script> tag that loads the tracker from
// the given path:
//
//	cqrshtmx.TrackerScriptTag("/-/tracker.js")
//	// => <script src="/-/tracker.js" defer></script>
//
// The path is a developer-controlled URL, never user input: a value that
// originates from untrusted input must be validated first, or it can inject
// attributes. A nonce-only Content-Security-Policy needs the nonce added to
// the tag by the page that renders it.
func TrackerScriptTag(path string) string {
	return `<script src="` + path + `" defer></script>`
}

// TrackingVersion returns the version of the embedded client tracker.
func TrackingVersion() string { return trackingVersion }
