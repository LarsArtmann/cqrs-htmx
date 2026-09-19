package dashboardui

import (
	"context"
	"strings"

	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
)

// healthKindToTone maps an internal health kind to a StatCard tone.
func healthKindToTone(kind string) display.StatTone {
	switch kind {
	case statusGood:
		return display.StatToneGreen
	case statusWarn:
		return display.StatToneYellow
	case statusBad:
		return display.StatToneRed
	default:
		return display.StatToneBlue
	}
}

// statCardHTML renders a library StatCard into a string. The valueID is the
// stable DOM hook (stat-<name> scheme) for future live-updating scripts —
// a markup refactor inside the card cannot break a script addressing the
// value through it.
func statCardHTML(ctx context.Context, valueID, value, label string, tone display.StatTone) string {
	var b strings.Builder

	//nolint:modernize // nested BaseProps is deliberate: promoted keys crash exhaustruct_v5 v5.0.3 (makeslice panic)
	props := display.StatCardProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Value:     value,
		Label:     label,
		Change:    "",
		Trend:     display.TrendNone,
		Tone:      tone,
		Icon:      "",
		Href:      "",
		HxGet:     "",
		HxTarget:  "",
		HxSwap:    "",
		ValueID:   valueID,
	}
	_ = display.StatCard(props).Render(ctx, &b)

	return b.String()
}
