package transport

import (
	"net/http"
	"time"
)

// ClearWriteDeadline removes any write deadline the server installed on the
// connection, so a long-lived Server-Sent Events stream is not severed by a
// route-wide [http.Server.WriteTimeout].
//
// A write timeout is a whole-connection limit: a server configured with, say,
// WriteTimeout 30s closes EVERY response 30 seconds after the request headers
// were read, including an SSE stream that is deliberately still open. There is
// no per-route override in [http.Server], so a streaming handler must clear the
// deadline on its own connection with [http.NewResponseController].
//
// The call is best-effort: writers that do not implement the deadline interface
// (notably [httptest.ResponseRecorder]) return [http.ErrNotSupported], which is
// deliberately ignored — a writer that cannot carry a deadline also cannot have
// one that needs clearing.
//
// Call it at the very start of the streaming handler, before any byte is
// written. [ServeDomainEvents] does this for you; use it directly only when
// writing your own SSE handler. It relaxes nothing outside the current request,
// so non-streaming routes keep the server-wide deadline.
func ClearWriteDeadline(w http.ResponseWriter) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
}
