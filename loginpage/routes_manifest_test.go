package loginpage

import (
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// TestRoutes_Manifest pins the page's single manifest row: public, at the
// mount root, with a reason.
func TestRoutes_Manifest(t *testing.T) {
	t.Parallel()

	handler, err := New(Config{Service: newTestService(t)})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	routes := handler.Routes()
	if len(routes) != 1 {
		t.Fatalf("login page manifest has %d rows, want 1", len(routes))
	}

	row := routes[0]
	if row.Method != "GET" || row.Pattern != "/" || row.Posture != cqrshtmx.PosturePublic || row.Why == "" {
		t.Fatalf("manifest row drifted: %+v", row)
	}
}
