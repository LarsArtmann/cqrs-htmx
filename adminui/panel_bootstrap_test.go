package adminui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

func TestNew_NilService_ReturnsError(t *testing.T) {
	_, err := New(Config{
		Service: nil,
	})
	if err == nil {
		t.Error("expected error for nil Service")
	}
}

func TestPanel_Mount(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	panel.Mount(mux, "/admin/")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code == http.StatusNotFound {
		t.Error("Mount should register /admin/users route")
	}
}

func TestPanel_MountCoexistsWithRootIndex(t *testing.T) {
	svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{})
	panel, err := New(Config{Service: svc, Authorizer: RequireAuthenticated()})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	panel.Mount(mux, "/admin/")
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", rec.Code)
	}
}
