package dashboardui

import (
	"fmt"
	"time"
)

// relativeTime renders a human-readable relative time like "2 minutes ago".
// The full RFC3339 timestamp should be rendered in a tooltip (title attribute).
func relativeTime(t time.Time) string {
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
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case d < 365*24*time.Hour:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(d.Hours() / 24 / 365)
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

// timeCell renders a table cell with relative time and full timestamp tooltip.
func timeCell(t time.Time) string {
	return fmt.Sprintf(`<td class="mono" title="%s">%s</td>`,
		esc(t.Format(time.RFC3339)),
		esc(relativeTime(t)),
	)
}

// encodingBadgeClass returns the CSS badge class for an event encoding.
func encodingBadgeClass(encoding string) string {
	switch encoding {
	case "json", "":
		return "badge badge-ok"
	case "cbor":
		return "badge badge-warn"
	case "raw":
		return "badge badge-neutral"
	default:
		return "badge badge-neutral"
	}
}
