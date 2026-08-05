package usermgmt

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// OAuth2Service encapsulates OAuth2/OIDC authentication: login flow, account
// linking, and unlinking.
//
// This is a focused sub-service extracted from *Service to validate the ADR-0038
// composition pattern for the eventual v5 Service decomposition. In v4, *Service
// delegates its OAuth2 methods here — the public API is unchanged. In v5, the
// delegate methods are removed and consumers access svc.OAuth2.BeginLogin(...)
// directly.
type OAuth2Service struct {
	provider OAuth2Provider
	states   OAuth2StateStore
	stateTTL time.Duration

	// Shared dependencies from the core Service
	readModel     *UserReadModel
	dispatcher    dispatcher
	sessions      SessionStore
	sessionTTL    time.Duration
	logger        *slog.Logger
	classifyError errorClassifier
}

// dispatcher is the subset of *command.Dispatcher that sub-services need.
type dispatcher interface {
	Dispatch(ctx context.Context, cmd command.Command) error
}

// classifyOAuth2DispatchError classifies a dispatch error from the OAuth2 flow.
// It delegates to the same logic as classifyDispatchError on *Service via a
// function field. In v5 they unify into a shared classifier on the focused service.
type errorClassifier func(err error, userID UserID, kv ...string) error

// NewOAuth2Service creates an OAuth2Service from the given configuration and
// shared dependencies. Returns nil if no provider is configured.
func NewOAuth2Service(
	provider OAuth2Provider,
	states OAuth2StateStore,
	stateTTL time.Duration,
	readModel *UserReadModel,
	dispatcher dispatcher,
	sessions SessionStore,
	sessionTTL time.Duration,
	logger *slog.Logger,
	classifyError errorClassifier,
) *OAuth2Service {
	if provider == nil {
		return nil
	}
	return &OAuth2Service{
		provider:      provider,
		states:        states,
		stateTTL:      stateTTL,
		readModel:     readModel,
		dispatcher:    dispatcher,
		sessions:      sessions,
		sessionTTL:    sessionTTL,
		logger:        logger,
		classifyError: classifyError,
	}
}

// BeginLogin starts the OAuth2 login flow for the given provider.
func (o *OAuth2Service) BeginLogin(ctx context.Context, provider string) (*BeginOAuthLoginResponse, error) {
	state, err := generateOAuth2State()
	if err != nil {
		return nil, errorfamily.NewTransient("usermgmt.oauth2.generate_state", "generate oauth2 state").WithCause(err)
	}

	redirectURL, pkceVerifier, err := o.provider.BeginLogin(ctx, provider, state)
	if err != nil {
		return nil, errorfamily.NewTransient("usermgmt.oauth2.begin_login", "oauth2 begin login").
			WithCause(err).WithContext("provider", provider)
	}

	if err := o.states.Save(state, provider, pkceVerifier, o.stateTTL); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.oauth2.save_state", "save oauth2 state").
			WithCause(err).WithContext("provider", provider)
	}

	o.logAuth("oauth_login_begin", UserID{}, "provider", provider)
	return &BeginOAuthLoginResponse{RedirectURL: redirectURL}, nil
}

// FinishLogin completes the OAuth2 login flow. It validates the state token,
// calls the provider to exchange the code and extract user info, and either:
//   - Links the external account to an existing user (matched by email), or
//   - Auto-registers a new user if no matching email is found.
//
// A session is created for the authenticated user. When the provider reports
// the email as verified, the user's email is marked as verified.
func (o *OAuth2Service) FinishLogin(
	ctx context.Context,
	provider, code, state string,
) (*FinishOAuthLoginResponse, error) {
	storedProvider, pkceVerifier, err := o.states.Consume(state)
	if err != nil {
		o.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "invalid_state")
		return nil, errorfamily.WrapRejection(err, "usermgmt.oauth.state_consume_failed", "consume oauth2 state").
			WithContext("provider", provider)
	}
	if storedProvider != provider {
		o.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "provider_mismatch")
		return nil, errorfamily.WrapRejection(ErrOAuthInvalidState, "usermgmt.oauth.provider_mismatch",
			"oauth2 provider mismatch between state and callback").WithContext("provider", provider)
	}

	userInfoJSON, err := o.provider.FinishLogin(ctx, provider, code, pkceVerifier)
	if err != nil {
		o.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "token_exchange")
		return nil, errorfamily.WrapTransient(err, "usermgmt.oauth.token_exchange", "exchange oauth2 token").
			WithContext("provider", provider)
	}

	var info OAuth2UserInfo
	if err := json.Unmarshal(userInfoJSON, &info); err != nil {
		return nil, errorfamily.NewInfrastructure("usermgmt.oauth.userinfo_unmarshal", "unmarshal user info").
			WithCause(err).WithContext("provider", provider)
	}

	if info.Email == "" {
		o.logAuth("oauth_login_failed", UserID{}, "provider", provider, "reason", "no_email")
		return nil, errorfamily.NewRejection("usermgmt.oauth_no_email",
			"OAuth2 provider did not return an email address").WithContext("provider", provider)
	}
	info.Email = strings.ToLower(strings.TrimSpace(info.Email))

	user, created, err := o.matchOrCreateUser(ctx, provider, info)
	if err != nil {
		return nil, err
	}

	session, err := o.createSession(ctx, user.ID)
	if err != nil {
		return nil, withUserIDContext(
			errorfamily.NewTransient("usermgmt.oauth2.create_session", "create oauth2 session").WithCause(err), user.ID,
		)
	}

	if created {
		o.logAuth("oauth_register", user.ID, "provider", provider, "email", info.Email)
	} else {
		o.logAuth("oauth_login", user.ID, "provider", provider)
	}

	return &FinishOAuthLoginResponse{User: user, Session: session}, nil
}

