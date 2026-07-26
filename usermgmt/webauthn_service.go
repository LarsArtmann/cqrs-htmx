package usermgmt

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"

	errorfamily "github.com/larsartmann/go-error-family"
)

// webAuthnUserData is the JSON shape the WebAuthnProvider expects for userJSON.
type webAuthnUserData struct {
	ID          string             `json:"id"`
	Email       string             `json:"email"`
	DisplayName string             `json:"display_name"`
	Credentials []webAuthnUserCred `json:"credentials"`
}

type webAuthnUserCred struct {
	ID              []byte   `json:"id"`
	PublicKey       []byte   `json:"public_key"`
	AttestationType string   `json:"attestation_type"`
	Transports      []string `json:"transports,omitempty"`
	AAGUID          []byte   `json:"aaguid,omitempty"`
	SignCount       uint32   `json:"sign_count"`
	BackupEligible  bool     `json:"backup_eligible"`
	BackupState     bool     `json:"backup_state"`
}

// marshalWebAuthnUser serializes a domain User to the JSON shape expected by
// the WebAuthnProvider.
func marshalWebAuthnUser(user *User) ([]byte, error) {
	creds := make([]webAuthnUserCred, len(user.Credentials))
	for i, c := range user.Credentials {
		creds[i] = webAuthnUserCred{
			ID:              c.ID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transports:      c.Transports,
			AAGUID:          c.AAGUID,
			SignCount:       c.SignCount,
			BackupEligible:  c.BackupEligible,
			BackupState:     c.BackupState,
		}
	}
	data, err := json.Marshal(webAuthnUserData{
		ID:          user.ID.Get().String(),
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Credentials: creds,
	})
	if err != nil {
		return nil, errorfamily.NewInfrastructure("internal", "marshal webauthn user data").WithCause(err)
	}
	return data, nil
}

// maxWebAuthnBodySize limits the authenticator response body to prevent abuse.
const maxWebAuthnBodySize = 1 << 20 // 1 MB

// webauthnBeginResponse is the shared shape returned by both WebAuthn begin
// ceremonies (registration and login): the JSON options to send to the client
// and the server-side session key. The two ceremonies expose it under
// distinct public names (BeginRegistrationResponse / BeginLoginResponse) for
// API clarity, but the wire shape is identical, so it is defined once here.
type webauthnBeginResponse struct {
	Options    jsontext.Value `json:"options"`
	SessionKey string         `json:"session_key"`
}

// BeginRegistrationResponse contains the credential creation options to send to the client.
type BeginRegistrationResponse = webauthnBeginResponse

// BeginRegistration starts the WebAuthn credential registration ceremony.
// The returned options must be sent to the client, which uses the browser
// WebAuthn API to create a credential. The session is stored server-side keyed by sessionKey.
func (s *Service) BeginRegistration(ctx context.Context, userID UserID) (*BeginRegistrationResponse, error) {
	if s.webauthn == nil {
		return nil, ErrWebAuthnNotConfigured
	}

	user, ok := s.readModel.FindByUserID(userID)
	if !ok {
		s.logger.Debug("usermgmt: begin registration failed – user not found", "user_id", userID)
		return nil, errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "begin registration")
	}

	userJSON, err := marshalWebAuthnUser(user)
	if err != nil {
		return nil, errorfamily.NewInfrastructure("usermgmt.webauthn.marshal_user", "marshal user data").WithCause(err)
	}

	options, sessionData, err := s.webauthn.BeginRegistration(ctx, userJSON)
	if err != nil {
		s.logger.Warn("usermgmt: begin registration ceremony failed",
			"user_id", userID, "error", err)
		return nil, errorfamily.NewTransient("internal", "begin webauthn registration").WithCause(err)
	}

	sessionKey := userID.Get().String()
	s.webauthnSessions.Save(sessionKey, sessionData)
	s.logger.Info("usermgmt: registration ceremony begun", "user_id", userID)

	return &BeginRegistrationResponse{
		Options:    options,
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
		return errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "finish registration")
	}

	sessionKey := userID.Get().String()
	sessionData, err := s.webauthnSessions.Get(sessionKey)
	if err != nil {
		s.logger.Warn("usermgmt: finish registration failed – session not found",
			"user_id", userID, "error", err)
		return err //nolint:wrapcheck // domain sentinel error
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebAuthnBodySize))
	if err != nil {
		return errorfamily.NewRejection("usermgmt.webauthn.body_read", "read attestation body").WithCause(err)
	}

	userJSON, err := marshalWebAuthnUser(user)
	if err != nil {
		return errorfamily.NewInfrastructure("usermgmt.webauthn.marshal_user", "marshal user data").WithCause(err)
	}

	credJSON, err := s.webauthn.FinishRegistration(ctx, userJSON, body, sessionData)
	if err != nil {
		s.logger.Warn("usermgmt: finish registration ceremony failed",
			"user_id", userID, "error", err)
		return errorfamily.NewRejection("usermgmt.webauthn.registration_failed",
			"credential registration failed").WithCause(err)
	}

	s.webauthnSessions.Delete(sessionKey)

	var cred CredentialCore
	if err := json.Unmarshal(credJSON, &cred); err != nil {
		return errorfamily.NewInfrastructure("usermgmt.webauthn.unmarshal_credential", "unmarshal credential").
			WithCause(err)
	}
	cred.Name = credentialName

	aggID, err := aggIDFromUser(userID)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.webauthn.userid_conversion_failed", "convert userID")
	}
	if err := s.dispatcher.Dispatch(
		ctx,
		NewAddCredentialCmd(aggID, WebAuthnCredential{CredentialCore: cred}),
	); err != nil {
		return errorfamily.Wrapf(err, errorfamily.Classify(err),
			"usermgmt.webauthn.dispatch_failed", "finish registration dispatch")
	}
	s.logger.Info("usermgmt: credential registered",
		"user_id", userID, "credential_name", credentialName)
	return nil
}

