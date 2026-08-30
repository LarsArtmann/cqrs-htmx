package usermgmt

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// Contract tests for the consumer-supplied store hatches on ServiceConfig:
// a WebAuthnSessionStore, VerificationTokenStore, or PendingTOTPStore passed
// by the consumer must be the object the service actually talks to. A hatch
// that silently ignores a custom store would strand multi-instance
// deployments (sessions/tokens/totp-secrets invisible to other replicas) —
// a security- and correctness-relevant lie.

// Sentinel miss errors: the counting stubs must surface failures WITHOUT
// depending on internal error constructors (stdlib error constructors are
// banned in non-test code; in tests they are fine).
type webAuthnSessionMissError struct{}

func (webAuthnSessionMissError) Error() string { return "webauthn session miss (test stub)" }

type verificationTokenMissError struct{}

func (verificationTokenMissError) Error() string { return "verification token miss (test stub)" }

type countingWebAuthnSessions struct {
	mu      sync.Mutex
	data    map[string][]byte
	saves   int
	gets    int
	deletes int
}

func newCountingWebAuthnSessions() *countingWebAuthnSessions {
	return &countingWebAuthnSessions{data: map[string][]byte{}}
}

func (s *countingWebAuthnSessions) Save(key string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saves++
	s.data[key] = bytes.Clone(data)
}

func (s *countingWebAuthnSessions) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gets++
	if d, ok := s.data[key]; ok {
		return bytes.Clone(d), nil
	}
	return nil, webAuthnSessionMissError{}
}

func (s *countingWebAuthnSessions) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes++
	delete(s.data, key)
}

type countingVerificationTokens struct {
	mu     sync.Mutex
	tokens map[string]UserID
	saves  int
}

func newCountingVerificationTokens() *countingVerificationTokens {
	return &countingVerificationTokens{tokens: map[string]UserID{}}
}

func (s *countingVerificationTokens) Save(userID UserID, _ string, _ time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saves++
	token := "tok-" + userID.Get().String()
	s.tokens[token] = userID
	return token, nil
}

func (s *countingVerificationTokens) Consume(token string) (UserID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	uid, ok := s.tokens[token]
	if !ok {
		return UserID{}, verificationTokenMissError{}
	}
	delete(s.tokens, token)
	return uid, nil
}

type countingPendingTOTP struct {
	mu       sync.Mutex
	pending  map[string][]byte
	saves    int
	consumes int
}

func newCountingPendingTOTP() *countingPendingTOTP {
	return &countingPendingTOTP{pending: map[string][]byte{}}
}

func (s *countingPendingTOTP) Save(userID string, secret []byte, _ time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saves++
	s.pending[userID] = bytes.Clone(secret)
}

func (s *countingPendingTOTP) Consume(userID string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consumes++
	secret, ok := s.pending[userID]
	delete(s.pending, userID)
	return secret, ok
}

func TestContract_WebAuthnSessionStore_HatchIsUsed(t *testing.T) {
	store := newCountingWebAuthnSessions()
	svc, err := NewService(ServiceConfig{
		WebAuthn:             testWebAuthnProvider{},
		WebAuthnSessionStore: store,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Stop()

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID:    NewUserID(strings.Repeat("c", 26)),
		Email: "hatch-wa@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := svc.BeginRegistration(context.Background(), reg.User.ID); err != nil {
		t.Fatalf("BeginRegistration: %v", err)
	}

	if store.saves != 1 {
		t.Fatalf("BeginRegistration must route the ceremony session through the SUPPLIED store, saves=%d", store.saves)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.data) == 0 {
		t.Fatal("supplied store holds no ceremony session — the service used a different store")
	}
}

func TestContract_VerificationTokenStore_HatchIsUsed(t *testing.T) {
	store := newCountingVerificationTokens()
	svc, err := NewService(ServiceConfig{
		EmailVerification:      &EmailVerificationConfig{},
		VerificationTokenStore: store,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Stop()

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID:    NewUserID(strings.Repeat("d", 26)),
		Email: "hatch-vt@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	token, err := svc.SendVerificationEmail(context.Background(), reg.User.ID)
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	if store.saves != 1 {
		t.Fatalf("SendVerificationEmail must issue the token through the SUPPLIED store, saves=%d", store.saves)
	}

	consumed, err := store.Consume(token)
	if err != nil {
		t.Fatalf("token not found in the SUPPLIED store (service used a different store): %v", err)
	}
	if consumed.Get() != reg.User.ID.Get() {
		t.Fatalf("consumed userID %q, want %q", consumed.Get(), reg.User.ID.Get())
	}
}

func TestContract_PendingTOTPStore_HatchIsUsed(t *testing.T) {
	store := newCountingPendingTOTP()
	svc, err := NewService(ServiceConfig{
		TOTP:             newTestTOTPProvider("Hatch"),
		PendingTOTPStore: store,
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	defer svc.Stop()

	reg, err := svc.Register(context.Background(), RegisterRequest{
		ID:    NewUserID(strings.Repeat("e", 26)),
		Email: "hatch-totp@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := svc.EnableTOTP(context.Background(), reg.User.ID); err != nil {
		t.Fatalf("EnableTOTP: %v", err)
	}
	if store.saves != 1 {
		t.Fatalf("EnableTOTP must park the secret through the SUPPLIED store, saves=%d", store.saves)
	}
	store.mu.Lock()
	secret, ok := store.pending[reg.User.ID.Get().String()]
	store.mu.Unlock()
	if !ok || len(secret) == 0 {
		t.Fatal("supplied store holds no pending secret — the service used a different store")
	}

	// VerifyTOTPSetup consumes the pending secret unconditionally (before
	// code validation) — an invalid code must still route through the
	// supplied store exactly once.
	_ = svc.VerifyTOTPSetup(context.Background(), reg.User.ID, "000000")
	if store.consumes != 1 {
		t.Fatalf("VerifyTOTPSetup must Consume through the SUPPLIED store, consumes=%d", store.consumes)
	}
	store.mu.Lock()
	_, still := store.pending[reg.User.ID.Get().String()]
	store.mu.Unlock()
	if still {
		t.Fatal("pending secret was not consumed from the SUPPLIED store")
	}
}
