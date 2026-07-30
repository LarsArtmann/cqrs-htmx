package dashboardui

import (
	"fmt"
	"net/http"
	"strconv"
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
}

// renderPagination renders pagination controls (Prev/Next links).
// basePath is the dashboard base path, path is the page path (e.g., "/events").
// Extra query params (filters, sort) are preserved across pagination links.
func renderPagination(basePath, path string, state paginationState, extraParams string) string {
	if !state.HasNext && !state.HasPrev {
		return ""
	}

	var b string

	params := ""
	if extraParams != "" {
		params = "&" + extraParams
	}

	if state.HasPrev {
		b = fmt.Sprintf(`<a href="%s%s" class="btn">← Previous</a>`, basePath, path)
	} else {
		b = `<span class="pagination disabled">← Previous</span>`
	}

	if state.HasNext {
		b += fmt.Sprintf(`<a href="%s%s?after=%s&limit=%d%s" class="btn btn-accent">Next →</a>`,
			basePath, path, state.NextCursor, state.PageSize, params)
	}

	return fmt.Sprintf(`<div class="pagination">%s</div>`, b)
}

// parsePageSize reads the ?limit= query param, clamped to [1, maxPageSize].
// Falls back to defaultPageSize when unset.
func parsePageSize(r *http.Request, defaultPageSize int) int {
	s := r.URL.Query().Get("limit")
	if s == "" {
		return defaultPageSize
	}

	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return defaultPageSize
	}

	if n > maxPageSize {
		return maxPageSize
	}

	return n
}
