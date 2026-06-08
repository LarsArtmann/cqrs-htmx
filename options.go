package cqrshtmx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// CommandDecoder decodes an HTTP request into a command.Command.
type CommandDecoder func(r *http.Request) (command.Command, error)

// QueryDecoder decodes an HTTP request into a query.Query.
type QueryDecoder func(r *http.Request) (query.Query, error)

// RenderFunc renders a query result as an HTTP response.
type RenderFunc func(w http.ResponseWriter, r *http.Request, result any) error

// TemplComponent matches templ.Component's interface without importing templ.
// Any type with Render(ctx, w) error satisfies this, including templ-generated components.
type TemplComponent interface {
	Render(ctx context.Context, w io.Writer) error
}

// HandlerOption configures a command or query handler.
type HandlerOption func(*handlerConfig)

// authMode defines the authorization mode for a handler.
type authMode int

const (
	authNone authMode = iota
	authRequired
	authAuthorized
)

func (m authMode) String() string {
	switch m {
	case authNone:
		return "none"
	case authRequired:
		return "required"
	case authAuthorized:
		return "authorized"
	default:
		return "unknown"
	}
}

type handlerConfig struct {
	authMode       authMode
	resource       string
	action         string
	commandDecoder CommandDecoder
	queryDecoder   QueryDecoder
	render         RenderFunc
	redirect       string
	trigger        string
	triggerDetail  map[string]any
	pushURL        string
	csrfConfig     *CSRFConfig
	maxBodySize    int64
	timeout        time.Duration
	successStatus  int
	requireMethod  string
	onError        func(*http.Request, error)
}

// hasNoExplicitBody returns true if the handler has no render function and
// no HTMX response fields that would produce body content.
// When true, the handler should return 204 No Content.
func (c *handlerConfig) hasNoExplicitBody() bool {
	return c.render == nil &&
		c.redirect == "" &&
		c.trigger == "" &&
		c.pushURL == "" &&
		len(c.triggerDetail) == 0
}

// decodeAndSet creates a HandlerOption that decodes a request body and sets
// the result on the handlerConfig. It collapses the 4 Decode* variants.
func decodeAndSet[T, R any](
	bodyDec func(*http.Request, int64) (T, error),
	mapper func(T) (R, error),
	setter func(*handlerConfig, func(*http.Request) (R, error)),
) HandlerOption {
	return func(cfg *handlerConfig) {
		setter(cfg, func(r *http.Request) (R, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return bodyDec(r, cfg.maxBodySize)
			}, mapper)
		})
	}
}

// DecodeJSON decodes a JSON request body into a command using the mapper.
func DecodeJSON[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return decodeAndSet(decodeJSONBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (command.Command, error)) {
			cfg.commandDecoder = dec
		})
}

// DecodeJSONQuery decodes a JSON request body into a query.
func DecodeJSONQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return decodeAndSet(decodeJSONBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (query.Query, error)) {
			cfg.queryDecoder = dec
		})
}

// DecodeForm decodes form data into a command using the mapper.
func DecodeForm[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return decodeAndSet(decodeFormBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (command.Command, error)) {
			cfg.commandDecoder = dec
		})
}

// DecodeFormQuery decodes form data into a query using the mapper.
func DecodeFormQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return decodeAndSet(decodeFormBody[T], mapper,
		func(cfg *handlerConfig, dec func(*http.Request) (query.Query, error)) {
			cfg.queryDecoder = dec
		})
}

// Render sets the render function for query results.
func Render(fn RenderFunc) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = fn
	}
}

// RenderTempl renders a fixed templ.Component for query results.
// The component is created before rendering and ignores the query result.
//
// Usage:
//
//	app.Query("GetPage", cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RenderTempl(myPageComponent()),
//	)
func RenderTempl(component TemplComponent) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, r *http.Request, _ any) error {
			return component.Render(r.Context(), w)
		}
	}
}

