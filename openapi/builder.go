package openapi

// New creates a Spec rooted at OpenAPI 3.1.0 with the given title and version,
// and an empty paths map ready to receive operations.
func New(title, version string) *Spec {
	return &Spec{
		OpenAPI: "3.1.0",
		Info:    Info{Title: title, Version: version},
		Paths:   map[string]*PathItem{},
	}
}

// WithDescription sets the API description and returns the spec.
func (s *Spec) WithDescription(desc string) *Spec {
	s.Info.Description = desc

	return s
}

// Schema registers a reusable component schema (referenced via Ref(name)) and
// returns the spec. This is the standard way to share a schema across multiple
// operations without inlining it everywhere.
func (s *Spec) Schema(name string, schema *Schema) *Spec {
	if s.Components == nil {
		s.Components = &Components{Schemas: map[string]*Schema{}}
	}

	s.Components.Schemas[name] = schema

	return s
}

// Path registers operations on a path and returns the spec. Each operation
// (Get, Post, ...) is constructed with the Get/Post/... constructors. If the
// path already exists, operations are merged into the existing PathItem.
func (s *Spec) Path(path string, operations ...*opBuilder) *Spec {
	item := s.Paths[path]

	if item == nil {
		item = &PathItem{}
		s.Paths[path] = item
	}

	for _, op := range operations {
		op.apply(item)
	}

	return s
}

// --- operation constructors ---

// opBuilder carries an HTTP method + Operation and applies it to a PathItem.
// Instances are created by Get, Post, Put, Patch, Delete, Head, Options.
type opBuilder struct {
	method string
	op     *Operation
}

func (b *opBuilder) apply(item *PathItem) {
	switch b.method {
	case "get":
		item.Get = b.op
	case "post":
		item.Post = b.op
	case "put":
		item.Put = b.op
	case "patch":
		item.Patch = b.op
	case "delete":
		item.Delete = b.op
	case "head":
		item.Head = b.op
	case "options":
		item.Options = b.op
	}
}

// Get starts a GET operation with the given operationId.
func Get(operationID string) *opBuilder { return newOp("get", operationID) }

// Post starts a POST operation with the given operationId.
func Post(operationID string) *opBuilder { return newOp("post", operationID) }

// Put starts a PUT operation with the given operationId.
func Put(operationID string) *opBuilder { return newOp("put", operationID) }

// Patch starts a PATCH operation with the given operationId.
func Patch(operationID string) *opBuilder { return newOp("patch", operationID) }

// Delete starts a DELETE operation with the given operationId.
func Delete(operationID string) *opBuilder { return newOp("delete", operationID) }

func newOp(method, operationID string) *opBuilder {
	return &opBuilder{
		method: method,
		op:     &Operation{OperationID: operationID, Responses: map[int]*Response{}},
	}
}

// --- Operation fluent mutators (return *opBuilder for chaining) ---

// Summary sets the operation summary.
func (b *opBuilder) Summary(text string) *opBuilder {
	b.op.Summary = text

	return b
}

// Desc sets the operation description.
func (b *opBuilder) Desc(text string) *opBuilder {
	b.op.Description = text

	return b
}

// Description is an alias for Desc for naming consistency with the field.
func (b *opBuilder) Description(text string) *opBuilder {
	b.op.Description = text

	return b
}

// Tag adds a tag to the operation.
func (b *opBuilder) Tag(tag string) *opBuilder {
	b.op.Tags = append(b.op.Tags, tag)

	return b
}

// Deprecated marks the operation as deprecated.
func (b *opBuilder) Deprecated() *opBuilder {
	b.op.Deprecated = true

	return b
}

// PathParam adds a required path parameter (in: path) with the given schema
// and description.
func (b *opBuilder) PathParam(name string, schema *Schema, description string) *opBuilder {
	return b.param(name, "path", schema, description, true)
}

// QueryParam adds an optional query parameter (in: query) with the given
// schema and description.
func (b *opBuilder) QueryParam(name string, schema *Schema, description string) *opBuilder {
	return b.param(name, "query", schema, description, false)
}

// QueryParamReq adds a required query parameter.
func (b *opBuilder) QueryParamReq(name string, schema *Schema, description string) *opBuilder {
	return b.param(name, "query", schema, description, true)
}

// HeaderParam adds a header parameter.
func (b *opBuilder) HeaderParam(name string, schema *Schema, description string) *opBuilder {
	return b.param(name, "header", schema, description, false)
}

func (b *opBuilder) param(name, in string, schema *Schema, description string, required bool) *opBuilder {
	b.op.Parameters = append(b.op.Parameters, Parameter{
		Name: name, In: in, Schema: schema, Description: description, Required: omitBool(required),
	})

	return b
}

// JSONBody sets a required application/json request body.
func (b *opBuilder) JSONBody(schema *Schema) *opBuilder {
	b.op.RequestBody = &RequestBody{
		Required: true,
		Content:  map[string]*MediaType{jsonMediaType: JSON(schema)},
	}

	return b
}

// JSONBodyOpt sets an optional application/json request body.
func (b *opBuilder) JSONBodyOpt(schema *Schema) *opBuilder {
	b.op.RequestBody = &RequestBody{
		Content: map[string]*MediaType{jsonMediaType: JSON(schema)},
	}

	return b
}

// Response adds a response for the given status code. The optional content
// MediaTypes (typically produced by JSON(schema)) attach response bodies.
func (b *opBuilder) Response(status int, description string, content ...*MediaType) *opBuilder {
	resp := &Response{Description: description}

	if len(content) > 0 {
		resp.Content = map[string]*MediaType{}

		for i, media := range content {
			// JSON is the conventional key; for the common single-JSON-body
			// case we use application/json explicitly.
			resp.Content[jsonMediaType] = media

			_ = i // placeholder: additional media types beyond JSON would key by their content type
		}
	}

	b.op.Responses[status] = resp

	return b
}

// NoContent adds a response for the given status code with no body (e.g. 204).
func (b *opBuilder) NoContent(status int, description string) *opBuilder {
	b.op.Responses[status] = &Response{Description: description}

	return b
}

// Op returns the built Operation, detached from the path-assembly builder. Use
// it to pass the operation to cqrshtmx.WithOpenAPI without registering a path:
//
//	cqrshtmx.WithOpenAPI(openapi.Post("CreateItem").Summary("...").Op())
//
// Callers cannot name the *opBuilder type (it is unexported), but they CAN
// call this method on the value returned by Get/Post/... and receive the
// exported Operation.
func (b *opBuilder) Op() Operation {
	return *b.op
}
