package huma_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	humav2 "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	cqrsadapter "github.com/larsartmann/cqrs-htmx/huma/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// TestMain silences the classified pipeline log: the adapter logs one record per
// failure by design, and a test suite that provokes dozens of failures would
// otherwise bury its own output.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.DiscardHandler))

	os.Exit(m.Run())
}

// ---- fixtures ---------------------------------------------------------------

// testAPI is a real Huma API over a net/http mux, so every assertion runs the
// same code path a consumer's server does.
type testAPI struct {
	api humav2.API
	mux *http.ServeMux
}

func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	mux := http.NewServeMux()

	return testAPI{api: humago.New(mux, humav2.DefaultConfig("test api", "0.1.0")), mux: mux}
}

// do issues a JSON request with ctx attached and returns the recorder.
func (a testAPI) do(ctx context.Context, method, path, body string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequestWithContext(ctx, method, path, reader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	a.mux.ServeHTTP(rec, req)

	return rec
}

// stubChecker is a PermissionChecker stub that records the question it was
// asked, so tests can assert what the pipeline actually asked the engine.
type stubChecker struct {
	allowed bool
	rules   []string
	err     error

	asked      bool
	subject    string
	domain     string
	resource   string
	actionName string
}

func (c *stubChecker) Check(subject, domain, resource, action string) (bool, []string, error) {
	c.asked = true
	c.subject = subject
	c.domain = domain
	c.resource = resource
	c.actionName = action

	if c.err != nil {
		return false, nil, c.err
	}

	return c.allowed, c.rules, nil
}

// greetingCommand is a command carrying metadata, so enrichment is observable.
type greetingCommand struct {
	*command.BasicCommand
	Name string
}

func newGreetingCommand(t *testing.T, name string) *greetingCommand {
	t.Helper()

	core, err := command.New("Greet", id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	return &greetingCommand{BasicCommand: core, Name: name}
}

// sumQuery is a query carrying metadata, so enrichment is observable.
type sumQuery struct {
	*query.BasicQuery
	A int
	B int
}

func newSumQuery(t *testing.T, a, b int) *sumQuery {
	t.Helper()

	core, err := query.New("Sum")
	if err != nil {
		t.Fatalf("query.New: %v", err)
	}

	return &sumQuery{BasicQuery: core, A: a, B: b}
}

// exampleError carries machine context, so the problem document's context
// extension can be asserted.
type exampleError struct{}

func (exampleError) Error() string { return "url is required" }

func (exampleError) ErrorContext() map[string]string {
	return map[string]string{"form_key": "url"}
}

// ---- success paths ----------------------------------------------------------

func TestRegisterCommand_DispatchesAndSetsDefaultStatus(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)

	cmdDisp := command.NewDispatcher()

	var got *greetingCommand

	registerErr := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, cmd *greetingCommand) error {
		got = cmd

		return nil
	})
	if registerErr != nil {
		t.Fatalf("register command: %v", registerErr)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(env.api, pipe,
		humav2.Operation{OperationID: "greet", Method: "POST", Path: "/greet", DefaultStatus: http.StatusCreated},
		func(_ context.Context, in *struct {
			Body struct {
				Name string `json:"name"`
			}
		},
		) (command.Command, error) {
			return newGreetingCommand(t, in.Body.Name), nil
		},
	)

	userID := cqrshtmx.GenerateUserID()
	rec := env.do(cqrshtmx.WithUserID(context.Background(), userID), http.MethodPost, "/greet", `{"name":"ada"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	if got == nil {
		t.Fatal("command handler never ran")
	}

	if got.Name != "ada" {
		t.Errorf("command Name = %q, want %q", got.Name, "ada")
	}

	// Enrichment: the user from the request context must reach the command's
	// metadata before dispatch.
	if got.Metadata().UserID != userID {
		t.Errorf("command metadata UserID = %v, want %v", got.Metadata().UserID, userID)
	}
}

func TestRegisterQuery_ReturnsTypedOutput(t *testing.T) {
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

	type sumOutput struct {
		Body struct {
			Total int `json:"total"`
		}
	}

	cqrsadapter.RegisterQuery(env.api, pipe,
		humav2.Operation{OperationID: "sum", Method: "GET", Path: "/sum"},
		func(_ context.Context, in *struct {
			A int `query:"a"`
			B int `query:"b"`
		},
		) (query.Query, error) {
			return newSumQuery(t, in.A, in.B), nil
		},
		func(_ context.Context, _ *struct {
			A int `query:"a"`
			B int `query:"b"`
		},
			result any,
		) (*sumOutput, error) {
			total, ok := result.(int)
			if !ok {
				return nil, errorfamily.NewCorruption("unexpected_result", "query returned an unexpected type")
			}

			out := &sumOutput{}
			out.Body.Total = total

			return out, nil
		},
	)

	rec := env.do(context.Background(), http.MethodGet, "/sum?a=2&b=3", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		Total int `json:"total"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body.Total != 5 {
		t.Errorf("total = %d, want 5", body.Total)
	}
}

// ---- authorization ----------------------------------------------------------

func TestAuthorization_Matrix(t *testing.T) {
	t.Parallel()

	allowedChecker := &stubChecker{allowed: true, rules: []string{"p, role_user, *, visits, create, allow"}}
	deniedChecker := &stubChecker{allowed: false, rules: []string{"p, role_user, *, visits, create, deny"}}
	brokenChecker := &stubChecker{err: errorfamily.NewTransient("enforce_failed", "engine offline")}

	anonymous := context.Background()
	authenticated := cqrshtmx.WithUserID(context.Background(), cqrshtmx.GenerateUserID())

	tests := []struct {
		name        string
		checker     cqrshtmx.PermissionChecker
		ctx         context.Context
		options     []cqrsadapter.Option
		wantStatus  int
		wantCode    string
		wantChecked bool
	}{
		{
			name:        "authenticated caller with a grant is dispatched",
			checker:     allowedChecker,
			ctx:         authenticated,
			options:     []cqrsadapter.Option{cqrsadapter.Authorize("visits", "create")},
			wantStatus:  http.StatusCreated,
			wantCode:    "",
			wantChecked: true,
		},
		{
			name:        "anonymous caller is unauthorized and never reaches the engine",
			checker:     allowedChecker,
			ctx:         anonymous,
			options:     []cqrsadapter.Option{cqrsadapter.Authorize("visits", "create")},
			wantStatus:  http.StatusUnauthorized,
			wantCode:    httperr.CodeUnauthorized,
			wantChecked: false,
		},
		{
			name:        "policy denial is forbidden",
			checker:     deniedChecker,
			ctx:         authenticated,
			options:     []cqrsadapter.Option{cqrsadapter.Authorize("visits", "create")},
			wantStatus:  http.StatusForbidden,
			wantCode:    httperr.CodeForbidden,
			wantChecked: true,
		},
		{
			name:        "engine failure is a 503, never a silent denial",
			checker:     brokenChecker,
			ctx:         authenticated,
			options:     []cqrsadapter.Option{cqrsadapter.Authorize("visits", "create")},
			wantStatus:  http.StatusServiceUnavailable,
			wantCode:    "enforce_failed",
			wantChecked: true,
		},
		{
			name:        "no checker configured fails closed",
			checker:     nil,
			ctx:         authenticated,
			options:     []cqrsadapter.Option{cqrsadapter.Authorize("visits", "create")},
			wantStatus:  http.StatusServiceUnavailable,
			wantCode:    "enforcer_nil",
			wantChecked: false,
		},
		{
			name:        "RequireAuth rejects an anonymous caller",
			checker:     nil,
			ctx:         anonymous,
			options:     []cqrsadapter.Option{cqrsadapter.RequireAuth()},
			wantStatus:  http.StatusUnauthorized,
			wantCode:    httperr.CodeUnauthorized,
			wantChecked: false,
		},
		{
			name:        "RequireAuth admits an authenticated caller without a policy question",
			checker:     nil,
			ctx:         authenticated,
			options:     []cqrsadapter.Option{cqrsadapter.RequireAuth()},
			wantStatus:  http.StatusCreated,
			wantCode:    "",
			wantChecked: false,
		},
		{
			name:        "an operation with no authorization posture is public",
			checker:     nil,
			ctx:         anonymous,
			options:     nil,
			wantStatus:  http.StatusCreated,
			wantCode:    "",
			wantChecked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestAPI(t)
			cmdDisp := command.NewDispatcher()

			if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, _ *greetingCommand) error {
				return nil
			}); err != nil {
				t.Fatalf("register command: %v", err)
			}

			pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp, Checker: tt.checker})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			cqrsadapter.RegisterCommand(
				env.api,
				pipe,
				humav2.Operation{
					OperationID:   "create-visit",
					Method:        "POST",
					Path:          "/visits",
					DefaultStatus: http.StatusCreated,
				},
				func(_ context.Context, _ *struct{}) (command.Command, error) {
					return newGreetingCommand(t, "visit"), nil
				},
				tt.options...,
			)

			rec := env.do(tt.ctx, http.MethodPost, "/visits", "{}")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			if tt.wantCode == "" {
				return
			}

			problem := decodeProblem(t, rec)

			if problem.Code != tt.wantCode {
				t.Errorf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}

			if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
				t.Errorf("Content-Type = %q, want application/problem+json", got)
			}

			if tt.wantChecked {
				stub, ok := tt.checker.(*stubChecker)
				if !ok {
					t.Fatalf("checker is %T, want *stubChecker", tt.checker)
				}

				if !stub.asked {
					t.Error("checker was never asked")
				}
			}
		})
	}
}