// RenderTemplResult creates a templ.Component from the query result and renders it.
// The mapper function converts the query result into a TemplComponent.
//
// Usage:
//
//	app.Query("ListUsers", cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RenderTemplResult(func(result any) cqrshtmx.TemplComponent {
//	        return userListPage(result.([]*User))
//	    }),
//	)
func RenderTemplResult[T any](mapper func(T) TemplComponent) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, r *http.Request, result any) error {
			typed, ok := result.(T)
			if !ok {
				return event.NewRejection("unexpected_result_type",
					fmt.Sprintf("unexpected result type %T", result)).WithCause(ErrDecodeFailed)
			}

			component := mapper(typed)
			return component.Render(r.Context(), w)
		}
	}
}

// Redirect sets a redirect URL for successful execution.
func Redirect(url string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.redirect = url
	}
}

// Trigger sets an HTMX client-side event to fire on success.
func Trigger(event string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.trigger = event
	}
}

// TriggerWithDetail sets an HTMX client-side event with JSON detail.
func TriggerWithDetail(event string, detail any) HandlerOption {
	return func(cfg *handlerConfig) {
		if cfg.triggerDetail == nil {
			cfg.triggerDetail = make(map[string]any)
		}

		cfg.triggerDetail[event] = detail
	}
}

// PushURL pushes a URL into browser history on success.
func PushURL(url string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.pushURL = url
	}
}

// validateDispatch wraps a decoder with a validation step.
// Validation errors are wrapped with ErrValidationFailed.
func validateDispatch[T any](
	getter func(*handlerConfig) func(*http.Request) (T, error),
	setter func(*handlerConfig, func(*http.Request) (T, error)),
	validator func(T) error,
	label string,
) HandlerOption {
	return func(cfg *handlerConfig) {
		original := getter(cfg)
		if original == nil {
			slog.Warn("cqrs-htmx: "+label+" applied before decoder",
				slog.String("hint", "apply after DecodeJSON/DecodeForm"))
			return
		}

		setter(cfg, func(r *http.Request) (T, error) {
			val, err := original(r)
			if err != nil {
				var zero T
				return zero, err
			}

			if valErr := validator(val); valErr != nil {
				var zero T
				return zero, fmt.Errorf("%w: %w", ErrValidationFailed, valErr)
			}

			return val, nil
		})
	}
}

// ValidateCommand wraps the command decoder with a validation step.
// The validator receives the decoded command and may return an error.
// Validation errors are wrapped with ErrValidationFailed.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.DecodeJSON(...),
//	    cqrshtmx.ValidateCommand(func(cmd command.Command) error {
//	        // e.g., check required fields
//	        return nil
//	    }),
//	)
func ValidateCommand(validator func(command.Command) error) HandlerOption {
	return validateDispatch(
		func(cfg *handlerConfig) func(*http.Request) (command.Command, error) { return cfg.commandDecoder },
		func(cfg *handlerConfig, dec func(*http.Request) (command.Command, error)) { cfg.commandDecoder = dec },
		validator,
		"ValidateCommand",
	)
}

// ValidateQuery wraps the query decoder with a validation step.
// The validator receives the decoded query and may return an error.
// Validation errors are wrapped with ErrValidationFailed.
func ValidateQuery(validator func(query.Query) error) HandlerOption {
	return validateDispatch(
		func(cfg *handlerConfig) func(*http.Request) (query.Query, error) { return cfg.queryDecoder },
		func(cfg *handlerConfig, dec func(*http.Request) (query.Query, error)) { cfg.queryDecoder = dec },
		validator,
		"ValidateQuery",
	)
}

// WithTimeout sets a per-handler timeout override.
// If > 0, it takes precedence over the App-level Config.Timeout.
// Zero or negative means fall back to App config (default: no timeout).
func WithTimeout(d time.Duration) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.timeout = d
	}
}

// WithMaxBodySize sets a per-handler maximum request body size override.
// If > 0, it takes precedence over the App-level Config.MaxBodySize.
// Zero or negative means fall back to App config (default: 10 MB).
func WithMaxBodySize(n int64) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.maxBodySize = n
	}
}

// WithSuccessStatus sets the HTTP status code for successful responses
// when no explicit body is written. Default is 204 No Content.
// Common values: 200 OK, 201 Created.
func WithSuccessStatus(code int) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.successStatus = code
	}
}

