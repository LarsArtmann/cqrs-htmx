package usermgmt

import (
	"context"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

// BeginRegistrationResponse contains the credential creation options to send to the client.
type BeginRegistrationResponse struct {
	Options    *protocol.CredentialCreation `json:"options"`
	SessionKey string                       `json:"session_key"`
}

// BeginRegistration starts the WebAuthn credential registration ceremony.
// The returned CredentialCreation must be sent to the client, which uses the browser
// WebAuthn API to create a credential. The session is stored server-side keyed by sessionKey.
func (s *Service) BeginRegistration(ctx context.Context, userID UserID) (*BeginRegistrationResponse, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logger.Debug("usermgmt: begin registration failed – user not found", "user_id", userID)
		return nil, event.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "begin registration")
	}

	waUser := &webauthnUser{user: user}
	creation, session, err := s.webauthn.BeginRegistration(waUser)
	if err != nil {
		s.logger.Warn("usermgmt: begin registration ceremony failed",
			"user_id", userID, "error", err)
		return nil, event.NewTransient("internal", "begin webauthn registration").WithCause(err)
	}

	sessionKey := userID.Get()
	s.webauthnSessions.Save(sessionKey, session)
	s.logger.Info("usermgmt: registration ceremony begun", "user_id", userID)

	return &BeginRegistrationResponse{
		Options:    creation,
		SessionKey: sessionKey,
	}, nil
}

// FinishRegistration completes the WebAuthn credential registration ceremony.
// The HTTP request must contain the attestation response from the authenticator.
// On success, the credential is persisted as a CredentialAdded event on the User aggregate.
func (s *Service) FinishRegistration(ctx context.Context, userID UserID, r *http.Request, credentialName string) error {
	if s.webauthn == nil {
		return ErrWebAuthnNotConfigured
	}

	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logger.Debug("usermgmt: finish registration failed – user not found", "user_id", userID)
		return event.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "finish registration")
	}

	sessionKey := userID.Get()
	session, err := s.webauthnSessions.Get(sessionKey)
	if err != nil {
		s.logger.Warn("usermgmt: finish registration failed – session not found",
			"user_id", userID, "error", err)
		return err
	}

	waUser := &webauthnUser{user: user}
	credential, err := s.webauthn.FinishRegistration(
		waUser,
		*session,
		r,
	)
	if err != nil {
		s.logger.Warn("usermgmt: finish registration ceremony failed",
			"user_id", userID, "error", err)
		return event.NewRejection("usermgmt.webauthn.registration_failed",
			"credential registration failed").WithCause(err)
	}

	s.webauthnSessions.Delete(sessionKey)

	domainCred := fromWebAuthnCredential(credential, credentialName)
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return event.WrapInfrastructure(err, "usermgmt.webauthn.userid_conversion_failed", "convert userID")
	}
	if err := s.dispatcher.Dispatch(ctx, NewAddCredentialCmd(aggID, domainCred)); err != nil {
		return event.Compose(
			event.Newf(event.Transient, "usermgmt.webauthn.dispatch_failed", "finish registration dispatch"),
			err,
		)
	}
	s.logger.Info("usermgmt: credential registered",
		"user_id", userID, "credential_name", credentialName)
	return nil
}

// BeginLoginResponse contains the assertion options to send to the client.
type BeginLoginResponse struct {
	Options    *protocol.CredentialAssertion `json:"options"`
	SessionKey string                        `json:"session_key"`
}

// BeginLogin starts the WebAuthn login ceremony.
// The user is looked up by email. The returned CredentialAssertion must be sent to
// the client, which uses the browser WebAuthn API to assert a credential.
// If account lockout is configured and the account is locked, ErrAccountLocked is returned.
func (s *Service) BeginLogin(_ context.Context, email string) (*BeginLoginResponse, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	if s.lockout != nil && s.lockout.IsLocked(email) {
		s.logger.Warn("usermgmt: login blocked – account locked", "email", email)
		return nil, ErrAccountLocked
	}

	user, ok := s.readModel.FindByEmail(email)
	if !ok {
		s.logger.Debug("usermgmt: login failed – user not found", "email", email)
		return nil, event.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "begin login")
	}

	if len(user.Credentials) == 0 {
		s.logger.Debug("usermgmt: login failed – no credentials", "email", email)
		return nil, ErrNoCredentials
	}

	waUser := &webauthnUser{user: user}
	assertion, session, err := s.webauthn.BeginLogin(waUser)
	if err != nil {
		s.logger.Warn("usermgmt: begin login ceremony failed", "email", email, "error", err)
		return nil, event.NewTransient("internal", "begin webauthn login").WithCause(err)
	}

	sessionKey := user.ID.Get()
	s.webauthnSessions.Save(sessionKey, session)
	s.logger.Debug("usermgmt: login ceremony begun", "email", email)

	return &BeginLoginResponse{
		Options:    assertion,
		SessionKey: sessionKey,
	}, nil
}

// FinishLoginResponse is an alias for [AuthResult], returned by [Service.FinishLogin].
type FinishLoginResponse = AuthResult

// FinishLogin completes the WebAuthn login ceremony.
// The HTTP request must contain the assertion response from the authenticator.
// On success, a session is created and returned.
func (s *Service) FinishLogin(ctx context.Context, userID UserID, r *http.Request) (*FinishLoginResponse, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logger.Debug("usermgmt: finish login failed – user not found", "user_id", userID)
		return nil, event.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "finish login")
	}

	sessionKey := userID.Get()
	session, err := s.webauthnSessions.Get(sessionKey)
	if err != nil {
		s.logger.Warn("usermgmt: finish login failed – session not found",
			"user_id", userID, "error", err)
		return nil, err
	}

	waUser := &webauthnUser{user: user}
	_, err = s.webauthn.FinishLogin(
		waUser,
		*session,
		r,
	)
	if err != nil {
		if s.lockout != nil {
			s.lockout.RecordFailure(user.Email)
		}
		s.logger.Warn("usermgmt: finish login ceremony failed",
			"user_id", userID, "email", user.Email, "error", err)
		return nil, event.NewRejection("usermgmt.webauthn.login_failed",
			"credential login failed").WithCause(err)
	}

	s.webauthnSessions.Delete(sessionKey)

	if s.lockout != nil {
		s.lockout.Reset(user.Email)
	}

	sess, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		return nil, withUserIDContext(
			event.NewTransient("internal", "create session").WithCause(err), user.ID,
		)
	}

	s.logger.Info("usermgmt: login successful", "user_id", userID, "email", user.Email)
	return &FinishLoginResponse{User: user, Session: sess}, nil
}
