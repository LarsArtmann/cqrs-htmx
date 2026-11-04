package cqrshtmx

import "github.com/larsartmann/cqrs-htmx/v4/internal/hx"

// Shared constants used across the error, response, and CSRF surfaces.
// Extracted to break the import cycle that was caused by these constants
// living in response.go while being needed by the error paths.
// See docs/modularization/2026-07-01_SOLLBRUCHSTELLEN.html for analysis.
//
// The values live in internal/hx so the extracted subpackages (htmx, httperr,
// eventcatalog) can use the identical bytes without importing this package.

// Content type constants for consistent HTTP response headers.
const (
	ContentTypePlain   = hx.ContentTypePlain
	ContentTypeHTML    = hx.ContentTypeHTML
	ContentTypeJSON    = hx.ContentTypeJSON
	ContentTypeProblem = hx.ContentTypeProblem
)

// JSON map key constants for consistent error/status response shapes.
// Exported so consumers can build matching response types without typos.
// Note: usermgmt is a separate Go module and cannot import these — it
// declares its own local statusKey/errorKey. The wire formats match.
const (
	JSONKeyError     = hx.JSONKeyError
	JSONKeyStatus    = hx.JSONKeyStatus
	JSONKeyCode      = hx.JSONKeyCode
	JSONKeyRequestID = hx.JSONKeyRequestID
)
