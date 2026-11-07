package openapi_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	cqrsopenapi "github.com/larsartmann/cqrs-htmx/v4/openapi"
)

// The RFC 10008 QUERY operation surface: the Query constructor, the `query`
// Path Item field (OpenAPI 3.2-only), the auto-bump on registration, and the
// Validate invariant that a query operation never rides a 3.1.0 document.
// Plan: docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md §W4.1.

func TestQueryOp_RendersKeyAndBumpsVersion(t *testing.T) {
	spec := cqrsopenapi.New("Demo API", "1.0.0").
		Path("/contacts/search",
			cqrsopenapi.Query("SearchContacts").
				Summary("Search contacts with a complex filter body").
				Tag("contacts").
				JSONBody(cqrsopenapi.Object(
					cqrsopenapi.Prop("filter", cqrsopenapi.String()),
				)).
				Response(200, "OK", cqrsopenapi.JSON(cqrsopenapi.Object(
					cqrsopenapi.Prop("id", cqrsopenapi.String()),
				))),
		)

	data, err := spec.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got := doc["openapi"]; got != "3.2.0" {
		t.Fatalf("openapi version = %v, want 3.2.0 (the query operation type is 3.2-only)", got)
	}

	paths, _ := doc["paths"].(map[string]any)
	item, _ := paths["/contacts/search"].(map[string]any)
	query, ok := item["query"].(map[string]any)
	if !ok {
		t.Fatalf("paths[\"/contacts/search\"][\"query\"] missing or not an object: %v", item)
	}

	if got := query["operationId"]; got != "SearchContacts" {
		t.Fatalf("query.operationId = %v, want SearchContacts", got)
	}
}

func TestQueryOp_PathOpAlsoBumps(t *testing.T) {
	op := cqrsopenapi.Query("SyncProbe").
		NoContent(204, "probed").
		Op()

	spec := cqrsopenapi.New("Demo", "1.0.0").PathOp("/probe", op)

	if spec.OpenAPI != "3.2.0" {
		t.Fatalf("OpenAPI = %q, want 3.2.0 after PathOp with a query operation", spec.OpenAPI)
	}
}

func TestQueryOp_NoQueryStays31(t *testing.T) {
	spec := cqrsopenapi.New("Demo", "1.0.0").
		Path("/items", cqrsopenapi.Get("ListItems").Response(200, "OK"))

	if spec.OpenAPI != "3.1.0" {
		t.Fatalf("OpenAPI = %q, want 3.1.0 when no query operation is registered", spec.OpenAPI)
	}
}

func TestQueryOp_HandBuiltNeeds32(t *testing.T) {
	spec := cqrsopenapi.New("Demo", "1.0.0")
	spec.Paths["/search"] = &cqrsopenapi.PathItem{
		Query: &cqrsopenapi.Operation{
			OperationID: "Search",
			Responses:   map[int]*cqrsopenapi.Response{200: {Description: "OK"}},
		},
	}

	err := spec.Validate()
	if err == nil {
		t.Fatalf("Validate: a query operation on a 3.1.0 document must fail")
	}

	if !strings.Contains(err.Error(), "3.2.0") {
		t.Fatalf("Validate error should name the 3.2.0 fix, got: %v", err)
	}

	spec.OpenAPI = "3.2.0"
	if err := spec.Validate(); err != nil {
		t.Fatalf("Validate after version fix: %v", err)
	}
}

func TestQueryOp_DoubleRegistrationPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("a second Query registration on the same path must panic, not silently replace")
		}
	}()

	_ = cqrsopenapi.New("Demo", "1.0.0").
		Path("/search",
			cqrsopenapi.Query("Search").Response(200, "OK"),
			cqrsopenapi.Query("SearchAgain").Response(200, "OK"),
		)
}

func TestQueryOp_ValidateCoversQueryForRequiredResponses(t *testing.T) {
	spec := cqrsopenapi.New("Demo", "1.0.0").
		Path("/search", cqrsopenapi.Query("NoResponses"))

	err := spec.Validate()
	if err == nil || !strings.Contains(err.Error(), "no responses") {
		t.Fatalf("a query operation without responses must fail Validate like every other method, got: %v", err)
	}
}