func applyHTMXResponse(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) bool {
	if cfg.redirect == "" && cfg.trigger == "" && cfg.pushURL == "" && len(cfg.triggerDetail) == 0 {
		return false
	}

	resp := NewResponse(w, r)

	if cfg.redirect != "" {
		resp.Redirect(cfg.redirect)
	}

	if cfg.trigger != "" {
		resp.Trigger(cfg.trigger)
	}

	for name, detail := range cfg.triggerDetail {
		resp.TriggerWithDetail(name, detail)
	}

	if cfg.pushURL != "" {
		resp.PushURL(cfg.pushURL)
	}

	return resp.Apply()
}

// OnError returns a HandlerOption that registers a per-handler error callback.
// The callback is invoked after the App-level error handler, allowing handlers
// to add custom logging, metrics, or cleanup for specific routes.
func OnError(fn func(r *http.Request, err error)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.onError = fn
	}
}

// RenderJSON renders query results as JSON with 200 OK and
// Content-Type: application/json. Use the type parameter to enforce
// compile-time documentation and runtime type checking.
//
// Usage:
//
//	app.Query("GetUser", cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RenderJSON[User](),
//	)
func RenderJSON[T any]() HandlerOption {
	return renderJSONWithStatus[T](http.StatusOK)
}

// RenderJSONStatus renders query results as JSON with a custom HTTP status code
// and Content-Type: application/json.
//
// Usage:
//
//	app.Query("CreateUser", cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RenderJSONStatus[User](http.StatusCreated),
//	)
func RenderJSONStatus[T any](status int) HandlerOption {
	return renderJSONWithStatus[T](status)
}

func renderJSONWithStatus[T any](status int) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, r *http.Request, result any) error {
			typed, ok := result.(T)
			if !ok {
				return event.NewRejection("unexpected_result_type",
					fmt.Sprintf("unexpected result type %T", result)).WithCause(ErrDecodeFailed)
			}

			return WriteJSON(w, status, typed)
		}
	}
}

// RequireMethod returns a HandlerOption that rejects requests with the wrong HTTP method.
// Returns 405 Method Not Allowed for mismatched methods.
//
// Usage:
//
//	app.Command("CreateUser",
//	    cqrshtmx.DecodeJSON(...),
//	    cqrshtmx.RequireMethod(http.MethodPost),
//	)
func RequireMethod(method string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.requireMethod = method
	}
}

// DecodePagination extracts page and page_size from HTTP query parameters.
// Uses go-cqrs-lite query.Pagination defaults (page=1, page_size=20, max=100)
// when parameters are missing or invalid.
//
// Query parameters:
//   - page: page number (1-based, default 1)
//   - page_size: items per page (default 20, max 100)
//
// Usage:
//
//	app.Query("ListItems",
//	    cqrshtmx.DecodeFormQuery(func(r *http.Request) (query.Query, error) {
//	        p := cqrshtmx.DecodePagination(r)
//	        return ListItemsQuery{Page: p.Page, PageSize: p.PageSize}, nil
//	    }),
//	)
func DecodePagination(r *http.Request) query.Pagination {
	var page, pageSize uint

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			page = uint(n)
		}
	}

	if v := r.URL.Query().Get("page_size"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			pageSize = uint(n)
		}
	}

	return query.NewPagination(page, pageSize)
}

// RenderPaginatedJSON renders a query.PaginatedResult[T] as JSON with 200 OK.
// Sets Content-Type and includes pagination metadata in the response body.
//
// Usage:
//
//	app.Query("ListUsers",
//	    cqrshtmx.DecodeFormQuery(...),
//	    cqrshtmx.RenderPaginatedJSON[User](),
//	)
func RenderPaginatedJSON[T any]() HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, _ *http.Request, result any) error {
			typed, ok := result.(query.PaginatedResult[T])
			if !ok {
				return event.NewRejection("unexpected_result_type",
					fmt.Sprintf("expected PaginatedResult[%T], got %T", *new(T), result)).WithCause(ErrDecodeFailed)
			}

			return WriteJSON(w, http.StatusOK, typed)
		}
	}
}
