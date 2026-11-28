package integration_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4"
)

// dashboardRoutePattern returns the GET pattern of the first route in the
// dashboard's own manifest whose path ends in suffix.
//
// The manifest is the source of truth because this module's tests run against
// two dashboardui trees: the workspace build resolves the local one and CI
// (GOWORK=off) resolves the published tag pinned in go.mod. Between a route
// change and the tag that carries it, the two disagree on a path, so a
// hard-coded one is red on exactly one side. Looking the path up keeps these
// tests about the wire contract, not the spelling of a route.
func dashboardRoutePattern(t *testing.T, dash *dashboardui.Dashboard, suffix string) string {
	t.Helper()

	for _, route := range dash.Routes() {
		if route.Method == http.MethodGet && strings.HasSuffix(route.Pattern, suffix) {
			return route.Pattern
		}
	}

	t.Fatalf("dashboardui exposes no GET route ending in %q", suffix)

	return ""
}
