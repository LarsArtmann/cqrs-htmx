// Package webauthn provides passkey (WebAuthn/FIDO2) authentication using the
// go-webauthn library.
//
// It implements the usermgmt.WebAuthnProvider interface via structural typing —
// no import of the core usermgmt package is required. Consumers inject a
// *Provider into usermgmt.ServiceConfig.WebAuthn to enable passkey
// registration and login.
package webauthn

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Config configures the WebAuthn Relying Party for passwordless auth.
type Config struct {
	RPID          string   // e.g. "example.com"
	RPDisplayName string   // e.g. "My App"
	RPOrigins     []string // e.g. []string{"https://example.com"}
}

// Provider implements WebAuthn passkey ceremonies using go-webauthn.
// It satisfies the usermgmt.WebAuthnProvider interface via structural typing.
type Provider struct {
	wa *webauthn.WebAuthn
}

// New creates a WebAuthn provider with the given Relying Party configuration.
func New(cfg Config) (*Provider, error) {
	wa, err := webauthn.New(&webauthn.Config{ //nolint:exhaustruct // only required fields set
		RPID:          cfg.RPID,
		RPDisplayName: cfg.RPDisplayName,
		RPOrigins:     cfg.RPOrigins,
	})
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "webauthn.create_instance", "create instance")
	}
	return &Provider{wa: wa}, nil
}

// --- internal types for JSON serialization ---

// userData is deserialized from the userJSON []byte passed by the Service.
// The field names match the JSON tags used by credentialCore in core.
type userData struct {
	ID          string           `json:"id"`
	Email       string           `json:"email"`
	DisplayName string           `json:"display_name"`
	Credentials []credentialData `json:"credentials"`
}

// credentialData mirrors the credentialCore struct in core usermgmt.
// The Name field is omitted — the Service adds it after FinishRegistration.
type credentialData struct {
	ID              []byte   `json:"id"`
	PublicKey       []byte   `json:"public_key"`
	AttestationType string   `json:"attestation_type"`
	Transports      []string `json:"transports,omitempty"`
	AAGUID          []byte   `json:"aaguid,omitempty"`
	SignCount       uint32   `json:"sign_count"`
	BackupEligible  bool     `json:"backup_eligible"`
	BackupState     bool     `json:"backup_state"`
}

// --- webauthn.User adapter ---

// webauthnUser adapts userData to the webauthn.User interface.
type webauthnUser struct {
	data userData
}

func (w *webauthnUser) WebAuthnID() []byte {
	return []byte(w.data.ID)
}

func (w *webauthnUser) WebAuthnName() string {
	return w.data.Email
}

func (w *webauthnUser) WebAuthnDisplayName() string {
	return w.data.DisplayName
}

func (w *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(w.data.Credentials))
	for i, c := range w.data.Credentials {
		creds[i] = toWebAuthnCredential(c)
	}
	return creds
}

// --- credential conversion ---

func toWebAuthnCredential(c credentialData) webauthn.Credential {
	//nolint:exhaustruct // only fields we track; rest are go-webauthn defaults
	return webauthn.Credential{
		ID:              c.ID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       toProtocolTransports(c.Transports),
		Flags: webauthn.CredentialFlags{
			UserPresent:    true,
			UserVerified:   true,
			BackupEligible: c.BackupEligible,
			BackupState:    c.BackupState,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:    c.AAGUID,
			SignCount: c.SignCount,
		},
	}
}

func fromWebAuthnCredential(c *webauthn.Credential) credentialData {
	return credentialData{
		ID:              c.ID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transports:      fromProtocolTransports(c.Transport),
		AAGUID:          c.Authenticator.AAGUID,
		SignCount:       c.Authenticator.SignCount,
		BackupEligible:  c.Flags.BackupEligible,
		BackupState:     c.Flags.BackupState,
	}
}

func toProtocolTransports(t []string) []protocol.AuthenticatorTransport {
	if t == nil {
		return nil
	}
	result := make([]protocol.AuthenticatorTransport, len(t))
	for i, v := range t {
		result[i] = protocol.AuthenticatorTransport(v)
	}
	return result
}

