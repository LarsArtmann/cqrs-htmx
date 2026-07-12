package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func setCommandDecoder(cfg *handlerConfig, dec func(*http.Request) (command.Command, error)) {
	cfg.commandDecoder = dec
}

func setQueryDecoder(cfg *handlerConfig, dec func(*http.Request) (query.Query, error)) {
	cfg.queryDecoder = dec
}

// DecodeJSON decodes a JSON request body into a command using the mapper.
// An empty body (e.g. a POST with no body, or Content-Length: 0) is treated as
// a zero-value T, not an error. This means GET-style queries or no-body POSTs
// work without sending "{}".
func DecodeJSON[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return decodeAndSet(decodeJSONBody[T], mapper, setCommandDecoder)
}

// DecodeJSONQuery decodes a JSON request body into a query.
// An empty body (e.g. a GET request with no body) is treated as a zero-value T,
// not an error. This is useful for queries that take no parameters or rely
// entirely on context (path values, headers, query params).
func DecodeJSONQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return decodeAndSet(decodeJSONBody[T], mapper, setQueryDecoder)
}

// DecodeForm decodes form data into a command using the mapper.
func DecodeForm[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return decodeAndSet(decodeFormBody[T], mapper, setCommandDecoder)
}

// DecodeFormQuery decodes form data into a query using the mapper.
func DecodeFormQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return decodeAndSet(decodeFormBody[T], mapper, setQueryDecoder)
}

// DecodeJSONWithRequest decodes a JSON request body into a command, giving the
// mapper access to the *http.Request. Use this when the mapper needs request-scoped
// data (cookies, headers, path values) — e.g., custom cookie-based auth:
//
//	app.Command("CreateItem",
//	    cqrshtmx.DecodeJSONWithRequest(func(r *http.Request, body createItemReq) (command.Command, error) {
//	        playerID := extractPlayerID(r) // read cookie/header
//	        return &createItemCmd{PlayerID: playerID, Name: body.Name}, nil
//	    }),
//	)
func DecodeJSONWithRequest[T any](mapper func(r *http.Request, body T) (command.Command, error)) HandlerOption {
	return decodeAndSetWithRequest(decodeJSONBody[T], mapper, setCommandDecoder)
}

// DecodeFormWithRequest decodes form data into a command, giving the mapper
// access to the *http.Request. Same as DecodeJSONWithRequest but for form bodies.
func DecodeFormWithRequest[T any](mapper func(r *http.Request, body T) (command.Command, error)) HandlerOption {
	return decodeAndSetWithRequest(decodeFormBody[T], mapper, setCommandDecoder)
}

// DecodeJSONQueryWithRequest decodes a JSON request body into a query, giving the
// mapper access to the *http.Request.
func DecodeJSONQueryWithRequest[T any](mapper func(r *http.Request, body T) (query.Query, error)) HandlerOption {
	return decodeAndSetWithRequest(decodeJSONBody[T], mapper, setQueryDecoder)
}

// DecodeFormQueryWithRequest decodes form data into a query, giving the mapper
// access to the *http.Request.
func DecodeFormQueryWithRequest[T any](mapper func(r *http.Request, body T) (query.Query, error)) HandlerOption {
	return decodeAndSetWithRequest(decodeFormBody[T], mapper, setQueryDecoder)
}
