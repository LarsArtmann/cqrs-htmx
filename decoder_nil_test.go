package cqrshtmx_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// TestCommandDecodeReturningNil_Returns500 verifies that a buggy command
// decoder which returns (nil, nil) is classified as Corruption (500) — a
// server-side wiring bug — rather than Transient (503, which clients would
// retry) or Rejection (400, which would blame the caller).
//
// This exercises the errDecoderReturnedNil sentinel added in v4.2.x; before
// that fix the path shared errDecoderMissing, an Infrastructure-family error
// that maps to 503 and would have triggered client retries of a request that
// can never succeed.
func TestCommandDecodeReturningNil_Returns500(t *testing.T) {
	t.Parallel()

	disp := command.NewDispatcher()
	_ = disp.Register("NilCmd", func(_ context.Context, _ command.Command) error {
		return nil
	})

	app := cqrshtmx.MustNew(cqrshtmx.Config{Commands: disp})

	// A decoder that returns (nil, nil): exactly the wiring bug the sentinel
	// targets. The mapper ignores its decoded body and returns no command.
	handler := app.Command("NilCmd", cqrshtmx.DecodeJSON(
		func(_ struct{}) (command.Command, error) { return nil, nil },
	))

	r := httptest.NewRequest(http.MethodPost, "/nil", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (Corruption/500, not 503 Transient); body=%q",
			w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

// TestQueryDecodeReturningNil_Returns500 mirrors the command path for queries:
// a query decoder returning (nil, nil) must also surface as 500 Corruption so
// clients do not retry a request that can never succeed.
func TestQueryDecodeReturningNil_Returns500(t *testing.T) {
	t.Parallel()

	app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: queryDispatcherThatNeverRuns(t)})

	handler := app.Query("NilQ", cqrshtmx.DecodeJSONQuery(
		func(_ struct{}) (query.Query, error) { return nil, nil },
	))

	r := httptest.NewRequest(http.MethodGet, "/nil-q", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d (Corruption/500, not 503 Transient); body=%q",
			w.Code, http.StatusInternalServerError, w.Body.String())
	}
}

// queryDispatcherThatNeverRuns returns a query dispatcher with a handler
// registered purely so the App is non-nil; the handler must never run because
// the test asserts the decoder-nil branch fires first. If the handler runs,
// the test fails by explicit Fatalf.
func queryDispatcherThatNeverRuns(t *testing.T) *query.Dispatcher {
	t.Helper()

	disp := query.NewDispatcher()
	_ = disp.Register("NilQ", func(_ context.Context, _ query.Query) (any, error) {
		t.Fatal("query handler ran; decoder-nil branch should have fired first")

		return nil, nil
	})

	return disp
}
