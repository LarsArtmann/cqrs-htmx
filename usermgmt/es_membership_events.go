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
// Carries ActorID/TenantID for projection self-containment (CasbinProjection needs
// to know which subject+domain to update without looking up the aggregate state).
type MemberRolesChangedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ActorKind     string `json:"actor_kind"`
	ActorID       string `json:"actor_id"`
	TenantID      string `json:"tenant_id"`
	Roles         []Role `json:"roles"`
}

// MemberRemovedPayload is emitted when an actor is removed from a tenant.
// Carries ActorID/TenantID for CasbinProjection cleanup.
type MemberRemovedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	ActorID       string `json:"actor_id"`
	TenantID      string `json:"tenant_id"`
}
