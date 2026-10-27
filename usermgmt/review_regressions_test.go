package usermgmt

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// TestUserReadModel_EmailChangedResetsTOTP pins the read-model/fold agreement
// on EventEmailChanged: both must reset TOTP. Before this was mirrored, the
// read model kept TOTPEnabled=true while the aggregate fold had already
// disabled it, locking users into an undisableable second factor.
func TestUserReadModel_EmailChangedResetsTOTP(t *testing.T) {
	rm := NewUserReadModel()
	ctx := context.Background()

	events := []event.Event{
		makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Email:         "totp-reset@example.com",
			DisplayName:   "Totp",
			Roles:         []Role{RoleUser},
		}),
		makeEvent(t, eventTOTPEnabled, 2, TOTPEnabledPayload{
			SchemaVersion: currentSchemaVersion,
			Secret:        []byte("secret-bytes"),
		}),
	}
	for _, evt := range events {
		if err := rm.Handle(ctx, evt); err != nil {
			t.Fatalf("Handle %s: %v", evt.Type(), err)
		}
	}

	user, ok := rm.FindByID(testAggID)
	if !ok {
		t.Fatal("user not found after register")
	}
	if !user.TOTPEnabled || len(user.TOTPSecret) == 0 {
		t.Fatal("precondition: TOTP must be enabled before email change")
	}

	evt := makeEvent(t, eventEmailChanged, 3, EmailChangedPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "new-email@example.com",
	})
	if err := rm.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle EmailChanged: %v", err)
	}

	user, ok = rm.FindByID(testAggID)
	if !ok {
		t.Fatal("user not found after email change")
	}
	if user.TOTPEnabled {
		t.Error("TOTPEnabled = true after EmailChanged, want false (mirror of FoldUser)")
	}
	if user.TOTPSecret != nil {
		t.Error("TOTPSecret not cleared after EmailChanged (mirror of FoldUser)")
	}
	if user.Email != "new-email@example.com" {
		t.Errorf("email = %q, want new-email@example.com", user.Email)
	}
	if user.EmailVerified {
		t.Error("EmailVerified = true after EmailChanged, want false")
	}
}

// unverifiedEmailOAuth2Stub reports a fixed email with email_verified=false,
// simulating a provider with user-controllable unverified emails.
type unverifiedEmailOAuth2Stub struct{}

func (unverifiedEmailOAuth2Stub) BeginLogin(_ context.Context, _, state string) (string, string, error) {
	return "https://provider.example.com/auth?state=" + state, "pkce", nil
}

func (unverifiedEmailOAuth2Stub) FinishLogin(_ context.Context, _, _, _ string) ([]byte, error) {
	return []byte(
		`{"subject":"attacker-sub","email":"victim@example.com","email_verified":false,"display_name":"Attacker"}`,
	), nil
}

// TestFinishOAuthLogin_UnverifiedEmailDoesNotLink pins the pre-account-takeover
// guard: an OAuth2 identity whose email is NOT verified by the provider must
// never be linked to the existing owner of that email address.
func TestFinishOAuthLogin_UnverifiedEmailDoesNotLink(t *testing.T) {
	svc, err := NewService(ServiceConfig{OAuth2: unverifiedEmailOAuth2Stub{}})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	t.Cleanup(svc.Stop)

	registerTestUser(t, svc, "victim", "victim@example.com")

	_, err = runTestOAuth2Login(t.Context(), svc, "github")
	if err != nil {
		t.Fatalf("FinishOAuthLogin: %v", err)
	}

	// The attacker lands on a NEWLY provisioned account, not the victim's.
	if got := svc.readModel.Count(); got != 2 {
		t.Errorf("read model count = %d, want 2 (new account, no linking)", got)
	}
	linked, ok := svc.readModel.FindByExternalAccount("github", "attacker-sub")
	if !ok {
		t.Fatal("attacker account not found — expected a newly provisioned account")
	}
	victim, ok := svc.readModel.FindByEmail("victim@example.com")
	if !ok {
		t.Fatal("victim not found")
	}
	if victim.ID == linked.ID {
		t.Fatal("unverified-email identity was linked to the existing victim account")
	}
}

// TestAuditLog_RecentNonPositive pins Recent against the negative-n
// slice-bounds panic.
func TestAuditLog_RecentNonPositive(t *testing.T) {
	a := NewAuditLog()
	evt := makeEvent(t, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "audit@example.com",
		DisplayName:   "Audit",
		Roles:         []Role{RoleUser},
	})
	if err := a.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	for _, n := range []int{-1, 0} {
		got := a.Recent(n)
		if len(got) != 0 {
			t.Errorf("Recent(%d) = %d entries, want 0", n, len(got))
		}
	}
}

// TestSQLSessionStore_SweeperStopIsIdempotent pins the once-guard on the
// sweeper's stop func (second call used to panic with close-of-closed-channel).
func TestSQLSessionStore_SweeperStopIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLSessionStore(context.Background(), db, "sqlite")
	if err != nil {
		t.Fatalf("NewSQLSessionStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	stop := store.StartCleanupSweeper(context.Background(), time.Hour)
	stop()
	stop() // must not panic
}
