package usermgmt

import "time"

// WebAuthnCredential represents a registered passkey/WebAuthn credential.
// Stored as part of the User aggregate state, updated via events.
type WebAuthnCredential struct {
	ID              []byte    `json:"id"`
	PublicKey       []byte    `json:"public_key"`
	AttestationType string    `json:"attestation_type"`
	Transports      []string  `json:"transports,omitempty"`
	AAGUID          []byte    `json:"aaguid,omitempty"`
	SignCount       uint32    `json:"sign_count"`
	BackupEligible  bool      `json:"backup_eligible"`
	BackupState     bool      `json:"backup_state"`
	Name            string    `json:"name,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// newCredentialFromPayload builds a WebAuthnCredential from a
// CredentialAddedPayload, defensively copying the Transports and AAGUID
// slices so the result does not alias event payload memory.
func newCredentialFromPayload(p CredentialAddedPayload, createdAt time.Time) WebAuthnCredential {
	return WebAuthnCredential{
		ID:              p.ID,
		PublicKey:       p.PublicKey,
		AttestationType: p.AttestationType,
		Transports:      append([]string(nil), p.Transports...),
		AAGUID:          append([]byte(nil), p.AAGUID...),
		SignCount:       p.SignCount,
		BackupEligible:  p.BackupEligible,
		BackupState:     p.BackupState,
		Name:            p.Name,
		CreatedAt:       createdAt,
	}
}

// Clone returns a deep copy of the credential with independent slice fields.
func (c WebAuthnCredential) Clone() WebAuthnCredential {
	cp := c
	cp.ID = append([]byte(nil), c.ID...)
	cp.PublicKey = append([]byte(nil), c.PublicKey...)
	cp.Transports = append([]string(nil), c.Transports...)
	cp.AAGUID = append([]byte(nil), c.AAGUID...)
	return cp
}
