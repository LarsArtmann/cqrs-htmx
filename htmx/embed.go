package htmx

import _ "embed"

// htmxVersion is the version of the embedded HTMX library.
const htmxVersion = "2.0.11"

//go:embed htmx.min.js
var htmxJS []byte

// Version returns the embedded HTMX library version string.
func Version() string { return htmxVersion }
