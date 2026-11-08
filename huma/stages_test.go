package huma_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	humav2 "github.com/danielgtaylor/huma/v2"
	cqrsadapter "github.com/larsartmann/cqrs-htmx/huma/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestNew_ReportsConfiguredDispatchers(t *testing.T) {
	t.Parallel()

	both, err := cqrsadapter.New(cqrsadapter.Config{
		Commands: command.NewDispatcher(),
		Queries:  query.NewDispatcher(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if !both.HasCommands() || !both.HasQueries() {
		t.Fatalf("HasCommands=%v HasQueries=%v, want both true", both.HasCommands(), both.HasQueries())
	}

	queriesOnly, err := cqrsadapter.New(cqrsadapter.Config{Queries: query.NewDispatcher()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if queriesOnly.HasCommands() {
		t.Error("HasCommands = true for a query-only adapter")
	}

	if !queriesOnly.HasQueries() {
		t.Error("HasQueries = false for a query-only adapter")
	}
}

func TestAdapterStages_QueryEnrichmentAndResult(t *testing.T) {
	t.Parallel()

	qryDisp := query.NewDispatcher()

	var got *sumQuery

	if err := query.RegisterTyped(qryDisp, "Sum", func(_ context.Context, q *sumQuery) (int, error) {
		got = q

		return q.A + q.B, nil
	}); err != nil {
		t.Fatalf("register query: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Queries: qryDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	userID := cqrshtmx.GenerateUserID()
	ctx := cqrshtmx.WithUserID(context.Background(), userID)

	result, err := pipe.DispatchQuery(ctx, newSumQuery(t, 2, 3))
	if err != nil {
		t.Fatalf("DispatchQuery: %v", err)
	}

	if total, ok := result.(int); !ok || total != 5 {
		t.Fatalf("result = %v, want 5", result)
	}

	if got == nil || got.Metadata().UserID != userID {
		t.Fatalf("query was not dispatched with the request identity: %+v", got)
	}
}

func TestAdapterStages_MissingQueryDispatcher(t *testing.T) {
	t.Parallel()

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: command.NewDispatcher()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := pipe.DispatchQuery(context.Background(), newSumQuery(t, 1, 1)); err == nil {
		t.Fatal("DispatchQuery with no dispatcher: want an error")
	} else if status := httperr.MapError(err); status != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", status)
	}

	// The command side of the same adapter has a dispatcher, so it reaches the
	// dispatcher and fails on the unregistered type instead of on wiring.
	if err := pipe.DispatchCommand(context.Background(), newGreetingCommand(t, "ada")); err == nil {
		t.Fatal("DispatchCommand with an unregistered type: want an error")
	}
}

func TestRegisterQuery_StageFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		build      func(ctx context.Context, in *struct{}) (query.Query, error)
		options    []cqrsadapter.Option
		wantStatus int
		wantCode   string
	}{
		{
			name: "build failure",
			build: func(_ context.Context, _ *struct{}) (query.Query, error) {
				return nil, errorfamily.NewRejection("query.invalid", "invalid query")
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "query.invalid",
		},
		{
			name: "build returns nil",
			build: func(_ context.Context, _ *struct{}) (query.Query, error) {
				return nil, nil
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   httperr.ErrorCode(httperr.ErrDecoderReturnedNil),
		},
		{
			name: "validation failure",
			build: func(_ context.Context, _ *struct{}) (query.Query, error) {
				return newSumQuery(t, 0, 0), nil
			},
			options: []cqrsadapter.Option{
				cqrsadapter.ValidateQuery(func(query.Query) error { return exampleError{} }),
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "cqrshtmx.validate.failed",
		},
		{
			name: "guard failure",
			build: func(_ context.Context, _ *struct{}) (query.Query, error) {
				return newSumQuery(t, 1, 1), nil
			},
			options: []cqrsadapter.Option{
				cqrsadapter.Guard(func(context.Context, any) error { return httperr.ErrForbidden }),
			},
			wantStatus: http.StatusForbidden,
			wantCode:   httperr.CodeForbidden,
		},
		{
			name: "dispatch failure",
			build: func(_ context.Context, _ *struct{}) (query.Query, error) {
				// Sum is registered by the test below; Unknown is not.
				return newSumQuery(t, 1, 1), nil
			},
			options: []cqrsadapter.Option{
				cqrsadapter.AuthorizeIn("acme", "sums", "read"),
				cqrsadapter.WithTimeout(time.Nanosecond),
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "context.deadline_exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestAPI(t)
			qryDisp := query.NewDispatcher()

			if err := query.RegisterTyped(qryDisp, "Sum", func(ctx context.Context, q *sumQuery) (int, error) {
				if err := ctx.Err(); err != nil {
					return 0, errorfamily.NewTransient("context.deadline_exceeded", err.Error()).WithCause(err)
				}

				return q.A + q.B, nil
			}); err != nil {
				t.Fatalf("register query: %v", err)
			}

			pipe, err := cqrsadapter.New(cqrsadapter.Config{
				Queries: qryDisp,
				Checker: &stubChecker{allowed: true},
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			cqrsadapter.RegisterQuery(env.api, pipe,
				humav2.Operation{OperationID: "sum", Method: "GET", Path: "/sum"},
				tt.build,
				func(_ context.Context, _ *struct{}, result any) (*struct{}, error) {
					if _, ok := result.(int); !ok {
						t.Errorf("respond received %T, want int", result)
					}

					return &struct{}{}, nil
				},
				tt.options...,
			)

			ctx := cqrshtmx.WithUserID(context.Background(), cqrshtmx.GenerateUserID())
			rec := env.do(ctx, http.MethodGet, "/sum", "")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			problem := decodeProblem(t, rec)

			if problem.Code != tt.wantCode {
				t.Errorf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}
		})
	}
}

func TestRegisterQuery_RespondErrorIsMapped(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	qryDisp := query.NewDispatcher()

	if err := query.RegisterTyped(qryDisp, "Sum", func(_ context.Context, q *sumQuery) (int, error) {
		return q.A + q.B, nil
	}); err != nil {
		t.Fatalf("register query: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Queries: qryDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterQuery(env.api, pipe,
		humav2.Operation{OperationID: "sum", Method: "GET", Path: "/sum"},
		func(_ context.Context, _ *struct{}) (query.Query, error) {
			return newSumQuery(t, 1, 1), nil
		},
		func(_ context.Context, _ *struct{}, _ any) (*struct{}, error) {
			return nil, errorfamily.NewConflict("result.moved", "the result moved")
		},
	)

	rec := env.do(context.Background(), http.MethodGet, "/sum", "")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestAuthorizeIn_AsksTheTenantScopedQuestion(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	checker := &stubChecker{allowed: true}
	cmdDisp := command.NewDispatcher()

	if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, _ *greetingCommand) error {
		return nil
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp, Checker: checker})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(env.api, pipe,
		humav2.Operation{OperationID: "greet", Method: "POST", Path: "/greet", DefaultStatus: http.StatusNoContent},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return newGreetingCommand(t, "ada"), nil
		},
		cqrsadapter.AuthorizeIn("acme", "visits", "create"),
	)

	ctx := cqrshtmx.WithUserID(context.Background(), cqrshtmx.GenerateUserID())

	if rec := env.do(ctx, http.MethodPost, "/greet", "{}"); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body.String())
	}

	if checker.domain != "acme" {
		t.Errorf("domain = %q, want acme", checker.domain)
	}

	if checker.resource != "visits" || checker.actionName != "create" {
		t.Errorf("question = %q/%q, want visits/create", checker.resource, checker.actionName)
	}
}

func TestWithTimeout_BoundsTheDispatch(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	if err := command.RegisterTyped(cmdDisp, "Greet", func(ctx context.Context, _ *greetingCommand) error {
		<-ctx.Done()

		return errorfamily.NewTransient("context.deadline_exceeded", ctx.Err().Error()).WithCause(ctx.Err())
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp, Timeout: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(env.api, pipe,
		humav2.Operation{OperationID: "greet", Method: "POST", Path: "/greet", DefaultStatus: http.StatusNoContent},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return newGreetingCommand(t, "ada"), nil
		},
		cqrsadapter.WithTimeout(5*time.Millisecond),
	)

	rec := env.do(context.Background(), http.MethodPost, "/greet", "{}")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestProblem_ContentTypes(t *testing.T) {
	t.Parallel()

	problem, ok := errors.AsType[*cqrsadapter.ProblemError](cqrsadapter.MapError(context.Background(),
		errorfamily.NewRejection("visit.invalid", "invalid")))
	if !ok {
		t.Fatalf("MapError did not return a *ProblemError")
	}

	if got := problem.ContentType("application/cbor"); got != "application/problem+cbor" {
		t.Errorf("ContentType(cbor) = %q, want application/problem+cbor", got)
	}

	if got := problem.ContentType("application/xml"); got != "application/xml" {
		t.Errorf("ContentType(unknown) = %q, want it echoed back", got)
	}

	// ProblemError keeps the underlying error reachable, so callers can still
	// ask "was this a denial?" through the problem document.
	if problem.Unwrap() == nil {
		t.Error("ProblemError lost its cause")
	}

	encoded, err := json.Marshal(problem)
	if err != nil {
		t.Fatalf("marshal problem: %v", err)
	}

	if string(encoded) == "" {
		t.Error("ProblemError marshalled to an empty document")
	}
}

func TestRequestID_ReachesTheProblemInstance(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, _ *greetingCommand) error {
		return errorfamily.NewRejection("visit.url_required", "url is required")
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(env.api, pipe,
		humav2.Operation{OperationID: "greet", Method: "POST", Path: "/greet", DefaultStatus: http.StatusCreated},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return newGreetingCommand(t, "ada"), nil
		},
	)

	reqID := cqrshtmx.NewRequestID()
	ctx := cqrshtmx.WithRequestID(context.Background(), reqID)

	rec := env.do(ctx, http.MethodPost, "/greet", "{}")

	problem := decodeProblem(t, rec)

	if problem.Instance != reqID.String() {
		t.Errorf("instance = %q, want the request ID %q", problem.Instance, reqID.String())
	}
}
