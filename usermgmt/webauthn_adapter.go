package usermgmt

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// webauthnUser adapts our domain User to the webauthn.User interface.
type webauthnUser struct {
	user *User
}

func (w *webauthnUser) WebAuthnID() []byte {
	return []byte(w.user.ID.Get())
}

func (w *webauthnUser) WebAuthnName() string {
	return w.user.Email
}

func (w *webauthnUser) WebAuthnDisplayName() string {
	return w.user.DisplayName
}

func (w *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(w.user.Credentials))
	for i, c := range w.user.Credentials {
		creds[i] = toWebAuthnCredential(c)
	}
	return creds
}

func toWebAuthnCredential(c WebAuthnCredential) webauthn.Credential {
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

func fromWebAuthnCredential(c *webauthn.Credential, name string) WebAuthnCredential {
	//nolint:exhaustruct // CreatedAt set by event system via OccurredAt
	return WebAuthnCredential{
		credentialCore: credentialCore{
			ID:              c.ID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transports:      fromProtocolTransports(c.Transport),
			AAGUID:          c.Authenticator.AAGUID,
			SignCount:       c.Authenticator.SignCount,
			BackupEligible:  c.Flags.BackupEligible,
			BackupState:     c.Flags.BackupState,
			Name:            name,
		},
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
