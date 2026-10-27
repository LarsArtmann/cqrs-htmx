package openapi

import "testing"

// TestOperationClone_DeepCopy covers every branch of Operation.Clone and its
// clone* helpers (populated slices/maps/pointers, nil entries) plus the
// isolation guarantee: mutating the clone never reaches the original.
func TestOperationClone_DeepCopy(t *testing.T) {
	t.Parallel()

	op := &Operation{
		Tags:        []string{"a", "b"},
		Summary:     "s",
		OperationID: "id",
		Parameters: []Parameter{
			{Name: "p1", In: "query", Schema: String()},
			{Name: "p2", In: "path"},
		},
		RequestBody: &RequestBody{
			Description: "body",
			Content: map[string]*MediaType{
				"application/json": {Schema: Object(Prop("x", String()))},
			},
		},
		Responses: map[int]*Response{
			200: {
				Description: "ok",
				Headers: map[string]*Header{
					"X-Rate": {Schema: Integer()},
					"X-None": nil,
				},
				Content: map[string]*MediaType{
					"application/json": {Schema: String()},
					"text/plain":       nil,
				},
			},
			404: nil,
		},
		HTTPMethod: "get",
	}

	clone := op.Clone()

	if len(clone.Tags) != 2 || clone.Tags[0] != "a" {
		t.Fatalf("tags not cloned: %v", clone.Tags)
	}

	if clone.Parameters[0].Schema == op.Parameters[0].Schema {
		t.Fatal("parameter schemas must be deep-copied, not aliased")
	}

	if clone.RequestBody.Content["application/json"] == op.RequestBody.Content["application/json"] {
		t.Fatal("request body content must be deep-copied")
	}

	if clone.Responses[200].Headers["X-Rate"] == op.Responses[200].Headers["X-Rate"] {
		t.Fatal("response headers must be deep-copied")
	}

	if clone.Responses[200].Headers["X-None"] != nil {
		t.Fatal("nil header entries must stay nil")
	}

	if clone.Responses[200].Content["application/json"] == op.Responses[200].Content["application/json"] {
		t.Fatal("response content must be deep-copied")
	}

	if clone.Responses[200].Content["text/plain"] != nil {
		t.Fatal("nil content entries must stay nil")
	}

	if clone.Responses[404] != nil {
		t.Fatal("nil response entries must stay nil")
	}

	clone.Tags[0] = "mutated"
	clone.Parameters[0].Schema.Type = "mutated"
	clone.Responses[200].Headers["X-Rate"].Description = "mutated"

	if op.Tags[0] != "a" || op.Parameters[0].Schema.Type != "string" ||
		op.Responses[200].Headers["X-Rate"].Description != "" {
		t.Fatal("clone mutations leaked into the original")
	}
}

// TestOperationClone_Empty covers the nil branch of every clone helper: an
// empty operation clones to empty without nil-map/nil-slice hazards.
func TestOperationClone_Empty(t *testing.T) {
	t.Parallel()

	empty := Operation{OperationID: "empty"}
	clone := empty.Clone()

	if clone.Tags != nil || clone.Parameters != nil || clone.RequestBody != nil || clone.Responses != nil {
		t.Fatalf("empty operation clone should stay nil-valued: %+v", clone)
	}
}

// TestSchemaClone_DeepCopy covers Schema.Clone: properties (including a nil
// entry), required, enum, nested items, and the mutation isolation.
func TestSchemaClone_DeepCopy(t *testing.T) {
	t.Parallel()

	s := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"a": Object(Prop("b", String())),
			"n": nil,
		},
		Required: []string{"a"},
		Enum:     []any{"x", 1},
		Items:    Array(Integer()),
	}

	clone := s.Clone()

	if clone.Properties["a"] == s.Properties["a"] {
		t.Fatal("properties must be deep-copied")
	}

	if clone.Properties["n"] != nil {
		t.Fatal("nil property entries must stay nil")
	}

	if clone.Items == s.Items {
		t.Fatal("items must be deep-copied")
	}

	clone.Properties["a"].Properties["b"].Type = "mutated"
	clone.Items.Type = "mutated"

	if s.Properties["a"].Properties["b"].Type != "string" || s.Items.Type != "array" {
		t.Fatal("schema clone mutations leaked into the original")
	}
}

// TestSchemaClone_SchemaAdditionalProperties covers the *Schema-typed
// AdditionalProperties branch (the falseSchema sentinel is copied as-is).
func TestSchemaClone_SchemaAdditionalProperties(t *testing.T) {
	t.Parallel()

	outer := &Schema{Type: "object", AdditionalProperties: Object(Prop("x", String()))}

	clone := outer.Clone()

	ap, ok := clone.AdditionalProperties.(*Schema)
	if !ok || ap == outer.AdditionalProperties {
		t.Fatalf("additionalProperties *Schema must be deep-copied, got %T", clone.AdditionalProperties)
	}
}
