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

// --- Handler helpers ---

func noOpCommandHandler(_ context.Context, _ command.Command) error { return nil }

func encodeJSONResult(w http.ResponseWriter, _ *http.Request, result any) error {
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

// --- HTTP request helpers ---

func newPostRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
}

func newGetRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodGet, "/users", strings.NewReader(body))
}

func newGetRequestWithPath(path, body string) *http.Request {
	return httptest.NewRequest(http.MethodGet, path, strings.NewReader(body))
}

func withHTMX(r *http.Request) *http.Request {
	r.Header.Set("HX-Request", cqrshtmx.HeaderTrue)
	return r
}

func withUser(r *http.Request, uid cqrshtmx.UserID) *http.Request {
	r.Header.Set("X-User", uid.String())
	return r
}

func withUserID(r *http.Request, uid cqrshtmx.UserID) *http.Request {
	r.Header.Set("X-User-ID", uid.String())
	return r
}
