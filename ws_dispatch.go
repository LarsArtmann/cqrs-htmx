package cqrshtmx

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

// WSCommandDecoder decodes raw WebSocket message bytes into a command.
// Use this with [App.DispatchWSCommand] to bridge WS messages to CQRS dispatch.
//
// Create one with [DecodeWSJSON] or write your own if you need custom parsing
// (e.g., msgpack, protobuf, or ParseWSMessageInto[T] for HTMX WS format):
//
//	decoder := func(data []byte) (command.Command, error) {
//	    msg, _, err := cqrshtmx.ParseWSMessageInto[CreateTaskInput](data)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return command.New("CreateTask", msg)
//	}
type WSCommandDecoder func(data []byte) (command.Command, error)

// WSQueryDecoder decodes raw WebSocket message bytes into a query.
// Use this with [App.DispatchWSQuery] to bridge WS messages to CQRS dispatch.
type WSQueryDecoder func(data []byte) (query.Query, error)

// DecodeWSJSON creates a WSCommandDecoder that unmarshals JSON into T,
// then maps T to a command.Command via the mapper function.
//
//	type CreateTaskInput struct {
//	    Title string `json:"title"`
//	}
//
//	decoder := cqrshtmx.DecodeWSJSON(func(t CreateTaskInput) (command.Command, error) {
//	    return command.New("CreateTask", t)
//	})
func DecodeWSJSON[T any](mapper func(T) (command.Command, error)) WSCommandDecoder {
	return makeWSDecoder("decode ws json", mapper)
}

// DecodeWSJSONQuery creates a WSQueryDecoder that unmarshals JSON into T,
// then maps T to a query.Query via the mapper function.
func DecodeWSJSONQuery[T any](mapper func(T) (query.Query, error)) WSQueryDecoder {
	return makeWSDecoder("decode ws json query", mapper)
}

// makeWSDecoder builds a JSON-decoding wrapper for both WSCommandDecoder and
// WSQueryDecoder. It unmarshals the raw bytes into T, then maps T to the
// target CQRS type (command or query) via mapper.
func makeWSDecoder[T, C any](errPrefix string, mapper func(T) (C, error)) func([]byte) (C, error) {
	return func(data []byte) (C, error) {
		var t T
		if err := json.Unmarshal(data, &t); err != nil {
			return zero[C](), event.Wrapf(err, event.Rejection, "cqrshtmx.ws.decode_failed", "%s", errPrefix)
		}
		return mapper(t)
	}
}

func zero[C any]() C {
	var c C
	return c
}

// DispatchWSCommand decodes a WebSocket message into a command and dispatches it.
//
// It is the WebSocket counterpart to [App.Command]: it decodes the raw message
// bytes, runs lifecycle hooks (BeforeDispatch/AfterDispatch), applies the App's
// timeout, and dispatches the command. It returns any error encountered.
//
// Unlike the HTTP path, DispatchWSCommand does NOT handle authorization, CSRF,
// or response writing — WebSocket connections are authenticated at upgrade time
// and responses are written by the consumer's WS library.
//
// The request r is the original HTTP upgrade request; it carries context
// (user ID, request ID) used by lifecycle hooks and event enrichment.
// Pass nil only if you truly have no request context.
//
// Example (gorilla/websocket):
//
//	for {
//	    _, data, err := conn.ReadMessage()
//	    if err != nil { return }
//	    if err := app.DispatchWSCommand(r, "CreateTask", decoder, data); err != nil {
//	        payload := cqrshtmx.NewStructuredError(err, r)
//	        _ = conn.WriteMessage(websocket.TextMessage, []byte(payload.JSON()))
//	        continue
//	    }
//	    wsBroadcaster.Broadcast(cqrshtmx.WSOOBHTML("tasks", renderTasks()))
//	}
func (a *App) DispatchWSCommand(
	r *http.Request,
	cmdType command.Type,
	decoder WSCommandDecoder,
	data []byte,
) error {
	if cmdType.IsZero() {
		panic("cqrs-htmx: command type must not be empty")
	}

	if a.commands == nil {
		return errCommandsNil
	}

	ctx := a.wsCallContext(r)

	cmd, err := decodeWSMessage(a, ctx, r, decoder, data,
		"cqrshtmx.ws.decode_command_failed", "decode command %s", cmdType)
	if err != nil {
		return err
	}

	ctx, cancel := a.timeoutCtx(ctx, nil)
	defer cancel()

	if dispatchErr := a.commands.Dispatch(ctx, cmd); dispatchErr != nil {
		return a.wrapWSDispatchErr(ctx, r, dispatchErr,
			"cqrshtmx.ws.dispatch_command_failed", "dispatch command %s", cmdType)
	}

	a.afterDispatchHook(ctx, r, nil)
	return nil
}

