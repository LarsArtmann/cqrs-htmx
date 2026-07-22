package cqrshtmx

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Validatable is the constraint used by DecodeAndValidate* helpers.
// Request-body types that implement Validate() error can be decoded and
// validated in a single step, catching validation failures before the
// command/query mapper runs.
type Validatable interface {
	Validate() error
}

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

// DecodeJSONTyped decodes a JSON request body directly into a typed command
// value of type Q. The body shape must unmarshal into Q, which must satisfy
// command.Command. Use with App.CommandTyped[Q].
func DecodeJSONTyped[Q command.Command]() HandlerOption {
	return DecodeJSON(func(q Q) (command.Command, error) {
		return q, nil
	})
}

// DecodeFormTyped decodes form data directly into a typed command value of type
// Q. Use with App.CommandTyped[Q].
func DecodeFormTyped[Q command.Command]() HandlerOption {
	return DecodeForm(func(q Q) (command.Command, error) {
		return q, nil
	})
}

// DecodeJSONQueryTyped decodes a JSON request body directly into a typed query
// value of type Q, which must satisfy query.Query. Use with App.QueryTyped[Q, R].
func DecodeJSONQueryTyped[Q query.Query]() HandlerOption {
	return DecodeJSONQuery(func(q Q) (query.Query, error) {
		return q, nil
	})
}

// DecodeFormQueryTyped decodes form data directly into a typed query value of
// type Q. Use with App.QueryTyped[Q, R].
func DecodeFormQueryTyped[Q query.Query]() HandlerOption {
	return DecodeFormQuery(func(q Q) (query.Query, error) {
		return q, nil
	})
}

// DecodeAndValidateJSON decodes a JSON request body into T, calls T.Validate(),
// then maps the validated body to a command. A validation failure returns the
// error from Validate(), which is surfaced as a 400 Bad Request by the default
// error handler. An empty body is treated as a zero-value T.
//
// Compile-time guarantee: T must implement Validate() error.
func DecodeAndValidateJSON[T Validatable](mapper func(T) (command.Command, error)) HandlerOption {
	return DecodeJSON(func(t T) (command.Command, error) {
		if err := t.Validate(); err != nil {
			return nil, errorfamily.Wrapf(
			err,
			event.Rejection,
			"cqrshtmx.decode.validation_failed",
			"validation failed",
		)
		}

		return mapper(t)
	})
}

// DecodeAndValidateJSONQuery decodes a JSON request body into T, calls
// T.Validate(), then maps the validated body to a query.
func DecodeAndValidateJSONQuery[T Validatable](mapper func(T) (query.Query, error)) HandlerOption {
	return DecodeJSONQuery(func(t T) (query.Query, error) {
		if err := t.Validate(); err != nil {
			return nil, errorfamily.Wrapf(
			err,
			event.Rejection,
			"cqrshtmx.decode.validation_failed",
			"validation failed",
		)
		}

		return mapper(t)
	})
}

// DecodeAndValidateForm decodes form data into T, calls T.Validate(), then maps
// the validated body to a command.
func DecodeAndValidateForm[T Validatable](mapper func(T) (command.Command, error)) HandlerOption {
	return DecodeForm(func(t T) (command.Command, error) {
		if err := t.Validate(); err != nil {
			return nil, errorfamily.Wrapf(
			err,
			event.Rejection,
			"cqrshtmx.decode.validation_failed",
			"validation failed",
		)
		}

		return mapper(t)
	})
}

// DecodeAndValidateFormQuery decodes form data into T, calls T.Validate(), then
// maps the validated body to a query.
func DecodeAndValidateFormQuery[T Validatable](mapper func(T) (query.Query, error)) HandlerOption {
	return DecodeFormQuery(func(t T) (query.Query, error) {
		if err := t.Validate(); err != nil {
			return nil, errorfamily.Wrapf(
			err,
			event.Rejection,
			"cqrshtmx.decode.validation_failed",
			"validation failed",
		)
		}

		return mapper(t)
	})
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
