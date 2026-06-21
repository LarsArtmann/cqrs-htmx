package usermgmt

// MemberAddedPayload is emitted when an actor joins a tenant with initial roles.
type MemberAddedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ActorKind     string `json:"actor_kind"`
	ActorID       string `json:"actor_id"`
	TenantID      string `json:"tenant_id"`
	Roles         []Role `json:"roles"`
}

// MemberRolesChangedPayload is emitted when an actor's roles change within a tenant.
type MemberRolesChangedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Roles         []Role `json:"roles"`
}

// MemberRemovedPayload is emitted when an actor is removed from a tenant.
type MemberRemovedPayload struct {
	SchemaVersion int `json:"schema_version"`
}
