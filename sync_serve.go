package cqrshtmx

import (
	"fmt"
	"net/http"
)

// SyncWorkerHandler returns an http.Handler that serves the embedded offline
// sync SharedWorker JavaScript. The worker queues failed command envelopes to
// IndexedDB when the network is down and tells tabs to retry when connectivity
// returns. Mount it at any path, e.g.:
//
//	mux.Handle("/sync-worker.js", cqrshtmx.SyncWorkerHandler())
//
// The worker URL must be communicated to the sync client (sync-client.js) so
// it can instantiate new SharedWorker(workerURL). The sync client derives the
// worker URL from its own <script src> path by default.
func SyncWorkerHandler() http.Handler {
	return serveJS(syncWorkerJS, fmt.Sprintf(`"cqrshtmx-sync-worker-%s"`, syncVersion))
}

// SyncClientHandler returns an http.Handler that serves the embedded offline
// sync tab-side JavaScript. The client listens to HTMX events, manages the
// sync indicator, connects SSE for ACK confirmations, and coordinates with the
// SharedWorker for offline command queueing. Mount it at any path, e.g.:
//
//	mux.Handle("/sync-client.js", cqrshtmx.SyncClientHandler())
//
// The client auto-initializes on DOMContentLoaded if a [data-sse-url]
// attribute is present on the <body> element. It derives the SharedWorker URL
// from its own <script src> path (replacing "sync-client.js" with
// "sync-worker.js").
func SyncClientHandler() http.Handler {
	return serveJS(syncClientJS, fmt.Sprintf(`"cqrshtmx-sync-client-%s"`, syncVersion))
}

// SyncWorkerScriptTag returns an HTML <script>-loadable tag for the sync
// SharedWorker, suitable for embedding in a template or templ component.
// The path should point to where SyncWorkerHandler is mounted.
//
//	cqrshtmx.SyncWorkerScriptTag("/sync-worker.js")
//
// Note: the SharedWorker is loaded programmatically by sync-client.js, not
// via a <script> tag. This helper exists for consumers who need to reference
// the worker URL in a CSP meta tag or for debugging.
func SyncWorkerScriptTag(path string) string {
	return `<script src="` + path + `"></script>`
}

// SyncClientScriptTag returns an HTML <script> tag that loads the sync client
// from the given path. Include it after the HTMX script tag.
//
//	cqrshtmx.SyncClientScriptTag("/sync-client.js")
//	// => <script src="/sync-client.js"></script>
func SyncClientScriptTag(path string) string {
	return `<script src="` + path + `"></script>`
}

// SyncVersion returns the version string of the embedded offline sync assets.
func SyncVersion() string { return syncVersion }
