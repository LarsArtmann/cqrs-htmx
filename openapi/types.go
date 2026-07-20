package openapi

// omitBool is a bool that serializes to `null` when false, so that `omitempty`
// (which omits null in encoding/json/v2) drops it entirely. OpenAPI consumers
// treat absent booleans as false, so this produces clean specs without leaking
// redundant `"required": false` / `"deprecated": false` on every field.
type omitBool bool

// MarshalJSON renders true as `true` and false as `null` (omitted by omitempty).
func (b omitBool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}

	return []byte("null"), nil
}

// Spec is the root OpenAPI 3.1 document. Construct one with New, then add
// paths and (optionally) reusable component schemas. Serialize with JSON.
type Spec struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

// Info describes the API metadata.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// Components holds reusable schemas referenced via $ref.
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty"`
}

// PathItem describes the operations available on a single path.
type PathItem struct {
	Summary     string      `json:"summary,omitempty"`
	Description string      `json:"description,omitempty"`
	Get         *Operation  `json:"get,omitempty"`
	Post        *Operation  `json:"post,omitempty"`
	Put         *Operation  `json:"put,omitempty"`
	Patch       *Operation  `json:"patch,omitempty"`
	Delete      *Operation  `json:"delete,omitempty"`
	Head        *Operation  `json:"head,omitempty"`
	Options     *Operation  `json:"options,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
}

// Operation describes a single HTTP operation on a path.
type Operation struct {
	Tags        []string           `json:"tags,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	Description string             `json:"description,omitempty"`
	OperationID string             `json:"operationId,omitempty"`
	Parameters  []Parameter        `json:"parameters,omitempty"`
	RequestBody *RequestBody       `json:"requestBody,omitempty"`
	Responses   map[int]*Response  `json:"responses"`
	Deprecated  omitBool           `json:"deprecated,omitempty"`
}

// Parameter is a path, query, header, or cookie parameter.
type Parameter struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Description string   `json:"description,omitempty"`
	Required    omitBool `json:"required,omitempty"`
	Schema      *Schema  `json:"schema,omitempty"`
}

// RequestBody describes the expected request body.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    omitBool             `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// Response describes a single HTTP response by status code.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

// MediaType pairs a content type with a schema (e.g. application/json).
type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

// jsonMediaType is the standard JSON content type.
const jsonMediaType = "application/json"

// JSON wraps a schema in an application/json MediaType, for use as a response
// or request body content entry.
func JSON(schema *Schema) *MediaType {
	return &MediaType{Schema: schema}
}
