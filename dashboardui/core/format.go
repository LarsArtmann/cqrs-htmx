package core

import (
	"time"

	"github.com/dustin/go-humanize"
)

// RelativeTime renders a human-readable relative time like "2 minutes ago".
// The full RFC3339 timestamp should be rendered in a tooltip (title attribute).
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	if t.After(time.Now().Add(-time.Minute)) {
		return "just now"
	}

	return humanize.RelTime(t, time.Now(), "ago", "from now")
}

// HumanByteSize renders a byte count as a human-readable string in IEC binary
// units (e.g., "12.3 KiB").
func HumanByteSize(bytes int) string {
	return humanize.IBytes(uint64(bytes))
}
