package dashboardui

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"
	"testing"

	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
)

// Benchmarks comparing the hand-rolled string-building renderers against the
// templ-components hybrid render path (component .Render into a
// strings.Builder). Run with:
//
//	go test -bench BenchmarkRender -benchmem -count=5 > bench.txt && benchstat bench.txt
//
// These quantify the adoption cost the program accepted per component family;
// see docs/benchmarks/dashboardui-render-2026-09-19.md for the recorded run
// (the 2026-09-17 artifact predates the honest statCard baseline and is kept
// as history only).

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

func benchHybridStatCard(b *testing.B) {
	b.Helper()

	ctx := context.Background()
	props := display.StatCardProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Value:     "1234",
		Label:     "Events",
		Change:    "",
		Trend:     display.TrendNone,
		Tone:      display.StatToneBlue,
		Icon:      "",
		Href:      "",
		HxGet:     "",
		HxTarget:  "",
		HxSwap:    "",
		ValueID:   "stat-total-events",
	}

	for b.Loop() {
		var out strings.Builder

		_ = display.StatCard(props).Render(ctx, &out)
		_ = out.String()
	}
}

func BenchmarkRenderStatCard(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledStatCard)
	b.Run("hybrid-library", benchHybridStatCard)
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

func benchHybridButtonLink(b *testing.B) {
	b.Helper()

	ctx := context.Background()

	for b.Loop() {
		var out strings.Builder

		link := buttonLink(
			ctx,
			"View",
			"/dashboard/events/e_01HXYZ",
			"View event",
			display.ButtonSecondary,
			false,
		)
		_, _ = out.WriteString(link)
		_ = out.String()
	}
}

// benchHandRolledButtonSubmit reproduces the pre-adoption danger submit
// button (git 6294d73e~1 dashboardui/handlers_dlq.go delete form).
func benchHandRolledButtonSubmit(b *testing.B) {
	b.Helper()

	esc := html.EscapeString

	for b.Loop() {
		var out strings.Builder

		fmt.Fprintf(
			&out,
			`<button type="submit" class="btn btn-danger" aria-label="%s">%s</button>`,
			esc("Delete dead letter"),
			esc("Delete"),
		)
		_ = out.String()
	}
}

func benchHybridButtonSubmit(b *testing.B) {
	b.Helper()

	ctx := context.Background()

	for b.Loop() {
		var out strings.Builder

		_, _ = out.WriteString(
			buttonSubmit(ctx, "Delete", "Delete dead letter", display.ButtonOutlineDanger, nil),
		)
		_ = out.String()
	}
}

func BenchmarkRenderButton(b *testing.B) {
	b.Run("link-hand-rolled", benchHandRolledButtonLink)
	b.Run("link-hybrid", benchHybridButtonLink)
	b.Run("submit-hand-rolled", benchHandRolledButtonSubmit)
	b.Run("submit-hybrid", benchHybridButtonSubmit)
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

func benchHybridEmptyState(b *testing.B) {
	b.Helper()

	ctx := context.Background()

	for b.Loop() {
		var out strings.Builder

		_, _ = out.WriteString(emptyState(ctx, "No events", "Adjust the filters"))
		_ = out.String()
	}
}

func BenchmarkRenderEmptyState(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledEmptyState)
	b.Run("hybrid-library", benchHybridEmptyState)
}

// The pre-M20 metadata markup (git 21c13e22 dashboardui/handler_overview.go
// metaRow/metaRowCopyable). metaRowCopyable already rendered the library
// CopyButton (M13 predates M20), so the measured delta is the
// table.meta-table → display.DefinitionList structure swap only - the copy
// button cost is identical on both sides.

func benchMetaRow(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, `<tr><td class="meta-key">%s</td><td class="meta-val">%s</td></tr>`, key, value)
}