func TestAuthorizeOn_DerivesTheQuestionFromTypedInput(t *testing.T) {
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

	type revokeInput struct {
		Tenant  string `header:"X-Tenant"`
		TokenID string `                  path:"token" doc:"token id"`
	}

	cqrsadapter.RegisterCommand(
		env.api,
		pipe,
		humav2.Operation{
			OperationID:   "revoke",
			Method:        "POST",
			Path:          "/tokens/{token}/revoke",
			DefaultStatus: http.StatusNoContent,
		},
		func(_ context.Context, _ *revokeInput) (command.Command, error) {
			return newGreetingCommand(t, "revoke"), nil
		},
		cqrsadapter.AuthorizeOn(func(_ context.Context, in *revokeInput) (cqrshtmx.Request, error) {
			return cqrshtmx.Request{
				Domain:   in.Tenant,
				Resource: cqrshtmx.Resource{Type: "tokens", ID: in.TokenID},
				Action:   "revoke",
			}, nil
		}),
	)

	ctx := cqrshtmx.WithUserID(context.Background(), cqrshtmx.GenerateUserID())
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tokens/tok-42/revoke", strings.NewReader(""))
	req.Header.Set("X-Tenant", "acme")

	rec := httptest.NewRecorder()
	env.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body.String())
	}

	if checker.domain != "acme" {
		t.Errorf("domain = %q, want acme", checker.domain)
	}

	if checker.resource != "tokens:tok-42" {
		t.Errorf("resource = %q, want tokens:tok-42", checker.resource)
	}

	if checker.actionName != "revoke" {
		t.Errorf("action = %q, want revoke", checker.actionName)
	}

	if checker.subject == "" {
		t.Error("subject was not derived from the request context")
	}
}

