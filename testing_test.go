package cqrshtmx_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/casbin/casbin/v3"
	"github.com/casbin/casbin/v3/model"
	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	. "github.com/onsi/gomega"
)

// --- Shared test types ---

type testCreateUserCmd struct {
	aggID id.AggregateID
	email string
	name  string
}

func (c *testCreateUserCmd) Type() command.Type          { return "CreateUser" }
func (c *testCreateUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *testCreateUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type testCreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type testGetUserQuery struct{}

func (q *testGetUserQuery) Type() query.Type { return "GetUser" }

type bddCreateUserReq struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type bddCreateUserCmd struct {
	aggID id.AggregateID
	email string
	name  string
}

func (c *bddCreateUserCmd) Type() command.Type          { return "CreateUser" }
func (c *bddCreateUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *bddCreateUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type bddDeleteUserCmd struct {
	aggID id.AggregateID
}

func (c *bddDeleteUserCmd) Type() command.Type          { return "DeleteUser" }
func (c *bddDeleteUserCmd) AggregateID() id.AggregateID { return c.aggID }
func (c *bddDeleteUserCmd) IdempotencyKey() string      { return c.aggID.String() }

type bddListUsersQuery struct{}

func (q *bddListUsersQuery) Type() query.Type { return "ListUsers" }

type bddDashboardQuery struct{}

func (q *bddDashboardQuery) Type() query.Type { return "GetDashboard" }

type bddTemplComponent struct {
	html string
}

func (m *bddTemplComponent) Render(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.html))
	return err
}

type getPageQuery struct{}

func (q *getPageQuery) Type() query.Type { return "GetPage" }

// --- Decoder helpers ---

func decodeCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID()}, nil
	})
}

func decodeCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: id.NewAggregateID(), email: req.Email, name: req.Name}, nil
	})
}

func decodeBDDCreateUserJSON() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{aggID: id.NewAggregateID()}, nil
	})
}

func decodeGetUserJSONQuery() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSONQuery(func(_ testGetUserQuery) (query.Query, error) {
		return &testGetUserQuery{}, nil
	})
}

func decodeCreateUserJSONWithAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(_ testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID}, nil
	})
}

func decodeCreateUserJSONWithBodyAndAggID(aggID id.AggregateID) cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req testCreateUserRequest) (command.Command, error) {
		return &testCreateUserCmd{aggID: aggID, email: req.Email, name: req.Name}, nil
	})
}

func decodeBDDCreateUserJSONWithBody() cqrshtmx.HandlerOption {
	return cqrshtmx.DecodeJSON(func(req bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{
			aggID: id.NewAggregateID(),
			email: req.Email,
			name:  req.Name,
		}, nil
	})
}

func decodeBDDCreateUserFormMapper() func(bddCreateUserReq) (command.Command, error) {
	return func(req bddCreateUserReq) (command.Command, error) {
		return &bddCreateUserCmd{
			aggID: id.NewAggregateID(),
			email: req.Email,
			name:  req.Name,
		}, nil
	}
}

// --- Handler helpers ---

func noOpCommandHandler(_ context.Context, _ command.Command) error { return nil }

func encodeJSONResult(w http.ResponseWriter, _ *http.Request, result any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(result)
}

func rejectionHandler(code, message string) func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		return event.NewRejection(code, message)
	}
}

func middlewareCaptureHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		*called = true
	})
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func createdHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
}

// --- Extractor helpers ---

func staticExtractor(uid cqrshtmx.UserID) cqrshtmx.UserIDExtractor {
	return func(_ *http.Request) (cqrshtmx.UserID, error) { return uid, nil }
}

func headerExtractor(header string) cqrshtmx.UserIDExtractor {
	return func(r *http.Request) (cqrshtmx.UserID, error) {
		return cqrshtmx.ParseUserID(r.Header.Get(header))
	}
}

// --- Tracking dispatcher helpers ---

func trackingCommandHandler(dispatched *bool) func(context.Context, command.Command) error {
	return func(_ context.Context, _ command.Command) error {
		*dispatched = true
		return nil
	}
}

// --- Test enforcer ---

func newTestEnforcer() *casbin.Enforcer {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "r.sub == p.sub && r.obj == p.obj && r.act == p.act")

	e, _ := casbin.NewEnforcer(m)
	_, _ = e.AddPolicy(adminUserID.String(), "users", "create")
	_, _ = e.AddPolicy(adminUserID.String(), "users", "read")
	_, _ = e.AddPolicy(viewerUserID.String(), "users", "read")

	return e
}

// --- App creation helpers ---

func newCommandApp() *cqrshtmx.App {
	disp := command.NewDispatcher()
	_ = disp.Register("CreateUser", noOpCommandHandler)
	cfg := cqrshtmx.Config{Commands: disp}
	app, err := cqrshtmx.New(cfg)
	Expect(err).NotTo(HaveOccurred())
	return app
}

// --- Request/response helpers ---

type testResponse struct {
	*httptest.ResponseRecorder
}

func newPostJSONRequest(body string, opts ...func(*http.Request)) *http.Request {
	r := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, "/users", strings.NewReader(body),
	)
	r.Header.Set("Content-Type", "application/json")
	for _, o := range opts {
		o(r)
	}
	return r
}

func newPostRequest(path, body string, opts ...func(*http.Request)) *http.Request {
	r := httptest.NewRequestWithContext(
		context.Background(), http.MethodPost, path, strings.NewReader(body),
	)
	for _, o := range opts {
		o(r)
	}
	return r
}

func withHTMX(r *http.Request) {
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
}

func withUserHeader(header string, uid cqrshtmx.UserID) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(header, uid.String()) }
}

func withHeader(key, value string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(key, value) }
}

func serve(handler http.Handler, r *http.Request) *testResponse {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return &testResponse{w}
}

func (r *testResponse) code() int { return r.Code }

func assertStatusCode(handler http.Handler, r *http.Request, expected int) {
	w := serve(handler, r)
	ExpectWithOffset(1, w.code()).To(Equal(expected))
}

func assertHTMXErrorRedirect(err error, loginRedirect, expectedRedirect string) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	cqrshtmx.DefaultErrorHandlerWithRedirect(w, r, err, loginRedirect)
	ExpectWithOffset(1, w.Header().Get("HX-Redirect")).To(Equal(expectedRedirect))
}

// --- HTTP interface mock helpers ---

type mockPusher struct {
	http.ResponseWriter
	pushFunc func(target string, opts *http.PushOptions) error
}

func (m *mockPusher) Push(target string, opts *http.PushOptions) error {
	return m.pushFunc(target, opts)
}

func newPusherRecorder(p *mockPusher) *pusherRecorder {
	return &pusherRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		pusher:           p,
	}
}

type pusherRecorder struct {
	*httptest.ResponseRecorder
	pusher *mockPusher
}

func (r *pusherRecorder) Push(target string, opts *http.PushOptions) error {
	return r.pusher.Push(target, opts)
}

type hijackRecorder struct {
	*httptest.ResponseRecorder
}

func newHijackRecorder() *hijackRecorder {
	return &hijackRecorder{ResponseRecorder: httptest.NewRecorder()}
}
