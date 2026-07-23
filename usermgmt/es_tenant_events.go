package usermgmt

import identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"

type (
	TenantCreatedPayload     = identitymodel.TenantCreatedPayload
	TenantSuspendedPayload   = identitymodel.TenantSuspendedPayload
	TenantReactivatedPayload = identitymodel.TenantReactivatedPayload
	TenantDeletedPayload     = identitymodel.TenantDeletedPayload
)
