package adminui

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// maxSearchBody bounds the body-carrying search request. The criteria form is
// a handful of short fields; a body beyond this is refused, not parsed.
const maxSearchBody = 64 << 10

// errSearchBody is the Rejection for an unreadable search body (oversized or
// malformed urlencoded data).
var errSearchBody error = errorfamily.NewRejection("adminui.search.body", "search body is malformed or too large")

// userSearch carries the criteria of one search exchange up the QUERY-method
// fallback ladder (docs/guides/http-query-method.md): the same three keys the
// GET route squeezes into URL params travel as a urlencoded request body on
// the QUERY and POST rungs.
type userSearch struct {
	Query  string
	Filter userFilter
	Page   int
}

// parseUserSearch reads the criteria from a urlencoded request body.
//
// The wire format is htmx's native form encoding, not JSON: the panel's
// client is htmx + the embedded cqrs-query extension, which upgrade the
// METHOD (QUERY, with POST+override fallback) but keep htmx's form
// serialization for the body — vendoring a JSON encoder extension would buy
// nothing here. The parse is Content-Type-lenient per the library principle
// (never enforce defaults consumers might disagree with); a consumer stacking
// RFC 10008-strict rejection wraps the route with RequireContentType.
func parseUserSearch(r *http.Request) (userSearch, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxSearchBody+1))
	if err != nil {
		return userSearch{}, errorfamily.WrapTransient(err, "adminui.search.body_read", "read search body")
	}
	if len(body) > maxSearchBody {
		return userSearch{}, errSearchBody
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return userSearch{}, errSearchBody
	}

	page := 0
	if raw := values.Get("page"); raw != "" {
		if page, err = strconv.Atoi(raw); err != nil {
			return userSearch{}, errSearchBody
		}
	}

	return userSearch{
		Query:  strings.TrimSpace(values.Get("q")),
		Filter: parseUserFilter(values.Get("filter")),
		Page:   page,
	}, nil
}

// userSearch serves the body-carrying search rungs — the native QUERY route
// and its POST fallback twin register the SAME handler, so every rung of the
// ladder answers byte-identically (pinned by TestUserSearch_LadderParity).
// The response is the users table fragment, the exact payload the search
// input's hx-target swap expects.
func (h *Handler) userSearch(w http.ResponseWriter, r *http.Request, _ *identitymodel.User) {
	search, err := parseUserSearch(r)
	if err != nil {
		h.writeErrorPage(
			w,
			r,
			http.StatusBadRequest,
			"Invalid search request",
			"The search criteria could not be read. Clear the search box and try again.",
		)
		return
	}

	data := h.usersSearchData(r.Context(), search.Query, search.Filter, search.Page)
	renderPartial(w, r, usersTableContent(data))
}

// usersSearchData builds the list data for one page of filtered users — the
// shared body of every search rung: the GET route (usersIndex) and the
// QUERY/POST twins. One filter order, one pagination clamp, one enrichment
// pass; the rungs cannot drift.
func (h *Handler) usersSearchData(ctx context.Context, q string, filter userFilter, page int) usersListData {
	matched := filterUsers(h.config.Service.ReadModel().AllUsers(), q, filter)
	if filter == userFilterAdmin {
		matched = h.filterPlatformAdmins(matched)
	}
	offset, limit, totalPages, current := pageBounds(page, len(matched), h.config.PageSize)
	visible := matched[offset : offset+limit]
	rows := make([]userRow, 0, len(visible))
	lastSeen := h.lastSeenIndex()
	for _, u := range visible {
		row := h.userRow(ctx, u)
		row.LastSeen = lastSeen[u.ID.Get().String()]
		rows = append(rows, row)
	}

	return usersListData{
		Users: rows, Total: len(matched), Search: q,
		Filter: filter, BasePath: h.config.BasePath, Page: current, TotalPages: totalPages,
	}
}
