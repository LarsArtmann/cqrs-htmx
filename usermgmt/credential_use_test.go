package usermgmt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// testUseCredential is the credential the WebAuthn stub's FinishLogin reports
// as used. Its raw ID is "test" (base64 "dGVzdA=="), matching the stub JSON.
func testUseCredential() WebAuthnCredential {
	return WebAuthnCredential{
		ID:              []byte("test"),
		PublicKey:       []byte("key"),
		AttestationType: "none",
	}
}

// beginAndFinishLogin drives the full login ceremony against the deterministic
// WebAuthn stub.
func beginAndFinishLogin(t *testing.T, svc *Service, userID UserID, email string) error {
	t.Helper()
	ctx := context.Background()

	if _, err := svc.BeginLogin(ctx, email); err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/webauthn/login/finish",
		strings.NewReader(`{"assertion":"stub"}`))

	_, err := svc.FinishLogin(ctx, userID, req)

	return err
}

// TestFinishLogin_PersistsSignCounterAndFlags is the S32 regression test: a
// successful assertion must advance the persisted signature counter and record
// the authenticator's real flags. Before the fix the updated credential was
// discarded, so the stored counter stayed at its registration value forever and
// clone detection could never fire.
func TestFinishLogin_PersistsSignCounterAndFlags(t *testing.T) {
	svc := newWebAuthnTestServiceWithConfig(t)
	reg := registerTestUser(t, svc, "wa-used", "wa-used@test.com")
	addTestCredential(t, svc, reg.User.ID, testUseCredential())

	if err := beginAndFinishLogin(t, svc, reg.User.ID, reg.User.Email); err != nil {
		t.Fatalf("FinishLogin: %v", err)
	}

	user, err := svc.GetUser(context.Background(), reg.User.ID)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if len(user.Credentials) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(user.Credentials))
	}
	cred := user.Credentials[0]
	if cred.SignCount != 1 {
		t.Errorf("persisted sign count = %d, want 1 (the assertion's counter)", cred.SignCount)
	}
	if !cred.UserPresent {
		t.Error("persisted UserPresent = false, want true (assertion reported UP)")
	}
	if !cred.UserVerified {
		t.Error("persisted UserVerified = false, want true (assertion reported UV)")
	}

	if findEvent(loadUserEvents(t, svc, reg.User.ID), eventCredentialUsed) == nil {
		t.Fatalf("no %s event on the user stream", eventCredentialUsed)
	}
}

// TestFinishLogin_RejectsCounterRegression proves the clone signal: an
// assertion whose counter does not advance must fail the login and leave the
// stored high-water mark untouched.
func TestFinishLogin_RejectsCounterRegression(t *testing.T) {
	svc := newWebAuthnTestServiceWithConfig(t)
	reg := registerTestUser(t, svc, "wa-clone", "wa-clone@test.com")

	cred := testUseCredential()
	cred.SignCount = 5
	addTestCredential(t, svc, reg.User.ID, cred)

	err := beginAndFinishLogin(t, svc, reg.User.ID, reg.User.Email)
	if err == nil {
		t.Fatal("expected a counter-regression error, got nil")
	}
	if !strings.Contains(err.Error(), "cloned") {
		t.Errorf("error should name the clone signal, got: %v", err)
	}

	user, getErr := svc.GetUser(context.Background(), reg.User.ID)
	if getErr != nil {
		t.Fatalf("GetUser: %v", getErr)
	}
	if got := user.Credentials[0].SignCount; got != 5 {
		t.Errorf("stored sign count = %d, want 5 (regression must not be written)", got)
	}
	if findEvent(loadUserEvents(t, svc, reg.User.ID), eventCredentialUsed) != nil {
		t.Error("a rejected regression must not emit a CredentialUsed event")
	}
}

