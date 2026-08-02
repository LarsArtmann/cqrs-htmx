package datastar

import (
	"net/http"

	sdk "github.com/starfederation/datastar-go/datastar"
)

// ReadSignals decodes Datastar client signals from the HTTP request into the
// target struct. For GET/DELETE requests, signals are read from the "datastar"
// query parameter. For all other methods, signals are read from the JSON body.
//
//	var s struct{ Title string `json:"title"` }
//	if err := ds.ReadSignals(r, &s); err != nil {
//	    ds.ErrorResponse(w, r, err)
//	    return
//	}
//
// This is a re-export of datastar.ReadSignals.
func ReadSignals(r *http.Request, signals any) error {
	return sdk.ReadSignals(r, signals)
}
