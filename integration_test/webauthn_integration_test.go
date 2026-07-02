package integration_test

import (
	"context"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"
)

// TestService_WebAuthn_BeginRegistration_Integration verifies the full
// cross-module flow: Service.BeginRegistration → marshalWebAuthnUser (JSON)
// → webauthn.Provider.BeginRegistration → go-webauthn → JSON response.
//
// This is the critical boundary test that the extraction introduced: the
// Service serializes a User to []byte, the Provider deserializes it,
// runs the ceremony, and returns options. If the JSON contract drifts
// between core usermgmt and the webauthn sub-module, this test fails.
func TestService_WebAuthn_BeginRegistration_Integration(t *testing.T) {
	t.Parallel()

	provider, err := webauthn.New(webauthn.Config{
		RPID:          "localhost",
		RPDisplayName: "Integration Test App",
		RPOrigins:     []string{"http://localhost:8080"},
	})
	if err != nil {
		t.Fatalf("webauthn.New: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		WebAuthn: provider,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	uid, err := usermgmt.ParseUserID("01HXKYGEG0QH8XJYQKZ3R8WZAA")
	if err != nil {
		t.Fatalf("ParseUserID: %v", err)
	}

	_, err = svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID:          uid,
		Email:       "integration@test.com",
		DisplayName: "Integration User",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resp, err := svc.BeginRegistration(context.Background(), uid)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}

	if len(resp.Options) == 0 {
		t.Error("expected non-empty options from BeginRegistration")
	}
	if resp.SessionKey == "" {
		t.Error("expected non-empty session key")
	}

	_, err = svc.GetUser(context.Background(), uid)
	if err != nil {
		t.Errorf("expected user to exist after registration, got: %v", err)
	}
}

// TestService_WebAuthn_NilProvider_Guards verifies that a Service without
// a WebAuthn provider returns clear errors, not panics.
func TestService_WebAuthn_NilProvider_Guards(t *testing.T) {
	t.Parallel()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	uid, err := usermgmt.ParseUserID("01HXKYGEG0QH8XJYQKZ3R8WZAB")
	if err != nil {
		t.Fatalf("ParseUserID: %v", err)
	}

	_, err = svc.BeginRegistration(context.Background(), uid)
	if err == nil {
		t.Error("expected error when WebAuthn is not configured")
	}

	err = svc.FinishRegistration(context.Background(), uid, nil, "test")
	if err == nil {
		t.Error("expected error when WebAuthn is not configured")
	}
}
