package dashboardui

import (
	"context"
	"fmt"
	"strings"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
)

// renderTempl renders a templ.Component to a string. Used to integrate
// templ-components into the existing string-builder rendering pipeline.
func renderTempl(ctx context.Context, component templ.Component) string {
	var b strings.Builder

	if err := component.Render(ctx, &b); err != nil {
		return ""
	}

	return b.String()
}

// renderStatCards renders stat cards using templ-components display.StatCard.
func renderStatCardsTempl(ctx context.Context, stats overviewStats) string {
	var b strings.Builder

	if stats.TotalEvents != "" {
		b.WriteString(renderTempl(ctx, display.StatCard(display.StatCardProps{
			Value: stats.TotalEvents,
			Label: "Events",
		})))
	}

	if stats.TotalAggregates != "" {
		b.WriteString(renderTempl(ctx, display.StatCard(display.StatCardProps{
			Value: stats.TotalAggregates,
			Label: "Aggregates",
		})))
	}

	if len(stats.Projections) > 0 {
		active := 0

		for _, p := range stats.Projections {
			if p.StatusKind == "good" {
				active++
			}
		}

		b.WriteString(renderTempl(ctx, display.StatCard(display.StatCardProps{
			Value: fmt.Sprintf("%d/%d", active, len(stats.Projections)),
			Label: "Projections",
		})))
	}

	return b.String()
}
