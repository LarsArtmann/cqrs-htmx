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

// RenderIf selects between two render functions based on a request predicate.
// When check returns true, match is used; otherwise noMatch is used.
//
// This is the composable primitive behind [RenderPartialOrFullFunc]. Use it
// directly when you need a custom predicate beyond partial-vs-full-page:
//
//	app.Query("GetProfile", decoder,
//	    cqrshtmx.RenderIf(
//	        func(r *http.Request) bool { return cqrshtmx.HTMXTarget(r) == "#avatar" },
//	        avatarPartial,
//	        fullProfile,
//	    ),
//	)
func RenderIf(check func(*http.Request) bool, match, noMatch RenderFunc) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, r *http.Request, result any) error {
			if check(r) {
				return match(w, r, result)
			}
			return noMatch(w, r, result)
		}
	}
}

// RenderPartialOrFullFunc selects between two render functions based on whether
// the request is an HTMX partial request ([RenderPartial]). When the request
// comes from HTMX (and is not a history restore), partial is used; otherwise
// full is used.
//
// This is the non-generic version for consumers who use html/template, raw
// string building, or any non-templ rendering. For templ users, prefer
// [RenderPartialOrFull].
func RenderPartialOrFullFunc(partial, full RenderFunc) HandlerOption {
	return RenderIf(RenderPartial, partial, full)
}

// RenderPartialOrFull renders a partial templ fragment for HTMX requests and a
// full-page templ component for regular requests, eliminating the manual
// if-[RenderPartial] branching that every HTMX handler otherwise needs.
//
// Both mappers receive the same typed query result. The partial mapper should
// return just the fragment that changed (e.g. a table body); the full mapper
// should return the complete page with that fragment embedded.
//
// Usage:
//
//	app.Query("ListUsers", decoder,
//	    cqrshtmx.RenderPartialOrFull(
//	        func(users []*User) cqrshtmx.TemplComponent { return userListPartial(users) },
//	        func(users []*User) cqrshtmx.TemplComponent { return usersPage(users) },
//	    ),
//	)
func RenderPartialOrFull[T any](partial, full func(T) TemplComponent) HandlerOption {
	return func(cfg *handlerConfig) {
		cfg.render = func(w http.ResponseWriter, r *http.Request, result any) error {
			typed, ok := result.(T)
			if !ok {
				return errorfamily.NewRejection("unexpected_result_type",
					fmt.Sprintf("unexpected result type %T", result)).WithCause(ErrDecodeFailed)
			}

			w.Header().Set("Content-Type", ContentTypeHTML)
			if RenderPartial(r) {
				return partial(typed).Render(r.Context(), w)
			}
			return full(typed).Render(r.Context(), w)
		}
	}
}
