// Demo-cast seeding: the second seed wave (plan T26). Where the seed() in
// main.go fills the happy paths (users, memberships, tenants, pagination),
// seedCast plants the edge states the panel's surfaces are built for: machine
// actors (bots), a fully-equipped human (passkey + MFA + a real login-audit
// trail), tenant lifecycle states (suspended, deleted), varied membership
// roles, and a user who cannot sign in at all.
//
// Everything here travels the published Service methods only — the same
// commands the panel dispatches — so every seeded fact is a fact the event
// journal genuinely holds. No read-model writes, no synthetic events.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 TOTP is SHA-1 by spec (HOTP/TOTP RFC 4226/6238)
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// The named cast. Fixed emails keep every boot deterministic (same derivation
// as the main.go seeds: "seed-" + email → SyntheticUserID → stable ULID).
const (
	// equippedUserEmail is the fully-equipped human: passkey + TOTP MFA plus
	// an honest login-audit trail (one failed verify, then a success).
	equippedUserEmail = "erin@acme.dev"
	// lockedOutEmail is registered with NO sign-in method: no passkey, no
	// TOTP, no external account. The panel's user detail shows the resulting
	// "cannot sign in" state honestly — exactly what an operator sees before
	// the user enrolls their own passkey on first login.
	lockedOutEmail = "mallory@acme.dev"
)

// seedCast plants the demo cast on top of the base seeds. Errors are logged,
// never fatal: an in-memory boot re-seeds from scratch every time, and a
// partial cast still demos every surface that did seed.
func seedCast(ctx context.Context, svc *usermgmt.Service) {
	seedBots(ctx, svc)
	seedEquippedUser(ctx, svc)
	seedTenantLifecycle(ctx, svc)
	seedRoleChanges(ctx, svc)
	seedLockedOutUser(ctx, svc)
}

// seedBots registers two machine actors. The raw API token returned by
// RegisterBot exists only in this moment — the service persists an HMAC hash
// — so the demo prints it once, prominently, exactly like a real operator
// flow. There is no second chance by design.
func seedBots(ctx context.Context, svc *usermgmt.Service) {
	adminID := identitymodel.SyntheticUserID(adminUserID)
	for _, b := range []struct {
		name   string
		scopes []string
	}{
		// The scope-less bot demos the unscoped state: the panel and the
		// API-token middleware can show "has no scopes" instead of guessing.
		{name: "demo-ci-bot"},
		// The scoped bot demos the read-only integration shape. Scopes are
		// free-form labels (no enforced vocabulary at this pin) — keep them
		// resource:verb so a future enforcement pass maps cleanly.
		{name: "demo-deploy-bot", scopes: []string{"users:read", "tenants:read"}},
	} {
		res, err := svc.RegisterBot(ctx, usermgmt.RegisterBotRequest{
			ID:      demoBotID(b.name),
			Name:    b.name,
			OwnerID: adminID,
			Scopes:  b.scopes,
		})
		if err != nil {
			log.Printf("seed bot %s: %v", b.name, err)

			continue
		}
		fmt.Printf("bot %q API token (SHOWN ONCE — only the HMAC hash is stored): %s\n", b.name, res.Token)
	}
}

// demoBotID derives a deterministic bot ID. Bot aggregates key their stream
// on a ULID (the service parses the raw BotID string as a stream ID), so the
// demo reuses the users' SyntheticUserID hashing to mint a stable ULID and
// wraps it as a BotID.
func demoBotID(name string) identitymodel.BotID {
	return identitymodel.NewBotID(identitymodel.SyntheticUserID("seed-bot-" + name).Get().String())
}

// seedEquippedUser registers erin and equips her through the published
// enrollment paths: AddCredential for a synthetic passkey fixture, then the
// EnableTOTP → VerifyTOTPSetup handshake with the demo playing the
// authenticator app (demoTOTPCode). Finally two real VerifyTOTP calls —
// wrong code, then right one — journal login_failed and login_succeeded so
// the panel's audit page shows actual login activity, not synthetic rows.
func seedEquippedUser(ctx context.Context, svc *usermgmt.Service) {
	uid := identitymodel.SyntheticUserID("seed-" + equippedUserEmail)
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: uid, Email: equippedUserEmail, DisplayName: "Erin (passkey + MFA)",
	}); err != nil {
		log.Printf("seed register %s: %v", equippedUserEmail, err)

		return
	}

	// Synthetic passkey via the operator enrollment path. AddCredential is
	// the same Service method an import/migration tool would call; the
	// browser WebAuthn ceremony (challenge → attestation → FinishLogin) is
	// out of scope for a demo process, so the fixture carries fixed bytes.
	// WebAuthnCredential is exempt from exhaustruct repo-wide.
	cred := identitymodel.WebAuthnCredential{
		CredentialCore: identitymodel.CredentialCore{
			ID:              []byte("admin-demo-passkey-erin-01"),
			PublicKey:       []byte("admin-demo-seed-public-key-not-cryptographically-verifiable"),
			AttestationType: "none",
			Transports:      []string{"internal", "hybrid"},
			AAGUID:          nil,
			SignCount:       0,
			UserPresent:     true,
			UserVerified:    true,
			BackupEligible:  false,
			BackupState:     false,
			Name:            "demo passkey (seeded)",
		},
		CreatedAt: time.Now(),
	}
	if err := svc.AddCredential(ctx, uid, cred); err != nil {
		log.Printf("seed passkey %s: %v", equippedUserEmail, err)

		return
	}

	// TOTP enrollment: EnableTOTP returns the base32 secret an authenticator
	// app would scan; VerifyTOTPSetup activates it once a valid code comes
	// back. The demo computes that code itself (see demoTOTPCode).
	setup, err := svc.EnableTOTP(ctx, uid)
	if err != nil {
		log.Printf("seed totp setup %s: %v", equippedUserEmail, err)

		return
	}
	if err := svc.VerifyTOTPSetup(ctx, uid, demoTOTPCode(setup.Secret, time.Now())); err != nil {
		log.Printf("seed totp verify %s: %v", equippedUserEmail, err)

		return
	}

	// Login-audit variety, produced by the same path a real second-factor
	// check uses. A rejected code journals login_failed (reason: invalid);
	// the accepted one journals login_succeeded. Neither call touches the
	// (not configured) lockout counter — the demo stays truthful about that
	// too: lockout badges come from a LockoutStore, which this demo has none of.
	_ = svc.VerifyTOTP(ctx, uid, "000000") // deliberately wrong → login_failed
	if err := svc.VerifyTOTP(ctx, uid, demoTOTPCode(setup.Secret, time.Now())); err != nil {
		log.Printf("seed login audit %s: %v", equippedUserEmail, err)
	}
}