// Unlink removes an external identity provider from a user.
// It enforces the last-auth-method guard: removing the link is rejected if the
// user has zero WebAuthn credentials and zero other external accounts.
func (o *OAuth2Service) Unlink(ctx context.Context, userID UserID, provider string) error {
	user, ok := o.readModel.FindByUserID(userID)
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

	if err := o.dispatcher.Dispatch(ctx, NewUnlinkExternalAccountCmd(
		aggID, provider, subject,
	)); err != nil {
		return classifyOAuth2DispatchError(o.classifyError, err, userID, "subject", subject)
	}

	o.logAuth("oauth_unlink", userID, "provider", provider)
	return nil
}

// matchOrCreateUser finds an existing user by provider+subject (stable ID),
// or by email, or auto-registers a new user if neither is found.
func (o *OAuth2Service) matchOrCreateUser(
	ctx context.Context,
	provider string,
	info OAuth2UserInfo,
) (*User, bool, error) {
	if existing, ok := o.readModel.FindByExternalAccount(provider, info.Subject); ok {
		return existing, false, nil
	}

	if user, found := o.readModel.FindByEmail(info.Email); found {
		if err := o.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
			return nil, false, err
		}
		user, _ = o.readModel.FindByUserID(user.ID)
		return user, false, nil
	}

	aggID := id.NewStreamID()
	userID := NewUserID(aggID.String())
	displayName := info.DisplayName
	if displayName == "" {
		displayName = info.Email
	}
	if err := o.dispatcher.Dispatch(ctx, NewRegisterUserCmd(
		aggID, info.Email, displayName, []Role{RoleViewer, RoleUser},
	)); err != nil {
		return nil, false, classifyOAuth2DispatchError(o.classifyError, err, userID)
	}

	user, ok := o.readModel.FindByID(aggID)
	if !ok {
		return nil, false, withUserIDContext(
			errorfamily.NewTransient(
				"usermgmt.oauth2.read_model_missing",
				"oauth2 user not in read model after register",
			),
			userID,
		)
	}

	if err := o.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
		return nil, false, err
	}

	user, _ = o.readModel.FindByID(aggID)
	return user, true, nil
}

// linkExternalAccount dispatches LinkExternalAccount if not already linked,
// and marks the email as verified if the provider confirmed it.
func (o *OAuth2Service) linkExternalAccount(
	ctx context.Context,
	userID UserID,
	provider string,
	info OAuth2UserInfo,
) error {
	for _, ea := range o.readUserExternalAccounts(userID) {
		if ea.Provider == provider && ea.Subject == info.Subject {
			return nil
		}
	}

	if o.isExternalAccountLinkedToOther(provider, info.Subject, userID) {
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
	if err := o.dispatcher.Dispatch(ctx, NewLinkExternalAccountCmd(
		aggID, provider, info.Subject, info.Email, info.DisplayName,
	)); err != nil {
		return classifyOAuth2DispatchError(o.classifyError, err, userID, "subject", info.Subject)
	}

	if info.EmailVerified {
		o.markEmailVerifiedIfMatch(ctx, aggID, userID, info.Email)
	}
	return nil
}

func (o *OAuth2Service) isExternalAccountLinkedToOther(provider, subject string, userID UserID) bool {
	existing, ok := o.readModel.FindByExternalAccount(provider, subject)
	return ok && existing.ID.Get() != userID.Get()
}

func (o *OAuth2Service) markEmailVerifiedIfMatch(ctx context.Context, aggID id.StreamID, userID UserID, email string) {
	user, ok := o.readModel.FindByUserID(userID)
	if !ok || user.EmailVerified || !strings.EqualFold(user.Email, email) {
		return
	}
	if err := o.dispatcher.Dispatch(ctx, NewVerifyEmailCmd(aggID)); err != nil {
		o.logger.Warn("verify email after oauth2 failed", "error", err, "user_id", userID.Get())
	}
}

func (o *OAuth2Service) readUserExternalAccounts(userID UserID) []ExternalAccount {
	user, ok := o.readModel.FindByUserID(userID)
	if !ok {
		return nil
	}
	return user.ExternalAccounts
}

func (o *OAuth2Service) createSession(ctx context.Context, userID UserID) (*Session, error) {
	session, err := NewSession(userID, o.sessionTTL)
	if err != nil {
		return nil, errorfamily.NewTransient("usermgmt.session.create", "create session").WithCause(err)
	}
	if err := o.sessions.Create(ctx, session); err != nil {
		return nil, errorfamily.NewTransient("usermgmt.session.store", "store session").WithCause(err)
	}
	return session, nil
}

func (o *OAuth2Service) logAuth(event string, userID UserID, attrs ...any) {
	args := make([]any, 0, 4+len(attrs))
	args = append(args, "event", event, "user_id", userID)
	args = append(args, attrs...)
	o.logger.Info("usermgmt: "+event, args...)
}

// classifyOAuth2DispatchError delegates to the shared error classifier.
func classifyOAuth2DispatchError(classify errorClassifier, err error, userID UserID, kv ...string) error {
	return classify(err, userID, kv...)
}