// ---- validation and guards --------------------------------------------------

func TestValidateCommand_IsARejectionWithTheSharedCode(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	dispatched := false

	if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, _ *greetingCommand) error {
		dispatched = true

		return nil
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(
		env.api,
		pipe,
		humav2.Operation{
			OperationID:   "create-visit",
			Method:        "POST",
			Path:          "/visits",
			DefaultStatus: http.StatusCreated,
		},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return newGreetingCommand(t, ""), nil
		},
		cqrsadapter.ValidateCommand(func(command.Command) error {
			// A plain error must NOT become a 503: the validation stage wraps it
			// as a rejection, exactly like the endpoint pipeline does.
			return exampleError{}
		}),
	)

	rec := env.do(context.Background(), http.MethodPost, "/visits", "{}")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}

	problem := decodeProblem(t, rec)

	if problem.Code != "cqrshtmx.validate.failed" {
		t.Errorf("problem code = %q, want cqrshtmx.validate.failed", problem.Code)
	}

	if problem.Context["form_key"] != "url" {
		t.Errorf(
			"problem context = %v, want form_key=url (client-fault context must survive the wrap)",
			problem.Context,
		)
	}

	if dispatched {
		t.Error("dispatch ran even though validation failed")
	}
}

func TestGuard_RunsAfterBuildAndCanDenyWithAnyStatus(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, _ *greetingCommand) error {
		return nil
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(
		env.api,
		pipe,
		humav2.Operation{
			OperationID:   "create-visit",
			Method:        "POST",
			Path:          "/visits",
			DefaultStatus: http.StatusCreated,
		},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return newGreetingCommand(t, "visit"), nil
		},
		cqrsadapter.Guard(func(_ context.Context, cmdOrQuery any) error {
			if _, ok := cmdOrQuery.(*greetingCommand); !ok {
				t.Errorf("guard received %T, want *greetingCommand", cmdOrQuery)
			}

			return httperr.WithHTTPStatus(httperr.ErrForbidden, http.StatusTeapot)
		}),
	)

	rec := env.do(context.Background(), http.MethodPost, "/visits", "{}")

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418 — an explicit status carrier must win", rec.Code)
	}
}

