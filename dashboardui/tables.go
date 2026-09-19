package dashboardui

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
)

// tableHTML renders the library Table into a string (the hybrid adoption
// path: display.Table.Render into the existing strings.Builder — no templ
// conversion). TypedHeaders drive aria-sort + server-side sort links; rows
// are plain data, so the templ-children limitation that blocked display.Grid
// does not apply. Cells containing markup (links, badges, copy buttons) pass
// through templ.Raw; plain text cells should prefer TableCell.Text (escaped).
// LazyRows is on: the dashboard's listings render content-visibility rows, so
// long tables skip off-screen row rendering.
func tableHTML(ctx context.Context, headers []display.TableHeader, rows []display.TableRow, bodyID string) string {
	var b strings.Builder

	props := display.TableProps{
		ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: "",
		Caption:      "",
		Headers:      nil,
		TypedHeaders: headers,
		Rows:         rows,
		Striped:      false,
		Hover:        true,
		Bordered:     false,
		Flush:        false,
		CellPadding:  display.TableCellPaddingCompact,
		LazyRows:     true,
		Body:         nil,
		BodyID:       bodyID,
	}
	_ = display.Table(props).Render(ctx, &b)

	return b.String()
}

// rawCell wraps pre-rendered cell HTML (links, copy buttons, badges) as a
// table cell. Only pass trusted, already-escaped markup.
func rawCell(html string) display.TableCell {
	return display.TableCell{Text: "", Content: templ.Raw(html)}
}

// textCell builds an escaped plain-text table cell.
func textCell(text string) display.TableCell {
	return display.TableCell{Text: text, Content: nil}
}

// plainHeaders builds non-sortable typed headers from labels. An empty label
// renders the headerless actions column convention.
func plainHeaders(labels ...string) []display.TableHeader {
	headers := make([]display.TableHeader, 0, len(labels))

	for _, l := range labels {
		headers = append(headers, display.TableHeader{
			Label:         l,
			Sortable:      false,
			SortDirection: display.SortNone,
			Href:          "",
		})
	}

	return headers
}

// tableHTMLRaw renders the library Table shell (wrapper, thead, tbody) around
// pre-rendered <tr> markup. Use for tables whose rows are built as strings by
// existing renderers; the headers still go through the typed path for
// consistent styling.
func tableHTMLRaw(ctx context.Context, headers []display.TableHeader, rowsHTML string) string {
	var b strings.Builder

	props := display.TableProps{
		ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: "",
		Caption:      "",
		Headers:      nil,
		TypedHeaders: headers,
		Rows:         nil,
		Striped:      false,
		Hover:        true,
		Bordered:     false,
		Flush:        false,
		CellPadding:  display.TableCellPaddingCompact,
		LazyRows:     true,
		Body:         templ.Raw(rowsHTML),
		BodyID:       "",
	}
	_ = display.Table(props).Render(ctx, &b)

	return b.String()
}
