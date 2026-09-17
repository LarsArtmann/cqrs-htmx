package dashboardui

import (
	"context"
	"strings"

	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
)

// badgeHTML renders a library Badge with an explicit type into a string.
// Single mapping point for every former hand-rolled badge span (M7).
func badgeHTML(ctx context.Context, text string, badgeType display.BadgeType) string {
	var b strings.Builder

	props := display.BadgeProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Text:      text,
		Type:      badgeType,
		Size:      display.BadgeSizeMD,
		Pill:      false,
		Dot:       true,
		Href:      "",
	}
	_ = display.Badge(props).Render(ctx, &b)

	return b.String()
}

// statusKindToBadgeType maps an internal health kind to a badge type.
// Unknown kinds fall back to neutral, matching the pre-adoption default.
func statusKindToBadgeType(kind string) display.BadgeType {
	switch kind {
	case statusGood:
		return display.BadgeSuccess
	case statusWarn:
		return display.BadgeWarning
	case statusBad:
		return display.BadgeError
	default:
		return display.BadgeNeutral
	}
}

// encodingBadgeType maps an event encoding to a badge type. JSON is neutral
// (default), CBOR warns (may need a decoder), raw is neutral.
func encodingBadgeType(encoding string) display.BadgeType {
	switch encoding {
	case "cbor":
		return display.BadgeWarning
	default:
		return display.BadgeNeutral
	}
}

// countBadgeType maps a DLQ dead-letter count to a badge type: any pending
// letters are an error signal, zero is neutral.
func countBadgeType(count int) display.BadgeType {
	if count > 0 {
		return display.BadgeError
	}

	return display.BadgeNeutral
}
