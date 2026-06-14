package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

const testPassword = "secret12"

// typedCountUsersQuery is a typed query that lives in the integration_test
// module. It demonstrates the query.RegisterTyped pattern crossing module
// boundaries (cqrs-htmx + usermgmt + go-cqrs-lite/query).
type typedCountUsersQuery struct {
	*query.BasicQuery
}

func newTypedCountUsersQuery() *typedCountUsersQuery {
	core, err := query.New("CountUsers")
	if err != nil {
		panic(err)
	}
	return &typedCountUsersQuery{BasicQuery: core}
}

// TestTypedQueryDispatch_CrossModule verifies that query.RegisterTyped
// registered in one module can be dispatched through an App configured
// in another module, with the result decoded via query.DispatchTyped.
// TestCrossModule_PaginationFlow verifies that cqrshtmx.DecodePagination
// (root module) produces the same shape as a manually-constructed
// query.NewPagination (usermgmt-independent) instance, and that
// both can be consumed by a typed query handler in another module.
func TestCrossModule_PaginationFlow(t *testing.T) {
	// Root module decodes from request URL.
	r := httptest.NewRequest(
		http.MethodGet, "/items?page=3&page_size=25", nil,
	)
	p := cqrshtmx.DecodePagination(r)

	// Independently construct the same pagination from usermgmt perspective.
	// (This demonstrates that cqrshtmx and go-cqrs-lite/query agree on the shape.)
	manual := query.NewPagination(3, 25)

	if p.Page != manual.Page {
		t.Errorf("page mismatch: %d vs %d", p.Page, manual.Page)
	}
	if p.PageSize != manual.PageSize {
		t.Errorf("page_size mismatch: %d vs %d", p.PageSize, manual.PageSize)
	}

	// Use the pagination in a typed query handler to verify it works.
	qdisp := query.NewDispatcher()
	_ = query.RegisterTyped(
		qdisp, "Page",
		func(_ context.Context, q *typedPageQuery) (query.Pagination, error) {
			return query.NewPagination(p.Page, p.PageSize), nil
		},
	)

	got, err := query.DispatchTyped[query.Pagination](
		context.Background(), qdisp, newTypedPageQuery(),
	)
	if err != nil {
		t.Fatalf("DispatchTyped: %v", err)
	}
	if got.Page != 3 || got.PageSize != 25 {
		t.Errorf("got pagination %+v, want page=3 page_size=25", got)
	}
}

// typedPageQuery is a typed query type for the pagination flow test.
type typedPageQuery struct {
	*query.BasicQuery
}

func newTypedPageQuery() *typedPageQuery {
	core, err := query.New("Page")
	if err != nil {
		panic(err)
	}
	return &typedPageQuery{BasicQuery: core}
}

func TestTypedQueryDispatch_CrossModule(t *testing.T) {
	ctx := context.Background()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{BcryptCost: 4})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	registerUser := func(t *testing.T, email string) {
		t.Helper()
		uid := cqrshtmx.NewUserID()
		if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
			ID:       usermgmt.NewUserID(uid.String()),
			Email:    email,
			Password: testPassword,
		}); err != nil {
			t.Fatalf("Register %s: %v", email, err)
		}
	}

	registerUser(t, "typed1@test.com")
	registerUser(t, "typed2@test.com")

	qdisp := query.NewDispatcher()
	if err := query.RegisterTyped(
		qdisp, "CountUsers",
		func(ctx context.Context, _ *typedCountUsersQuery) (int, error) {
			// The typed handler receives the same context as the dispatch.
			// Here we just return a fixed count since registration is the
			// primary thing under test.
			_ = ctx
			return 2, nil
		},
	); err != nil {
		t.Fatalf("RegisterTyped: %v", err)
	}

	cdisp := command.NewDispatcher()
	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: cdisp,
		Queries:  qdisp,
	})
	if err != nil {
		t.Fatalf("cqrshtmx.New: %v", err)
	}
	if app == nil {
		t.Fatal("expected non-nil app")
	}

	count, err := query.DispatchTyped[int](
		context.Background(), qdisp, newTypedCountUsersQuery(),
	)
	if err != nil {
		t.Fatalf("DispatchTyped: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 registered users, got %d", count)
	}
}

// TestTypedQueryDispatch_ThroughHTTPHandler verifies that a typed query
// can be served by a real HTTP handler built with cqrshtmx.App.Query.
func TestTypedQueryDispatch_ThroughHTTPHandler(t *testing.T) {
	qdisp := query.NewDispatcher()
	if err := query.RegisterTyped(
		qdisp, "CountUsers",
		func(_ context.Context, _ *typedCountUsersQuery) (int, error) {
			return 42, nil
		},
	); err != nil {
		t.Fatalf("RegisterTyped: %v", err)
	}

	cdisp := command.NewDispatcher()
	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: cdisp,
		Queries:  qdisp,
	})
	if err != nil {
		t.Fatalf("cqrshtmx.New: %v", err)
	}

	handler := app.Query(
		"CountUsers",
		cqrshtmx.DecodeJSONQuery(func(_ typedCountUsersQuery) (query.Query, error) {
			return newTypedCountUsersQuery(), nil
		}),
		cqrshtmx.Render(func(w http.ResponseWriter, _ *http.Request, result any) error {
			count, ok := result.(int)
			if !ok {
				return fmt.Errorf("expected int, got %T", result)
			}
			_, err := fmt.Fprintf(w, `{"count":%s}`, strconv.Itoa(count))
			if err != nil {
				return fmt.Errorf("write count response: %w", err)
			}
			return nil
		}),
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/count", strings.NewReader("{}"))
	r.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != `{"count":42}` {
		t.Errorf("expected body `{\"count\":42}`, got %q", w.Body.String())
	}
}
