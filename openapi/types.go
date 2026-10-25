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

// hasMethod reports whether the PathItem already carries an operation for the
// lowercase HTTP method ("get", "post", ...). Used by the builder's
// double-registration guard.
func (p *PathItem) hasMethod(method string) bool {
	switch method {
	case "get":
		return p.Get != nil
	case "post":
		return p.Post != nil
	case "put":
		return p.Put != nil
	case "patch":
		return p.Patch != nil
	case "delete":
		return p.Delete != nil
	case "head":
		return p.Head != nil
	case "options":
		return p.Options != nil
	}

	return false
}

// operations iterates every non-nil operation with its lowercase method name.
// Used by Spec.Validate.
func (p *PathItem) operations() map[string]*Operation {
	return map[string]*Operation{
		"get":     p.Get,
		"post":    p.Post,
		"put":     p.Put,
		"patch":   p.Patch,
		"delete":  p.Delete,
		"head":    p.Head,
		"options": p.Options,
	}
}

// Operation describes a single HTTP operation on a path.
type Operation struct {
	Tags        []string          `json:"tags,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Description string            `json:"description,omitempty"`
	OperationID string            `json:"operationId,omitempty"`
	Parameters  []Parameter       `json:"parameters,omitempty"`
	RequestBody *RequestBody      `json:"requestBody,omitempty"`
	Responses   map[int]*Response `json:"responses"`
	Deprecated  omitBool          `json:"deprecated,omitempty"`

	// HTTPMethod is the lowercase HTTP method the builder was created with
	// ("get", "post", ...). It is set by Get/Post/Put/Patch/Delete/Head/Options
	// and excluded from serialization; a literally constructed Operation has
	// an empty HTTPMethod.
	HTTPMethod string `json:"-"`
}

// Clone returns a deep copy of the operation: every slice, map, and pointer
// reachable from it is duplicated, so mutating the copy cannot reach the
// original. Handlers that expose collected operations (e.g.
// cqrshtmx.App.OpenAPIRoutes) use this to keep their internal state isolated.
func (o Operation) Clone() Operation {
	c := o
	c.Tags = cloneStrings(o.Tags)
	c.Parameters = cloneParameters(o.Parameters)
	if o.RequestBody != nil {
		rb := *o.RequestBody
		rb.Content = cloneContent(o.RequestBody.Content)
		c.RequestBody = &rb
	}

	if o.Responses != nil {
		c.Responses = make(map[int]*Response, len(o.Responses))
		for code, resp := range o.Responses {
			if resp == nil {
				c.Responses[code] = nil

				continue
			}

			rc := *resp
			rc.Headers = cloneHeaders(resp.Headers)
			rc.Content = cloneContent(resp.Content)
			c.Responses[code] = &rc
		}
	}

	return c
}

func cloneStrings(in []string) []string {
	if in == nil {
		return nil
	}

	out := make([]string, len(in))
	copy(out, in)

	return out
}

func cloneParameters(in []Parameter) []Parameter {
	if in == nil {
		return nil
	}

	out := make([]Parameter, len(in))
	for i, p := range in {
		pc := p
		if p.Schema != nil {
			sc := p.Schema.Clone()
			pc.Schema = &sc
		}

		out[i] = pc
	}

	return out
}

func cloneHeaders(in map[string]*Header) map[string]*Header {
	if in == nil {
		return nil
	}

	out := make(map[string]*Header, len(in))
	for name, h := range in {
		if h == nil {
			out[name] = nil

			continue
		}

		hc := *h
		if h.Schema != nil {
			sc := h.Schema.Clone()
			hc.Schema = &sc
		}

		out[name] = &hc
	}

	return out
}

func cloneContent(in map[string]*MediaType) map[string]*MediaType {
	if in == nil {
		return nil
	}

	out := make(map[string]*MediaType, len(in))
	for ct, mt := range in {
		if mt == nil {
			out[ct] = nil

			continue
		}

		mc := *mt
		if mt.Schema != nil {
			sc := mt.Schema.Clone()
			mc.Schema = &sc
		}

		out[ct] = &mc
	}

	return out
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
	Description string                `json:"description,omitempty"`
	Required    omitBool              `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// Response describes a single HTTP response by status code.
type Response struct {
	Description string                `json:"description"`
	Headers     map[string]*Header    `json:"headers,omitempty"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

// Header describes a single response header (OpenAPI's header object: a
// description plus the header value's schema).
type Header struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
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

// eventStreamMediaType is the SSE content type.
const eventStreamMediaType = "text/event-stream"

// EventStream returns a text/event-stream MediaType — the body shape of SSE
// endpoints (an OpenAPI Schema cannot describe the framing, so the schema is
// nil and the description lives on the Response).
func EventStream() *MediaType {
	return &MediaType{}
}

// EventStreamContentType is the media-type key for SSE bodies, for use with
// [opBuilder.ResponseWithContent].
const EventStreamContentType = eventStreamMediaType

// HTML returns a text/html MediaType for HTMX partial responses.
func HTML(schema *Schema) *MediaType {
	return &MediaType{Schema: schema}
}

// HTMLContentType is the media-type key for HTML bodies, for use with
// [opBuilder.ResponseWithContent].
const HTMLContentType = "text/html"
