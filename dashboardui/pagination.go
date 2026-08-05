package dashboardui

import (
	"fmt"
	"strconv"
	"strings"
)

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
	b.WriteString(renderPageSizeSelector(basePath, path, state, extraParams))

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

var pageSizeOptions = []int{25, 50, 100, 200}

// renderPageSizeSelector renders a dropdown for choosing items per page.
// Changing the selection navigates to the same path with the new limit,
// preserving active filters but resetting cursor position.
func renderPageSizeSelector(basePath, path string, state paginationState, extraParams string) string {
	current := state.PageSize
	if current == 0 {
		current = defaultPageSize
	}

	var b strings.Builder
	b.WriteString(
		`<span class="page-size-selector"><label>Per page: <select onchange="window.location.href=this.value">`,
	)

	for _, opt := range pageSizeOptions {
		selected := ""
		if opt == current {
			selected = " selected"
		}

		query := "limit=" + strconv.Itoa(opt)
		if extraParams != "" {
			query += "&" + extraParams
		}

		fmt.Fprintf(&b, `<option value="%s%s?%s"%s>%d</option>`, basePath, path, query, selected, opt)
	}

	b.WriteString(`</select></label></span>`)

	return b.String()
}
