package identitymodel

import (
	"encoding/json/v2"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// --- User event payloads ---

type UserRegisteredPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name,omitempty"`
	Roles         []Role `json:"roles"`
}

// RolesUpdatedPayload is the legacy payload for the RolesUpdated event.
type RolesUpdatedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Roles         []Role `json:"roles"`
	Domain        string `json:"domain"`
}

type EmailChangedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
}

type DisplayNameChangedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	DisplayName   string `json:"display_name"`
}

type UserDeletedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}

type CredentialAddedPayload struct {
	SchemaVersion int `json:"schema_version"`
	CredentialCore
}

type CredentialRemovedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ID            []byte `json:"id"`
}

// CredentialUsedPayload records the credential state observed during one
// successful WebAuthn assertion: the signature counter (the clone-detection
// high-water mark) and the flags the authenticator reported.
type CredentialUsedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	CredentialID  []byte `json:"credential_id"`
	SignCount     uint32 `json:"sign_count"`
	UserPresent   bool   `json:"user_present,omitempty"`
	UserVerified  bool   `json:"user_verified,omitempty"`
	BackupState   bool   `json:"backup_state"`
}

type EmailVerifiedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
}

type TOTPEnabledPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Secret        []byte `json:"secret"`
}

type TOTPDisabledPayload struct {
	SchemaVersion int `json:"schema_version"`
}

type ExternalAccountLinkedPayload struct {
	SchemaVersion int `json:"schema_version"`
	ExternalAccountCore
}

type ExternalAccountUnlinkedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Provider      string `json:"provider"`
	Subject       string `json:"subject"`
}

// --- Membership event payloads ---

type MemberAddedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ActorKind     string `json:"actor_kind"`
	ActorID       string `json:"actor_id"`
	TenantID      string `json:"tenant_id"`
	Roles         []Role `json:"roles"`
}

type MemberRolesChangedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ActorKind     string `json:"actor_kind"`
	ActorID       string `json:"actor_id"`
	TenantID      string `json:"tenant_id"`
	Roles         []Role `json:"roles"`
}

type MemberRemovedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ActorID       string `json:"actor_id"`
	TenantID      string `json:"tenant_id"`
}

// --- Tenant event payloads ---

type TenantCreatedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
}

type TenantSuspendedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}

type TenantReactivatedPayload struct {
	SchemaVersion int `json:"schema_version"`
}

type TenantDeletedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}

// --- Bot event payloads ---

type BotRegisteredPayload struct {
	SchemaVersion int      `json:"schema_version"`
	Name          string   `json:"name"`
	OwnerID       UserID   `json:"owner_id"`
	TokenHash     []byte   `json:"token_hash"`
	Scopes        []string `json:"scopes"`
}

type BotDeletedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}

// --- Auth audit event payloads ---
//
// The three auth events (LoginSucceeded, LoginFailed, RegistrationClosed)
// are SECURITY AUDIT facts, not user state: they are appended to dedicated
// Auth-typed streams (one fresh stream per event) so login volume never
// grows the user aggregate a command must fold on every dispatch. Never
// place them on user streams — see docs/guides/actor-and-audit-trail.md.

// AuthMethod is the bounded vocabulary of authentication methods recorded
// in auth audit payloads.
type AuthMethod string

const (
	AuthMethodWebAuthN AuthMethod = "webauthn"
	AuthMethodTOTP     AuthMethod = "totp"
	AuthMethodOAuth2   AuthMethod = "oauth2"
)

// AuthFailureReason is the bounded vocabulary of login failure reason
// classes. Values are coarse classifications, never provider error text or
// credential material — the payload must be safe to journal verbatim.
type AuthFailureReason string

const (
	AuthFailureNoCredentials  AuthFailureReason = "no_credentials"
	AuthFailureCeremonyFailed AuthFailureReason = "ceremony_failed"
	AuthFailureCodeInvalid    AuthFailureReason = "code_invalid"
	AuthFailureStateInvalid   AuthFailureReason = "state_invalid"
	AuthFailureProviderMismat AuthFailureReason = "provider_mismatch"
	AuthFailureTokenExchange  AuthFailureReason = "token_exchange"
	AuthFailureNoEmail        AuthFailureReason = "no_email"
)

