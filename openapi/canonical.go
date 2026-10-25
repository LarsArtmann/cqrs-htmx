package openapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"strconv"
)

// Deterministic serialization (S43): encoding/json/v2 does NOT sort map keys,
// so a Spec carrying maps serialized to a different byte order per process —
// which churned the content-derived ETag of OpenAPISpecHandler on every
// restart, defeating the restart-stable caching the handler was designed for.
//
// The types below give every map-bearing type a MarshalJSONTo that emits the
// same members in sorted key order. Values inside the canonical maps are the
// public types themselves, so recursion needs no shadow copies and the wire
// shape stays defined by the public structs' tags.

// canonPair is one sorted-key entry of a canonical object.
type canonPair[V any] struct {
	key string
	val V
}

// canonMap is a string-keyed map serialized in sorted key order.
type canonMap[V any] []canonPair[V]

func (m canonMap[V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err //nolint:wrapcheck // passthrough encoder error; openapi is dependency-free
	}

	for _, p := range m {
		if err := enc.WriteToken(jsontext.String(p.key)); err != nil {
			return err //nolint:wrapcheck
		}

		if err := json.MarshalEncode(enc, p.val); err != nil {
			return err //nolint:wrapcheck
		}
	}

	return enc.WriteToken(jsontext.EndObject) //nolint:wrapcheck
}

// canonIntPair is one sorted entry of a canonical int-keyed object.
type canonIntPair[V any] struct {
	key int
	val V
}

// canonIntMap is an int-keyed map (e.g. Responses by status code) serialized
// in ascending numeric key order, keys rendered as JSON strings ("200").
type canonIntMap[V any] []canonIntPair[V]

func (m canonIntMap[V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err //nolint:wrapcheck
	}

	for _, p := range m {
		if err := enc.WriteToken(jsontext.String(strconv.Itoa(p.key))); err != nil {
			return err //nolint:wrapcheck
		}

		if err := json.MarshalEncode(enc, p.val); err != nil {
			return err //nolint:wrapcheck
		}
	}

	return enc.WriteToken(jsontext.EndObject) //nolint:wrapcheck
}

// canonStrings converts a string-keyed map into sorted canonMap form.
func canonStrings[V any](m map[string]V) canonMap[V] {
	if m == nil {
		return nil
	}

	out := make(canonMap[V], 0, len(m))
	for k, v := range m {
		out = append(out, canonPair[V]{key: k, val: v})
	}

	slices.SortFunc(out, func(a, b canonPair[V]) int { return compareStrings(a.key, b.key) })

	return out
}

// canonInts converts an int-keyed map into ascending canonIntMap form.
func canonInts[V any](m map[int]V) canonIntMap[V] {
	if m == nil {
		return nil
	}

	out := make(canonIntMap[V], 0, len(m))
	for k, v := range m {
		out = append(out, canonIntPair[V]{key: k, val: v})
	}

	slices.SortFunc(out, func(a, b canonIntPair[V]) int { return a.key - b.key })

	return out
}

func compareStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// specShadow mirrors Spec's field set with canonical maps. The parity test
// (TestCanonical_ParityWithPlainMarshal) fails if the two shapes drift.
type specShadow struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      canonMap[*PathItem] `json:"paths"`
	Components *Components         `json:"components,omitempty"`
}

// MarshalJSONTo serializes the spec deterministically (sorted paths). See the
// package-level comment on deterministic serialization.
func (s *Spec) MarshalJSONTo(enc *jsontext.Encoder) error {
	shadow := specShadow{
		OpenAPI: s.OpenAPI,
		Info:    s.Info,
		Paths:   canonStrings(s.Paths),
	}

	if s.Components != nil {
		// Components carries its own MarshalJSONTo (sorted schemas).
		shadow.Components = s.Components
	}

	//nolint:wrapcheck // passthrough; openapi is dependency-free
	return json.MarshalEncode(enc, shadow)
}

type componentsShadow struct {
	Schemas canonMap[*Schema] `json:"schemas,omitempty"`
}

// MarshalJSONTo serializes components with schemas in sorted name order.
func (c *Components) MarshalJSONTo(enc *jsontext.Encoder) error {
	//nolint:wrapcheck
	return json.MarshalEncode(enc, componentsShadow{Schemas: canonStrings(c.Schemas)})
}

