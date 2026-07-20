package openapi

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// JSON serializes the spec to indented OpenAPI 3.1 JSON. The output is suitable
// for writing to an openapi.json file or serving from a /openapi.json endpoint.
func (s *Spec) JSON() ([]byte, error) {
	var buf bytes.Buffer

	if err := json.MarshalWrite(&buf, s, jsontext.WithIndent("  ")); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
