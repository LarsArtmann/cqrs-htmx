package cqrshtmx

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// Embedded HTMX extension JavaScript files (minified).
//
//go:embed extensions/sse.min.js
var extSSEJS []byte

//go:embed extensions/ws.min.js
var extWSJS []byte

//go:embed extensions/idiomorph-ext.min.js
var extIdiomorphJS []byte

// HTMX extension name constants for use with HTMXExtensionHandler and
// HTMXExtensionsHandler. These are the extensions that have direct server-side
// counterparts in cqrs-htmx (SSE + WS building blocks) or are commonly paired
// with them (idiomorph for morph-based partial swaps).
const (
	// HTMXExtSSE is the htmx Server-Sent Events extension (htmx-ext-sse).
	// Pairs with cqrs-htmx SSEStream, Broadcaster, JournalSSEStore.
	HTMXExtSSE = "sse"

	// HTMXExtWS is the htmx WebSocket extension (htmx-ext-ws).
	// Pairs with cqrs-htmx WSMessage, WSBroadcaster, DispatchWSCommand.
	HTMXExtWS = "ws"

	// HTMXExtIdiomorph provides the morph swap strategy via idiomorph.
	// Commonly used alongside SSE for efficient partial DOM updates.
	HTMXExtIdiomorph = "idiomorph"
)

// htmxExtension holds an embedded HTMX extension's JS payload and version.
type htmxExtension struct {
	js      []byte
	version string
}

// htmxExtensions is the registry of all embedded HTMX extensions.
//
//nolint:gochecknoglobals // embedded assets + their metadata; immutable after init.
var htmxExtensions = map[string]htmxExtension{
	HTMXExtSSE:       {js: extSSEJS, version: "2.2.4"},
	HTMXExtWS:        {js: extWSJS, version: "2.0.4"},
	HTMXExtIdiomorph: {js: extIdiomorphJS, version: "0.7.4"},
}

// HTMXExtensionHandler returns an http.Handler that serves the embedded
// HTMX extension JavaScript for the given extension name.
//
// The handler sets Content-Type, ETag, and Cache-Control headers, supports
// conditional GET (304 Not Modified), and accepts GET/HEAD only (405 otherwise).
//
// Known extension names: HTMXExtSSE ("sse"), HTMXExtWS ("ws"),
// HTMXExtIdiomorph ("idiomorph"). Call HTMXExtensionNames() to list all.
//
// Panics if name is not a known embedded extension — this is a programming
// error caught at startup, not a runtime condition.
//
//	mux.Handle("GET /ext/sse.js", cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtSSE))
//	mux.Handle("GET /ext/ws.js", cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtWS))
//	mux.Handle("GET /ext/idiomorph.js", cqrshtmx.HTMXExtensionHandler(cqrshtmx.HTMXExtIdiomorph))
func HTMXExtensionHandler(name string) http.Handler {
	ext, ok := htmxExtensions[name]
	if !ok {
		msg := fmt.Sprintf("cqrshtmx: unknown htmx extension %q (available: %s)", name, strings.Join(htmxExtensionNames(), ", "))
		panic(msg) //cqrs-lint:ignore(C009) startup-time programmer error: unknown extension name
	}

	return serveJS(ext.js, fmt.Sprintf(`"htmx-ext-%s-%s"`, name, ext.version))
}

// HTMXExtensionsHandler returns an http.Handler that serves a concatenated
// bundle of the named HTMX extensions in a single HTTP response — one request
// instead of N. Extensions are concatenated in the given order, each prefixed
// with a version comment for debuggability.
//
// The ETag is a composite of all extension names and versions, so conditional
// GET works correctly across the whole bundle.
//
// Panics if any name is not a known embedded extension, or if no names are given.
//
//	mux.Handle("GET /ext/bundle.js",
//	    cqrshtmx.HTMXExtensionsHandler(cqrshtmx.HTMXExtSSE, cqrshtmx.HTMXExtWS, cqrshtmx.HTMXExtIdiomorph))
func HTMXExtensionsHandler(names ...string) http.Handler {
	if len(names) == 0 {
		panic("cqrshtmx: HTMXExtensionsHandler requires at least one extension name") //cqrs-lint:ignore(C009) startup-time programmer error: empty variadic call
	}

	var buf bytes.Buffer

	etagParts := make([]string, 0, len(names))
	for _, name := range names {
		ext, ok := htmxExtensions[name]
		if !ok {
			msg := fmt.Sprintf("cqrshtmx: unknown htmx extension %q (available: %s)", name, strings.Join(htmxExtensionNames(), ", "))
			panic(msg) //cqrs-lint:ignore(C009) startup-time programmer error: unknown extension name
		}

		fmt.Fprintf(&buf, "/* htmx-ext-%s %s */\n", name, ext.version)
		buf.Write(ext.js)
		buf.WriteByte('\n')

		etagParts = append(etagParts, name+"-"+ext.version)
	}

	return serveJS(buf.Bytes(), fmt.Sprintf(`"htmx-ext-bundle-%s"`, strings.Join(etagParts, ",")))
}

// HTMXExtensionVersion returns the version string of the embedded extension
// with the given name, or empty string if the name is unknown.
//
//	cqrshtmx.HTMXExtensionVersion(cqrshtmx.HTMXExtSSE) // "2.2.4"
func HTMXExtensionVersion(name string) string {
	ext, ok := htmxExtensions[name]
	if !ok {
		return ""
	}

	return ext.version
}

// htmxExtensionNames returns the names of all embedded HTMX extensions,
// sorted alphabetically. Shared by the exported HTMXExtensionNames and
// panic messages in HTMXExtensionHandler/HTMXExtensionsHandler.
func htmxExtensionNames() []string {
	names := make([]string, 0, len(htmxExtensions))
	for name := range htmxExtensions {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// HTMXExtensionNames returns the names of all available embedded HTMX
// extensions, sorted alphabetically.
func HTMXExtensionNames() []string {
	return htmxExtensionNames()
}

// htmxExtensionCDNURLs maps extension names to their unpkg CDN paths.
//
//nolint:gochecknoglobals // immutable lookup table.
var htmxExtensionCDNURLs = map[string]string{
	HTMXExtSSE:       "https://unpkg.com/htmx-ext-sse@%s/dist/sse.min.js",
	HTMXExtWS:        "https://unpkg.com/htmx-ext-ws@%s/dist/ws.min.js",
	HTMXExtIdiomorph: "https://unpkg.com/idiomorph@%s/dist/idiomorph-ext.min.js",
}

// HTMXExtensionCDNScriptTag returns an HTML <script> tag that loads the given
// HTMX extension from the unpkg CDN, using the embedded version. For consumers
// who prefer CDN over self-hosting.
//
//	cqrshtmx.HTMXExtensionCDNScriptTag(cqrshtmx.HTMXExtSSE)
//	// => <script src="https://unpkg.com/htmx-ext-sse@2.2.4/dist/sse.min.js"></script>
func HTMXExtensionCDNScriptTag(name string) string {
	version := HTMXExtensionVersion(name)
	if version == "" {
		return ""
	}

	tmpl, ok := htmxExtensionCDNURLs[name]
	if !ok {
		return ""
	}

	return `<script src="` + fmt.Sprintf(tmpl, version) + `"></script>`
}