// ---- error mapping ----------------------------------------------------------

func TestDispatchErrors_MapToTheirFamily(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "rejection",
			err:        errorfamily.NewRejection("visit.url_required", "url is required"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "visit.url_required",
		},
		{
			name:       "conflict",
			err:        errorfamily.NewConflict("visit.exists", "visit already recorded"),
			wantStatus: http.StatusConflict,
			wantCode:   "visit.exists",
		},
		{
			name:       "transient",
			err:        errorfamily.NewTransient("store_unavailable", "store is offline"),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "store_unavailable",
		},
		{
			name:       "corruption",
			err:        errorfamily.NewCorruption("state_corrupt", "state is inconsistent"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "state_corrupt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := newTestAPI(t)
			cmdDisp := command.NewDispatcher()

			if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, _ *greetingCommand) error {
				return tt.err
			}); err != nil {
				t.Fatalf("register command: %v", err)
			}

			pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			cqrsadapter.RegisterCommand(
				env.api,
				pipe,
				humav2.Operation{
					OperationID:   "create-visit",
					Method:        "POST",
					Path:          "/visits",
					DefaultStatus: http.StatusCreated,
				},
				func(_ context.Context, _ *struct{}) (command.Command, error) {
					return newGreetingCommand(t, "visit"), nil
				},
			)

			rec := env.do(context.Background(), http.MethodPost, "/visits", "{}")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}

			problem := decodeProblem(t, rec)

			if problem.Code != tt.wantCode {
				t.Errorf("problem code = %q, want %q", problem.Code, tt.wantCode)
			}

			if problem.Status != tt.wantStatus {
				t.Errorf("problem status = %d, want %d", problem.Status, tt.wantStatus)
			}
		})
	}
}

func TestBuildErrors_PassHumaStatusErrorsThrough(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(
		env.api,
		pipe,
		humav2.Operation{
			OperationID:   "create-visit",
			Method:        "POST",
			Path:          "/visits",
			DefaultStatus: http.StatusCreated,
		},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			// The consumer already decided the response shape; the adapter must
			// not relabel it.
			return nil, humav2.Error422UnprocessableEntity("url is not a valid URL", nil)
		},
	)

	rec := env.do(context.Background(), http.MethodPost, "/visits", "{}")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestBuildNilCommand_IsAServerFault(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(
		env.api,
		pipe,
		humav2.Operation{
			OperationID:   "create-visit",
			Method:        "POST",
			Path:          "/visits",
			DefaultStatus: http.StatusCreated,
		},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return nil, nil
		},
	)

	rec := env.do(context.Background(), http.MethodPost, "/visits", "{}")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 — a build function returning nil is a wiring bug", rec.Code)
	}
}

