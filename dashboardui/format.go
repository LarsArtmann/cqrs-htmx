package dashboardui

import (
	"time"

	"github.com/dustin/go-humanize"
)

// relativeTime renders a human-readable relative time like "2 minutes ago".
// The full RFC3339 timestamp should be rendered in a tooltip (title attribute).
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	if d := time.Since(t); d < time.Minute {
		return "just now"
	}

	return humanize.RelTime(t, time.Now(), "ago", "from now")
}

// humanByteSize renders a byte count as a human-readable string in IEC binary units (e.g., "12.3 KiB").
func humanByteSize(bytes int) string {
	return humanize.IBytes(uint64(bytes))
}

// encodingBadgeClass returns the CSS badge class for an event encoding.
// JSON is neutral (default), CBOR is a warning (may need decoder), raw is neutral.
func encodingBadgeClass(encoding string) string {
	switch encoding {
	case "json", "":
		return badgeNeutral
	case "cbor":
		return badgeWarn
	case "raw":
		return badgeNeutral
	default:
		return badgeNeutral
	}
}
