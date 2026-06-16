package usermgmt

import (
	"encoding/json"
	"fmt"
)

type UserRegisteredPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name,omitempty"`
	Roles         []Role `json:"roles"`
}

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
	SchemaVersion   int      `json:"schema_version"`
	ID              []byte   `json:"id"`
	PublicKey       []byte   `json:"public_key"`
	AttestationType string   `json:"attestation_type"`
	Transports      []string `json:"transports,omitempty"`
	AAGUID          []byte   `json:"aaguid,omitempty"`
	SignCount       uint32   `json:"sign_count"`
	BackupEligible  bool     `json:"backup_eligible"`
	BackupState     bool     `json:"backup_state"`
	Name            string   `json:"name,omitempty"`
}

type CredentialRemovedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ID            []byte `json:"id"`
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

func marshalPayload(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return b, nil
}
