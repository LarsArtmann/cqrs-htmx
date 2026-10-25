package adminui

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"
)

// errorComponent is a templ.Component whose Render always fails, standing in
// for the mid-render writer/template failures renderPage/renderPartial must
// translate into non-200 responses.
type errorComponent struct{}

func (errorComponent) Render(context.Context, templ.Writer) error {
	return errors.New("boom") // #nosec G101 -- test sentinel, not a credential
}

// TestRenderPartial_RenderErrorAnswers500 pins S36/w02.4: a failed partial
// render must answer 500. Pre-fix, renderPartial only logged the error, the
// response stayed a truncated 200, and htmx silently swapped the broken
// fragment into the DOM.
func TestRenderPartial_RenderErrorAnswers500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/partials/stats", nil)

	renderPartial(rec, req, errorComponent{})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("renderPartial error status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := rec.Body.String(); body == "" {
		t.Error("renderPartial error body must not be empty")
	}
}

// TestRenderPage_RenderErrorAnswers500 pins the page-side twin so the two
// render paths cannot drift apart again.
func TestRenderPage_RenderErrorAnswers500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)

	renderPage(rec, req, errorComponent{})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("renderPage error status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
