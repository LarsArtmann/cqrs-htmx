package usermgmt

import (
	"encoding/json"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

type UserRegisteredPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
	DisplayName   string `json:"display_name,omitempty"`
	Roles         []Role `json:"roles"`
}

// RolesUpdatedPayload is the legacy payload for the RolesUpdated event.
// No longer emitted (roles are managed by the Membership aggregate), but
// retained for decoding existing events in stores and the migration tool.
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
	credentialCore
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

type ExternalAccountLinkedPayload struct {
	SchemaVersion int `json:"schema_version"`
	externalAccountCore
}

type ExternalAccountUnlinkedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Provider      string `json:"provider"`
	Subject       string `json:"subject"`
}

func marshalPayload(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "usermgmt.payload.marshal_failed", "marshal payload")
	}
	return b, nil
}
