package identitymodel

import (
	"encoding/json/v2"

	errorfamily "github.com/larsartmann/go-error-family"
)

// --- User event payloads ---

type UserRegisteredPayload struct {
	SchemaVersion int `json:"schema_version"`
	//cqrs-lint:ignore(F006) domain types module: encryption is infrastructure concern applied by consumers
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Roles       []Role `json:"roles"`
}

// RolesUpdatedPayload is the legacy payload for the RolesUpdated event.
type RolesUpdatedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Roles         []Role `json:"roles"`
	Domain        string `json:"domain"`
}

// cqrs-lint:ignore(S002) domain types module: encryption is an infrastructure concern applied by consumers via encryption.EncryptMiddleware
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

// cqrs-lint:ignore(P009) JSON codec chosen for cross-language interoperability; CBOR optional per consumer
type CredentialRemovedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ID            []byte `json:"id"`
}

type EmailVerifiedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
}

// cqrs-lint:ignore(P009) JSON codec chosen for cross-language interoperability; CBOR optional per consumer
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

// cqrs-lint:ignore(P009) JSON codec chosen for cross-language interoperability; CBOR optional per consumer
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
