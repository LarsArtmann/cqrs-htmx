package dashboardui

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
)

// updateGolden is set with -update to rewrite the golden files after an
// intentional markup change: go test ./... -run TestGolden -update
// Review the diff before committing regenerated goldens — they pin the exact
// markup the dashboard's live-updating scripts and CSP contract rely on.
var updateGolden = flag.Bool("update", false, "rewrite golden testdata files")

// goldenRender renders a component to a string through the hybrid path
// (the same call shape production renderers use: Render into a
// strings.Builder via the io.Writer interface).
func goldenRender(t *testing.T, c templ.Component) string {
	t.Helper()

	var b strings.Builder

	if err := c.Render(context.Background(), &b); err != nil {
		t.Fatalf("render: %v", err)
	}

	return b.String()
}

// TestGolden_ComponentMarkup pins the exact markup of the library components
// the dashboard adopts (status badge, stat card, empty state, button). Drift
// in these files = a markup change that can break ValueID-targeting scripts,
// CSP attributes, or the compiled CSS class coverage.
func TestGolden_ComponentMarkup(t *testing.T) {
	statusBadge := display.StatusBadge("healthy")
	badge := display.Badge(display.BadgeProps{
		ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: "",
		Text: "42",
		Type: display.BadgeNeutral,
		Size: display.BadgeSizeMD,
		Pill: false,
		Dot:  true,
		Href: "",
	})
	statCard := display.StatCard(display.StatCardProps{
		ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: "",
		Value:    "1234",
		Label:    "Events",
		Change:   "",
		Trend:    display.TrendNone,
		Tone:     display.StatToneBlue,
		Icon:     "",
		Href:     "",
		HxGet:    "",
		HxTarget: "",
		HxSwap:   "",
		ValueID:  "stat-total-events",
	})
	emptyState := display.EmptyState(display.EmptyStateProps{
		ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: "",
		Title:       "No events yet",
		TitleTag:    "h2",
		Description: "Events will appear here.",
		Icon:        "inbox",
		ActionText:  "",
		ActionHref:  "",
		ActionAttrs: nil,
	})

	cases := map[string]string{
		"status_badge_healthy.golden": goldenRender(t, statusBadge),
		"badge_neutral_dot.golden":    goldenRender(t, badge),
		"stat_card_value_id.golden":   goldenRender(t, statCard),
		"empty_state_h2.golden":       goldenRender(t, emptyState),
	}

	for name, got := range cases {
		path := filepath.Join("testdata", "golden", name)

		if *updateGolden {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}

			if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
				t.Fatalf("write golden: %v", err)
			}

			continue
		}

		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read golden %s (run with -update to create): %v", name, err)
		}

		if got != string(want) {
			t.Errorf("golden %s drifted — regenerate with -update and review the diff", name)
		}
	}
}