// TestRecordCredentialUse_AttachesUserIDContext mechanizes the invariant the
// FinishLogin verbatim passthrough relies on: every error recordCredentialUse
// returns carries user_id, so propagating it unchanged loses no attribution
// context (the erraudit boundary annotation documents exactly this).
func TestRecordCredentialUse_AttachesUserIDContext(t *testing.T) {
	assertUserIDContext := func(t *testing.T, err error, want UserID) {
		t.Helper()

		famErr, ok := errors.AsType[*errorfamily.Error](err)
		if !ok {
			t.Fatalf("expected *errorfamily.Error, got %T: %v", err, err)
		}
		if !famErr.HasContext("user_id") {
			t.Fatalf("expected user_id context, got %#v", famErr.ErrorContext())
		}
		if got := famErr.ContextValue("user_id"); got != want.Get().String() {
			t.Errorf("user_id context = %q, want %q", got, want.Get().String())
		}
	}

	t.Run("unmarshal failure", func(t *testing.T) {
		svc := newWebAuthnTestServiceWithConfig(t)
		reg := registerTestUser(t, svc, "wa-ctx-unmarshal", "wa-ctx-unmarshal@test.com")

		err := svc.recordCredentialUse(context.Background(), reg.User.ID, []byte("{not json"))
		if err == nil {
			t.Fatal("expected an unmarshal failure, got nil")
		}
		assertUserIDContext(t, err, reg.User.ID)
	})

	t.Run("decider rejection", func(t *testing.T) {
		svc := newWebAuthnTestServiceWithConfig(t)
		reg := registerTestUser(t, svc, "wa-ctx-decider", "wa-ctx-decider@test.com")

		cred := testUseCredential()
		cred.SignCount = 5
		addTestCredential(t, svc, reg.User.ID, cred)

		err := beginAndFinishLogin(t, svc, reg.User.ID, reg.User.Email)
		if err == nil {
			t.Fatal("expected a counter-regression rejection, got nil")
		}
		assertUserIDContext(t, err, reg.User.ID)
	})
}

// TestCeremonySessions_NamespacedPerCeremony proves registration and login
// challenges for the same user cannot overwrite each other.
func TestCeremonySessions_NamespacedPerCeremony(t *testing.T) {
	svc := newWebAuthnTestServiceWithConfig(t)
	reg := registerTestUser(t, svc, "wa-ns", "wa-ns@test.com")
	addTestCredential(t, svc, reg.User.ID, testUseCredential())

	ctx := context.Background()
	userID := reg.User.ID.Get().String()

	regResp, err := svc.BeginRegistration(ctx, reg.User.ID)
	if err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}
	loginResp, err := svc.BeginLogin(ctx, reg.User.Email)
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	if regResp.SessionKey != registrationSessionPrefix+userID {
		t.Errorf("registration session key = %q, want %q", regResp.SessionKey, registrationSessionPrefix+userID)
	}
	if loginResp.SessionKey != loginSessionPrefix+userID {
		t.Errorf("login session key = %q, want %q", loginResp.SessionKey, loginSessionPrefix+userID)
	}

	for _, key := range []string{regResp.SessionKey, loginResp.SessionKey} {
		if _, err := svc.webauthnSessions.Get(key); err != nil {
			t.Errorf("session %q missing after both ceremonies began: %v", key, err)
		}
	}
}

func TestDecideRecordCredentialUse_NoOpForCounterlessAuthenticator(t *testing.T) {
	t.Parallel()

	cred := testUseCredential() // sign_count 0, flags false
	state := UserState{
		Email:       "counterless@test.com",
		Credentials: []WebAuthnCredential{cred},
	}

	events, err := decideRecordCredentialUse(id.NewStreamID(), CredentialUse{CredentialID: cred.ID})(
		state, event.Version(1),
	)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("counter-less no-op emitted %d events, want 0", len(events))
	}
}

func TestDecideRecordCredentialUse_Rejections(t *testing.T) {
	t.Parallel()

	cred := testUseCredential()
	withCred := UserState{Email: "u@test.com", Credentials: []WebAuthnCredential{cred}}

	tests := map[string]struct {
		state UserState
		use   CredentialUse
	}{
		"unknown credential": {
			state: withCred,
			use:   CredentialUse{CredentialID: []byte("other"), SignCount: 1},
		},
		"deleted user": {
			state: UserState{Email: "u@test.com", Deleted: true},
			use:   CredentialUse{CredentialID: cred.ID, SignCount: 1},
		},
		"missing user": {
			state: UserState{},
			use:   CredentialUse{CredentialID: cred.ID, SignCount: 1},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := decideRecordCredentialUse(id.NewStreamID(), tc.use)(
				tc.state, event.Version(1),
			); err == nil {
				t.Error("expected a rejection, got nil")
			}
		})
	}
}

func TestDecideRecordCredentialUse_AdvancesCounter(t *testing.T) {
	t.Parallel()

	cred := testUseCredential()
	state := UserState{Email: "u@test.com", Credentials: []WebAuthnCredential{cred}}

	events, err := decideRecordCredentialUse(id.NewStreamID(), CredentialUse{
		CredentialID: cred.ID,
		SignCount:    3,
		UserPresent:  true,
	})(state, event.Version(1))
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type() != eventCredentialUsed {
		t.Errorf("event type = %s, want %s", events[0].Type(), eventCredentialUsed)
	}
}
