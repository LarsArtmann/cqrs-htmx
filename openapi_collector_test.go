package cqrshtmx_test

import (
	"net/http"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// TestOpenAPIRoutesCollected proves the WithOpenAPI metadata is actually
// readable back: registrations surface in order with kind, method, and the
// operation summary, metadata-less handlers contribute nothing, and the
// accessor returns a detached copy.
func TestOpenAPIRoutesCollected(t *testing.T) {
	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: command.NewDispatcher(),
		Queries:  query.NewDispatcher(),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_ = app.Command("CollectCmd",
		cqrshtmx.WithOpenAPI(openapi.Post("CollectCmd").Summary("create it").Op()),
	)
	_ = app.Query("CollectQuery",
		cqrshtmx.WithOpenAPI(openapi.Get("CollectQuery").Summary("read it").Op()),
	)
	// A handler WITHOUT WithOpenAPI contributes nothing.
	_ = app.Command("CollectCmd")

	routes := app.OpenAPIRoutes()
	if len(routes) != 2 {
		t.Fatalf("collected %d routes, want 2 (metadata-less handlers excluded)", len(routes))
	}

	if routes[0].Kind != "command" || !strings.EqualFold(routes[0].Method, http.MethodPost) ||
		routes[0].Operation.Summary != "create it" {
		t.Errorf("route[0] = %+v", routes[0])
	}

	if routes[1].Kind != "query" || !strings.EqualFold(routes[1].Method, http.MethodGet) ||
		routes[1].Operation.Summary != "read it" {
		t.Errorf("route[1] = %+v", routes[1])
	}

	// The accessor returns a copy: mutating it must not affect the App.
	routes[0].Operation.Summary = "mutated"
	if app.OpenAPIRoutes()[0].Operation.Summary != "create it" {
		t.Fatal("OpenAPIRoutes did not return a detached copy")
	}
}