// DispatchWSQuery decodes a WebSocket message into a query, dispatches it,
// and returns the query result. It is the WebSocket counterpart to [App.Query].
//
// Like [DispatchWSCommand], it does NOT handle authorization, CSRF, or response
// writing. The caller serializes the result (or StructuredError on failure)
// back to the WS connection.
//
// Example:
//
//	result, err := app.DispatchWSQuery(r, "GetTasks", queryDecoder, data)
//	if err != nil {
//	    payload := cqrshtmx.NewStructuredError(err, r)
//	    _ = conn.WriteMessage(websocket.TextMessage, []byte(payload.JSON()))
//	    return
//	}
//	jsonBytes, _ := json.Marshal(result)
//	_ = conn.WriteMessage(websocket.TextMessage, jsonBytes)
func (a *App) DispatchWSQuery(
	r *http.Request,
	qryType query.Type,
	decoder WSQueryDecoder,
	data []byte,
) (any, error) {
	if qryType.IsZero() {
		panic("cqrs-htmx: query type must not be empty")
	}

	if a.queries == nil {
		return nil, errQueriesNil
	}

	ctx := a.wsCallContext(r)

	qry, err := decodeWSMessage(a, ctx, r, decoder, data,
		"cqrshtmx.ws.decode_query_failed", "decode query %s", qryType)
	if err != nil {
		return nil, err
	}

	ctx, cancel := a.timeoutCtx(ctx, nil)
	defer cancel()

	result, dispatchErr := a.queries.Dispatch(ctx, qry)
	if dispatchErr != nil {
		return nil, a.wrapWSDispatchErr(ctx, r, dispatchErr,
			"cqrshtmx.ws.dispatch_query_failed", "dispatch query %s", qryType)
	}

	a.afterDispatchHook(ctx, r, nil)
	return result, nil
}

// wsContext returns the request context if r is non-nil, otherwise context.Background.
func wsContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

// wsCallContext returns the request context enriched by the BeforeDispatch hook,
// or context.Background() when no request is supplied. Shared by every
// WebSocket dispatch entry point.
func (a *App) wsCallContext(r *http.Request) context.Context {
	ctx := wsContext(r)
	if a.beforeDispatch != nil && r != nil {
		ctx = a.beforeDispatch(ctx, r)
	}
	return ctx
}

// decodeWSMessage runs the shared decode → wrap → nil-check pipeline used by
// DispatchWSCommand and DispatchWSQuery. On any failure the AfterDispatch hook
// fires and the returned error is non-nil (so the caller can `return err`).
//
// code and msgFormat are forwarded to event.Wrapf so the caller can produce a
// domain-specific error code (e.g. cqrshtmx.ws.decode_command_failed).
func decodeWSMessage[T any](
	a *App,
	ctx context.Context,
	r *http.Request,
	decoder func([]byte) (T, error),
	data []byte,
	code, msgFormat string,
	msgArgs ...any,
) (T, error) {
	var zero T
	v, err := decoder(data)
	if err != nil {
		wrapped := event.Wrapf(err, event.Rejection, code, msgFormat, msgArgs...)
		a.afterDispatchHook(ctx, r, wrapped)
		return zero, wrapped
	}
	if isWSValueNil(v) {
		a.afterDispatchHook(ctx, r, errDecoderMissing)
		return zero, errDecoderMissing
	}
	return v, nil
}

// isWSValueNil reports whether a freshly decoded CQRS message is the zero
// value for its type. Generics can't compare arbitrary types with `==` (some
// are incomparable), so we route through reflect: nil for reference kinds,
// otherwise false (we trust the decoder to produce a meaningful value).
func isWSValueNil(v any) bool {
	if v == nil {
		return true
	}
	k := reflect.ValueOf(v).Kind()
	switch k { //nolint:exhaustive // only reference kinds can be nil
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return reflect.ValueOf(v).IsNil()
	}
	return false
}

// wrapWSDispatchErr wraps a dispatch error preserving its error family (so a
// domain Rejection isn't forced into a Transient), fires the AfterDispatch
// hook, and returns the wrapped error.
func (a *App) wrapWSDispatchErr(
	ctx context.Context, r *http.Request, err error,
	code, msgFormat string, msgArgs ...any,
) error {
	wrapped := event.Wrapf(err, event.Classify(err), code, msgFormat, msgArgs...)
	a.afterDispatchHook(ctx, r, wrapped)
	return wrapped
}
