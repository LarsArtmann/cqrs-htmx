package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
	"github.com/larsartmann/go-cqrs-lite/command/v2"
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
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{BcryptCost: 4})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	uid := cqrshtmx.NewUserID()
	ctx := context.Background()

	reg, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID:       usermgmt.NewUserID(uid.String()),
		Email:    "cycle@test.com",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := svc.Authenticate(ctx, reg.Session.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	parsedID, parseErr := cqrshtmx.ParseUserID(user.ID.Get())
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
