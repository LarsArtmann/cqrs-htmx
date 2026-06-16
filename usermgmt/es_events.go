package usermgmt

import (
	"encoding/json"
	"fmt"
)

type UserRegisteredPayload struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name,omitempty"`
	Roles       []Role `json:"roles"`
}

type RolesUpdatedPayload struct {
	Roles  []Role `json:"roles"`
	Domain string `json:"domain"`
}

type EmailChangedPayload struct {
	Email string `json:"email"`
}

type DisplayNameChangedPayload struct {
	DisplayName string `json:"display_name"`
}

type UserDeletedPayload struct {
	Reason string `json:"reason"`
}

type CredentialAddedPayload struct {
	ID              []byte   `json:"id"`
	PublicKey       []byte   `json:"public_key"`
	AttestationType string   `json:"attestation_type"`
	Transports      []string `json:"transports,omitempty"`
	AAGUID          []byte   `json:"aaguid,omitempty"`
	BackupEligible  bool     `json:"backup_eligible"`
	BackupState     bool     `json:"backup_state"`
	Name            string   `json:"name,omitempty"`
}

type CredentialRemovedPayload struct {
	ID []byte `json:"id"`
}

func marshalPayload(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	return b, nil
}
