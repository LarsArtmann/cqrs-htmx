package usermgmt

import (
	"context"
)

// testWebAuthnProvider is a deterministic WebAuthnProvider stub for testing the
// Service's WebAuthn domain logic without importing the real go-webauthn-based
// webauthn module. It returns canned responses for all ceremonies.
type testWebAuthnProvider struct{}

// testWebAuthnOptions is the canned ceremony options returned by the stub.
var testWebAuthnOptions = []byte(
	`{"publicKey":{"challenge":"dGVzdA==","rpId":"localhost","user":{"id":"dGVzdA==","name":"test@test.com","displayName":"Test"}}}`,
)

// testWebAuthnCredential is the canned credential returned by FinishRegistration.
var testWebAuthnCredential = []byte(
	`{"id":"dGVzdA==","public_key":"dGVzdA==","attestation_type":"none","sign_count":0}`,
)

// testWebAuthnSession is the canned opaque session data.
var testWebAuthnSession = []byte(`{"challenge":"dGVzdA==","user_id":"dGVzdA==","expires":"2099-01-01T00:00:00Z"}`)

func (testWebAuthnProvider) BeginRegistration(_ context.Context, _ []byte) ([]byte, []byte, error) {
	return testWebAuthnOptions, testWebAuthnSession, nil
}

func (testWebAuthnProvider) FinishRegistration(_ context.Context, _, _, _ []byte) ([]byte, error) {
	return testWebAuthnCredential, nil
}

func (testWebAuthnProvider) BeginLogin(_ context.Context, _ []byte) ([]byte, []byte, error) {
	return testWebAuthnOptions, testWebAuthnSession, nil
}

func (testWebAuthnProvider) FinishLogin(_ context.Context, _, _, _ []byte) error {
	return nil
}
