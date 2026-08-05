package core

import (
	"net/http"
	"strconv"
	"strings"
)

// PageState tracks cursor-based pagination state for rendering.
type PageState struct {
	HasNext     bool
	NextCursor  string
	PageSize    int
	HasPrev     bool
	After       string
	PrevHistory string
	TotalCount  string
	PageStart   int
	PageLen     int
}

// PushCursor appends cursor to the comma-separated history, skipping empty
// values (the implicit first-page cursor is never stored).
func PushCursor(history, cursor string) string {
	if cursor == "" {
		return history
	}

	if history == "" {
		return cursor
	}

	return history + "," + cursor
}

// PopCursor splits the history into the last entry and the remaining prefix.
// Returns ("", "") when history is empty.
func PopCursor(history string) (string, string) {
	if history == "" {
		return "", ""
	}

	idx := strings.LastIndex(history, ",")

	if idx == -1 {
		return history, ""
	}

	return history[idx+1:], history[:idx]
}

// PaginationQuery builds the query string for a pagination link from the
// after cursor, prev history, page size, and any extra filter params.
func PaginationQuery(after, prevHistory string, pageSize int, extraParams string) string {
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

// ComputePageStart estimates the 1-based index of the first item on the
// current page from the cursor history. Assumes consistent page sizes.
func ComputePageStart(state PageState) int {
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

// WithCountInfo fills in PageStart, PageLen, and TotalCount (when known).
// TotalCount is only set on the last page (HasNext=false) because that's the
// only case where the total is known exactly without an extra count query.
func (s PageState) WithCountInfo(itemLen int) PageState {
	s.PageStart = ComputePageStart(s)
	s.PageLen = itemLen

	if !s.HasNext && itemLen > 0 {
		s.TotalCount = strconv.Itoa(s.PageStart + itemLen - 1)
	}

	return s
}

// ParsePageSize reads the ?limit= query param, clamped to [1, MaxPageSize].
// Falls back to DefaultPageSize when unset.
func ParsePageSize(r *http.Request, defaultSize int) int {
	s := r.URL.Query().Get("limit")
	if s == "" {
		return defaultSize
	}

	num, err := strconv.Atoi(s)
	if err != nil || num < 1 {
		return defaultSize
	}

	if num > maxPageSize {
		return maxPageSize
	}

	return num
}

// ParseCursorParams extracts the after cursor and prev history from the
// request query string. Shared by all paginated index handlers.
func ParseCursorParams(r *http.Request) (string, string, bool) {
	after := r.URL.Query().Get("after")
	prevHistory := r.URL.Query().Get("prev")
	hasPrev := after != ""

	return after, prevHistory, hasPrev
}

// DefaultPageSize returns the default page size used across the dashboard.
func DefaultPageSize() int { return defaultPageSize }

// MaxPageSize returns the maximum allowed page size.
func MaxPageSize() int { return maxPageSize }
