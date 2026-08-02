package datastar

import (
	"net/http"
)

// ErrorResponse sends an error to the client as a Datastar signal patch. The
// error message is placed in the "notification" signal with level "error",
// matching the pattern used by the datastar-demo and providing a consistent
// error UX for Datastar frontends.
//
//	if err := ds.ReadSignals(r, &s); err != nil {
//	    ds.ErrorResponse(w, r, err)
//	    return
//	}
func ErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	NewResponse(w, r).PatchSignals(map[string]any{
		"notification": map[string]string{
			"level":   "error",
			"message": err.Error(),
		},
	})
}

// NotificationResponse sends a notification signal patch to the client.
// Level should be "success", "warning", "error", or "info".
//
//	ds.NotificationResponse(w, r, "success", "Todo created!")
func NotificationResponse(w http.ResponseWriter, r *http.Request, level, message string) {
	NewResponse(w, r).PatchSignals(map[string]any{
		"notification": map[string]string{
			"level":   level,
			"message": message,
		},
	})
}
