package dashboardui

import (
	"regexp"
	"strings"
	"testing"
)

// TestDashboardJSSSEBaseURL pins the client-side base-URL derivation in
// dashboardJS. The served script derives its mount base from its own src
// ("<base>/-/dashboard.js"), strips the "/dashboard.js" suffix, then strips
// the trailing "/-" segment before appending "/-/events/stream".
//
// Regression: the strip pattern was /\/-\/$/, which requires a slash AFTER
// the dash. The stripped src ends in "/-" (no trailing slash), so the strip
// never matched and every client requested "<base>/-/-/events/stream" —
// a guaranteed 404 on every deployment (found live on a consumer's
// https://host/cqrs/queries page, 2026-09-18).
func TestDashboardJSSSEBaseURL(t *testing.T) {
	// The broken and fixed patterns, extracted from the served script.
	brokenPattern := `var base = path.replace(/\/-\/$/, "");`
	if strings.Contains(dashboardJS, brokenPattern) {
		t.Fatalf("dashboardJS still contains the broken base-strip pattern %q (matches nothing; base keeps the trailing /- and the SSE URL gains a spurious /-/)", brokenPattern)
	}
	fixedPattern := `var base = path.replace(/\/-$/, "");`
	if !strings.Contains(dashboardJS, fixedPattern) {
		t.Fatalf("dashboardJS must derive base by stripping the trailing /- with %q", fixedPattern)
	}

	// Port the exact JS computation (the two replaces above, then the
	// stream URL suffix) to Go regexp and assert the wire outcome for the
	// URL shapes every deployment produces.
	scriptSuffix := regexp.MustCompile(`/dashboard\.js$`)
	baseStrip := regexp.MustCompile(`/-$`)
	streamURL := func(scriptSrc string) string {
		path := scriptSuffix.ReplaceAllString(scriptSrc, "")
		return baseStrip.ReplaceAllString(path, "") + "/-/events/stream"
	}

	tests := []struct {
		name      string
		scriptSrc string
		want      string
	}{
		{
			name:      "consumer mount /cqrs",
			scriptSrc: "https://host/cqrs/-/dashboard.js",
			want:      "https://host/cqrs/-/events/stream",
		},
		{
			name:      "default mount /dashboard",
			scriptSrc: "https://host/dashboard/-/dashboard.js",
			want:      "https://host/dashboard/-/events/stream",
		},
		{
			name:      "root mount without /- degrades unchanged",
			scriptSrc: "https://host/dash/dashboard.js",
			want:      "https://host/dash/-/events/stream",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := streamURL(tt.scriptSrc); got != tt.want {
				t.Fatalf("streamURL(%q) = %q, want %q", tt.scriptSrc, got, tt.want)
			}
		})
	}
}
