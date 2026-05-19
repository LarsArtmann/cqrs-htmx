package cqrshtmx

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
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
}

// hasNoExplicitBody returns true if the handler has no render function and
// no HTMX response fields that would produce body content.
// When true, the handler should return 204 No Content.
func (c *handlerConfig) hasNoExplicitBody() bool {
	return c.redirect == "" && c.trigger == "" && c.pushURL == "" && len(c.triggerDetail) == 0
}

// DecodeJSON decodes a JSON request body into a command using the mapper.
func DecodeJSON[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.commandDecoder = func(r *http.Request) (command.Command, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return decodeJSONBody[T](r, cfg.maxBodySize)
			}, mapper)
		}
	}
}

// DecodeJSONQuery decodes a JSON request body into a query.
func DecodeJSONQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.queryDecoder = func(r *http.Request) (query.Query, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return decodeJSONBody[T](r, cfg.maxBodySize)
			}, mapper)
		}
	}
}

// DecodeForm decodes form data into a command using the mapper.
func DecodeForm[T any](mapper func(T) (command.Command, error)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.commandDecoder = func(r *http.Request) (command.Command, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return decodeFormBody[T](r, cfg.maxBodySize)
			}, mapper)
		}
	}
}

// DecodeFormQuery decodes form data into a query using the mapper.
func DecodeFormQuery[T any](mapper func(T) (query.Query, error)) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.queryDecoder = func(r *http.Request) (query.Query, error) {
			return decodeRequest(r, func(r *http.Request) (T, error) {
				return decodeFormBody[T](r, cfg.maxBodySize)
			}, mapper)
		}
	}
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
				return fmt.Errorf("%w: unexpected result type %T", ErrDecodeFailed, result)
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
	return func(cfg *handlerConfig) {
		if cfg.commandDecoder == nil {
			return
		}

		original := cfg.commandDecoder
		cfg.commandDecoder = func(r *http.Request) (command.Command, error) {
			cmd, err := original(r)
			if err != nil {
				return nil, err
			}

			if valErr := validator(cmd); valErr != nil {
				return nil, fmt.Errorf("%w: %w", ErrValidationFailed, valErr)
			}

			return cmd, nil
		}
	}
}

// ValidateQuery wraps the query decoder with a validation step.
// The validator receives the decoded query and may return an error.
// Validation errors are wrapped with ErrValidationFailed.
func ValidateQuery(validator func(query.Query) error) HandlerOption {
	return func(cfg *handlerConfig) {
		if cfg.queryDecoder == nil {
			return
		}

		original := cfg.queryDecoder
		cfg.queryDecoder = func(r *http.Request) (query.Query, error) {
			qry, err := original(r)
			if err != nil {
				return nil, err
			}

			if valErr := validator(qry); valErr != nil {
				return nil, fmt.Errorf("%w: %w", ErrValidationFailed, valErr)
			}

			return qry, nil
		}
	}
}

func applyHTMXResponse(w http.ResponseWriter, r *http.Request, cfg *handlerConfig) bool {
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
