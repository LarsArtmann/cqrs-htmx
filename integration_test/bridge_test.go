package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v3"
	"github.com/larsartmann/go-cqrs-lite/command/v3"
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

func TestUsermgmtBridge_FullRegisterAuthCycle(t *testing.T) {
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	ctx := context.Background()
	uid := cqrshtmx.NewUserID()
	reg := registerTestUser(t, svc, uid, "cycle@test.com")

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	parsedID, parseErr := cqrshtmx.ParseUserID(user.ID.Get().String())
	if parseErr != nil {
		t.Fatalf("ParseUserID: %v", parseErr)
	}

	enforcer, eErr := usermgmt.NewAuthz()
	if eErr != nil {
		t.Fatalf("NewAuthz: %v", eErr)
	}

	disp := command.NewDispatcher()
	app, appErr := cqrshtmx.New(cqrshtmx.Config{
		Commands: disp,
		Enforcer: enforcer.AsEnforcer(),
	})
	if appErr != nil {
		t.Fatalf("New App: %v", appErr)
	}

	_ = app

	ctx = cqrshtmx.WithUserID(ctx, parsedID)
	extracted := cqrshtmx.UserIDFromContext(ctx)
	if extracted != parsedID {
		t.Errorf("expected %v, got %v", parsedID, extracted)
	}
}
