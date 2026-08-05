package usermgmt

import "context"

// BeginOAuthLoginResponse contains the redirect URL for the OAuth2 authorization flow.
type BeginOAuthLoginResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// FinishOAuthLoginResponse is an alias for [AuthResult], returned by [Service.FinishOAuthLogin].
type FinishOAuthLoginResponse = AuthResult

// BeginOAuthLogin starts the OAuth2 login flow for the given provider.
// Delegates to the focused [OAuth2Service] — see ADR-0038 for the decomposition plan.
func (s *Service) BeginOAuthLogin(ctx context.Context, provider string) (*BeginOAuthLoginResponse, error) {
	if s.oauth2Svc == nil {
		return nil, ErrOAuthNotConfigured
	}
	return s.oauth2Svc.BeginLogin(ctx, provider)
}

// FinishOAuthLogin completes the OAuth2 login flow.
// Delegates to the focused [OAuth2Service] — see ADR-0038 for the decomposition plan.
func (s *Service) FinishOAuthLogin(
	ctx context.Context,
	provider, code, state string,
) (*FinishOAuthLoginResponse, error) {
	if s.oauth2Svc == nil {
		return nil, ErrOAuthNotConfigured
	}
	return s.oauth2Svc.FinishLogin(ctx, provider, code, state)
}

// UnlinkExternalAccount removes an external identity provider from a user.
// Delegates to the focused [OAuth2Service] — see ADR-0038 for the decomposition plan.
func (s *Service) UnlinkExternalAccount(ctx context.Context, userID UserID, provider string) error {
	if s.oauth2Svc == nil {
		return ErrOAuthNotConfigured
	}
	return s.oauth2Svc.Unlink(ctx, userID, provider)
}
