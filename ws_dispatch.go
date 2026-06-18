package cqrshtmx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
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
	return func(data []byte) (command.Command, error) {
		var t T
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("decode ws json: %w", err)
		}
		return mapper(t)
	}
}

// DecodeWSJSONQuery creates a WSQueryDecoder that unmarshals JSON into T,
// then maps T to a query.Query via the mapper function.
func DecodeWSJSONQuery[T any](mapper func(T) (query.Query, error)) WSQueryDecoder {
	return func(data []byte) (query.Query, error) {
		var t T
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, fmt.Errorf("decode ws json query: %w", err)
		}
		return mapper(t)
	}
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

	ctx := wsContext(r)

	if a.beforeDispatch != nil && r != nil {
		ctx = a.beforeDispatch(ctx, r)
	}

	cmd, err := decoder(data)
	if err != nil {
		wrappedErr := fmt.Errorf("%w: %s: %w", ErrDecodeFailed, cmdType, err)
		a.afterDispatchHook(ctx, r, wrappedErr)
		return wrappedErr
	}

	if cmd == nil {
		a.afterDispatchHook(ctx, r, errDecoderMissing)
		return errDecoderMissing
	}

	ctx, cancel := a.timeoutCtx(ctx, nil)
	defer cancel()

	if err = a.commands.Dispatch(ctx, cmd); err != nil {
		wrappedErr := fmt.Errorf("%w: %s: %w", ErrDispatchFailed, cmdType, err)
		a.afterDispatchHook(ctx, r, wrappedErr)
		return wrappedErr
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

	ctx := wsContext(r)

	if a.beforeDispatch != nil && r != nil {
		ctx = a.beforeDispatch(ctx, r)
	}

	qry, err := decoder(data)
	if err != nil {
		wrappedErr := fmt.Errorf("%w: %s: %w", ErrDecodeFailed, qryType, err)
		a.afterDispatchHook(ctx, r, wrappedErr)
		return nil, wrappedErr
	}

	if qry == nil {
		a.afterDispatchHook(ctx, r, errDecoderMissing)
		return nil, errDecoderMissing
	}

	ctx, cancel := a.timeoutCtx(ctx, nil)
	defer cancel()

	result, err := a.queries.Dispatch(ctx, qry)
	if err != nil {
		wrappedErr := fmt.Errorf("%w: %s: %w", ErrDispatchFailed, qryType, err)
		a.afterDispatchHook(ctx, r, wrappedErr)
		return nil, wrappedErr
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
