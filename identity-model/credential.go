package identitymodel

import "time"

// CredentialCore contains the shared WebAuthn credential fields used by both
// WebAuthnCredential (domain model) and CredentialAddedPayload (event payload).
type CredentialCore struct {
	ID              []byte   `json:"id"`
	PublicKey       []byte   `json:"public_key"`
	AttestationType string   `json:"attestation_type"`
	Transports      []string `json:"transports,omitempty"`
	AAGUID          []byte   `json:"aaguid,omitempty"`
	SignCount       uint32   `json:"sign_count"`
	UserPresent     bool     `json:"user_present,omitempty"`
	UserVerified    bool     `json:"user_verified,omitempty"`
	BackupEligible  bool     `json:"backup_eligible"`
	BackupState     bool     `json:"backup_state"`
	Name            string   `json:"name,omitempty"`
}

// WebAuthnCredential represents a registered passkey/WebAuthn credential.
type WebAuthnCredential struct {
	CredentialCore
	CreatedAt time.Time `json:"created_at"`
}

// NewCredentialFromPayload builds a WebAuthnCredential from a
// CredentialAddedPayload, defensively copying the Transports and AAGUID
// slices so the result does not alias event payload memory.
func NewCredentialFromPayload(p CredentialAddedPayload, createdAt time.Time) WebAuthnCredential {
	return WebAuthnCredential{
		ID:              p.ID,
		PublicKey:       p.PublicKey,
		AttestationType: p.AttestationType,
		Transports:      append([]string(nil), p.Transports...),
		AAGUID:          append([]byte(nil), p.AAGUID...),
		SignCount:       p.SignCount,
		UserPresent:     p.UserPresent,
		UserVerified:    p.UserVerified,
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

// CredentialUse is the credential state observed during one successful
// WebAuthn assertion. Recorded per login so the signature counter and flags
// can be persisted — without it, clone detection compares every assertion
// against the registration-time counter and never fires.
type CredentialUse struct {
	CredentialID []byte
	SignCount    uint32
	UserPresent  bool
	UserVerified bool
	BackupState  bool
}

// CounterRegressed reports whether next is a signature-counter regression, the
// WebAuthn clone signal (§7.2 step 17). Authenticators without a counter
// always report 0, so 0 → 0 is not a regression; any non-zero count that fails
// to advance is.
func (c WebAuthnCredential) CounterRegressed(next uint32) bool {
	return next <= c.SignCount && (next != 0 || c.SignCount != 0)
}

// ApplyUse returns a copy of the credential with the assertion's observed
// state applied. The caller is responsible for rejecting a regression first
// (see CounterRegressed); applying a regressed count would destroy the stored
// high-water mark.
func (c WebAuthnCredential) ApplyUse(u CredentialUse) WebAuthnCredential {
	cp := c.Clone()
	cp.SignCount = u.SignCount
	cp.UserPresent = u.UserPresent
	cp.UserVerified = u.UserVerified
	cp.BackupState = u.BackupState
	return cp
}
