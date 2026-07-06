package usermgmt

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
	errorfamily "github.com/larsartmann/go-error-family"
)

// BeginOAuthLoginResponse contains the redirect URL for the OAuth2 authorization flow.
type BeginOAuthLoginResponse struct {
	RedirectURL string `json:"redirect_url"`
}

// BeginOAuthLogin starts the OAuth2 login flow for the given provider.
// It generates a CSRF state token, calls the provider to build the authorization
// URL (with PKCE), stores the state, and returns the redirect URL.
func (s *Service) BeginOAuthLogin(ctx context.Context, provider string) (*BeginOAuthLoginResponse, error) {
	if s.oauth2 == nil {
		return nil, ErrOAuthNotConfigured
	}

	state, err := generateOAuth2State()
	if err != nil {
		return nil, errorfamily.NewTransient("internal", "generate oauth2 state").WithCause(err)
	}

	redirectURL, pkceVerifier, err := s.oauth2.BeginLogin(ctx, provider, state)
	if err != nil {
		return nil, errorfamily.NewTransient("internal", "oauth2 begin login").WithCause(err)
	}

	if err := s.oauth2States.Save(state, provider, pkceVerifier, s.oauth2StateTTL); err != nil {
		return nil, errorfamily.NewTransient("internal", "save oauth2 state").WithCause(err)
	}

	s.logAuth("oauth_login_begin", UserID{}, "provider", provider)
	return &BeginOAuthLoginResponse{RedirectURL: redirectURL}, nil
}

// FinishOAuthLoginResponse is an alias for [AuthResult], returned by [Service.FinishOAuthLogin].
type FinishOAuthLoginResponse = AuthResult

// FinishOAuthLogin completes the OAuth2 login flow. It validates the state token,
// calls the provider to exchange the code and extract user info, and either:
//   - Links the external account to an existing user (matched by email), or
//   - Auto-registers a new user if no matching email is found.
//
// A session is created for the authenticated user. When the provider reports
// the email as verified, the user's email is marked as verified.
func (s *Service) FinishOAuthLogin(
	ctx context.Context,
	provider, code, state string,
) (*FinishOAuthLoginResponse, error) {
	if s.oauth2 == nil {
		return nil, ErrOAuthNotConfigured
	}

	storedProvider, pkceVerifier, err := s.oauth2States.Consume(state)
	if err != nil {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "invalid_state")
		return nil, errorfamily.WrapRejection(err, "usermgmt.oauth.state_consume_failed", "consume oauth2 state")
	}
	if storedProvider != provider {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "provider_mismatch")
		return nil, ErrOAuthInvalidState
	}

	userInfoJSON, err := s.oauth2.FinishLogin(ctx, provider, code, pkceVerifier)
	if err != nil {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "token_exchange")
		return nil, errorfamily.WrapTransient(err, "usermgmt.oauth.token_exchange", "exchange oauth2 token")
	}

	var info OAuth2UserInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		return nil, errorfamily.NewInfrastructure("usermgmt.oauth.userinfo_unmarshal", "unmarshal user info").
			WithCause(err)
	}

	if info.Email == "" {
		s.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "no_email")
		return nil, errorfamily.NewRejection("usermgmt.oauth_no_email",
			"OAuth2 provider did not return an email address")
	}
	info.Email = strings.ToLower(strings.TrimSpace(info.Email))

	user, created, err := s.matchOrCreateUser(ctx, provider, info)
	if err != nil {
		return nil, err
	}

	session, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, withUserIDContext(
			errorfamily.NewTransient("internal", "create oauth2 session").WithCause(err), user.ID,
		)
	}

	if created {
		s.logAuth("oauth_register", user.ID, "provider", provider, "email", info.Email)
	} else {
		s.logAuth("oauth_login", user.ID, "provider", provider)
	}

	return &FinishOAuthLoginResponse{User: user, Session: session}, nil
}

// matchOrCreateUser finds an existing user by provider+subject (stable ID),
// or by email, or auto-registers a new user if neither is found.
// Returns the user and true if a new user was created.
func (s *Service) matchOrCreateUser(
	ctx context.Context,
	provider string,
	info OAuth2UserInfo,
) (*User, bool, error) {
	// 1. Try to find by provider+subject (stable identifier)
	if existing, ok := s.readModel.FindByExternalAccount(provider, info.Subject); ok {
		return existing, false, nil
	}

	// 2. Try to find by email
	if user, found := s.readModel.FindByEmail(info.Email); found {
		if err := s.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
			return nil, false, err
		}
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

	user, ok := s.readModel.FindByID(aggID)
	if !ok {
		return nil, false, withUserIDContext(
			errorfamily.NewTransient("internal", "oauth2 user not in read model after register"), userID,
		)
	}

	if err := s.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
		return nil, false, err
	}

	user, _ = s.readModel.FindByID(aggID)
	return user, true, nil
}

// linkExternalAccount dispatches LinkExternalAccount if not already linked,
// and marks the email as verified if the provider confirmed it.
func (s *Service) linkExternalAccount(
	ctx context.Context,
	userID UserID,
	provider string,
	info OAuth2UserInfo,
) error {
	for _, ea := range s.readUserExternalAccounts(userID) {
		if ea.Provider == provider && ea.Subject == info.Subject {
			return nil
		}
	}

	if s.isExternalAccountLinkedToOther(provider, info.Subject, userID) {
		return ErrExternalAccountAlreadyLinked
	}

	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return errorfamily.WrapInfrastructure(
			err,
			"usermgmt.service.userid_conversion_failed",
			"convert userID for link",
		)
	}
	if err := s.dispatcher.Dispatch(ctx, NewLinkExternalAccountCmd(
		aggID, provider, info.Subject, info.Email, info.DisplayName,
	)); err != nil {
		return s.classifyDispatchError(err, userID, "subject", info.Subject)
	}

	if info.EmailVerified {
		s.markEmailVerifiedIfMatch(ctx, aggID, userID, info.Email)
	}
	return nil
}

func (s *Service) isExternalAccountLinkedToOther(provider, subject string, userID UserID) bool {
	existing, ok := s.readModel.FindByExternalAccount(provider, subject)
	return ok && existing.ID.Get() != userID.Get()
}

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
		return errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.service.user_not_found", "unlink external account")
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
		return errorfamily.WrapInfrastructure(
			err,
			"usermgmt.service.userid_conversion_failed",
			"convert userID for unlink",
		)
	}

	if err := s.dispatcher.Dispatch(ctx, NewUnlinkExternalAccountCmd(
		aggID, provider, subject,
	)); err != nil {
		return s.classifyDispatchError(err, userID, "subject", subject)
	}

	s.logAuth("oauth_unlink", userID, "provider", provider)
	return nil
}

func (s *Service) readUserExternalAccounts(userID UserID) []ExternalAccount {
	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return nil
	}
	return user.ExternalAccounts
}
