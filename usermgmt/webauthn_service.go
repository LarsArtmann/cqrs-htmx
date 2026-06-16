package usermgmt

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("begin registration: %w", ErrUserNotFound)
	}

	waUser := &webauthnUser{user: user}
	creation, session, err := s.webauthn.BeginRegistration(waUser)
	if err != nil {
		return nil, event.NewTransient("internal", "begin webauthn registration").WithCause(err)
	}

	sessionKey := userID.Get()
	s.webauthnSessions.Save(sessionKey, session)

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
		return fmt.Errorf("finish registration: %w", ErrUserNotFound)
	}

	sessionKey := userID.Get()
	session, err := s.webauthnSessions.Get(sessionKey)
	if err != nil {
		return err
	}

	waUser := &webauthnUser{user: user}
	credential, err := s.webauthn.FinishRegistration(
		waUser,
		*session,
		r,
	) //nolint:contextcheck // WebAuthn reads HTTP request body
	if err != nil {
		return event.NewRejection("usermgmt.webauthn.registration_failed",
			"credential registration failed").WithCause(err)
	}

	s.webauthnSessions.Delete(sessionKey)

	domainCred := fromWebAuthnCredential(credential, credentialName)
	aggID := aggIDFromUser(userID)
	return s.dispatcher.Dispatch(ctx, NewAddCredentialCmd(aggID, domainCred))
}

// BeginLoginResponse contains the assertion options to send to the client.
type BeginLoginResponse struct {
	Options    *protocol.CredentialAssertion `json:"options"`
	SessionKey string                        `json:"session_key"`
}

// BeginLogin starts the WebAuthn login ceremony.
// The user is looked up by email. The returned CredentialAssertion must be sent to
// the client, which uses the browser WebAuthn API to assert a credential.
func (s *Service) BeginLogin(_ context.Context, email string) (*BeginLoginResponse, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	user, ok := s.readModel.FindByEmail(email)
	if !ok {
		return nil, fmt.Errorf("begin login: %w", ErrUserNotFound)
	}

	if len(user.Credentials) == 0 {
		return nil, ErrNoCredentials
	}

	waUser := &webauthnUser{user: user}
	assertion, session, err := s.webauthn.BeginLogin(waUser)
	if err != nil {
		return nil, event.NewTransient("internal", "begin webauthn login").WithCause(err)
	}

	sessionKey := user.ID.Get()
	s.webauthnSessions.Save(sessionKey, session)

	return &BeginLoginResponse{
		Options:    assertion,
		SessionKey: sessionKey,
	}, nil
}

// FinishLoginResponse contains the session created after successful WebAuthn login.
type FinishLoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

// FinishLogin completes the WebAuthn login ceremony.
// The HTTP request must contain the assertion response from the authenticator.
// On success, a session is created and returned.
func (s *Service) FinishLogin(ctx context.Context, userID UserID, r *http.Request) (*FinishLoginResponse, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		return nil, fmt.Errorf("finish login: %w", ErrUserNotFound)
	}

	sessionKey := userID.Get()
	session, err := s.webauthnSessions.Get(sessionKey)
	if err != nil {
		return nil, err
	}

	waUser := &webauthnUser{user: user}
	_, err = s.webauthn.FinishLogin(
		waUser,
		*session,
		r,
	) //nolint:contextcheck // WebAuthn reads HTTP request body
	if err != nil {
		return nil, event.NewRejection("usermgmt.webauthn.login_failed",
			"credential login failed").WithCause(err)
	}

	s.webauthnSessions.Delete(sessionKey)

	sess, err := s.sessions.Create(ctx, user.ID, s.sessionTTL)
	if err != nil {
		return nil, withUserIDContext(
			event.NewTransient("internal", "create session").WithCause(err), user.ID,
		)
	}

	return &FinishLoginResponse{User: user, Session: sess}, nil
}
