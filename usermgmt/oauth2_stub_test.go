package usermgmt

import (
	"context"
)

// testOAuth2Provider is a deterministic OAuth2Provider stub for testing the
// Service's OAuth2 domain logic without importing the real oauth2 module.
type testOAuth2Provider struct{}

func (testOAuth2Provider) BeginLogin(_ context.Context, provider, state string) (string, string, error) {
	return "https://provider.example.com/auth?client_id=test&state=" + state + "&provider=" + provider, "test-pkce-verifier", nil
}

func (testOAuth2Provider) FinishLogin(_ context.Context, provider, code, pkceVerifier string) ([]byte, error) {
	return []byte(
		`{"subject":"test-subject-` + provider + `","email":"oauth@test.com","email_verified":true,"display_name":"Test User"}`,
	), nil
}