func benchMetaRowCopyable(
	b *strings.Builder,
	ctx context.Context,
	key, displayValue, rawValue string,
) {
	fmt.Fprintf(
		b,
		`<tr><td class="meta-key">%s</td><td class="meta-val">%s %s</td></tr>`,
		key,
		displayValue,
		copyButtonHTML(ctx, rawValue, ""),
	)
}

func benchHandRolledDefinitionList(b *testing.B) {
	b.Helper()

	ctx := context.Background()

	for b.Loop() {
		var out strings.Builder

		out.WriteString(`<div><h3>Metadata</h3><table class="meta-table">`)
		benchMetaRow(&out, "Stream Type", esc("user"))
		benchMetaRowCopyable(&out, ctx, "Stream ID", esc("01HXYZ"), "01HXYZ")
		benchMetaRow(&out, "Version", esc("42"))
		benchMetaRow(&out, "Schema Version", esc("1"))
		benchMetaRow(&out, "Encoding", esc("json"))
		benchMetaRow(&out, "Occurred At", esc("2026-09-19T10:00:00Z"))
		benchMetaRowCopyable(&out, ctx, "Correlation ID", esc("01HCORR"), "01HCORR")
		out.WriteString(`</table></div>`)
		_ = out.String()
	}
}

func benchHybridDefinitionList(b *testing.B) {
	b.Helper()

	ctx := context.Background()

	for b.Loop() {
		var out strings.Builder

		_, _ = out.WriteString(definitionListHTML(ctx, []display.DefinitionItem{
			defItem("Stream Type", "user"),
			defItemCopy("Stream ID", "01HXYZ", "01HXYZ"),
			defItem("Version", "42"),
			defItem("Schema Version", "1"),
			defItem("Encoding", "json"),
			defItem("Occurred At", "2026-09-19T10:00:00Z"),
			defItemCopy("Correlation ID", "01HCORR", "01HCORR"),
		}))
		_ = out.String()
	}
}

func BenchmarkRenderDefinitionList(b *testing.B) {
	b.Run("hand-rolled", benchHandRolledDefinitionList)
	b.Run("hybrid-library", benchHybridDefinitionList)
}

// benchTableDataRows measures the typed data-row Table path (tableHTML +
// TableCell values): 10 rows mixing escaped text and pre-rendered cells.
func benchTableDataRows(b *testing.B) {
	b.Helper()

	ctx := context.Background()
	headers := plainHeaders("ID", "Type", "Version", "")

	for b.Loop() {
		rows := make([]display.TableRow, 0, 10)

		for j := range 10 {
			rowID := fmt.Sprintf("01HXYZ%02d", j)
			rows = append(rows, display.TableRow{
				Cells: []display.TableCell{
					textCell(rowID),
					textCell("user.created"),
					textCell(strconv.Itoa(j)),
					rawCell(`<a href="/dashboard/events/e` + rowID + `" class="btn">View</a>`),
				},
			})
		}

		var out strings.Builder

		_, _ = out.WriteString(tableHTML(ctx, headers, rows, "bench-tbody"))
		_ = out.String()
	}
}

// benchTableRawBody measures the raw-body Table path (tableHTMLRaw): the same
// 10 rows pre-rendered as <tr> markup around the library shell.
func benchTableRawBody(b *testing.B) {
	b.Helper()

	ctx := context.Background()
	headers := plainHeaders("ID", "Type", "Version", "")

	for b.Loop() {
		var rows strings.Builder

		for j := range 10 {
			fmt.Fprintf(
				&rows,
				`<tr><td class="mono">%s</td><td>%s</td><td>%d</td><td><a href="/dashboard/events/e%02d" class="btn">View</a></td></tr>`,
				fmt.Sprintf("01HXYZ%02d", j),
				"user.created",
				j,
				j,
			)
		}

		var out strings.Builder

		_, _ = out.WriteString(tableHTMLRaw(ctx, headers, rows.String()))
		_ = out.String()
	}
}

func BenchmarkRenderTablePaths(b *testing.B) {
	b.Run("data-row", benchTableDataRows)
	b.Run("raw-body", benchTableRawBody)
}
