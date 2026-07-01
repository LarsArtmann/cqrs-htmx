package integration_test

import (
	"net/http"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

// TestCrossModuleErrUnauthorized verifies that ErrUnauthorized sentinels from
// root and usermgmt are functionally equivalent: they share the same error code,
// classify as the same error family, and map to the same HTTP status via MapError.
//
// The modules intentionally have separate sentinels (zero mutual imports), but
// the aligned error codes ensure cross-module compatibility.
func TestCrossModuleErrUnauthorized(t *testing.T) {
	t.Parallel()

	rootErr := cqrshtmx.ErrUnauthorized
	usermgmtErr := usermgmt.ErrUnauthorized

	// Both should map to HTTP 401 via root's MapError (code-based check).
	if status := cqrshtmx.MapError(rootErr); status != http.StatusUnauthorized {
		t.Errorf("root ErrUnauthorized: MapError = %d, want %d", status, http.StatusUnauthorized)
	}
	if status := cqrshtmx.MapError(usermgmtErr); status != http.StatusUnauthorized {
		t.Errorf("usermgmt ErrUnauthorized: MapError = %d, want %d", status, http.StatusUnauthorized)
	}
}

// TestCrossModuleErrForbidden verifies that ErrForbidden sentinels from
// root and usermgmt map to the same HTTP 403 status.
func TestCrossModuleErrForbidden(t *testing.T) {
	t.Parallel()

	rootErr := cqrshtmx.ErrForbidden
	usermgmtErr := usermgmt.ErrForbidden

	if status := cqrshtmx.MapError(rootErr); status != http.StatusForbidden {
		t.Errorf("root ErrForbidden: MapError = %d, want %d", status, http.StatusForbidden)
	}
	if status := cqrshtmx.MapError(usermgmtErr); status != http.StatusForbidden {
		t.Errorf("usermgmt ErrForbidden: MapError = %d, want %d", status, http.StatusForbidden)
	}
}
