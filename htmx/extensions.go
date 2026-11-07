package htmx

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/larsartmann/cqrs-htmx/v4/internal/hx"
)

// Embedded HTMX extension JavaScript files (minified).
//
//go:embed extensions/sse.min.js
var extSSEJS []byte

//go:embed extensions/idiomorph-ext.min.js
var extIdiomorphJS []byte

//go:embed extensions/cqrs-query.js
var extCQRSQueryJS []byte

// HTMX extension name constants for use with ExtensionHandler and
// ExtensionsHandler. These are the extensions that have direct server-side
// counterparts in cqrs-htmx (SSE building blocks) or are commonly paired
// with them (idiomorph for morph-based partial swaps).
const (
	// ExtSSE is the htmx Server-Sent Events extension (htmx-ext-sse).
	// Pairs with cqrs-htmx SSEStream, Broadcaster, JournalSSEStore.
	ExtSSE = "sse"

	// ExtIdiomorph provides the morph swap strategy via idiomorph.
	// Commonly used alongside SSE for efficient partial DOM updates.
	ExtIdiomorph = "idiomorph"

	// ExtCQRSQuery upgrades htmx 2 requests to the native HTTP QUERY method
	// (RFC 10008) with automatic downgrade to POST + X-Http-Method-Override
	// when an intermediary answers 405/501. cqrs-htmx's own extension — pairs
	// with MethodOverrideMiddleware on the server.
	ExtCQRSQuery = "cqrs-query"
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
	ExtSSE:       {js: extSSEJS, version: "2.2.4"},
	ExtIdiomorph: {js: extIdiomorphJS, version: "0.7.4"},
	ExtCQRSQuery: {js: extCQRSQueryJS, version: "1.0.0"},
}

// ExtensionHandler returns an http.Handler that serves the embedded
// HTMX extension JavaScript for the given extension name.
//
// The handler sets Content-Type, ETag, and Cache-Control headers, supports
// conditional GET (304 Not Modified), and accepts GET/HEAD only (405 otherwise).
//
// Known extension names: ExtSSE ("sse"), ExtIdiomorph ("idiomorph").
// Call ExtensionNames() to list all.
//
// Panics if name is not a known embedded extension — this is a programming
// error caught at startup, not a runtime condition.
//
//	mux.Handle("GET /ext/sse.js", cqrshtmx.ExtensionHandler(cqrshtmx.ExtSSE))
//	mux.Handle("GET /ext/idiomorph.js", cqrshtmx.ExtensionHandler(cqrshtmx.ExtIdiomorph))
func ExtensionHandler(name string) http.Handler {
	ext, ok := htmxExtensions[name]
	if !ok {
		msg := fmt.Sprintf(
			"cqrshtmx: unknown htmx extension %q (available: %s)",
			name,
			strings.Join(htmxExtensionNames(), ", "),
		)
		//cqrs-lint:ignore(C009) startup-time programmer error: unknown extension name
		panic(msg)
	}

	return hx.ServeJS(ext.js, fmt.Sprintf(`"htmx-ext-%s-%s"`, name, ext.version))
}

// ExtensionsHandler returns an http.Handler that serves a concatenated
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
//	    cqrshtmx.ExtensionsHandler(cqrshtmx.ExtSSE, cqrshtmx.ExtIdiomorph))
func ExtensionsHandler(names ...string) http.Handler {
	if len(names) == 0 {
		//cqrs-lint:ignore(C009) startup-time programmer error: empty variadic call
		panic(
			"cqrshtmx: ExtensionsHandler requires at least one extension name",
		)
	}

	var buf bytes.Buffer

	etagParts := make([]string, 0, len(names))
	for _, name := range names {
		ext, ok := htmxExtensions[name]
		if !ok {
			msg := fmt.Sprintf(
				"cqrshtmx: unknown htmx extension %q (available: %s)",
				name,
				strings.Join(htmxExtensionNames(), ", "),
			)
			//cqrs-lint:ignore(C009) startup-time programmer error: unknown extension name
			panic(msg)
		}

		fmt.Fprintf(&buf, "/* htmx-ext-%s %s */\n", name, ext.version)
		buf.Write(ext.js)
		buf.WriteByte('\n')

		etagParts = append(etagParts, name+"-"+ext.version)
	}

	return hx.ServeJS(buf.Bytes(), fmt.Sprintf(`"htmx-ext-bundle-%s"`, strings.Join(etagParts, ",")))
}

// ExtensionVersion returns the version string of the embedded extension
// with the given name, or empty string if the name is unknown.
//
//	cqrshtmx.ExtensionVersion(cqrshtmx.ExtSSE) // "2.2.4"
func ExtensionVersion(name string) string {
	ext, ok := htmxExtensions[name]
	if !ok {
		return ""
	}

	return ext.version
}

// htmxExtensionNames returns the names of all embedded HTMX extensions,
// sorted alphabetically. Shared by the exported ExtensionNames and
// panic messages in ExtensionHandler/ExtensionsHandler.
func htmxExtensionNames() []string {
	names := make([]string, 0, len(htmxExtensions))
	for name := range htmxExtensions {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// ExtensionNames returns the names of all available embedded HTMX
// extensions, sorted alphabetically.
func ExtensionNames() []string {
	return htmxExtensionNames()
}

// htmxExtensionCDNURLs maps extension names to their unpkg CDN paths.
//
//nolint:gochecknoglobals // immutable lookup table.
var htmxExtensionCDNURLs = map[string]string{
	ExtSSE:       "https://unpkg.com/htmx-ext-sse@%s/dist/sse.min.js",
	ExtIdiomorph: "https://unpkg.com/idiomorph@%s/dist/idiomorph-ext.min.js",
}

// ExtensionCDNScriptTag returns an HTML <script> tag that loads the given
// HTMX extension from the unpkg CDN, using the embedded version. For consumers
// who prefer CDN over self-hosting.
//
//	cqrshtmx.ExtensionCDNScriptTag(cqrshtmx.ExtSSE)
//	// => <script src="https://unpkg.com/htmx-ext-sse@2.2.4/dist/sse.min.js"></script>
func ExtensionCDNScriptTag(name string) string {
	version := ExtensionVersion(name)
	if version == "" {
		return ""
	}

	tmpl, ok := htmxExtensionCDNURLs[name]
	if !ok {
		return ""
	}

	return `<script src="` + fmt.Sprintf(tmpl, version) + `"></script>`
}
