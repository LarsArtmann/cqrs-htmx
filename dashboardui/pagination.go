package dashboardui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// paginationState tracks cursor-based pagination state for rendering.
type paginationState struct {
	// HasNext is true when more results exist beyond the current page.
	HasNext bool
	// NextCursor is the cursor value for the next page (e.g., last event ID).
	NextCursor string
	// PageSize is the number of items per page.
	PageSize int
	// HasPrev is true when a cursor was used (i.e., not on the first page).
	HasPrev bool
	// After is the cursor that loaded the current page (the ?after= value).
	// Empty on the first page.
	After string
	// PrevHistory is the comma-separated stack of cursors for pages before
	// the current one. The implicit first-page cursor ("") is never stored;
	// an empty history with a non-empty After means "Previous goes to page 1".
	PrevHistory string
	// TotalCount, when non-empty, renders a "Showing X–Y of Z" label.
	// A trailing "+" means the count hit the limit (e.g. "500+").
	TotalCount string
	// PageStart is the 1-based index of the first item on the current page.
	PageStart int
	// PageLen is the number of items actually shown on the current page.
	PageLen int
}

// pushCursor appends cursor to the comma-separated history, skipping empty
// values (the implicit first-page cursor is never stored).
func pushCursor(history, cursor string) string {
	if cursor == "" {
		return history
	}

	if history == "" {
		return cursor
	}

	return history + "," + cursor
}

// popCursor splits the history into the last entry and the remaining prefix.
// Returns ("", "") when history is empty.
func popCursor(history string) (string, string) {
	if history == "" {
		return "", ""
	}

	idx := strings.LastIndex(history, ",")

	if idx == -1 {
		return history, ""
	}

	return history[idx+1:], history[:idx]
}

// paginationQuery builds the query string for a pagination link from the
// after cursor, prev history, page size, and any extra filter params.
func paginationQuery(after, prevHistory string, pageSize int, extraParams string) string {
	var parts []string

	if after != "" {
		parts = append(parts, "after="+after)
	}

	if prevHistory != "" {
		parts = append(parts, "prev="+prevHistory)
	}

	parts = append(parts, "limit="+strconv.Itoa(pageSize))

	if extraParams != "" {
		parts = append(parts, extraParams)
	}

	return strings.Join(parts, "&")
}

// renderPagination renders Prev/Next links with cursor-history tracking.
// basePath is the dashboard base path, path is the page path (e.g., "/events").
// Extra query params (filters, sort) are preserved across pagination links.
func renderPagination(basePath, path string, state paginationState, extraParams string) string {
	if !state.HasNext && !state.HasPrev {
		return ""
	}

	var b strings.Builder

	b.WriteString(`<div class="pagination">`)

	if state.HasPrev {
		prevAfter, prevHistory := popCursor(state.PrevHistory)
		query := paginationQuery(prevAfter, prevHistory, state.PageSize, extraParams)
		fmt.Fprintf(&b, `<a href="%s%s?%s" class="btn">← Previous</a>`, basePath, path, query)
	} else {
		b.WriteString(`<span class="pagination disabled">← Previous</span>`)
	}

	b.WriteString(renderPaginationInfo(state))

	if state.HasNext {
		nextHistory := pushCursor(state.PrevHistory, state.After)
		query := paginationQuery(state.NextCursor, nextHistory, state.PageSize, extraParams)
		fmt.Fprintf(&b, `<a href="%s%s?%s" class="btn btn-accent">Next →</a>`, basePath, path, query)
	}

	b.WriteString(`</div>`)

	return b.String()
}

// renderPaginationInfo renders the "Showing X–Y of Z" label when available.
func renderPaginationInfo(state paginationState) string {
	if state.PageLen == 0 {
		return ""
	}

	end := state.PageStart + state.PageLen - 1
	if state.PageStart < 1 {
		state.PageStart = 1
	}
	if end < state.PageStart {
		end = state.PageStart
	}

	if state.TotalCount != "" {
		return fmt.Sprintf(`<span class="pagination-info">Showing %d–%d of %s</span>`,
			state.PageStart, end, esc(state.TotalCount))
	}

	return fmt.Sprintf(`<span class="pagination-info">Showing %d–%d</span>`,
		state.PageStart, end)
}

// computePageStart estimates the 1-based index of the first item on the current
// page from the cursor history. Assumes consistent page sizes across pages.
func computePageStart(state paginationState) int {
	entriesInPrev := 0
	if state.PrevHistory != "" {
		entriesInPrev = strings.Count(state.PrevHistory, ",") + 1
	}

	traversed := entriesInPrev
	if state.After != "" {
		traversed++
	}

	if state.PageSize <= 0 {
		state.PageSize = defaultPageSize
	}

	return traversed*state.PageSize + 1
}

// withCountInfo fills in PageStart, PageLen, and TotalCount (when known).
// TotalCount is only set on the last page (HasNext=false) because that's the
// only case where the total is known exactly without an extra count query.
func (s paginationState) withCountInfo(itemLen int) paginationState {
	s.PageStart = computePageStart(s)
	s.PageLen = itemLen

	if !s.HasNext && itemLen > 0 {
		s.TotalCount = strconv.Itoa(s.PageStart + itemLen - 1)
	}

	return s
}

// parsePageSize reads the ?limit= query param, clamped to [1, maxPageSize].
// Falls back to defaultPageSize when unset.
func parsePageSize(r *http.Request, defaultPageSize int) int {
	s := r.URL.Query().Get("limit")
	if s == "" {
		return defaultPageSize
	}

	num, err := strconv.Atoi(s)
	if err != nil || num < 1 {
		return defaultPageSize
	}

	if num > maxPageSize {
		return maxPageSize
	}

	return num
}

// parseCursorParams extracts the after cursor and prev history from the
// request query string. Shared by all paginated index handlers.
func parseCursorParams(r *http.Request) (string, string, bool) {
	after := r.URL.Query().Get("after")
	prevHistory := r.URL.Query().Get("prev")
	hasPrev := after != ""

	return after, prevHistory, hasPrev
}
