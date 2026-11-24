package main

import (
	"context"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	totp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// newSeedTestService builds the same in-memory service the demo's main() uses
// (audit projection + TOTP strategy + bot token pepper) so seed() runs the
// real published command paths.
func newSeedTestService(t *testing.T) *usermgmt.Service {
	t.Helper()

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{ //nolint:exhaustruct // demo/test uses in-memory defaults
		AuditLog:    usermgmt.NewAuditLog(),
		TOTP:        totp.New(totp.Config{Issuer: "cqrs-htmx Demo Test"}),
		TokenPepper: usermgmt.TokenPepper("admin-demo-test-pepper-not-production"),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	return svc
}

// TestSeed_CastIsVisibleAndHonest walks the seeded cast through the same
// read APIs the panel renders from (GetUser/GetBot/GetTenant/AuditLog) and
// pins the demo's honesty contract: every seeded fact must be a fact the
// service genuinely derived from journaled commands — no read-model writes.
func TestSeed_CastIsVisibleAndHonest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newSeedTestService(t)
	seed(ctx, svc) // log.Fatalf inside on hard failure; in-memory always succeeds

	// F188 — bots: two, one with scopes, one deliberately scope-less.
	scoped, err := svc.GetBot(ctx, demoBotID("demo-deploy-bot"))
	if err != nil {
		t.Fatalf("GetBot demo-deploy-bot: %v", err)
	}
	if len(scoped.Scopes) == 0 {
		t.Errorf("demo-deploy-bot has no scopes, want users:read + tenants:read")
	}
	unscoped, err := svc.GetBot(ctx, demoBotID("demo-ci-bot"))
	if err != nil {
		t.Fatalf("GetBot demo-ci-bot: %v", err)
	}
	if len(unscoped.Scopes) != 0 {
		t.Errorf("demo-ci-bot scopes = %v, want none (the unscoped demo state)", unscoped.Scopes)
	}

	// F189 — the equipped user: passkey fixture + active TOTP.
	erin := identitymodel.SyntheticUserID("seed-" + equippedUserEmail)
	u, err := svc.GetUser(ctx, erin)
	if err != nil {
		t.Fatalf("GetUser erin: %v", err)
	}
	if len(u.Credentials) != 1 {
		t.Errorf("erin credentials = %d, want 1 (the seeded passkey fixture)", len(u.Credentials))
	}
	if !u.TOTPEnabled {
		t.Error("erin TOTPEnabled = false, want true (EnableTOTP + VerifyTOTPSetup ran)")
	}

	// F190 — tenant lifecycle: suspended stays visible with its badge state;
	// deleted leaves the read model (the pinned in-memory projection drops the
	// row — the TenantDeleted tombstone lives in the journal only, so the demo
	// asserts the pinned semantics rather than an idealized tombstone row).
	suspended, err := svc.GetTenant(ctx, identitymodel.NewTenantID("suspendo"))
	if err != nil {
		t.Fatalf("GetTenant suspendo: %v", err)
	}
	if !suspended.Suspended || suspended.Deleted {
		t.Errorf("suspendo = {Suspended:%v Deleted:%v}, want suspended only", suspended.Suspended, suspended.Deleted)
	}
	if _, err := svc.GetTenant(ctx, identitymodel.NewTenantID("vanisho")); err == nil {
		t.Error("GetTenant vanisho returned a tenant — a deleted tenant must leave the read model")
	}

	// F191 (role variety) — the UpdateMemberRoles seeds landed in the
	// membership read model: bob now admin in acme, carol user in globex.
	members := svc.TenantMembers(ctx, identitymodel.NewTenantID("acme"))
	bobRoles := memberRoles(members, "seed-bob@acme.dev")
	if !bobRoles[identitymodel.RoleAdmin] || bobRoles[identitymodel.RoleUser] {
		t.Errorf("bob acme roles = %v, want exactly admin after the seeded role change", bobRoles)
	}

	// F192 — the tenant-admin demo path: alice keeps her owner role in acme.
	alice := identitymodel.ActorIDFromUser(identitymodel.SyntheticUserID("seed-alice@acme.dev"))
	if !memberRoles(members, "seed-alice@acme.dev")[identitymodel.RoleOwner] {
		t.Errorf("alice (actor %v) lost the owner role in acme — the tenant-admin demo path would break", alice)
	}
}

// memberRoles collects a member's roles from the tenant members list by the
// seed-derived user ID ("seed-" + email → SyntheticUserID → actor).
func memberRoles(members []*usermgmt.Membership, seedEmail string) map[identitymodel.Role]bool {
	actor := identitymodel.ActorIDFromUser(identitymodel.SyntheticUserID(seedEmail))
	roles := make(map[identitymodel.Role]bool)
	for _, m := range members {
		if m.ActorID == actor {
			for _, r := range m.Roles {
				roles[r] = true
			}
		}
	}

	return roles
}

// TestSeed_AuditVarietyIsJournaled pins F191: the seed produces audit entries
// through real published paths only — login activity via VerifyTOTP (one
// deliberate failure, then a success), enrollment via AddCredential and the
// TOTP handshake, sessions via Register. Role changes journal MemberRolesChanged
// on the membership stream, which the pinned AuditLog projection does NOT map
// to an audit action (its roles_updated entry keys on the legacy user-stream
// RolesUpdated event no published command emits at this pin) — so the read
// model carries the role change, and the audit page honestly shows none.
// Lockout failures are NOT faked: this demo configures no LockoutStore.
func TestSeed_AuditVarietyIsJournaled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newSeedTestService(t)
	seed(ctx, svc)

	wanted := map[string]bool{
		usermgmt.AuditActionLoginFailed:     false,
		usermgmt.AuditActionLoginSucceeded:  false,
		usermgmt.AuditActionCredentialAdded: false,
		usermgmt.AuditActionTOTPEnabled:     false,
		usermgmt.AuditActionSessionStarted:  false,
	}
	for _, e := range svc.AuditLog().Entries() {
		if _, tracked := wanted[e.Action]; tracked {
			wanted[e.Action] = true
		}
	}
	for action, seen := range wanted {
		if !seen {
			t.Errorf("audit log has no %q entry — the seed for it did not journal", action)
		}
	}

	// The locked-out user demos absence, not presence: registered, but with
	// no sign-in method whatsoever.
	mallory := identitymodel.SyntheticUserID("seed-" + lockedOutEmail)
	u, err := svc.GetUser(ctx, mallory)
	if err != nil {
		t.Fatalf("GetUser mallory: %v", err)
	}
	if len(u.Credentials) != 0 || len(u.ExternalAccounts) != 0 || u.TOTPEnabled {
		t.Errorf("mallory must have no sign-in methods, got credentials=%d external=%d totp=%v",
			len(u.Credentials), len(u.ExternalAccounts), u.TOTPEnabled)
	}
}
