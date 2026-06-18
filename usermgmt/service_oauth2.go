package usermgmt

import (
	"context"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"golang.org/x/oauth2"
)

// BeginOAuthLoginResponse contains the redirect URL for the OAuth2 authorization flow.
type BeginOAuthLoginResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// BeginOAuthLogin starts the OAuth2 login flow for the given provider.
// It generates a CSRF state token and PKCE verifier, stores them, and returns
// the provider's authorization URL. The browser should be redirected to this URL.
func (s *Service) BeginOAuthLogin(_ context.Context, provider string) (*BeginOAuthLoginResponse, error) {
	prov, err := s.getOAuth2Provider(provider)
	if err != nil {
		return nil, err
	}

	pkceVerifier := oauth2.GenerateVerifier()
	state, err := s.oauth2States.Save(provider, pkceVerifier, s.oauth2StateTTL)
	if err != nil {
		return nil, event.NewTransient("internal", "generate oauth2 state").WithCause(err)
	}

	redirectURL := prov.authCodeURL(state, pkceVerifier)
	s.logAuth("oauth_login_begin", UserID{}, "provider", provider)
	return &BeginOAuthLoginResponse{RedirectURL: redirectURL}, nil
}

// FinishOAuthLoginResponse contains the authenticated user and session.
type FinishOAuthLoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// FinishOAuthLogin completes the OAuth2 login flow. It validates the state token,
// exchanges the authorization code for tokens, extracts user info, and either:
//   - Links the external account to an existing user (matched by email), or
//   - Auto-registers a new user if no matching email is found.
//
// A session is created for the authenticated user. When the provider reports
// the email as verified, the user's email is marked as verified.
func (s *Service) FinishOAuthLogin(
	ctx context.Context,
	provider, code, state string,
) (*FinishOAuthLoginResponse, error) {
	prov, err := s.getOAuth2Provider(provider)
	if err != nil {
		return nil, err
	}

	storedProvider, pkceVerifier, err := s.oauth2States.Consume(state)
	if err != nil {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "invalid_state")
		return nil, err
	}
	if storedProvider != provider {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "provider_mismatch")
		return nil, ErrOAuthInvalidState
	}

	userInfo, err := prov.exchangeAndExtractUser(ctx, code, pkceVerifier)
	if err != nil {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "token_exchange")
		return nil, fmt.Errorf("%w: %w", ErrOAuthTokenExchange, err)
	}

	if userInfo.Email == "" {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "no_email")
		return nil, event.NewRejection("usermgmt.oauth_no_email",
			"OAuth2 provider did not return an email address")
	}
	userInfo.Email = strings.ToLower(strings.TrimSpace(userInfo.Email))

	user, created, err := s.matchOrCreateUser(ctx, provider, userInfo)
	if err != nil {
		return nil, err
	}

	session, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		return nil, withUserIDContext(
			event.NewTransient("internal", "create oauth2 session").WithCause(err), user.ID,
		)
	}

	if created {
		s.logAuth("oauth_register", user.ID, "provider", provider, "email", userInfo.Email)
	} else {
		s.logAuth("oauth_login", user.ID, "provider", provider)
	}

	return &FinishOAuthLoginResponse{User: user, Session: session}, nil
}

// matchOrCreateUser finds an existing user by provider+subject (stable ID),
// or by email, or auto-registers a new user if neither is found.
// Returns the user and true if a new user was created.
//
// Lookup order:
//  1. By provider+subject — recognizes returning users even if their email changed
//  2. By email — links new provider to existing user with matching email
//  3. Auto-register — creates a new user if no match found
func (s *Service) matchOrCreateUser(
	ctx context.Context,
	provider string,
	info oauth2UserInfo,
) (*User, bool, error) {
	// 1. Try to find by provider+subject (stable identifier)
	if existing, ok := s.readModel.FindByExternalAccount(provider, info.Subject); ok {
		// Already linked — idempotent re-login
		return existing, false, nil
	}

	// 2. Try to find by email
	if user, found := s.readModel.FindByEmail(info.Email); found {
		if err := s.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
			return nil, false, err
		}
		// Re-read to get the updated ExternalAccounts
		user, _ = s.readModel.FindByUserID(user.ID)
		return user, false, nil
	}

	// 3. Auto-register a new user
	aggID := id.NewAggregateID()
	userID := NewUserID(aggID.String())
	displayName := info.DisplayName
	if displayName == "" {
		displayName = info.Email
	}
	if err := s.dispatcher.Dispatch(ctx, NewRegisterUserCmd(
		aggID, info.Email, displayName, []Role{RoleViewer, RoleUser},
	)); err != nil {
		return nil, false, s.classifyDispatchError(err, userID)
	}

	// Read model is updated synchronously (MemoryBus blocks until handlers complete)
	user, ok := s.readModel.FindByID(aggID)
	if !ok {
		return nil, false, withUserIDContext(
			event.NewTransient("internal", "oauth2 user not in read model after register"), userID,
		)
	}

	// Link the external account to the newly created user
	if err := s.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
		return nil, false, err
	}

	// Re-read to get the updated ExternalAccounts
	user, _ = s.readModel.FindByID(aggID)
	return user, true, nil
}

