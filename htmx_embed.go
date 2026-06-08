package cqrshtmx

import _ "embed"

//go:embed htmx.min.js
var htmxJS []byte

// HTMXVersion returns the embedded HTMX library version string.
func HTMXVersion() string { return "2.0.9" }