func TestMissingDispatcher_Answers503(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	qryDisp := query.NewDispatcher()

	// Adapter is configured for queries only, so the command route has no
	// dispatcher to reach.
	pipe, err := cqrsadapter.New(cqrsadapter.Config{Queries: qryDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(
		env.api,
		pipe,
		humav2.Operation{
			OperationID:   "create-visit",
			Method:        "POST",
			Path:          "/visits",
			DefaultStatus: http.StatusCreated,
		},
		func(_ context.Context, _ *struct{}) (command.Command, error) {
			return newGreetingCommand(t, "visit"), nil
		},
	)

	rec := env.do(context.Background(), http.MethodPost, "/visits", "{}")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body: %s)", rec.Code, rec.Body.String())
	}

	problem := decodeProblem(t, rec)

	if problem.Code != httperr.ErrorCode(httperr.ErrCommandsNil) {
		t.Errorf("problem code = %q, want %q", problem.Code, httperr.ErrorCode(httperr.ErrCommandsNil))
	}
}

// ---- exported stages --------------------------------------------------------

func TestAdapterStages_AreUsableStandalone(t *testing.T) {
	t.Parallel()

	cmdDisp := command.NewDispatcher()

	var got *greetingCommand

	if err := command.RegisterTyped(cmdDisp, "Greet", func(_ context.Context, cmd *greetingCommand) error {
		got = cmd

		return nil
	}); err != nil {
		t.Fatalf("register command: %v", err)
	}

	checker := &stubChecker{allowed: true}

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp, Checker: checker})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	userID := cqrshtmx.GenerateUserID()
	ctx := cqrshtmx.WithUserID(context.Background(), userID)

	if err := pipe.Authorize(ctx, cqrshtmx.Request{
		Resource: cqrshtmx.Resource{Type: "visits"},
		Action:   "create",
	}); err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	if checker.subject != userID.String() {
		t.Errorf("subject = %q, want %q", checker.subject, userID.String())
	}

	if err := pipe.DispatchCommand(ctx, newGreetingCommand(t, "ada")); err != nil {
		t.Fatalf("DispatchCommand: %v", err)
	}

	if got == nil || got.Metadata().UserID != userID {
		t.Fatalf("command was not dispatched with the request identity: %+v", got)
	}
}

