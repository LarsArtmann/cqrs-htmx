package dashboardui

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/htmx"
)

// Spike (2026-09-22, TODO_LIST round-9): validates the templ-components
// v1.19.1 escape hatch for children-slot components in dashboardui's hybrid
// (strings.Builder) rendering path. PolledRegion's { children... } slot
// renders empty when the component is rendered straight into a builder —
// historically the reason the projection-health polling region stays
// hand-rolled (see README "Deliberate exclusions"). templ.WithChildren
// populates the slot programmatically; if this test passes, adoption of
// PolledRegion (and display.Grid) on the polling region is unblocked
// pending a golden/e2e churn decision.
func TestSpikePolledRegionWithChildrenInHybridPath(t *testing.T) {
	panelChild := panelSpikeChild()
	ctx := templ.WithChildren(context.Background(), panelChild)

	var b strings.Builder

	props := htmx.PolledRegionProps{
		ID:    "projection-health",
		URL:   "/-/partials/projection-health",
		Every: "10s",
		Swap:  htmx.SwapOuterHTML,
	}
	if err := htmx.PolledRegion(props).Render(ctx, &b); err != nil {
		t.Fatalf("render PolledRegion: %v", err)
	}

	html := b.String()

	for _, want := range []string{
		`id="projection-health"`,
		`hx-get="/-/partials/projection-health"`,
		`hx-trigger="every 10s"`,
		`hx-swap="outerHTML"`,
		`aria-live="polite"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered region missing %q\nhtml: %s", want, html)
		}
	}

	if !strings.Contains(html, "data-spike-child-marker") {
		t.Errorf("children slot rendered EMPTY (escape hatch not effective)\nhtml: %s", html)
	}
}

// panelSpikeChild stands in for the projection-health panel content
// (title + table) that would become the polled region's child.
func panelSpikeChild() templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := w.Write([]byte(`<div class="panel-title" data-spike-child-marker>Projection Health</div>`))

		return err
	})
}
