package openapi

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// JSON serializes the spec to indented OpenAPI 3.1 JSON. The output is suitable
// for writing to an openapi.json file or serving from a /openapi.json endpoint.
// The spec is validated first (see [Spec.Validate]) — an invalid spec fails
// here, once, instead of serving broken JSON to API consumers.
func (s *Spec) JSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	//nolint:wrapcheck // openapi is dependency-free; the caller (OpenAPISpecHandler) classifies and wraps this as Infrastructure.
	if err := json.MarshalWrite(&buf, s, jsontext.WithIndent("  ")); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
