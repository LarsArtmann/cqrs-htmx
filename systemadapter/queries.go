package systemadapter

import (
	"context"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// Query helpers provide Go-friendly access to the declarative projections.
// They wrap system.Get[R] / system.Find[R] calls with typed signatures.

// -------------------------------------------------------------------------
// Tenant queries
// -------------------------------------------------------------------------

// FindTenantByID returns the tenant with the given stream ID.
func FindTenantByID(ctx context.Context, sys *system.System, tenantID string) (TenantView, error) {
	return system.Get[TenantView](ctx, sys, "tenant_by_id", tenantID)
}

// FindTenantByName returns the first tenant with the given name.
func FindTenantByName(ctx context.Context, sys *system.System, name string) (TenantView, error) {
	tenants, err := system.Find[TenantView](ctx, sys, "tenants",
		system.Where("Name", name))
	if err != nil {
		return TenantView{}, err
	}
	if len(tenants) == 0 {
		return TenantView{}, system.ErrNotFound
	}
	return tenants[0], nil
}

// AllTenants returns all tenants.
func AllTenants(ctx context.Context, sys *system.System) ([]TenantView, error) {
	return system.Find[TenantView](ctx, sys, "tenants")
}

// -------------------------------------------------------------------------
// Bot queries
// -------------------------------------------------------------------------

// FindBotByID returns the bot with the given stream ID.
func FindBotByID(ctx context.Context, sys *system.System, botID string) (BotView, error) {
	return system.Get[BotView](ctx, sys, "bot_by_id", botID)
}

// FindBotsByOwner returns all bots owned by the given user ID.
func FindBotsByOwner(ctx context.Context, sys *system.System, ownerID string) ([]BotView, error) {
	return system.Find[BotView](ctx, sys, "bots",
		system.Where("OwnerID", ownerID))
}

// FindBotByTokenHash returns the bot ID for the given token hash (hex-encoded).
func FindBotByTokenHash(ctx context.Context, sys *system.System, tokenHashHex string) (BotView, error) {
	links, err := system.Find[BotTokenView](ctx, sys, "bot_tokens",
		system.Where("TokenHash", tokenHashHex))
	if err != nil {
		return BotView{}, err
	}
	if len(links) == 0 {
		return BotView{}, system.ErrNotFound
	}
	return system.Get[BotView](ctx, sys, "bot_by_id", links[0].BotID)
}

// -------------------------------------------------------------------------
// Membership queries
// -------------------------------------------------------------------------

// FindMembershipByID returns the membership with the given stream ID.
func FindMembershipByID(ctx context.Context, sys *system.System, membershipID string) (MembershipView, error) {
	return system.Get[MembershipView](ctx, sys, "membership_by_id", membershipID)
}

// FindMembershipsByActor returns all memberships for the given actor.
func FindMembershipsByActor(ctx context.Context, sys *system.System, actorID string) ([]MembershipView, error) {
	return system.Find[MembershipView](ctx, sys, "memberships",
		system.Where("ActorID", actorID))
}

// FindMembershipsByTenant returns all memberships in the given tenant.
func FindMembershipsByTenant(ctx context.Context, sys *system.System, tenantID string) ([]MembershipView, error) {
	return system.Find[MembershipView](ctx, sys, "memberships",
		system.Where("TenantID", tenantID))
}

// -------------------------------------------------------------------------
// User queries
// -------------------------------------------------------------------------

// FindUserByID returns the user with the given stream ID.
func FindUserByID(ctx context.Context, sys *system.System, userID string) (UserView, error) {
	return system.Get[UserView](ctx, sys, "user_by_id", userID)
}

// FindUserByEmail returns the first user with the given email.
func FindUserByEmail(ctx context.Context, sys *system.System, email string) (UserView, error) {
	users, err := system.Find[UserView](ctx, sys, "users",
		system.Where("Email", email))
	if err != nil {
		return UserView{}, err
	}
	if len(users) == 0 {
		return UserView{}, system.ErrNotFound
	}
	return users[0], nil
}

// AllUsers returns all users, sorted by creation time ascending.
func AllUsers(ctx context.Context, sys *system.System) ([]UserView, error) {
	return system.Find[UserView](ctx, sys, "users")
}

// FindUserByExternalAccount returns the user linked to the given external account.
func FindUserByExternalAccount(ctx context.Context, sys *system.System, provider, subject string) (UserView, error) {
	links, err := system.Find[ExternalAccountLink](ctx, sys, "external_account_links",
		system.Where("ProviderSubject", provider+":"+subject))
	if err != nil {
		return UserView{}, err
	}
	if len(links) == 0 {
		return UserView{}, system.ErrNotFound
	}
	return system.Get[UserView](ctx, sys, "user_by_id", links[0].UserID)
}

// -------------------------------------------------------------------------
// Authz queries
// -------------------------------------------------------------------------

// FindPolicyByStreamID returns the policy entry for the given aggregate stream ID.
func FindPolicyByStreamID(ctx context.Context, sys *system.System, streamID string) (PolicyEntry, error) {
	return system.Get[PolicyEntry](ctx, sys, "authz_policy_by_id", streamID)
}

// FindPolicies returns all policy entries matching the given subject and/or domain.
// Pass empty strings to skip a filter.
func FindPolicies(ctx context.Context, sys *system.System, subject, domain string) ([]PolicyEntry, error) {
	var opts []system.FindOption
	if subject != "" {
		opts = append(opts, system.Where("Subject", subject))
	}
	if domain != "" {
		opts = append(opts, system.Where("Domain", domain))
	}
	return system.Find[PolicyEntry](ctx, sys, "authz_policies", opts...)
}

// Enforce checks whether the subject holds any role in the given domain that
// grants the requested action. Role inheritance is applied: admin implies user,
// user implies viewer, super_admin implies all.
func Enforce(ctx context.Context, sys *system.System, subject, domain, action string) (bool, error) {
	entries, err := FindPolicies(ctx, sys, subject, domain)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		for _, role := range entry.Roles {
			if roleGrantsAction(identitymodel.Role(role), action) {
				return true, nil
			}
		}
	}
	return false, nil
}

