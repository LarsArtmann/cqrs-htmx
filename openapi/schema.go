package openapi

// Schema is a subset of JSON Schema (draft 2020-12) sufficient for OpenAPI 3.1
// request bodies, parameters, and responses. All fields are optional and
// omitted from the serialized output when zero.
//
// Construct schemas with the helper constructors (String, Object, Array, etc.)
// rather than struct literals — the helpers set the Type field consistently
// and are far more readable.
type Schema struct {
	// Type is the JSON Schema type: "string", "integer", "number", "boolean",
	// "object", "array", or "null".
	Type string `json:"type,omitempty"`

	// Format restricts a type further: "int64", "uuid", "date-time", "email",
	// "byte" (base64), "uri", etc. OpenAPI defines a standard set; consumers
	// may use any string.
	Format string `json:"format,omitempty"`

	// Description is a human-readable description.
	Description string `json:"description,omitempty"`

	// Properties maps object property names to their schemas. Only meaningful
	// when Type == "object".
	Properties map[string]*Schema `json:"properties,omitempty"`

	// Required lists property names that must be present. Only meaningful when
	// Type == "object".
	Required []string `json:"required,omitempty"`

	// Items is the schema for array elements. Only meaningful when
	// Type == "array".
	Items *Schema `json:"items,omitempty"`

	// Enum constrains the value to one of the listed constants (any JSON type).
	Enum []any `json:"enum,omitempty"`

	// MinLength is the minimum string length (string type only).
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength is the maximum string length (string type only).
	MaxLength *int `json:"maxLength,omitempty"`

	// Minimum is the inclusive numeric minimum (number/integer type only).
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum is the inclusive numeric maximum (number/integer type only).
	Maximum *float64 `json:"maximum,omitempty"`

	// AdditionalProperties controls unmapped object keys. Pass a *Schema to
	// constrain their type, or openapi.False() to forbid them entirely.
	AdditionalProperties any `json:"additionalProperties,omitempty"`

	// Ref is a JSON reference ($ref) into the components/schemas map, e.g.
	// "#/components/schemas/Item". Use Ref(name) to construct one. When set,
	// all other fields are ignored on serialization (per JSON Reference).
	Ref string `json:"$ref,omitempty"`
}

// String returns a schema with type "string".
func String() *Schema { return &Schema{Type: "string"} }

// Integer returns a schema with type "integer".
func Integer() *Schema { return &Schema{Type: "integer"} }

// Number returns a schema with type "number".
func Number() *Schema { return &Schema{Type: "number"} }

// Boolean returns a schema with type "boolean".
func Boolean() *Schema { return &Schema{Type: "boolean"} }

// Array returns a schema with type "array" and the given item schema.
func Array(items *Schema) *Schema { return &Schema{Type: "array", Items: items} }

// Object returns a schema with type "object" and the given properties.
// Use Prop to build each property.
func Object(properties ...Property) *Schema {
	schema := &Schema{Type: "object", Properties: make(map[string]*Schema, len(properties))}

	for _, p := range properties {
		schema.Properties[p.Name] = p.Schema

		if p.Required {
			schema.Required = append(schema.Required, p.Name)
		}
	}

	return schema
}

// FreeForm returns an object schema that allows any properties (empty object).
// Useful for "any JSON" payloads.
func FreeForm() *Schema {
	return &Schema{Type: "object", AdditionalProperties: map[string]any{}}
}

// Property is a name+schema pair, optionally required, used by Object.
type Property struct {
	Name     string
	Schema   *Schema
	Required bool
}

// Prop builds an optional object property.
func Prop(name string, schema *Schema) Property {
	return Property{Name: name, Schema: schema}
}

// PropReq builds a required object property.
func PropReq(name string, schema *Schema) Property {
	return Property{Name: name, Schema: schema, Required: true}
}

// Ref returns a schema that references a named schema in components/schemas.
// The argument is the schema name (NOT the full "#/components/schemas/..." path);
// the path prefix is added automatically.
func Ref(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

// falseSchema is a sentinel that serializes to the JSON literal `false`, used
// for AdditionalProperties: false. It implements json.Marshaler.
type falseSchema struct{}

func (falseSchema) MarshalJSON() ([]byte, error) { return []byte("false"), nil }

// False returns a sentinel usable as AdditionalProperties to forbid any
// unmapped object property.
func False() any { return falseSchema{} }

// ErrorSchema returns a standard RFC 7807-style problem-details object schema,
// convenient for error responses.
func ErrorSchema() *Schema {
	return Object(
		PropReq("status", Integer()),
		PropReq("title", String()),
		Prop("detail", String()),
		Prop("instance", String()),
		Prop("code", String()),
	)
}

// --- fluent Schema mutators ---

// WithDescription sets the description and returns the schema.
func (s *Schema) WithDescription(desc string) *Schema {
	s.Description = desc

	return s
}

// WithFormat sets the format and returns the schema.
func (s *Schema) WithFormat(format string) *Schema {
	s.Format = format

	return s
}

// WithMinLength sets minLength (string type) and returns the schema.
func (s *Schema) WithMinLength(n int) *Schema {
	s.MinLength = &n

	return s
}

// WithMaxLength sets maxLength (string type) and returns the schema.
func (s *Schema) WithMaxLength(n int) *Schema {
	s.MaxLength = &n

	return s
}

// WithMin sets the inclusive numeric minimum and returns the schema.
func (s *Schema) WithMin(n float64) *Schema {
	s.Minimum = &n

	return s
}

// WithMax sets the inclusive numeric maximum and returns the schema.
func (s *Schema) WithMax(n float64) *Schema {
	s.Maximum = &n

	return s
}

// WithEnum constrains the value to the given constants and returns the schema.
func (s *Schema) WithEnum(values ...any) *Schema {
	s.Enum = values

	return s
}
