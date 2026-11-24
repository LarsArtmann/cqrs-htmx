package adminui

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// errorComponent is a templ.Component that always returns an error from Render.
type errorComponent struct{}

func (errorComponent) Render(_ context.Context, _ io.Writer) error {
	return errors.New("render failed")
}

func TestRenderPage_RenderError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	renderPage(w, r, errorComponent{})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("renderPage with error component: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRenderPartial_RenderError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	renderPartial(w, r, errorComponent{})
	// S36: a failed partial render must answer 500 — htmx swaps 200 responses
	// into the DOM, so a truncated fragment would silently corrupt the page.
	if w.Code != http.StatusInternalServerError {
		t.Errorf("renderPartial with error component: status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