// linkExternalAccount dispatches LinkExternalAccount if not already linked,
// and marks the email as verified if the provider confirmed it.
// Enforces global uniqueness: rejects if the provider+subject is linked to a different user.
func (s *Service) linkExternalAccount(
	ctx context.Context,
	userID UserID,
	provider string,
	info oauth2UserInfo,
) error {
	// Check if already linked to THIS user (idempotent — re-login shouldn't fail)
	for _, ea := range s.readUserExternalAccounts(userID) {
		if ea.Provider == provider && ea.Subject == info.Subject {
			return nil // already linked
		}
	}

	// Check if linked to a DIFFERENT user (global uniqueness enforcement)
	if s.isExternalAccountLinkedToOther(provider, info.Subject, userID) {
		return ErrExternalAccountAlreadyLinked
	}

	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID for link: %w", err)
	}
	if err := s.dispatcher.Dispatch(ctx, NewLinkExternalAccountCmd(
		aggID, provider, info.Subject, info.Email, info.DisplayName,
	)); err != nil {
		return s.classifyDispatchError(err, userID)
	}

	if info.EmailVerified {
		s.markEmailVerifiedIfMatch(ctx, aggID, userID, info.Email)
	}
	return nil
}

// isExternalAccountLinkedToOther checks if the provider+subject is linked to
// a user OTHER than the given userID. Used for global uniqueness enforcement.
func (s *Service) isExternalAccountLinkedToOther(provider, subject string, userID UserID) bool {
	existing, ok := s.readModel.FindByExternalAccount(provider, subject)
	return ok && existing.ID.Get() != userID.Get()
}

// markEmailVerifiedIfMatch dispatches VerifyEmailCmd if the provider-reported
// email matches the user's current email and isn't already verified.
func (s *Service) markEmailVerifiedIfMatch(ctx context.Context, aggID id.AggregateID, userID UserID, email string) {
	user, ok := s.readModel.FindByUserID(userID)
	if !ok || user.EmailVerified || !strings.EqualFold(user.Email, email) {
		return
	}
	if err := s.dispatcher.Dispatch(ctx, NewVerifyEmailCmd(aggID)); err != nil {
		s.logger.Warn("verify email after oauth2 failed", "error", err, "user_id", userID.Get())
	}
}

// UnlinkExternalAccount removes an external identity provider from a user.
// It enforces the last-auth-method guard: removing the link is rejected if the
// user has zero WebAuthn credentials and zero other external accounts.
func (s *Service) UnlinkExternalAccount(ctx context.Context, userID UserID, provider string) error {
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return fmt.Errorf("unlink external account: %w", ErrUserNotFound)
	}

	var subject string
	for _, ea := range user.ExternalAccounts {
		if ea.Provider == provider {
			subject = ea.Subject
			break
		}
	}
	if subject == "" {
		return ErrOAuthProviderNotFound
	}

	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return fmt.Errorf("convert userID for unlink: %w", err)
	}

	if err := s.dispatcher.Dispatch(ctx, NewUnlinkExternalAccountCmd(
		aggID, provider, subject,
	)); err != nil {
		return s.classifyDispatchError(err, userID)
	}

	s.logAuth("oauth_unlink", userID, "provider", provider)
	return nil
}

func (s *Service) getOAuth2Provider(provider string) (*oauth2Provider, error) {
	if s.oauth2Providers == nil {
		return nil, ErrOAuthNotConfigured
	}
	prov, ok := s.oauth2Providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrOAuthProviderNotFound, provider)
	}
	return prov, nil
}

// readUserExternalAccounts returns the user's external accounts from the read model.
// Helper to avoid nil-slice issues when checking for existing links.
func (s *Service) readUserExternalAccounts(userID UserID) []ExternalAccount {
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return nil
	}
	return user.ExternalAccounts
}
