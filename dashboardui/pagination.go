package dashboardui

import (
	"strconv"

	"github.com/larsartmann/templ-components/forms"
)

var pageSizeOptions = []int{25, 50, 100, 200}

// prevPaginationHref builds the Previous link target, popping the most
// recent cursor off the history stack.
func prevPaginationHref(basePath, path string, state paginationState, extraParams string) string {
	prevAfter, prevHistory := popCursor(state.PrevHistory)
	query := paginationQuery(prevAfter, prevHistory, state.PageSize, extraParams)

	return basePath + path + "?" + query
}

// nextPaginationHref builds the Next link target, pushing the current
// cursor onto the history stack.
func nextPaginationHref(basePath, path string, state paginationState, extraParams string) string {
	nextHistory := pushCursor(state.PrevHistory, state.After)
	query := paginationQuery(state.NextCursor, nextHistory, state.PageSize, extraParams)

	return basePath + path + "?" + query
}

// paginationInfoText renders the "Showing X–Y of Z" label text when a page
// was cut. Empty when the whole result set fits (no pagination bar info).
func paginationInfoText(state paginationState) string {
	if state.PageLen == 0 {
		return ""
	}

	start := state.PageStart
	if start < 1 {
		start = 1
	}

	end := start + state.PageLen - 1
	if end < start {
		end = start
	}

	label := "Showing " + strconv.Itoa(start) + "–" + strconv.Itoa(end)
	if state.TotalCount != "" {
		label += " of " + state.TotalCount
	}

	return label
}

// pageSizeOptionsFor builds the library select options. Each option
// navigates to the same path with the new limit, preserving active filters
// but resetting cursor position.
func pageSizeOptionsFor(basePath, path string, state paginationState, extraParams string) []forms.SelectOption {
	current := state.PageSize
	if current == 0 {
		current = defaultPageSize
	}

	options := make([]forms.SelectOption, 0, len(pageSizeOptions))

	for _, opt := range pageSizeOptions {
		query := "limit=" + strconv.Itoa(opt)
		if extraParams != "" {
			query += "&" + extraParams
		}

		options = append(options, forms.SelectOption{
			Value:    basePath + path + "?" + query,
			Label:    strconv.Itoa(opt),
			Disabled: false,
			Selected: opt == current,
		})
	}

	return options
}
