package dashboardui

import (
	"fmt"
	"time"
)

const (
	hoursPerDay        = 24
	approxDaysPerMonth = 30
	daysPerYear        = 365
)

// relativeTime renders a human-readable relative time like "2 minutes ago".
// The full RFC3339 timestamp should be rendered in a tooltip (title attribute).
func relativeTime(t time.Time) string { //nolint:cyclop // natural switch over time ranges
	if t.IsZero() {
		return ""
	}

	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", minutes)
	case d < hoursPerDay*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	case d < approxDaysPerMonth*hoursPerDay*time.Hour:
		days := int(d.Hours() / hoursPerDay)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	case d < daysPerYear*hoursPerDay*time.Hour:
		months := int(d.Hours() / hoursPerDay / approxDaysPerMonth)
		if months == 1 {
			return "1 month ago"
		}

		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / hoursPerDay / daysPerYear)
		if years == 1 {
			return "1 year ago"
		}

		return fmt.Sprintf("%d years ago", years)
	}
}

// humanByteSize renders a byte count as a human-readable string (e.g., "12.3 KB").
func humanByteSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
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
