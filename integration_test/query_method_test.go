package integration_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/stretchr/testify/require"
)

// httpMethodQuery is the HTTP QUERY method (RFC 10008). net/http has no
// MethodQuery constant yet; the literal is a valid method token everywhere.
const httpMethodQuery = "QUERY"

// The HTTP QUERY method (RFC 10008) parity suite: one query handler serves
// the same read over three transports, and the responses must be identical.
//
//	rung 1: native QUERY + JSON body     (safe, idempotent, body-carrying)
//	rung 2: POST with a JSON body        (the pre-middleware fallback shape)
//	rung 3: GET + URL params             (small queries, bookmarkable)
//
// This pins plan fact S2 (App.Query and the decoders are method-agnostic) at
// the integration level, on PUBLISHED root API only — no unpublished symbols,
// so the hermetic module-isolation leg stays green.
// Plan: docs/planning/2026-11-07_07-21_SUPERB-http-query-method-pareto-plan.md.

type searchContactsQuery struct {
	Filter string
}

func (q *searchContactsQuery) Type() query.Type { return "SearchContacts" }

type searchRequest struct {
	Filter string `json:"filter"`
}

type contact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var contactsFixture = []contact{
	{Name: "Ada Lovelace", Email: "ada@example.com"},
	{Name: "Alan Turing", Email: "alan@example.com"},
	{Name: "Grace Hopper", Email: "grace@example.com"},
}

// searchContacts is the single registered query handler, so the dispatcher
// only ever passes *searchContactsQuery here.
func searchContacts(_ context.Context, qry query.Query) (any, error) {
	sq := qry.(*searchContactsQuery) //nolint:errcheck // single registered type

	if sq.Filter == "" {
		return contactsFixture, nil
	}
	matched := make([]contact, 0, len(contactsFixture))
	for _, c := range contactsFixture {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(sq.Filter)) {
			matched = append(matched, c)
		}
	}
	return matched, nil
}

func newQueryMethodServer(t *testing.T) *httptest.Server {
	t.Helper()

	qryDisp := query.NewDispatcher()
	qryDisp.Register("SearchContacts", searchContacts)

	app := cqrshtmx.MustNew(cqrshtmx.Config{Queries: qryDisp})

	jsonDecoder := cqrshtmx.DecodeJSONQuery(func(req searchRequest) (query.Query, error) {
		return &searchContactsQuery{Filter: req.Filter}, nil
	})
	formDecoder := cqrshtmx.DecodeFormQueryWithRequest(func(r *http.Request, _ struct{}) (query.Query, error) {
		return &searchContactsQuery{Filter: r.URL.Query().Get("filter")}, nil
	})

	mux := http.NewServeMux()
	// One handler, three rungs: native QUERY, the POST fallback shape, and
	// the URL-param GET for small queries. The mux pattern carries the method.
	mux.Handle(httpMethodQuery+" /contacts", app.Query("SearchContacts", jsonDecoder, cqrshtmx.RenderJSON[[]contact]()))
	mux.Handle(http.MethodPost+" /contacts", app.Query("SearchContacts", jsonDecoder, cqrshtmx.RenderJSON[[]contact]()))
	mux.Handle(http.MethodGet+" /contacts", app.Query("SearchContacts", formDecoder, cqrshtmx.RenderJSON[[]contact]()))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

func TestQueryMethod_NativeQueryWithBody(t *testing.T) {
	t.Parallel()
	srv := newQueryMethodServer(t)

	req, err := http.NewRequest(httpMethodQuery, srv.URL+"/contacts",
		strings.NewReader(`{"filter":"turing"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Contains(t, res.Header.Get("Content-Type"), "application/json")
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "Alan Turing")
}

func TestQueryMethod_FallbackParity(t *testing.T) {
	t.Parallel()
	srv := newQueryMethodServer(t)

	do := func(method, body, contentType, rawQuery string) (int, string) {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+"/contacts"+rawQuery, strings.NewReader(body))
		require.NoError(t, err)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		res, err := srv.Client().Do(req)
		require.NoError(t, err)
		raw, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		_ = res.Body.Close()
		return res.StatusCode, string(raw)
	}

	// The same filtered read over rung 1 (QUERY), rung 2 (POST), and
	// rung 3 (GET + params) must produce byte-identical responses.
	queryStatus, queryBody := do(httpMethodQuery, `{"filter":"a"}`, "application/json", "")
	postStatus, postBody := do(http.MethodPost, `{"filter":"a"}`, "application/json", "")
	getStatus, getBody := do(http.MethodGet, "", "", "?filter=a")

	require.Equal(t, http.StatusOK, queryStatus)
	require.Equal(t, queryStatus, postStatus, "override POST (rung 2) must match native QUERY")
	require.Equal(t, queryStatus, getStatus, "GET twin (rung 3) must match native QUERY")
	require.Equal(t, queryBody, postBody)
	require.Equal(t, queryBody, getBody)
}

func TestQueryMethod_EmptyBodyIsZeroValueFilter(t *testing.T) {
	t.Parallel()
	srv := newQueryMethodServer(t)

	req, err := http.NewRequest(httpMethodQuery, srv.URL+"/contacts", nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close() //nolint:errcheck

	// Empty body decodes to the zero-value query (no filter): the full list,
	// not an error — the documented decodeJSONBody behavior.
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestQueryMethod_MuxAdvertisesQueryInAllow(t *testing.T) {
	t.Parallel()
	srv := newQueryMethodServer(t)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/contacts", nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close() //nolint:errcheck

	// RFC 10008 discovery: a resource advertises QUERY through OPTIONS/Allow.
	// The stdlib mux already emits Allow on its 405 — pin that QUERY is in it.
	require.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	require.Contains(t, res.Header.Get("Allow"), "QUERY")
}
