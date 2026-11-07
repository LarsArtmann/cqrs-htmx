package cqrshtmx

import (
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	httputil "github.com/larsartmann/httputil"
)

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
	return func(config *handlerConfig) {
		config.render = func(w http.ResponseWriter, r *http.Request, result any) error {
			typed, ok := result.(T)
			if !ok {
				return errorfamily.NewRejection("unexpected_result_type",
					fmt.Sprintf("unexpected result type %T", result)).WithCause(httperr.ErrDecodeFailed)
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
	return func(config *handlerConfig) {
		config.requireMethod = method
	}
}

// RequireAnyMethod returns a HandlerOption that rejects requests whose HTTP
// method is not one of the given methods — the multi-method generalization of
// [RequireMethod], for handlers that legitimately serve several transports of
// the same operation (the RFC 10008 QUERY fallback ladder: QUERY plus the
// POST fallback twin, plus a GET twin for URL-sized queries). A mismatch
// answers 405 with an Allow header listing the configured methods.
//
// Usage:
//
//	app.Query("SearchContacts",
//	    cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RequireAnyMethod("QUERY", http.MethodPost),
//	)
func RequireAnyMethod(methods ...string) HandlerOption {
	return func(config *handlerConfig) {
		config.requireMethods = methods
	}
}

// RequireContentType returns a HandlerOption that enforces the RFC 10008
// media-type ladder on requests: a body-carrying request without a declared
// Content-Type answers 400 (the server must not guess the media type), and a
// declared type different from the required one answers 415
// ([httperr.ErrUnsupportedMediaType]). Parameters such as "; charset=utf-8"
// are ignored in the comparison. A request without a body is always allowed
// (there is nothing to mis-declare).
//
// The library default is deliberately lenient (decoders accept any bytes they
// can parse); this option is the opt-in for routes that want the strict,
// spec-exact behavior. Usage:
//
//	app.Query("SearchContacts",
//	    cqrshtmx.DecodeJSONQuery(...),
//	    cqrshtmx.RequireContentType("application/json"),
//	)
func RequireContentType(mediaType string) HandlerOption {
	return func(config *handlerConfig) {
		config.requiredContentType = mediaType
	}
}

// checkRequiredContentType enforces config.requiredContentType after the
// security checks, before decode (see executePreDispatchChecks).
func (a *App) checkRequiredContentType(w http.ResponseWriter, r *http.Request, config *handlerConfig) error {
	if config.requiredContentType == "" {
		return nil
	}

	declared := r.Header.Get("Content-Type")
	if declared == "" {
		if r.Body == nil || r.ContentLength == 0 {
			return nil
		}

		a.errorHandler(w, r, errorfamily.Wrapf(httperr.ErrDecodeFailed, event.Rejection,
			"cqrshtmx.handler.content_type_missing",
			"request carries a body but no Content-Type; want %s", config.requiredContentType))

		return httperr.ErrDecodeFailed
	}

	got, _, err := mime.ParseMediaType(declared)
	if err != nil {
		a.errorHandler(w, r, errorfamily.Wrapf(httperr.ErrDecodeFailed, event.Rejection,
			"cqrshtmx.handler.content_type_invalid",
			"malformed Content-Type %q: %v", declared, err))

		return httperr.ErrDecodeFailed
	}

	if !strings.EqualFold(got, config.requiredContentType) {
		a.errorHandler(w, r, errorfamily.Wrapf(httperr.ErrUnsupportedMediaType, event.Rejection,
			"cqrshtmx.handler.unsupported_media_type",
			"got %q, want %q", got, config.requiredContentType))

		return httperr.ErrUnsupportedMediaType
	}

	return nil
}

// AcceptQuery returns a HandlerOption that ADDS media types to the
// Accept-Query response header (RFC 10008 discovery: which query formats the
// resource accepts). The JSON body-query decoders already advertise
// application/json themselves, so this option is only needed for consumer
// decoders that accept additional formats.
func AcceptQuery(mediaTypes ...string) HandlerOption {
	return func(config *handlerConfig) {
		for _, mt := range mediaTypes {
			if !slices.Contains(config.acceptQuery, mt) {
				config.acceptQuery = append(config.acceptQuery, mt)
			}
		}
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
	page := httputil.ParseUintQuery(r, "page")
	pageSize := httputil.ParseUintQuery(r, "page_size")

	return query.NewPagination(page, pageSize)
}

// DecodePaginationStrict extracts page and page_size from HTTP query
// parameters, returning a rejection error for malformed or out-of-range
// values instead of silently coercing them to defaults like
// [DecodePagination] does.
//
// Missing parameters keep the go-cqrs-lite defaults (page=1, page_size=20).
// Explicit values must satisfy query.Pagination.Validate: page >= 1 and
// page_size in [1, 100]. Malformed numbers, zero, and oversized page sizes
// return an errorfamily Rejection — routed through the App's error handler
// it surfaces as a client-facing 400 with the offending value.
//
// Usage (inside a query decoder — the returned error aborts the request):
//
//	p, err := cqrshtmx.DecodePaginationStrict(r)
//	if err != nil {
//	    return nil, err
//	}
func DecodePaginationStrict(r *http.Request) (query.Pagination, error) {
	pagination := query.NewPagination(0, 0)

	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return query.Pagination{}, errorfamily.Newf(event.Rejection,
				"cqrshtmx.pagination.page_malformed",
				"page must be a non-negative integer, got %s", raw)
		}

		pagination.Page = uint(parsed)
	}

	if raw := r.URL.Query().Get("page_size"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return query.Pagination{}, errorfamily.Newf(event.Rejection,
				"cqrshtmx.pagination.page_size_malformed",
				"page_size must be a non-negative integer, got %s", raw)
		}

		pagination.PageSize = uint(parsed)
	}

	if err := pagination.Validate(); err != nil {
		return query.Pagination{}, errorfamily.Wrapf(err, event.Rejection,
			"cqrshtmx.pagination.invalid", "invalid pagination parameters")
	}

	return pagination, nil
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
	return func(config *handlerConfig) {
		config.render = func(w http.ResponseWriter, _ *http.Request, result any) error {
			typed, ok := result.(query.PaginatedResult[T])
			if !ok {
				return errorfamily.NewRejection(
					"unexpected_result_type",
					fmt.Sprintf(
						"expected PaginatedResult[%T], got %T",
						*new(T),
						result,
					),
				).WithCause(httperr.ErrDecodeFailed)
			}

			return WriteJSON(w, http.StatusOK, typed)
		}
	}
}
