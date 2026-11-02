package setup

import (
	"log/slog"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// grantGlobalAdminSnippet is the copy-paste fix printed by the boot diagnostic.
// It is the durable path ([usermgmt.Service.GrantRoles] over AddMember), not the
// in-memory enforcer write that loses the grant on restart.
const grantGlobalAdminSnippet = "svc.GrantRoles(ctx, usermgmt.ActorIDFromUser(userID), " +
	"identitymodel.NewTenantID(identitymodel.WildcardDomain), " +
	"[]identitymodel.Role{identitymodel.RoleSuperAdmin})"

// warnOnUnreachableAdminPanel logs a boot diagnostic when the admin panel is
// mounted but no actor holds an admin-tier role in the wildcard domain.
//
// The failure it prevents is silent and expensive: the bundle mounts /admin/
// unconditionally whenever the panel is enabled, the panel's default authorizer
// requires super_admin or admin in "*", and a consumer that never grants one
// ships a dashboard that answers 403 to every request — including the operator's
// own. Three consumption audits found exactly that (one service grants it once
// at boot behind a "users exist" guard and loses it on restart, one re-grants it
// on every drain, one never grants it at all).
//
// The warning is deliberately a warning and not an error: a fresh installation
// legitimately has no admin until the first user is promoted, and refusing to
// start there would be worse than starting with a closed panel.
//
// It is skipped under AsyncStartup, where the projections are still replaying
// when New returns and an existing grant may simply not be materialized yet.
func (b *Bundle) warnOnUnreachableAdminPanel() {
	if b.Admin == nil || b.Service == nil {
		return
	}

	if b.config.asyncStartup() {
		return
	}

	logger := b.config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	domain := identitymodel.NewTenantID(identitymodel.WildcardDomain)

	hasAdmin, err := b.Service.HasAdminGrant(domain)
	if err != nil {
		logger.Warn(
			"setup: admin panel is mounted but its admin grant could not be verified",
			"admin_path", b.config.AdminPath,
			"error", err,
		)

		return
	}

	if hasAdmin {
		return
	}

	logger.Warn(
		"setup: admin panel is mounted but nobody holds admin or super_admin in the wildcard domain — every panel request will be refused with 403",
		"admin_path",
		b.config.AdminPath,
		"fix",
		grantGlobalAdminSnippet,
	)
}

// asyncStartup reports whether the service was constructed without waiting for
// the projections to drain, from either the flattened field or the
// ServiceConfig escape hatch.
func (c Config) asyncStartup() bool {
	if c.AsyncStartup {
		return true
	}

	return c.ServiceConfig != nil && c.ServiceConfig.AsyncStartup
}
