package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/docserver"
	"github.com/larsartmann/go-cqrs-lite/catalog/v3/simple"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

type catalogUserCmd struct {
	Email string `json:"email" doc:"User email address"`
	Name  string `json:"name"  doc:"Display name"`
}

type catalogUserEvent struct {
	UserID string `json:"user_id" doc:"New user identifier"`
	Email  string `json:"email"   doc:"User email"`
}

type catalogUserQuery struct {
	ID string `json:"id" doc:"User ID to retrieve"`
}

// testCatalogVersion is the version string used across all catalog test fixtures.
const testCatalogVersion = "1.0.0"

func buildCatalogForApp(app *cqrshtmx.App) *catalog.Catalog {
	b := simple.New(
		app.ServiceName(),
		testCatalogVersion,
		simple.WithServiceSummary("Integration test service"),
	)

	simple.Command[catalogUserCmd](
		b, "create-user",
		simple.WithOperation("POST", "/api/users"),
		catalog.WithSummary("Create a new user account"),
	)

	simple.Event[catalogUserEvent](
		b, "user.created", catalog.Sends,
		catalog.WithSummary("A user was created"),
	)

	simple.Query[catalogUserQuery](
		b, "get-user",
		simple.WithOperation("GET", "/api/users/{id}"),
		catalog.WithSummary("Retrieve a user by ID"),
	)

	return b.Build()
}

// TestCatalog_WithApp tests that the catalog sub-package correctly generates
// documentation for types used with cqrs-htmx App handlers.
func TestCatalog_WithApp(t *testing.T) {
	t.Parallel()

	cmds := command.NewDispatcher()
	queries := query.NewDispatcher()

	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands:    cmds,
		Queries:     queries,
		ServiceName: "integration-test-service",
	})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	cat := buildCatalogForApp(app)

	if len(cat.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(cat.Services))
	}

	svc := cat.Services[0]
	if len(svc.Commands) != 1 || len(svc.Events) != 1 || len(svc.Queries) != 1 {
		t.Errorf("expected 1/1/1 cmd/evt/qry, got %d/%d/%d",
			len(svc.Commands), len(svc.Events), len(svc.Queries))
	}

	assertOpenAPI(t, cat)
	assertAsyncAPI(t, cat)
	assertD2ContainsService(t, cat)
}

func assertOpenAPI(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "Test",
		Version:     testCatalogVersion,
	})
	handler := ds.OpenAPISpec()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	doc := unmarshalJSONBody(t, w)

	if doc["openapi"] != "3.0.3" {
		t.Errorf("expected openapi 3.0.3, got %v", doc["openapi"])
	}
}

func assertAsyncAPI(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	ds := docserver.NewDocsServer(func() *catalog.Catalog { return cat }, docserver.Config{
		ServiceName: "Test",
		Version:     testCatalogVersion,
	})
	handler := ds.AsyncAPISpec()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func assertD2ContainsService(t *testing.T, cat *catalog.Catalog) {
	t.Helper()

	handler := docserver.D2Handler(cat)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "integration") {
		t.Errorf("D2 diagram should contain service identifier")
	}
}
