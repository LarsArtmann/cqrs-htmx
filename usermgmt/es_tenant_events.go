package usermgmt

// TenantCreatedPayload is emitted when a new tenant is created.
type TenantCreatedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	DisplayName   string `json:"display_name"`
}

// TenantSuspendedPayload is emitted when a tenant is temporarily suspended.
type TenantSuspendedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}

// TenantReactivatedPayload is emitted when a suspended tenant is restored.
type TenantReactivatedPayload struct {
	SchemaVersion int `json:"schema_version"`
}

// TenantDeletedPayload is emitted when a tenant is permanently deleted.
type TenantDeletedPayload struct {
	SchemaVersion int    `json:"schema_version"`
	Reason        string `json:"reason"`
}
