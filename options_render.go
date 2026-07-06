package cqrshtmx

import (
	"fmt"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Render sets a custom render function for query results.
// The function receives the query result and writes the HTTP response.
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

// RenderHTML renders a static HTML string as the query response.
// The HTML is written with Content-Type text/html; charset=utf-8.
// For dynamic HTML based on the query result, use Render with a custom function.
//
// Usage:
//
//	app.Query("GetSnippet", cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RenderHTML("<div>Hello, HTMX!</div>"),
//	)
func RenderHTML(html string) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, _ *http.Request, _ any) error {
			w.Header().Set("Content-Type", ContentTypeHTML)
			_, _ = w.Write([]byte(html))
			return nil
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
				return errorfamily.NewRejection("unexpected_result_type",
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