func fromProtocolTransports(t []protocol.AuthenticatorTransport) []string {
	if t == nil {
		return nil
	}
	result := make([]string, len(t))
	for i, v := range t {
		result[i] = string(v)
	}
	return result
}

var _ webauthn.User = (*webauthnUser)(nil)

// --- ceremony implementations ---

// marshalCeremonyResponse serializes the (options, session) pair returned by a
// WebAuthn Begin* call into two JSON blobs with consistent error wrapping.
// Shared by BeginRegistration and BeginLogin.
func marshalCeremonyResponse(options, session any) ([]byte, []byte, error) {
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "webauthn.marshal_options", "marshal options")
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return nil, nil, errorfamily.WrapInfrastructure(err, "webauthn.marshal_session", "marshal session")
	}

	return optionsJSON, sessionJSON, nil
}

// BeginRegistration starts the credential registration ceremony.
// userJSON contains serialized user data (id, email, display_name, credentials).
// Returns the CredentialCreation options (as JSON) and opaque session data.
func (p *Provider) BeginRegistration(_ context.Context, userJSON []byte) (options, sessionData []byte, err error) {
	user, err := parseUser(userJSON)
	if err != nil {
		return nil, nil, err
	}

	creation, session, err := p.wa.BeginRegistration(&webauthnUser{data: user})
	if err != nil {
		return nil, nil, errorfamily.WrapTransient(err, "webauthn.begin_registration", "begin registration")
	}

	return marshalCeremonyResponse(creation, session)
}

// FinishRegistration completes the credential registration ceremony.
// body is the raw attestation response from the authenticator.
// sessionData is the opaque session returned by BeginRegistration.
// Returns the new credential data as JSON.
func (p *Provider) FinishRegistration(ctx context.Context, userJSON, body, sessionData []byte) ([]byte, error) {
	user, err := parseUser(userJSON)
	if err != nil {
		return nil, err
	}

	session, err := parseSession(sessionData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "webauthn.create_request", "create request")
	}

	credential, err := p.wa.FinishRegistration(&webauthnUser{data: user}, session, req)
	if err != nil {
		return nil, errorfamily.WrapRejection(err, "webauthn.finish_registration", "finish registration")
	}

	credJSON, err := json.Marshal(fromWebAuthnCredential(credential))
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "webauthn.marshal_credential", "marshal credential")
	}

	return credJSON, nil
}

// BeginLogin starts the authentication ceremony.
// userJSON contains serialized user data (id, email, display_name, credentials).
// Returns the CredentialAssertion options (as JSON) and opaque session data.
func (p *Provider) BeginLogin(_ context.Context, userJSON []byte) (options, sessionData []byte, err error) {
	user, err := parseUser(userJSON)
	if err != nil {
		return nil, nil, err
	}

	assertion, session, err := p.wa.BeginLogin(&webauthnUser{data: user})
	if err != nil {
		return nil, nil, errorfamily.WrapTransient(err, "webauthn.begin_login", "begin login")
	}

	return marshalCeremonyResponse(assertion, session)
}

// FinishLogin completes the authentication ceremony.
// body is the raw assertion response from the authenticator.
// sessionData is the opaque session returned by BeginLogin.
func (p *Provider) FinishLogin(ctx context.Context, userJSON, body, sessionData []byte) error {
	user, err := parseUser(userJSON)
	if err != nil {
		return err
	}

	session, err := parseSession(sessionData)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(body))
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "webauthn.create_request", "create request")
	}

	_, err = p.wa.FinishLogin(&webauthnUser{data: user}, session, req)
	if err != nil {
		return errorfamily.WrapRejection(err, "webauthn.finish_login", "finish login")
	}

	return nil
}

// --- helpers ---

func parseUser(userJSON []byte) (userData, error) {
	var user userData
	if err := json.Unmarshal(userJSON, &user); err != nil {
		return userData{}, errorfamily.WrapCorruption(err, "webauthn.unmarshal_user", "unmarshal user data")
	}
	return user, nil
}

func parseSession(sessionData []byte) (webauthn.SessionData, error) {
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return webauthn.SessionData{}, errorfamily.WrapCorruption(
			err,
			"webauthn.unmarshal_session",
			"unmarshal session data",
		)
	}
	return session, nil
}