// BeginLoginResponse contains the assertion options to send to the client.
type BeginLoginResponse = webauthnBeginResponse

// BeginLogin starts the WebAuthn login ceremony.
// The user is looked up by email. The returned options must be sent to
// the client, which uses the browser WebAuthn API to assert a credential.
// If account lockout is configured and the account is locked, ErrAccountLocked is returned.
func (s *Service) BeginLogin(ctx context.Context, email string) (*BeginLoginResponse, error) {
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
		return nil, errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "begin login")
	}

	if len(user.Credentials) == 0 {
		s.logger.Debug("usermgmt: login failed – no credentials", "email", email)
		return nil, ErrNoCredentials
	}

	userJSON, err := marshalWebAuthnUser(user)
	if err != nil {
		return nil, errorfamily.NewInfrastructure("usermgmt.webauthn.marshal_user", "marshal user data").WithCause(err)
	}

	options, sessionData, err := s.webauthn.BeginLogin(ctx, userJSON)
	if err != nil {
		s.logger.Warn("usermgmt: begin login ceremony failed", "email", email, "error", err)
		return nil, errorfamily.NewTransient("internal", "begin webauthn login").WithCause(err)
	}

	sessionKey := user.ID.Get().String()
	s.webauthnSessions.Save(sessionKey, sessionData)
	s.logger.Debug("usermgmt: login ceremony begun", "email", email)

	return &BeginLoginResponse{
		Options:    options,
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
		return nil, errorfamily.WrapRejection(ErrUserNotFound, "usermgmt.webauthn.user_not_found", "finish login")
	}

	sessionKey := userID.Get().String()
	sessionData, err := s.webauthnSessions.Get(sessionKey)
	if err != nil {
		s.logger.Warn("usermgmt: finish login failed – session not found",
			"user_id", userID, "error", err)
		return nil, err //nolint:wrapcheck // domain sentinel error
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebAuthnBodySize))
	if err != nil {
		return nil, errorfamily.NewRejection("usermgmt.webauthn.body_read", "read assertion body").WithCause(err)
	}

	userJSON, err := marshalWebAuthnUser(user)
	if err != nil {
		return nil, errorfamily.NewInfrastructure("usermgmt.webauthn.marshal_user", "marshal user data").WithCause(err)
	}

	if err := s.webauthn.FinishLogin(ctx, userJSON, body, sessionData); err != nil {
		if s.lockout != nil {
			s.lockout.RecordFailure(user.Email)
		}
		s.logger.Warn("usermgmt: finish login ceremony failed",
			"user_id", userID, "email", user.Email, "error", err)
		return nil, errorfamily.NewRejection("usermgmt.webauthn.login_failed",
			"credential login failed").WithCause(err)
	}

	s.webauthnSessions.Delete(sessionKey)

	if s.lockout != nil {
		s.lockout.Reset(user.Email)
	}

	sess, err := s.createSession(ctx, user.ID)
	if err != nil {
		return nil, withUserIDContext(
			errorfamily.NewTransient("internal", "create session").WithCause(err), user.ID,
		)
	}

	s.logger.Info("usermgmt: login successful", "user_id", userID, "email", user.Email)
	return &FinishLoginResponse{User: user, Session: sess}, nil
}
