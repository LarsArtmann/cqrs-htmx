package openapi

import (
	"encoding/json/jsontext"
	"errors"
	"testing"
)

// failWriter fails every write, driving the passthrough error branches of the
// canonical MarshalJSONTo encoders (S43: encoder failures must propagate, not
// be swallowed or half-serialized).
type failWriter struct{}

var errFailWrite = errors.New("write failed")

func (failWriter) Write([]byte) (int, error) { return 0, errFailWrite }

func TestCanonical_EncodeErrorsPropagate(t *testing.T) {
	t.Parallel()

	spec := &Spec{
		OpenAPI: "3.1.0",
		Info:    Info{Title: "T", Version: "1.0.0"},
		Paths: map[string]*PathItem{
			"/x": {Get: &Operation{
				OperationID: "X",
				Parameters:  []Parameter{{Name: "id", In: "query", Schema: String()}},
				RequestBody: &RequestBody{
					Content: map[string]*MediaType{"application/json": {Schema: String()}},
				},
				Responses: map[int]*Response{
					200: {
						Description: "ok",
						Headers:     map[string]*Header{"X-H": {Schema: Integer()}},
						Content:     map[string]*MediaType{"application/json": {Schema: String()}},
					},
				},
			}},
		},
		Components: &Components{Schemas: map[string]*Schema{"S": String()}},
	}

	operation := spec.Paths["/x"].Get

	cases := []struct {
		name   string
		encode func(*jsontext.Encoder) error
	}{
		{"spec", spec.MarshalJSONTo},
		{"components", spec.Components.MarshalJSONTo},
		{"operation", operation.MarshalJSONTo},
		{"requestBody", operation.RequestBody.MarshalJSONTo},
		{"response", operation.Responses[200].MarshalJSONTo},
		{"schema", String().MarshalJSONTo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			enc := jsontext.NewEncoder(&failWriter{})

			if err := tc.encode(enc); err == nil {
				t.Fatalf("%s: expected the encoder write failure to propagate", tc.name)
			}
		})
	}
}
