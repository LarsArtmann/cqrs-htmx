package setup_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/larsartmann/cqrs-htmx/setup/v4"
	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
)

// Contract tests for the consumer-supplied store hatch: a SessionStore and
// LockoutStore passed via Config.ServiceConfig must be the objects the
// service actually talks to — registration persists sessions through the
// supplied store, the session middleware resolves them from it, and login
// consults the supplied lockout. A DI hatch that silently ignores custom
// stores would be a security-relevant lie (lockout bypass).

type countingSessionStore struct {
	inner       *usermgmt.InMemorySessionStore
	creates     atomic.Int64
	finds       atomic.Int64
	deletes     atomic.Int64
	deleteByUID atomic.Int64
}

func newCountingSessionStore() *countingSessionStore {
	return &countingSessionStore{inner: usermgmt.NewInMemorySessionStore()}
}

func (s *countingSessionStore) Create(ctx context.Context, session *usermgmt.Session) error {
	s.creates.Add(1)
	return s.inner.Create(ctx, session)
}

func (s *countingSessionStore) Find(ctx context.Context, token string) (*usermgmt.Session, error) {
	s.finds.Add(1)
	return s.inner.Find(ctx, token)
}

func (s *countingSessionStore) Delete(ctx context.Context, token string) error {
	s.deletes.Add(1)
	return s.inner.Delete(ctx, token)
}

func (s *countingSessionStore) DeleteByUserID(ctx context.Context, userID usermgmt.UserID) error {
	s.deleteByUID.Add(1)
	return s.inner.DeleteByUserID(ctx, userID)
}

type countingLockout struct {
	inner         *usermgmt.AccountLockout
	isLocked      atomic.Int64
	recordFailure atomic.Int64
	resets        atomic.Int64
}

func newCountingLockout() *countingLockout {
	return &countingLockout{inner: usermgmt.NewAccountLockout()}
}

func (l *countingLockout) IsLocked(email string) bool {
	l.isLocked.Add(1)
	return l.inner.IsLocked(email)
}

func (l *countingLockout) RecordFailure(email string) bool {
	l.recordFailure.Add(1)
	return l.inner.RecordFailure(email)
}

func (l *countingLockout) Reset(email string) {
	l.resets.Add(1)
	l.inner.Reset(email)
}

// stubWebAuthn lets BeginLogin reach the lockout check without a real
// passkey ceremony; the service only needs BeginLogin to return options.
type stubWebAuthn struct{}

func (stubWebAuthn) BeginRegistration(_ context.Context, _ []byte) ([]byte, []byte, error) {
	return []byte(`{}`), []byte(`{}`), nil
}

func (stubWebAuthn) FinishRegistration(_ context.Context, _, _, _ []byte) ([]byte, error) {
	return []byte(`{}`), nil
}

func (stubWebAuthn) BeginLogin(_ context.Context, _ []byte) ([]byte, []byte, error) {
	return []byte(`{"publicKey":{}}`), []byte(`{}`), nil
}

func (stubWebAuthn) FinishLogin(_ context.Context, _, _, _ []byte) error {
	return nil
}

func TestServiceConfig_CustomStores_AreTheOnesUsed(t *testing.T) {
	sessions := newCountingSessionStore()
	lockout := newCountingLockout()

	bundle, err := setup.New(setup.Config{
		ServiceConfig: &usermgmt.ServiceConfig{
			SessionStore: sessions,
			Lockout:      lockout,
			WebAuthn:     stubWebAuthn{},
		},
	})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	ctx := context.Background()

	// 1. Registration persists the issued session through the SUPPLIED store.
	reg, err := bundle.Service.Register(ctx, usermgmt.RegisterRequest{
		Email: "stores@example.com", DisplayName: "Store Contract",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if got := sessions.creates.Load(); got == 0 {
		t.Fatal("registration did not create a session through the supplied SessionStore")
	}
	if _, err := sessions.Find(ctx, reg.Session.Token); err != nil {
		t.Fatalf("supplied store cannot resolve the registration session token: %v", err)
	}

	// 2. The session middleware resolves the session from the SUPPLIED store:
	// an authenticated request against the gated dashboard must succeed and
	// bump the find counter.
	findsBefore := sessions.finds.Load()
	mux := http.NewServeMux()
	bundle.Mount(mux)
	srv := httptest.NewServer(bundle.Middleware()(mux))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/dashboard/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Cookie", "session="+reg.Session.Token)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /dashboard/: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /dashboard/ with supplied-store session: status %d, want 200", resp.StatusCode)
	}
	if sessions.finds.Load() <= findsBefore {
		t.Fatal("session middleware did not consult the supplied SessionStore")
	}

	// 3. Login consults the SUPPLIED lockout before anything else. The user
	// has no passkey yet, so BeginLogin rejects with no_credentials — the
	// contract under test is that the supplied lockout was consulted first.
	_, _ = bundle.Service.BeginLogin(ctx, "stores@example.com")
	if lockout.isLocked.Load() == 0 {
		t.Fatal("BeginLogin did not consult the supplied LockoutStore")
	}
}

func TestServiceConfig_CustomSessionStore_LogoutDeletesThroughIt(t *testing.T) {
	sessions := newCountingSessionStore()

	bundle, err := setup.New(setup.Config{
		ServiceConfig: &usermgmt.ServiceConfig{
			SessionStore: sessions,
		},
	})
	if err != nil {
		t.Fatalf("setup.New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	ctx := context.Background()
	reg, err := bundle.Service.Register(ctx, usermgmt.RegisterRequest{
		Email: "logout@example.com", DisplayName: "Logout Contract",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Logout must delete the session through the supplied store, after which
	// the token stops resolving.
	if err := bundle.Service.Logout(ctx, reg.Session.Token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if sessions.deletes.Load() == 0 && sessions.deleteByUID.Load() == 0 {
		t.Fatal("Logout did not delete through the supplied SessionStore")
	}
	if _, findErr := sessions.inner.Find(ctx, reg.Session.Token); findErr == nil {
		t.Fatal("session still resolves after Logout — custom store bypassed")
	}
}
