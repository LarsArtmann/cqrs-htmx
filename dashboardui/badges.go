package dashboardui

import (
	"github.com/larsartmann/templ-components/display"
)

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