func TestAuthorizeStage_DenialsAreClassified(t *testing.T) {
	t.Parallel()

	pipe, err := cqrsadapter.New(cqrsadapter.Config{
		Commands: command.NewDispatcher(),
		Checker:  &stubChecker{allowed: false},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	anonymousErr := pipe.Authorize(context.Background(), cqrshtmx.Request{
		Resource: cqrshtmx.Resource{Type: "visits"},
		Action:   "create",
	})
	if anonymousErr == nil {
		t.Fatal("Authorize: anonymous request was allowed")
	}

	if !cqrshtmx.IsUnauthorized(anonymousErr) {
		t.Errorf("anonymous denial is not an unauthorized error: %v", anonymousErr)
	}

	if status := httperr.MapError(anonymousErr); status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}

	authenticatedErr := pipe.Authorize(
		cqrshtmx.WithUserID(context.Background(), cqrshtmx.GenerateUserID()),
		cqrshtmx.Request{Resource: cqrshtmx.Resource{Type: "visits"}, Action: "create"},
	)
	if authenticatedErr == nil {
		t.Fatal("Authorize: policy denial was allowed")
	}

	if !cqrshtmx.IsForbidden(authenticatedErr) {
		t.Errorf("policy denial is not a forbidden error: %v", authenticatedErr)
	}

	if status := httperr.MapError(authenticatedErr); status != http.StatusForbidden {
		t.Errorf("status = %d, want 403", status)
	}
}

func TestMapError_DocumentsMapToProblemDetails(t *testing.T) {
	t.Parallel()

	if err := cqrsadapter.MapError(context.Background(), nil); err != nil {
		t.Fatalf("MapError(nil) = %v, want nil", err)
	}

	var problem *cqrsadapter.ProblemError
	if !errors.As(cqrsadapter.MapError(context.Background(),
		errorfamily.NewConflict("visit.exists", "visit already recorded")), &problem) {
		t.Fatalf("MapError did not return a *ProblemError")
	}

	if problem.GetStatus() != http.StatusConflict {
		t.Errorf("GetStatus = %d, want 409", problem.GetStatus())
	}

	if problem.Code != "visit.exists" {
		t.Errorf("Code = %q, want visit.exists", problem.Code)
	}

	if got := problem.ContentType("application/json"); got != "application/problem+json" {
		t.Errorf("ContentType = %q, want application/problem+json", got)
	}

	// Server faults must not leak internal detail to a public client.
	var corruption *cqrsadapter.ProblemError
	if !errors.As(cqrsadapter.MapError(context.Background(),
		errorfamily.NewCorruption("state_corrupt", "stream 01HZZ internal detail")), &corruption) {
		t.Fatalf("MapError did not return a *ProblemError")
	}

	if strings.Contains(corruption.Detail, "01HZZ") {
		t.Errorf("5xx detail leaked internal context: %q", corruption.Detail)
	}

	if corruption.Why == "" || corruption.Fix == "" {
		t.Error("5xx problem lost the family's why/fix guidance")
	}

	// An error Huma can already render must pass through untouched so its
	// status survives.
	humaErr := humav2.Error404NotFound("no such visit")

	if mapped := cqrsadapter.MapError(context.Background(), humaErr); !errors.Is(mapped, humaErr) {
		t.Errorf("MapError rewrapped a Huma status error: %v", mapped)
	}
}

// ---- registration guards ----------------------------------------------------

func TestNew_RequiresADispatcher(t *testing.T) {
	t.Parallel()

	if _, err := cqrsadapter.New(cqrsadapter.Config{}); err == nil {
		t.Fatal("New with no dispatchers: want an error, got nil")
	}
}

func TestRegistration_PanicsOnProgrammerErrors(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: command.NewDispatcher()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	op := humav2.Operation{OperationID: "x", Method: "POST", Path: "/x"}

	build := func(_ context.Context, _ *struct{}) (command.Command, error) {
		return nil, nil
	}

	tests := []struct {
		name string
		run  func()
	}{
		{
			name: "nil API",
			run: func() {
				cqrsadapter.RegisterCommand[struct{}](nil, pipe, op, build)
			},
		},
		{
			name: "nil adapter",
			run: func() {
				cqrsadapter.RegisterCommand[struct{}](env.api, nil, op, build)
			},
		},
		{
			name: "nil build",
			run: func() {
				cqrsadapter.RegisterCommand[struct{}](env.api, pipe, op, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Errorf("%s: want a panic at registration", tt.name)
				}
			}()

			tt.run()
		})
	}
}

// ---- Huma contract ----------------------------------------------------------

func TestOpenAPI_IsStillGeneratedByHuma(t *testing.T) {
	t.Parallel()

	env := newTestAPI(t)
	cmdDisp := command.NewDispatcher()

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Commands: cmdDisp})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cqrsadapter.RegisterCommand(env.api, pipe,
		humav2.Operation{
			OperationID:   "create-visit",
			Method:        "POST",
			Path:          "/visits",
			Summary:       "Record a visit",
			DefaultStatus: http.StatusCreated,
		},
		func(_ context.Context, in *struct {
			Body struct {
				URL string `json:"url" doc:"the visited URL"`
			}
		},
		) (command.Command, error) {
			return newGreetingCommand(t, in.Body.URL), nil
		},
	)

	spec, err := env.api.OpenAPI().MarshalJSON()
	if err != nil {
		t.Fatalf("marshal OpenAPI: %v", err)
	}

	var doc struct {
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}

	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatalf("decode OpenAPI: %v", err)
	}

	if _, ok := doc.Paths["/visits"]; !ok {
		t.Fatalf("operation missing from the generated spec: %s", spec)
	}

	// The error contract still belongs to Huma: the default error response must
	// keep pointing at Huma's ErrorModel schema.
	if _, ok := doc.Components.Schemas["ErrorModel"]; !ok {
		t.Error("generated spec lost Huma's ErrorModel schema")
	}

	if !strings.Contains(string(spec), "components/schemas/ErrorModel") {
		t.Error("generated spec no longer references ErrorModel for error responses")
	}
}

// ---- helpers ----------------------------------------------------------------

// decodeProblem reads the problem document out of a response recorder.
func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) cqrsadapter.ProblemError {
	t.Helper()

	var problem cqrsadapter.ProblemError
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem document: %v (body: %s)", err, rec.Body.String())
	}

	return problem
}
