package htmx

import _ "embed"

// HTMX4Version is the version of the embedded HTMX 4 library served by
// [ScriptHandler4]. HTMX 4 ships native `hx-query` support (RFC 10008); it
// rides ALONGSIDE the embedded 2.0.11 default, which stays the default until
// the v5 window (ROADMAP question 35) — consumers opt in per route.
const HTMX4Version = "4.0.1"

//go:embed htmx4.min.js
var htmx4JS []byte