type operationShadow struct {
	Tags        []string               `json:"tags,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	OperationID string                 `json:"operationId,omitempty"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
	RequestBody *RequestBody           `json:"requestBody,omitempty"`
	Responses   canonIntMap[*Response] `json:"responses"`
	Deprecated  omitBool               `json:"deprecated,omitempty"`
	HTTPMethod  string                 `json:"-"`
}

// MarshalJSONTo serializes the operation with responses in ascending status
// order.
func (o *Operation) MarshalJSONTo(enc *jsontext.Encoder) error {
	shadow := operationShadow{
		Tags:        o.Tags,
		Summary:     o.Summary,
		Description: o.Description,
		OperationID: o.OperationID,
		Parameters:  o.Parameters,
		RequestBody: o.RequestBody,
		Responses:   canonInts(o.Responses),
		Deprecated:  o.Deprecated,
		HTTPMethod:  o.HTTPMethod,
	}

	//nolint:wrapcheck
	return json.MarshalEncode(enc, shadow)
}

type requestBodyShadow struct {
	Description string               `json:"description,omitempty"`
	Required    omitBool             `json:"required,omitempty"`
	Content     canonMap[*MediaType] `json:"content"`
}

// MarshalJSONTo serializes the request body with content in sorted media-type
// order.
func (rb *RequestBody) MarshalJSONTo(enc *jsontext.Encoder) error {
	shadow := requestBodyShadow{
		Description: rb.Description,
		Required:    rb.Required,
		Content:     canonStrings(rb.Content),
	}

	//nolint:wrapcheck
	return json.MarshalEncode(enc, shadow)
}

type responseShadow struct {
	Description string               `json:"description"`
	Headers     canonMap[*Header]    `json:"headers,omitempty"`
	Content     canonMap[*MediaType] `json:"content,omitempty"`
}

// MarshalJSONTo serializes the response with headers and content in sorted
// key order.
func (r *Response) MarshalJSONTo(enc *jsontext.Encoder) error {
	shadow := responseShadow{
		Description: r.Description,
		Headers:     canonStrings(r.Headers),
		Content:     canonStrings(r.Content),
	}

	//nolint:wrapcheck
	return json.MarshalEncode(enc, shadow)
}

type schemaShadow struct {
	Type                 string            `json:"type,omitempty"`
	Format               string            `json:"format,omitempty"`
	Description          string            `json:"description,omitempty"`
	Properties           canonMap[*Schema] `json:"properties,omitempty"`
	Required             []string          `json:"required,omitempty"`
	Items                *Schema           `json:"items,omitempty"`
	MinItems             *uint64           `json:"minItems,omitempty"`
	MaxItems             *uint64           `json:"maxItems,omitempty"`
	Enum                 []any             `json:"enum,omitempty"`
	MinLength            *int              `json:"minLength,omitempty"`
	MaxLength            *int              `json:"maxLength,omitempty"`
	Minimum              *float64          `json:"minimum,omitempty"`
	Maximum              *float64          `json:"maximum,omitempty"`
	AdditionalProperties any               `json:"additionalProperties,omitempty"`
	Ref                  string            `json:"$ref,omitempty"`
}

// MarshalJSONTo serializes the schema with properties in sorted name order
// (recursively — each property is a *Schema that serializes the same way).
func (s *Schema) MarshalJSONTo(enc *jsontext.Encoder) error {
	shadow := schemaShadow{
		Type:                 s.Type,
		Format:               s.Format,
		Description:          s.Description,
		Properties:           canonStrings(s.Properties),
		Required:             s.Required,
		Items:                s.Items,
		MinItems:             s.MinItems,
		MaxItems:             s.MaxItems,
		Enum:                 s.Enum,
		MinLength:            s.MinLength,
		MaxLength:            s.MaxLength,
		Minimum:              s.Minimum,
		Maximum:              s.Maximum,
		AdditionalProperties: s.AdditionalProperties,
		Ref:                  s.Ref,
	}

	//nolint:wrapcheck
	return json.MarshalEncode(enc, shadow)
}
