package openapi_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	cqrsopenapi "github.com/larsartmann/cqrs-htmx/v4/openapi"
)

// TestSpec_BasicShape verifies the minimal OpenAPI document structure.
func TestSpec_BasicShape(t *testing.T) {
	spec := cqrsopenapi.New("Demo API", "1.0.0")

	data, err := spec.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, data)
	}

	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}

	info, _ := doc["info"].(map[string]any)
	if info["title"] != "Demo API" || info["version"] != "1.0.0" {
		t.Errorf("info = %+v, want title=Demo API version=1.0.0", info)
	}

	if _, ok := doc["paths"].(map[string]any); !ok {
		t.Error("paths is not an object")
	}
}

// TestSpec_FluentBuild exercises the full fluent API end-to-end and asserts the
// serialized output contains the expected fragments. This is the primary
// correctness test for the builder.
func TestSpec_FluentBuild(t *testing.T) {
	spec := cqrsopenapi.New("Items API", "2.0.0").
		WithDescription("A demo items service").
		Schema("Item", cqrsopenapi.Object(
			cqrsopenapi.PropReq("id", cqrsopenapi.String().WithFormat("uuid")),
			cqrsopenapi.PropReq("name", cqrsopenapi.String().WithMinLength(1)),
		)).
		Path("/items",
			cqrsopenapi.Post("CreateItem").
				Summary("Create a new item").
				Tag("items").
				JSONBody(cqrsopenapi.Object(
					cqrsopenapi.PropReq("name", cqrsopenapi.String().WithMinLength(1)),
				)).
				Response(201, "Created", cqrsopenapi.JSON(cqrsopenapi.Ref("Item"))).
				Response(400, "Bad request", cqrsopenapi.JSON(cqrsopenapi.ErrorSchema())),
		).
		Path("/items/{id}",
			cqrsopenapi.Get("GetItem").
				Summary("Get an item by id").
				Tag("items").
				PathParam("id", cqrsopenapi.String().WithFormat("uuid"), "the item id").
				QueryParam("expand", cqrsopenapi.String(), "expand related resources").
				Response(200, "OK", cqrsopenapi.JSON(cqrsopenapi.Ref("Item"))).
				Response(404, "Not found"),
		).
		Path("/items/{id}",
			cqrsopenapi.Delete("DeleteItem").
				Tag("items").
				PathParam("id", cqrsopenapi.String().WithFormat("uuid"), "the item id").
				NoContent(204, "Deleted"),
		)

	data, err := spec.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	out := string(data)

	for _, want := range []string{
		`"openapi": "3.1.0"`,
		`"title": "Items API"`,
		`"/items"`,
		`"/items/{id}"`,
		`"operationId": "CreateItem"`,
		`"operationId": "GetItem"`,
		`"operationId": "DeleteItem"`,
		`"$ref": "#/components/schemas/Item"`,
		`"name": "id"`,
		`"in": "path"`,
		`"required": true`,
		`"format": "uuid"`,
		`"minLength": 1`,
		`"204"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}

	// The /items/{id} path must contain BOTH Get and Delete (merge behavior).
	itemsPath := spec.Paths["/items/{id}"]
	if itemsPath.Get == nil || itemsPath.Delete == nil {
		t.Error("Path merge failed: /items/{id} should have both Get and Delete")
	}
}

// TestSpec_Golden pins the exact serialized output of a representative spec.
// Update the golden constant only when the builder's output intentionally
// changes; any drift indicates a regression in serialization.
func TestSpec_Golden(t *testing.T) {
	spec := cqrsopenapi.New("Demo", "0.1.0").
		Path("/widgets",
			cqrsopenapi.Post("CreateWidget").
				JSONBody(cqrsopenapi.Object(
					cqrsopenapi.PropReq("name", cqrsopenapi.String()),
				)).
				Response(201, "Created"),
		)

	data, err := spec.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	got := string(data)
	const want = `{
  "openapi": "3.1.0",
  "info": {
    "title": "Demo",
    "version": "0.1.0"
  },
  "paths": {
    "/widgets": {
      "post": {
        "operationId": "CreateWidget",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name": {
                    "type": "string"
                  }
                },
                "required": [
                  "name"
                ]
              }
            }
          }
        },
        "responses": {
          "201": {
            "description": "Created"
          }
        }
      }
    }
  }
}`

	if got != want {
		t.Errorf("golden mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestSchema_Constructors verifies the schema helpers produce correct JSON.
func TestSchema_Constructors(t *testing.T) {
	cases := []struct {
		name   string
		schema *cqrsopenapi.Schema
		want   string
	}{
		{"string", cqrsopenapi.String(), `"type":"string"`},
		{"integer-format", cqrsopenapi.Integer().WithFormat("int64"), `"type":"integer","format":"int64"`},
		{"array", cqrsopenapi.Array(cqrsopenapi.String()), `"type":"array","items":{"type":"string"}`},
		{"ref", cqrsopenapi.Ref("Foo"), `"$ref":"#/components/schemas/Foo"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.schema)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			if !strings.Contains(string(data), tc.want) {
				t.Errorf("schema %s: %s missing %s", tc.name, data, tc.want)
			}
		})
	}
}

// TestSchema_FalseAdditionalProperties verifies the False() sentinel serializes
// to the JSON literal `false` (forbidding extra properties).
func TestSchema_FalseAdditionalProperties(t *testing.T) {
	s := cqrsopenapi.Object(cqrsopenapi.PropReq("k", cqrsopenapi.String()))
	s.AdditionalProperties = cqrsopenapi.False()

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if !strings.Contains(string(data), `"additionalProperties":false`) {
		t.Errorf("expected additionalProperties:false in %s", data)
	}
}

// TestSpec_NestedObject verifies nested object schemas serialize correctly.
func TestSpec_NestedObject(t *testing.T) {
	nested := cqrsopenapi.Object(
		cqrsopenapi.PropReq("outer", cqrsopenapi.Object(
			cqrsopenapi.PropReq("inner", cqrsopenapi.Integer()),
		)),
	)

	data, err := json.Marshal(nested)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, `"properties":{"outer":{"type":"object"`) {
		t.Errorf("nested outer object missing: %s", out)
	}
	if !strings.Contains(out, `"inner":{"type":"integer"}`) {
		t.Errorf("nested inner property missing: %s", out)
	}
}
