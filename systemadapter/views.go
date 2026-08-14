package systemadapter

import "time"

// View structs are the read-model projections maintained by the metaengine.
// They are keyed by stream ID (string) and updated declaratively via fold
// handlers registered in evolutions.go.

// TenantView is the read-model projection for tenants.
type TenantView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Suspended   bool   `json:"suspended"`
}

// BotView is the read-model projection for bots.
type BotView struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	OwnerID   string   `json:"owner_id"`
	TokenHash []byte   `json:"token_hash"`
	Scopes    []string `json:"scopes"`
}

// MembershipView is the read-model projection for memberships.
type MembershipView struct {
	ID       string   `json:"id"`
	ActorID  string   `json:"actor_id"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
}

// CredentialView is a sub-struct of UserView for WebAuthn credentials.
type CredentialView struct {
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

// ExternalAccountView is a sub-struct of UserView for OAuth2/OIDC linked accounts.
type ExternalAccountView struct {
	Provider    string `json:"provider"`
	Subject     string `json:"subject"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

// UserView is the read-model projection for users.
// TOTPSecret is intentionally excluded for security.
type UserView struct {
	ID               string                `json:"id"`
	Email            string                `json:"email"`
	DisplayName      string                `json:"display_name,omitempty"`
	Credentials      []CredentialView      `json:"credentials,omitempty"`
	ExternalAccounts []ExternalAccountView `json:"external_accounts,omitempty"`
	EmailVerified    bool                  `json:"email_verified"`
	TOTPEnabled      bool                  `json:"totp_enabled"`
	//cqrs-lint:ignore(C013) metaengine serializes views as JSON; time.Time marshals to RFC3339 with timezone — nothing is lost
	CreatedAt time.Time `json:"created_at"`
	//cqrs-lint:ignore(C013) metaengine serializes views as JSON; time.Time marshals to RFC3339 with timezone — nothing is lost
	UpdatedAt time.Time `json:"updated_at"`
}

// BotTokenView is a secondary index projection for looking up bots by token hash.
type BotTokenView struct {
	TokenHash string `json:"token_hash"`
	BotID     string `json:"bot_id"`
}

// ExternalAccountLink is a secondary index projection for looking up users by
// external account (provider, subject) pair.
type ExternalAccountLink struct {
	ProviderSubject string `json:"provider_subject"`
	UserID          string `json:"user_id"`
}

// PolicyEntry represents a single authorization policy: which roles a subject
// holds in a domain. Keyed by the stream ID of the aggregate that produced it.
type PolicyEntry struct {
	Key     string   `json:"key"`
	Subject string   `json:"subject"`
	Domain  string   `json:"domain"`
	Roles   []string `json:"roles"`
}

// AuditEntryView is a single entry in the append-only audit log.
type AuditEntryView struct {
	EventType   string `json:"event_type"`
	AggregateID string `json:"aggregate_id"`
	//cqrs-lint:ignore(C013) metaengine serializes views as JSON; time.Time marshals to RFC3339 with timezone — nothing is lost
	OccurredAt time.Time `json:"occurred_at"`
	Action     string    `json:"action"`
}
