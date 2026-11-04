package hx

// Shared HTTP wire-format constants.
//
// These are re-exported by the root cqrshtmx package, which is where consumers
// should read them: the constants are public API there and unchanged. They live
// here so subpackages can use the exact same values without importing the root
// package (which would be an import cycle).

// Content type constants for consistent HTTP response headers.
const (
	ContentTypePlain   = "text/plain; charset=utf-8"
	ContentTypeHTML    = "text/html; charset=utf-8"
	ContentTypeJSON    = "application/json; charset=utf-8"
	ContentTypeProblem = "application/problem+json; charset=utf-8"
)

// JSON map key constants for consistent error/status response shapes.
const (
	JSONKeyError     = "error"
	JSONKeyStatus    = "status"
	JSONKeyCode      = "code"
	JSONKeyRequestID = "request_id"
)