// demoTOTPCode computes the current RFC 6238 code for a base32 secret using
// the pinned totp provider's defaults (HMAC-SHA-1, 6 digits, 30-second
// steps). The provider exposes ValidateCode only — no code generator — so
// the demo plays the authenticator app with ~15 lines of stdlib instead of
// importing anything new. A decode failure yields a code that cannot verify,
// which the seed logs and moves on from.
func demoTOTPCode(base32Secret string, at time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(base32Secret)
	if err != nil {
		log.Printf("seed totp decode: %v", err)

		return ""
	}

	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(at.Unix())/30)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (uint32(sum[offset])&0x7f)<<24 | // dynamic truncation per RFC 4226 §5.3
		uint32(sum[offset+1])<<16 |
		uint32(sum[offset+2])<<8 |
		uint32(sum[offset+3])

	return fmt.Sprintf("%06d", code%1_000_000)
}

// seedTenantLifecycle creates two dedicated tenants and drives them into the
// lifecycle states the tenants page renders: suspended (read-only mode,
// badge + reason) and deleted. The base tenants (acme, globex, initech) stay
// healthy — alice's tenant-admin panel depends on it. At this pin the
// in-memory read model DROPS deleted tenants (the TenantDeleted event lives
// in the journal only), so vanisho demos the delete path rather than a
// visible tombstone row.
func seedTenantLifecycle(ctx context.Context, svc *usermgmt.Service) {
	for _, t := range []struct {
		id, name, display string
		suspend           bool
		reason            string
	}{
		{
			id: "suspendo", name: "suspendo", display: "Suspendo Ltd", suspend: true,
			reason: "demo seed: terms-of-service review",
		},
		{
			id: "vanisho", name: "vanisho", display: "Vanisho GmbH", suspend: false,
			reason: "demo seed: tenant-requested deletion",
		},
	} {
		tid := identitymodel.NewTenantID(t.id)
		if _, err := svc.CreateTenant(ctx, usermgmt.CreateTenantRequest{
			ID: tid, Name: t.name, DisplayName: t.display,
		}); err != nil {
			log.Printf("seed tenant %s: %v", t.id, err)

			continue
		}
		if t.suspend {
			if err := svc.SuspendTenant(ctx, tid, t.reason); err != nil {
				log.Printf("seed suspend %s: %v", t.id, err)
			}

			continue
		}
		if err := svc.DeleteTenant(ctx, tid, t.reason); err != nil {
			log.Printf("seed delete %s: %v", t.id, err)
		}
	}
}

// seedRoleChanges journals real role transitions on existing members so the
// members table shows varied roles (bob gains admin in acme, carol becomes a
// user in globex). At this pin the change journals a MemberRolesChanged event
// on the membership stream — deliberately NOT claimed as an audit-page entry:
// the AuditLog projection maps only the legacy user-stream RolesUpdated event,
// which no published command emits here, so the audit page honestly shows
// nothing for a membership role change at this pin.
func seedRoleChanges(ctx context.Context, svc *usermgmt.Service) {
	for _, rc := range []struct {
		email, tenant string
		roles         []identitymodel.Role
	}{
		{"bob@acme.dev", "acme", []identitymodel.Role{identitymodel.RoleAdmin}},
		{"carol@other.dev", "globex", []identitymodel.Role{identitymodel.RoleUser}},
	} {
		uid := identitymodel.SyntheticUserID("seed-" + rc.email)
		err := svc.UpdateMemberRoles(
			ctx,
			identitymodel.ActorIDFromUser(uid),
			identitymodel.NewTenantID(rc.tenant),
			rc.roles,
		)
		if err != nil {
			log.Printf("seed role change %s/%s: %v", rc.email, rc.tenant, err)
		}
	}
}

// seedLockedOutUser registers mallory with no credential, no TOTP, and no
// external account: the user detail page shows the can't-sign-in state every
// passwordless import starts from. Deliberately not given a session beyond
// the registration default — the point is that there is nothing to sign in WITH.
func seedLockedOutUser(ctx context.Context, svc *usermgmt.Service) {
	uid := identitymodel.SyntheticUserID("seed-" + lockedOutEmail)
	if _, err := svc.Register(ctx, usermgmt.RegisterRequest{
		ID: uid, Email: lockedOutEmail, DisplayName: "Mallory (no sign-in methods)",
	}); err != nil {
		log.Printf("seed register %s: %v", lockedOutEmail, err)
	}
}
