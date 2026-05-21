package integration_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx"
	"github.com/larsartmann/cqrs-htmx/usermgmt"
	"github.com/larsartmann/go-cqrs-lite/core/command"
)

func TestUsermgmtBridge_AsEnforcer(t *testing.T) {
	authz, err := usermgmt.NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}

	enforcer := authz.AsEnforcer()
	disp := command.NewDispatcher()
	app, err := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
		Enforcer: enforcer,
	})
	if err != nil {
		t.Fatalf("New App: %v", err)
	}
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestUsermgmtBridge_UserIDFromRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	id := usermgmt.UserIDFromRequest(req)
	if id != "" {
		t.Errorf("expected empty ID from unauthenticated request, got %q", id)
	}
}
