package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casbin/casbin/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// newHTMXRequest creates an httptest.Request with HX-Request set to true.
func newHTMXRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("HX-Request", "true")
	return r
}

// serveHandler serves a request through the given handler and returns the response.
func serveHandler(handler http.Handler, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

// createTestApp creates a CQRS app with a command dispatcher and optional enforcer.
func createTestApp(disp *command.Dispatcher, enforcer *casbin.Enforcer) *cqrshtmx.App {
	cfg := cqrshtmx.Config{Commands: disp}
	if enforcer != nil {
		cfg.Enforcer = enforcer
	}
	app, _ := cqrshtmx.New(cfg)
	return app
}

// createTestAppWithExtractor creates a CQRS app with a user-ID extractor.
func createTestAppWithExtractor(
	disp *command.Dispatcher,
	extractor func(*http.Request) string,
) *cqrshtmx.App {
	app, _ := cqrshtmx.New(cqrshtmx.Config{
		Commands:        disp,
		UserIDExtractor: extractor,
	})
	return app
}

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

// newTestCommandDispatcher creates a command dispatcher that registers name with handler.
func newTestCommandDispatcher(
	name command.Type,
	handler func(context.Context, command.Command) error,
) *command.Dispatcher {
	disp := command.NewDispatcher()
	_ = disp.Register(name, handler)
	return disp
}

// newTestQueryDispatcher creates a query dispatcher that registers name with handler.
func newTestQueryDispatcher(
	name query.Type,
	handler func(context.Context, query.Query) (any, error),
) *query.Dispatcher {
	disp := query.NewDispatcher()
	_ = disp.Register(name, handler)
	return disp
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