type LoginSucceededPayload struct {
	SchemaVersion int        `json:"schema_version"`
	Method        AuthMethod `json:"method"`
	// Provider qualifies AuthMethodOAuth2 (e.g. "google"); empty otherwise.
	Provider string `json:"provider,omitempty"`
}

type LoginFailedPayload struct {
	SchemaVersion int               `json:"schema_version"`
	Method        AuthMethod        `json:"method"`
	Reason        AuthFailureReason `json:"reason"`
	// Provider qualifies AuthMethodOAuth2; empty otherwise.
	Provider string `json:"provider,omitempty"`
}

type RegistrationClosedPayload struct {
	SchemaVersion int `json:"schema_version"`
}

// SessionEndCause is the bounded vocabulary of why a session ended, recorded
// in SessionEndedPayload. Coarse classes only — never raw error text — so the
// journal stays safe to display verbatim.
type SessionEndCause string

const (
	// SessionEndLogout: the session holder logged out explicitly.
	SessionEndLogout SessionEndCause = "logout"
	// SessionEndExpired: a lazy expiry check removed the session after its
	// TTL passed.
	SessionEndExpired SessionEndCause = "expired"
	// SessionEndImpersonationEnded: an impersonation session was ended via
	// EndImpersonation.
	SessionEndImpersonationEnded SessionEndCause = "impersonation_ended"
	// SessionEndRevoked: all sessions for a user were revoked in one sweep
	// (user deletion or admin revoke-all).
	SessionEndRevoked SessionEndCause = "revoked"
	// SessionEndDeleted is the fallback for deletions that carry no specific
	// cause.
	SessionEndDeleted SessionEndCause = "deleted"
)

// SessionStartedPayload records that a session was established — the durable
// half of the session lifecycle audit journal (cqrs-htmx#25). Sessions stay
// ephemeral by design; these events make impersonation forensics answerable
// long after the session rows are gone: "who impersonated user X, when, and
// why" reads from the journal, not from the session store.
type SessionStartedPayload struct {
	SchemaVersion int `json:"schema_version"`
	// ActorID is the session's actor in "kind:raw" form — for an
	// impersonation session this is the TARGET being impersonated.
	ActorID string `json:"actor_id"`
	// Impersonator is the admin who initiated the impersonation ("kind:raw");
	// empty for direct logins.
	Impersonator string `json:"impersonator,omitempty"`
	// Reason is the mandatory impersonation justification; empty for direct
	// logins.
	Reason string `json:"reason,omitempty"`
	// StartedAt is when the session was established (session CreatedAt).
	StartedAt time.Time `json:"started_at"`
	// ExpiresAt is when the session would expire — the impersonation window.
	ExpiresAt time.Time `json:"expires_at"`
	// TokenHash is the base64url SHA-256 of the session token: a correlation
	// key linking SessionStarted and SessionEnded for the same session. The
	// token itself is 256-bit random and NEVER journaled; the digest cannot
	// be reversed or brute-forced.
	TokenHash string `json:"token_hash"`
}

// SessionEndedPayload records that a session (or a user's whole session set)
// ended, with a bounded cause. ActorID and TokenHash are omitted for
// user-scoped revocations, which end every session a user holds at once.
type SessionEndedPayload struct {
	SchemaVersion int `json:"schema_version"`
	// ActorID is the ended session's actor in "kind:raw" form; empty when the
	// event covers a user-scoped revocation without a single session (the
	// payload then names the user via the event's actor metadata).
	ActorID      string           `json:"actor_id,omitempty"`
	Impersonator string           `json:"impersonator,omitempty"`
	TokenHash    string           `json:"token_hash,omitempty"`
	Cause        SessionEndCause  `json:"cause"`
}

// marshalJSONOrWrap serializes v with encoding/json and wraps any error as
// an Infrastructure failure with the caller's error code and human-readable
// message. Shared by MarshalPayload, ActorID.MarshalJSON, and User.MarshalJSON.
func marshalJSONOrWrap(v any, errCode, errMsg string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, errCode, errMsg)
	}
	return b, nil
}

// MarshalPayload serializes an event payload to JSON.
func MarshalPayload(v any) ([]byte, error) {
	return marshalJSONOrWrap(v, "usermgmt.payload.marshal_failed", "marshal payload")
}
