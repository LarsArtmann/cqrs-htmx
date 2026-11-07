package huma_test

import (
	"context"
	"net/http"
	"testing"

	huma "github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	cqrsadapter "github.com/larsartmann/cqrs-htmx/huma/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

type probeSearchQuery struct{}

func (q *probeSearchQuery) Type() query.Type { return "Search" }

// NV3 verdict, pinned (QUERY plan §3): huma v2 — latest v2.40.0 — does NOT
// support the RFC 10008 QUERY method. Registration panics with
// "unknown method QUERY" from the OpenAPI Path Item builder (huma
// openapi.go's eight-method switch; no Query case exists upstream). This test
// pins that verdict mechanically: the day huma ships a Query case, this turns
// red and huma/README.md § QUERY + this test get the upgrade together.
// Workaround meanwhile: serve QUERY routes through the route builder
// (app.Query + MethodOverrideMiddleware) next to the Huma API.
func TestQueryMethod_HumaRegistrationPanicsUntilUpstreamSupportsIt(t *testing.T) {
	defer func() {
		r := recover()

		if r == nil {
			t.Fatalf("huma v2.40.0 accepted a QUERY operation — upstream added support; " +
				"update huma/README.md § QUERY and replace this verdict test with a round-trip test")
		}

		msg, ok := r.(string)
		if !ok || msg != "unknown method QUERY" {
			t.Fatalf("unexpected panic (expected the known method-whitelist panic): %v", r)
		}
	}()

	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("probe", "1.0.0"))

	qryDisp := query.NewDispatcher()
	_ = qryDisp.Register("Search", func(_ context.Context, _ query.Query) (any, error) {
		return "found", nil
	})

	pipe, err := cqrsadapter.New(cqrsadapter.Config{Queries: qryDisp})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}

	type searchInput struct {
		Body struct {
			Filter string `json:"filter"`
		}
	}

	type searchOutput struct {
		Message string `json:"message"`
	}

	cqrsadapter.RegisterQuery[searchInput, searchOutput](api, pipe,
		huma.Operation{OperationID: "search", Method: "QUERY", Path: "/search"},
		func(_ context.Context, _ *searchInput) (query.Query, error) {
			return &probeSearchQuery{}, nil
		},
		func(_ context.Context, _ *searchInput, result any) (*searchOutput, error) {
			return &searchOutput{Message: result.(string)}, nil
		},
	)
}
