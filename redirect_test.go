package cqrshtmx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSafeRedirectPath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", "/"},
		{"root", "/", "/"},
		{"simple path", "/users", "/users"},
		{"nested path", "/admin/users/42", "/admin/users/42"},
		{"query only path", "/?next=/x", "/?next=/x"},
		{"protocol-relative", "//evil.com", "/"},
		{"protocol-relative with path", "//evil.com/a", "/"},
		{"backslash bypass", `/\/evil.com`, "/"},
		{"backslash only", `/\evil.com`, "/"},
		{"backslash after segment", "/a/\\evil.com", "/a/\\evil.com"},
		{"scheme-bearing", "https://evil.com", "/"},
		{"relative", "users", "/"},
		{"dot-dot stays relative", "/a/../b", "/a/../b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeRedirectPath(tt.in); got != tt.want {
				t.Errorf("SafeRedirectPath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHTMXRedirect(t *testing.T) {
	t.Run("HTMX request sets Hx-Redirect header", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		req.Header.Set("Hx-Request", "true")

		HTMXRedirect(rec, req, "/dashboard")

		if got := rec.Header().Get("Hx-Redirect"); got != "/dashboard" {
			t.Errorf("Hx-Redirect = %q, want /dashboard", got)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("non-HTMX request issues 303", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/login", nil)

		HTMXRedirect(rec, req, "/dashboard")

		if got := rec.Header().Get("Location"); got != "/dashboard" {
			t.Errorf("Location = %q, want /dashboard", got)
		}
		if rec.Code != http.StatusSeeOther {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusSeeOther)
		}
	})
}
