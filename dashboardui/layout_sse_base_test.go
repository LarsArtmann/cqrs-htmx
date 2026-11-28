package dashboardui

import (
	"regexp"
	"strings"
	"testing"
)

// TestDashboardJSSSEBaseURL pins the client-side base-URL derivation in
// dashboardJS. The served script derives its mount base from its own src
// ("<base>/assets/dashboard.js") by stripping the "/assets/dashboard.js"
// suffix, then appends "/events/stream".
func TestDashboardJSSSEBaseURL(t *testing.T) {
	const basePattern = `var base = scriptSrc.replace(/\/assets\/dashboard\.js$/, "");`
	if !strings.Contains(dashboardJS, basePattern) {
		t.Fatalf("dashboardJS must derive base with %q", basePattern)
	}

	if strings.Contains(dashboardJS, "/-/") {
		t.Fatalf("dashboardJS must not reference the retired /-/ route prefix")
	}

	// Port the exact JS computation to Go regexp and assert the wire
	// outcome for the URL shapes every deployment produces.
	scriptSuffix := regexp.MustCompile(`/assets/dashboard\.js$`)
	streamURL := func(scriptSrc string) string {
		return scriptSuffix.ReplaceAllString(scriptSrc, "") + "/events/stream"
	}

	tests := []struct {
		name      string
		scriptSrc string
		want      string
	}{
		{
			name:      "consumer mount /cqrs",
			scriptSrc: "https://host/cqrs/assets/dashboard.js",
			want:      "https://host/cqrs/events/stream",
		},
		{
			name:      "default mount /dashboard",
			scriptSrc: "https://host/dashboard/assets/dashboard.js",
			want:      "https://host/dashboard/events/stream",
		},
		{
			name:      "root mount",
			scriptSrc: "https://host/assets/dashboard.js",
			want:      "https://host/events/stream",
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
