package dashboardui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/forms"
	"github.com/larsartmann/templ-components/utils"
)

// renderPagination renders Prev/Next links with cursor-history tracking.
// basePath is the dashboard base path, path is the page path (e.g., "/events").
// Extra query params (filters, sort) are preserved across pagination links.
func renderPagination(
	ctx context.Context,
	basePath, path string,
	state paginationState,
	extraParams string,
) string {
	if !state.HasNext && !state.HasPrev {
		return ""
	}

	var b strings.Builder

	b.WriteString(`<div class="pagination">`)

	if state.HasPrev {
		prevAfter, prevHistory := popCursor(state.PrevHistory)
		query := paginationQuery(prevAfter, prevHistory, state.PageSize, extraParams)
		b.WriteString(
			buttonLink(
				ctx,
				"← Previous",
				fmt.Sprintf("%s%s?%s", basePath, path, query),
				"",
				display.ButtonSecondary,
				false,
			),
		)
	} else {
		b.WriteString(`<span class="pagination disabled">← Previous</span>`)
	}

	b.WriteString(renderPaginationInfo(state))
	b.WriteString(renderPageSizeSelector(ctx, basePath, path, state, extraParams))

	if state.HasNext {
		nextHistory := pushCursor(state.PrevHistory, state.After)
		query := paginationQuery(state.NextCursor, nextHistory, state.PageSize, extraParams)
		b.WriteString(
			buttonLink(
				ctx,
				"Next →",
				fmt.Sprintf("%s%s?%s", basePath, path, query),
				"",
				display.ButtonOutlineInfo,
				false,
			),
		)
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
func renderPageSizeSelector(
	ctx context.Context,
	basePath, path string,
	state paginationState,
	extraParams string,
) string {
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

	var b strings.Builder

	props := forms.SelectProps{
		BaseProps: utils.BaseProps{
			ID:        "",
			Class:     "",
			Attrs:     nil,
			AriaLabel: "",
			Nonce:     "",
		},
		Name:     "limit",
		Label:    "Per page:",
		Options:  options,
		Groups:   nil,
		Required: false,
		Disabled: false,
		Stylable: false,
		Error:    "",
		HelpText: "",
	}
	_ = forms.Select(props).Render(ctx, &b)

	return b.String()
}
