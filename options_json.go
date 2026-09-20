package cqrshtmx

import (
	"fmt"
	"net/http"
	"strconv"

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
	return func(config *handlerConfig) {
		config.requireMethod = method
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
				return errorfamily.NewRejection("unexpected_result_type",
					fmt.Sprintf("expected PaginatedResult[%T], got %T", *new(T), result)).WithCause(ErrDecodeFailed)
			}

			return WriteJSON(w, http.StatusOK, typed)
		}
	}
}
