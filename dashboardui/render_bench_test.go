package dashboardui

import (
	"context"
	"fmt"
	"html"
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
// see docs/benchmarks/dashboardui-render-2026-09-17.txt for the recorded run.

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
