package cqrshtmx

import _ "embed"

// htmxVersion is the version of the embedded HTMX library.
const htmxVersion = "2.0.9"

//go:embed htmx.min.js
var htmxJS []byte

// HTMXVersion returns the embedded HTMX library version string.
func HTMXVersion() string { return htmxVersion }
