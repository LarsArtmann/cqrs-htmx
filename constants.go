package cqrshtmx

// Shared constants used across errors.go, response.go, and csrf_config.go.
// Extracted to break the import cycle that was caused by these constants
// living in response.go while being needed by errors.go and csrf_config.go.
// See docs/modularization/2026-07-01_SOLLBRUCHSTELLEN.html for analysis.

// Content type constants for consistent HTTP response headers.
const (
	ContentTypePlain   = "text/plain; charset=utf-8"
	ContentTypeHTML    = "text/html; charset=utf-8"
	ContentTypeJSON    = "application/json; charset=utf-8"
	ContentTypeProblem = "application/problem+json; charset=utf-8"
)

// JSON map key constants for consistent error/status response shapes.
// Exported so consumers can build matching response types without typos.
// Note: usermgmt is a separate Go module and cannot import these — it
// declares its own local statusKey/errorKey. The wire formats match.
const (
	JSONKeyError  = "error"
	JSONKeyStatus = "status"
	JSONKeyCode   = "code"
)
