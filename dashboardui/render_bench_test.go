package dashboardui

import (
	"context"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/icons"
)

// Benchmarks comparing the pre-adoption hand-rolled string-building renderers
// against the production templ component path (the same components the pages
// render). Run with:
//
//	go test -bench BenchmarkRender -benchmem -count=5 > bench.txt && benchstat bench.txt
//
// The 2026-09-19 artifact (docs/benchmarks/dashboardui-render-2026-09-19.md)
// measured the strings.Builder hybrid bridge; the "templ" arms here measure
// the full-templ migration path (2026-09-22) and are not directly comparable
// to the recorded hybrid numbers.

// benchRender renders a component into a reused builder per iteration — the
// same call shape production HTMX responses use.
func benchRender(b *testing.B, c templ.Component) {
	b.Helper()

	ctx := context.Background()

	for b.Loop() {
		var out strings.Builder

		if err := c.Render(ctx, &out); err != nil {
			b.Fatal(err)
		}

		_ = out.String()
	}
}

// benchHandRolledStatCard reproduces the pre-adoption statCard helper
// verbatim (git show 81088b64:dashboardui/handler_overview.go), including the
// html escaping that was part of the real per-card work - the original
// benchmark under-counted by rendering an unescaped, context-free approximation.
func benchHandRolledStatCard(b *testing.B) {
	b.Helper()

	esc := html.EscapeString

	for b.Loop() {
		var out strings.Builder

		fmt.Fprintf(&out, `<div class="stat-card">`)
		fmt.Fprintf(&out, `<div class="stat-card-value">%s</div>`, esc("1234"))
		fmt.Fprintf(&out, `<div class="stat-card-label">%s</div>`, esc("Events"))
		out.WriteString(`</div>`)
		_ = out.String()
	}
}

func benchTemplStatCard(b *testing.B) {
	b.Helper()

	statCard := statCard("stat-total-events", "1234", "Events", display.StatToneBlue)

	benchRender(b, statCard)
}

func BenchmarkRenderStatCard(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledStatCard)
	b.Run("templ", benchTemplStatCard)
}

// benchHandRolledButtonLink reproduces the pre-adoption anchor-button markup
// (git 6294d73e~1 dashboardui/handlers_events.go, "Previous"/"Next" nav),
// including the escaping the real sites performed.
func benchHandRolledButtonLink(b *testing.B) {
	b.Helper()

	esc := html.EscapeString

	for b.Loop() {
		var out strings.Builder

		fmt.Fprintf(
			&out,
			`<a href="%s" class="btn">%s</a>`,
			esc("/dashboard/events/e_01HXYZ"),
			esc("View"),
		)
		_ = out.String()
	}
}

func benchTemplButtonLink(b *testing.B) {
	b.Helper()

	benchRender(b, buttonLink(
		"View",
		"/dashboard/events/e_01HXYZ",
		"View event",
		display.ButtonSecondary,
		false,
	))
}

func BenchmarkRenderButtonLink(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledButtonLink)
	b.Run("templ", benchTemplButtonLink)
}

// benchHandRolledEmptyState reproduces the pre-adoption empty-state panel
// verbatim (git 6294d73e~1 dashboardui/render.go emptyState, pre-M9).
func benchHandRolledEmptyState(b *testing.B) {
	b.Helper()

	esc := html.EscapeString

	for b.Loop() {
		var out strings.Builder

		fmt.Fprintf(
			&out,
			`<div class="empty-state"><h2>%s</h2><p>%s</p></div>`,
			esc("No events"),
			esc("Adjust the filters"),
		)
		_ = out.String()
	}
}

func benchTemplEmptyState(b *testing.B) {
	b.Helper()

	benchRender(b, emptyStatePanel(icons.Inbox, "No events", "Adjust the filters"))
}

func BenchmarkRenderEmptyState(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledEmptyState)
	b.Run("templ", benchTemplEmptyState)
}