// roleGrantsAction checks whether a role grants the requested action, applying
// the inheritance hierarchy: super_admin > admin > user > viewer.
func roleGrantsAction(role identitymodel.Role, action string) bool {
	levels := map[identitymodel.Role]int{
		identitymodel.RoleSuperAdmin: 4,
		identitymodel.RoleAdmin:      3,
		identitymodel.RoleOwner:      3,
		identitymodel.RoleUser:       2,
		identitymodel.RoleViewer:     1,
	}
	required, ok := actionLevel(action)
	if !ok {
		return false
	}
	granted, ok := levels[role]
	if !ok {
		return false
	}
	return granted >= required
}

func actionLevel(action string) (int, bool) {
	switch strings.ToLower(action) {
	case "view", "read":
		return 1, true
	case "use", "write", "edit":
		return 2, true
	case "manage", "admin":
		return 3, true
	case "super", "everything":
		return 4, true
	default:
		return 0, false
	}
}

// -------------------------------------------------------------------------
// AuditLog queries
// -------------------------------------------------------------------------

// AuditEntries returns all audit entries, sorted by occurred-at descending.
func AuditEntries(ctx context.Context, sys *system.System) ([]AuditEntryView, error) {
	return system.Find[AuditEntryView](ctx, sys, "audit_log")
}

// AuditEntriesFor returns all audit entries for the given aggregate ID.
func AuditEntriesFor(ctx context.Context, sys *system.System, aggregateID string) ([]AuditEntryView, error) {
	return system.Find[AuditEntryView](ctx, sys, "audit_log",
		system.Where("AggregateID", aggregateID))
}

// RecentAuditEntries returns the N most recent audit entries.
func RecentAuditEntries(ctx context.Context, sys *system.System, n int) ([]AuditEntryView, error) {
	return system.Find[AuditEntryView](ctx, sys, "audit_log",
		system.OrderBy("OccurredAt", system.Desc),
		system.Limit(n))
}
