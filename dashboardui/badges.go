package dashboardui

import (
	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
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

// fateBadgeType maps a command fate to a badge type: failures and dead
// letters are errors, rejections and retries are warnings, completions are
// successes. Unknown fates (lifecycle event types newer than this build)
// stay neutral so schema drift is visible without coloring judgment.
func fateBadgeType(fate string) display.BadgeType {
	switch fate {
	case core.FateFailed, core.FateDeadLettered:
		return display.BadgeError
	case core.FateRejected, core.FateRetried:
		return display.BadgeWarning
	case core.FateCompleted:
		return display.BadgeSuccess
	default:
		return display.BadgeNeutral
	}
}
