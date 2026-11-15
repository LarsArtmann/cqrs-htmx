// Directory search API: the admin-demo's App-dispatched JSON surface.
//
// The panel's user search (adminui) is HTML-over-the-wire; this sibling
// endpoint is the MACHINE client story: the same directory read dispatched
// through a root App over the RFC 10008 QUERY ladder (JSON body rungs),
// documented by a served OpenAPI spec built from the per-route WithOpenAPI
// metadata. Together they prove both spec-construction forms the library
// ships: this file uses the per-route option form (metadata travels with the
// handler definition, App.OpenAPIRoutes collects it), sync-demo uses the
// explicit builder form.
package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// directorySearchBody is the JSON body of the QUERY rung. The keys mirror the
// panel search's URL params (q, filter, page) so a consumer can move between
// the HTML and JSON surfaces without learning a second vocabulary.
type directorySearchBody struct {
	Q      string `json:"q"`
	Filter string `json:"filter"`
	Page   int    `json:"page"`
}

// directorySearchQuery is the dispatched query type.
type directorySearchQuery struct {
	Q      string
	Filter string
	Page   int
}

func (q *directorySearchQuery) Type() query.Type { return "SearchDirectory" }

// directoryUser is the machine-facing projection of one user: the fields an
// external directory consumer decides on, not the panel's row internals.
type directoryUser struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Verified    bool   `json:"verified"`
}

// directoryResult is bounded: a directory search serves humans and
// integrations, not bulk export — the panel's export endpoints own that job.
type directoryResult struct {
	Total int             `json:"total"`
	Users []directoryUser `json:"users"`
}

// directorySearchPageSize caps one response.
const directorySearchPageSize = 50

// newDirectoryQueryDispatcher registers the SearchDirectory query over the
// service's read model. The filter is demo-level (substring over email and
// display name, status filter by verification), deliberately NOT the panel's
// adminui internals — the demo depends on published usermgmt API only.
func newDirectoryQueryDispatcher(svc *usermgmt.Service) *query.Dispatcher {
	qryDisp := query.NewDispatcher()
	//cqrs-lint:ignore(C028) demo wiring: a failed registration surfaces via the tests that exercise the query
	_ = query.RegisterTyped(qryDisp, "SearchDirectory",
		func(_ context.Context, q *directorySearchQuery) (directoryResult, error) {
			all := svc.ReadModel().AllUsers()
			needle := strings.ToLower(strings.TrimSpace(q.Q))

			users := make([]directoryUser, 0, len(all))
			for _, u := range all {
				if q.Filter == "verified" && !u.EmailVerified {
					continue
				}
				if q.Filter == "unverified" && u.EmailVerified {
					continue
				}
				if needle != "" &&
					!strings.Contains(strings.ToLower(u.Email), needle) &&
					!strings.Contains(strings.ToLower(u.DisplayName), needle) {
					continue
				}
				users = append(users, directoryUser{
					Email:       u.Email,
					DisplayName: u.DisplayName,
					Verified:    u.EmailVerified,
				})
			}

			total := len(users)
			start := max(0, (q.Page-1)*directorySearchPageSize)
			if start > total {
				start = total
			}
			end := min(total, start+directorySearchPageSize)

			return directoryResult{Total: total, Users: users[start:end]}, nil
		})

	return qryDisp
}

// directorySearchOpenAPI is the per-route metadata: it travels with the
// handler definition (the WithOpenAPI option) and the App collects it for the
// served spec. The QUERY op auto-bumps the spec to OpenAPI 3.2.0 (the query
// operation type does not exist in 3.1) — the version bump is the builder's
// job, not the demo's. The POST fallback rung needs no separate spec entry:
// it is the same operation over a friendlier transport, named in the
// description.
func directorySearchOpenAPI() openapi.Operation {
	bodySchema := openapi.Object(
		openapi.Prop("q", openapi.String().WithDescription("case-insensitive substring over email and display name")),
		openapi.Prop("filter", openapi.String().WithDescription("verified | unverified (omit for all)")),
		openapi.Prop("page", openapi.Integer().WithDescription("1-based page ("+strconv.Itoa(directorySearchPageSize)+" users per page)")),
	)

	resultSchema := openapi.Object(
		openapi.PropReq("total", openapi.Integer()),
		openapi.PropReq("users", openapi.Array(openapi.Object(
			openapi.PropReq("email", openapi.String()),
			openapi.Prop("display_name", openapi.String()),
			openapi.PropReq("verified", openapi.Boolean()),
		))),
	)

	return openapi.Query("SearchDirectory").
		Summary("Search the user directory").
		Desc("Safe, idempotent read over the event-sourced directory. " +
			"Send the same body as a plain POST (the fallback rung) when an intermediary refuses the QUERY verb.").
		Tag("directory").
		JSONBody(bodySchema).
		Response(200, "the matching directory page", openapi.JSON(resultSchema)).
		Response(400, "malformed query").
		Response(401, "no session").
		Op()
}

// directorySearchHandler builds the App route for one ladder rung. App.Query
// is method-agnostic — the mux pattern carries the method, so both rungs are
// the same builder with a different registration pattern. Only the native
// QUERY rung carries the WithOpenAPI metadata (one spec entry per operation).
func directorySearchHandler(app *cqrshtmx.App, withOpenAPI bool) http.Handler {
	options := []cqrshtmx.HandlerOption{
		cqrshtmx.DecodeJSONQuery(func(body directorySearchBody) (query.Query, error) {
			return &directorySearchQuery{Q: body.Q, Filter: body.Filter, Page: body.Page}, nil
		}),
		cqrshtmx.RenderJSON[directoryResult](),
	}
	if withOpenAPI {
		options = append(options, cqrshtmx.WithOpenAPI(directorySearchOpenAPI()))
	}

	return app.Query("SearchDirectory", options...)
}

// newDirectoryApp wires the App and the served spec. The QUERY rung must be
// built (app.Query) BEFORE OpenAPIRoutes is read — the collector captures the
// metadata when the handler is built.
func newDirectoryApp(svc *usermgmt.Service) (*cqrshtmx.App, *openapi.Spec) {
	app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: newDirectoryQueryDispatcher(svc)})
	_ = directorySearchHandler(app, true)

	spec := openapi.New("admin-demo API", "1.0.0")
	for _, route := range app.OpenAPIRoutes() {
		spec.PathOp("/api/users/search", route.Operation)
	}

	return app, spec
}

// requireSession refuses unauthenticated API calls with the shared problem
// document shape. NewSessionMiddleware only ENRICHES the request context —
// it never gates — so a machine-facing surface must gate itself (the panel's
// routes are gated internally by adminui's guard; this one is outside it).
func requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if usermgmt.UserIDFromRequest(r) == "" {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"about:blank","title":"unauthorized","status":401,"detail":"sign in to search the directory"}` + "\n"))

			return
		}
		next.ServeHTTP(w, r)
	})
}
