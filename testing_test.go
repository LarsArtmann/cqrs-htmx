package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// decodeCreateUserJSON returns a HandlerOption that decodes a request to
// a testCreateUserCmd with a fresh AggregateID (request body is ignored).
func decodeCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
	})
}

// decodeCreateUserJSONWithBody returns a HandlerOption that decodes a request to
// a testCreateUserCmd, populating email and name from the request body.
func decodeCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID(), email: req.Email, name: req.Name}, nil
	})
}

// decodeBDDCreateUserJSON returns a HandlerOption that decodes a request to
// a bddCreateUserCmd with a fresh AggregateID (request body is ignored).
func decodeBDDCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{aggID: id.NewAggregateID()}, nil
	})
}

// decodeGetUserJSONQuery returns a HandlerOption that decodes a JSON query request.
func decodeGetUserJSONQuery() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
		return &testGetUserQuery{}, nil
	})
}

// decodeCreateUserJSONWithAggID returns a HandlerOption that uses the provided AggregateID.
func decodeCreateUserJSONWithAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID}, nil
	})
}

// decodeCreateUserJSONWithBodyAndAggID returns a HandlerOption that decodes body and uses the provided AggregateID.
func decodeCreateUserJSONWithBodyAndAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
	})
}

// noOpCommandHandler returns a handler that always succeeds.
func noOpCommandHandler(_ context.Context, _ command.Command) error { return nil }

// encodeJSONResult writes result as JSON to the response writer.
func encodeJSONResult(w http.ResponseWriter, _ *http.Request, result any) error {
	return json.NewEncoder(w).Encode(result)
}

// rejectionHandler returns a handler that returns a CQRS rejection.
func rejectionHandler(code, message string) func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return event.NewRejection(code, message)
	}
}

// middlewareCaptureHandler returns an http.Handler that sets called to true.
func middlewareCaptureHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		*called = true
	})
}

// okHandler returns an http.Handler that writes 200 OK.
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// createdHandler returns an http.Handler that writes 201 Created.
func createdHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
}

// staticExtractor returns a UserIDExtractor that always returns the given user ID.
func staticExtractor(uid cqrshtmx.UserID) cqrshtmx.UserIDExtractor {
	return func(_ *http.Request) (cqrshtmx.UserID, error) { return uid, nil }
}