// The pre-M20 metadata markup (git 21c13e22 dashboardui/handler_overview.go
// metaRow/metaRowCopyable). metaRowCopyable already rendered the library
// CopyButton (M13 predates M20), so the measured delta is the
// table.meta-table → display.DefinitionList structure swap only - the copy
// button cost is identical on both sides.

func benchMetaRow(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, `<tr><td class="meta-key">%s</td><td class="meta-val">%s</td></tr>`, key, value)
}

func benchMetaRowCopyable(b *strings.Builder, key, displayValue, rawValue string) {
	fmt.Fprintf(
		b,
		`<tr><td class="meta-key">%s</td><td class="meta-val">%s @copy(%s)</td></tr>`,
		key,
		displayValue,
		rawValue,
	)
}

func benchHandRolledDefinitionList(b *testing.B) {
	b.Helper()

	for b.Loop() {
		var out strings.Builder

		out.WriteString(`<div><h3>Metadata</h3><table class="meta-table">`)
		benchMetaRow(&out, "Stream Type", esc("user"))
		benchMetaRowCopyable(&out, "Stream ID", esc("01HXYZ"), "01HXYZ")
		benchMetaRow(&out, "Version", esc("42"))
		benchMetaRow(&out, "Schema Version", esc("1"))
		benchMetaRow(&out, "Encoding", esc("json"))
		benchMetaRow(&out, "Occurred At", esc("2026-09-19T10:00:00Z"))
		benchMetaRowCopyable(&out, "Correlation ID", esc("01HCORR"), "01HCORR")
		out.WriteString(`</table></div>`)
		_ = out.String()
	}
}

func benchTemplDefinitionList(b *testing.B) {
	b.Helper()

	benchRender(b, definitionList([]display.DefinitionItem{
		defItem("Stream Type", "user"),
		defItemCopy("Stream ID", "01HXYZ", "01HXYZ"),
		defItem("Version", "42"),
		defItem("Schema Version", "1"),
		defItem("Encoding", "json"),
		defItem("Occurred At", "2026-09-19T10:00:00Z"),
		defItemCopy("Correlation ID", "01HCORR", "01HCORR"),
	}))
}

func BenchmarkRenderDefinitionList(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledDefinitionList)
	b.Run("templ", benchTemplDefinitionList)
}

// benchTableDataRows measures the typed data-row Table path (display.Table
// with TableCell values): 10 rows mixing escaped text and pre-rendered cells.
func benchTableDataRows(b *testing.B) {
	b.Helper()

	headers := plainHeaders("ID", "Type", "Version", "")

	for b.Loop() {
		rows := make([]display.TableRow, 0, 10)

		for j := range 10 {
			rowID := fmt.Sprintf("01HXYZ%02d", j)
			rows = append(rows, display.TableRow{
				Cells: []display.TableCell{
					{Text: rowID, Content: nil},
					{Text: "user.created", Content: nil},
					{Text: strconv.Itoa(j), Content: nil},
					{Text: "", Content: templ.Raw(`<a href="/dashboard/events/e` + rowID + `" class="btn">View</a>`)},
				},
			})
		}

		benchRender(b, dataTable(headers, rows, "bench-tbody"))
	}
}

// benchTableRawBody measures the raw-body Table path (rawDataTable): the same
// 10 rows as a component of <tr> markup around the library shell.
func benchTableRawBody(b *testing.B) {
	b.Helper()

	headers := plainHeaders("ID", "Type", "Version", "")

	for b.Loop() {
		benchRender(b, rawDataTable(headers, "bench-tbody", benchRowsComponent(10)))
	}
}

func benchRowsComponent(n int) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for j := range n {
			if _, err := fmt.Fprintf(
				w,
				`<tr><td class="mono">%s</td><td>%s</td><td>%d</td><td><a href="/dashboard/events/e%02d" class="btn">View</a></td></tr>`,
				fmt.Sprintf("01HXYZ%02d", j),
				"user.created",
				j,
				j,
			); err != nil {
				return err
			}
		}

		return nil
	})
}

func BenchmarkRenderTablePaths(b *testing.B) {
	b.Run("data-row", benchTableDataRows)
	b.Run("raw-body", benchTableRawBody)
}
