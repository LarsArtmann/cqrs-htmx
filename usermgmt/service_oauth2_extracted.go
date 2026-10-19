package usermgmt

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
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

	// maxUsers mirrors ServiceConfig.MaxUsers: when greater than zero,
	// auto-provisioning of new users via first-login is rejected once the read
	// model reaches this count. Logins of existing users (matched by external
	// account or email) are never affected.
	maxUsers int
	// registrationMu is the Service's shared registration lock. It serializes
	// the count-check-then-dispatch critical section across Register and
	// matchOrCreateUser so concurrent first logins cannot both pass the gate.
	registrationMu *sync.Mutex

	// Shared dependencies from the core Service
	readModel     *UserReadModel
	dispatcher    dispatcher
	sessions      SessionStore
	sessionTTL    time.Duration
	logger        *slog.Logger
	classifyError errorClassifier

	// store + bus carry the auth audit journal (cqrs-htmx#23): the
	// OAuth2Service journals login outcomes and registration-closed probes
	// through the same appendAuthAuditEvent path as the core Service.
	store event.Store
	bus   event.Bus
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
//
// maxUsers and registrationMu carry the Service's registration gate: pass the
// same values the core Service was built with (ServiceConfig.MaxUsers and the
// Service's registration mutex) so both flows enforce one shared user limit.
// registrationMu may be nil, in which case the count check is advisory only.
// store + bus carry the auth audit journal; pass the Service's values.
func NewOAuth2Service(
	provider OAuth2Provider,
	states OAuth2StateStore,
	stateTTL time.Duration,
	maxUsers int,
	registrationMu *sync.Mutex,
	readModel *UserReadModel,
	dispatcher dispatcher,
	sessions SessionStore,
	sessionTTL time.Duration,
	logger *slog.Logger,
	classifyError errorClassifier,
	store event.Store,
	bus event.Bus,
) *OAuth2Service {
	if provider == nil {
		return nil
	}
	if registrationMu == nil {
		registrationMu = &sync.Mutex{}
	}
	return &OAuth2Service{
		provider:       provider,
		states:         states,
		stateTTL:       stateTTL,
		maxUsers:       maxUsers,
		registrationMu: registrationMu,
		readModel:      readModel,
		dispatcher:     dispatcher,
		sessions:       sessions,
		sessionTTL:     sessionTTL,
		logger:         logger,
		classifyError:  classifyError,
		store:          store,
		bus:            bus,
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

	o.journalLoginSuccess(ctx, user.ID, provider)

	return &FinishOAuthLoginResponse{User: user, Session: session}, nil
}

// Unlink removes an external identity provider from a user.
// It enforces the last-auth-method guard: removing the link is rejected if the
// user has zero WebAuthn credentials and zero other external accounts.
func (o *OAuth2Service) Unlink(ctx context.Context, userID UserID, provider string) error {
	user, ok := o.readModel.FindByUserID(userID)
	if !ok {
		return errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.service.user_not_found", "unlink external account").
			WithContextAny("user_id", userID)
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

	// Email-match linking is gated on EmailVerified (see matchVerifiedEmail).
	matched, matchedByEmail, err := o.matchVerifiedEmail(ctx, provider, info)
	if err != nil {
		return nil, false, err
	}
	if matchedByEmail {
		return matched, false, nil
	}

	return o.provisionLinkedUser(ctx, provider, info)
}

// provisionLinkedUser auto-registers a fresh user for the OAuth2 identity and
// links the external account to it. The shared registration lock (when
// MaxUsers is set) is held from the count check through the final read so a
// concurrent Register or first-login sees the updated count once this returns.
func (o *OAuth2Service) provisionLinkedUser(
	ctx context.Context,
	provider string,
	info OAuth2UserInfo,
) (*User, bool, error) {
	if o.maxUsers > 0 {
		o.registrationMu.Lock()
		defer o.registrationMu.Unlock()

		if o.readModel.Count() >= o.maxUsers {
			o.logAuth("oauth_register_rejected", UserID{}, "provider", provider, "reason", "registration_closed")
			o.journalRegistrationClosed(ctx)

			return nil, false, errorfamily.NewRejection(
				"usermgmt.oauth2.registration_closed",
				"registration is closed",
			).WithCause(ErrRegistrationClosed).WithContext("provider", provider)
		}
	}

	aggID := id.NewStreamID()
	userID := NewUserID(aggID.Get())
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
		return nil, false, errorfamily.Wrapf(err, errorfamily.Classify(err),
			"usermgmt.oauth2.link_failed", "link external account").
			WithContextAny("user_id", user.ID).WithContext("provider", provider)
	}

	user, ok = o.readModel.FindByID(aggID)
	if !ok {
		return nil, false, withUserIDContext(
			errorfamily.NewTransient(
				"usermgmt.oauth2.read_model_missing",
				"oauth2 user not in read model after register+link",
			),
			userID,
		)
	}

	return user, true, nil
}

// matchVerifiedEmail links the OAuth2 identity to an existing account by
// email address and returns the refreshed user (ok=true). Linking is gated on
// EmailVerified: linking an external identity to an existing account by an
// unverified, user-controllable email (GitHub noreply, generic OIDC) is a
// pre-account-takeover vector (NIST 800-63B forbids linking on unverified
// email). Unverified emails report no match so matchOrCreateUser falls
// through to subject-only matching / auto-registration instead.
func (o *OAuth2Service) matchVerifiedEmail(
	ctx context.Context,
	provider string,
	info OAuth2UserInfo,
) (*User, bool, error) {
	if !info.EmailVerified {
		return nil, false, nil
	}

	user, found := o.readModel.FindByEmail(info.Email)
	if !found {
		return nil, false, nil
	}

	if err := o.linkExternalAccount(ctx, user.ID, provider, info); err != nil {
		return nil, true, err
	}

	refreshed, ok := o.readModel.FindByUserID(user.ID)
	if !ok {
		return nil, true, withUserIDContext(
			errorfamily.NewTransient(
				"usermgmt.oauth2.read_model_missing",
				"oauth2 user not in read model after email-match link",
			),
			user.ID,
		)
	}

	return refreshed, true, nil
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
	return createAndStoreSession(ctx, o.sessions, o.sessionTTL, userID)
}

func (o *OAuth2Service) logAuth(event string, userID UserID, attrs ...any) {
	logAuthEvent(o.logger, event, userID, attrs...)
}

// classifyOAuth2DispatchError delegates to the shared error classifier.
func classifyOAuth2DispatchError(classify errorClassifier, err error, userID UserID, kv ...string) error {
	return classify(err, userID, kv...)
}
