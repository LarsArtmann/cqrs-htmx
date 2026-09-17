package dashboardui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	"github.com/larsartmann/templ-components/errorpage"
)

func TestStatusToFamily(t *testing.T) {
	cases := []struct {
		status int
		want   errorpage.Family
	}{
		{http.StatusBadRequest, errorpage.FamilyRejection},
		{http.StatusNotFound, errorpage.FamilyRejection},
		{http.StatusForbidden, errorpage.FamilyRejection},
		{http.StatusConflict, errorpage.FamilyConflict},
		{http.StatusServiceUnavailable, errorpage.FamilyTransient},
		{http.StatusBadGateway, errorpage.FamilyTransient},
		{http.StatusGatewayTimeout, errorpage.FamilyTransient},
		{http.StatusInternalServerError, errorpage.FamilyInfrastructure},
		{http.StatusNotImplemented, errorpage.FamilyInfrastructure},
	}

	for _, tc := range cases {
		if got := statusToFamily(tc.status); got != tc.want {
			t.Errorf("statusToFamily(%d) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestRenderError_StyledFamilyPage(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	d.renderError(rec, httptest.NewRequest(http.MethodGet, "/dashboard/events", nil),
		http.StatusInternalServerError, "failed to load events")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"<!DOCTYPE html>",
		"/-/dashboard-tw.css",
		"error-shell",
		"failed to load events",
		"Internal Server Error",
		// family styling: infrastructure renders in gray
		"bg-gray-100",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("error page missing %q", want)
		}
	}
}

func TestRenderError_HTMXGetsBareCard(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/events", nil)
	req.Header.Set("Hx-Request", "true")

	rec := httptest.NewRecorder()
	d.renderError(rec, req, http.StatusNotFound, "event not found")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	body := rec.Body.String()
	if strings.Contains(body, "<!DOCTYPE") {
		t.Error("HTMX error response must not contain the full shell")
	}

	if !strings.Contains(body, "event not found") || !strings.Contains(body, "bg-amber-") {
		t.Error("HTMX error response missing card content or rejection styling")
	}
}

func TestRenderError_NilRequestStillWrites(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	d.renderError(rec, nil, http.StatusBadRequest, "invalid input")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "invalid input") {
		t.Error("nil-request error response missing message")
	}
}

func TestNotFound404_LibraryComponentShape(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	d.Mount(mux, "/dashboard/")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard/definitely-not-a-page", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Page not found", "404", "Overview", "/dashboard/"} {
		if !strings.Contains(body, want) {
			t.Errorf("404 page missing %q", want)
		}
	}
}

func TestPageData_CarriesNonce(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard/", nil)
	p := d.page("Overview", "/", req)

	// No nonce middleware ran, so the nonce is empty but the field is plumbed.
	if p.Nonce != "" {
		t.Errorf("Nonce = %q, want empty without middleware", p.Nonce)
	}
}
