package usermgmt

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestService_Register(t *testing.T) {
	svc, err := NewService(newTestServiceConfig())
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("user-1"), Email: "test@example.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User.ID != NewUserID("user-1") {
		t.Errorf("expected user ID user-1, got %s", resp.User.ID)
	}
	if resp.Session == nil {
		t.Error("expected non-nil session")
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	registerTestUser(t, svc, "u1", "a@b.com")

	_, err := svc.Register(context.Background(), RegisterRequest{
		ID: NewUserID("u2"), Email: "a@b.com",
	})
	assertErrorIs(t, err, ErrEmailExists, "ErrEmailExists")
}

func TestService_Register_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Empty ID is auto-generated — registration should succeed.
	resp, err := svc.Register(ctx, RegisterRequest{
		ID: NewUserID(""), Email: "autoid@test.com",
	})
	if err != nil {
		t.Fatalf("Register with empty ID should auto-generate: %v", err)
	}
	if resp.User.ID.IsZero() {
		t.Error("expected auto-generated user ID, got zero")
	}

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u1"), Email: "invalid",
	})
	assertErrorIs(t, err, ErrValidation, "bad email")
}

func TestService_Register_TrimmedEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    NewUserID("u1"),
		Email: "  spaced@test.com  ",
	})
	if err != nil {
		t.Fatalf("Register with trimmed email: %v", err)
	}
	if resp.User.Email != "spaced@test.com" {
		t.Errorf("expected trimmed email 'spaced@test.com', got %q", resp.User.Email)
	}
}

func TestService_Register_TrimmedDisplayName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:          NewUserID("u1"),
		Email:       "trimdisplay@test.com",
		DisplayName: "  Spaced Name  ",
	})
	if err != nil {
		t.Fatalf("Register with trimmed display name: %v", err)
	}
	if resp.User.DisplayName != "Spaced Name" {
		t.Errorf("expected trimmed display name 'Spaced Name', got %q", resp.User.DisplayName)
	}
}

func TestService_Register_NoDisplayName(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Register(ctx, RegisterRequest{
		ID:    NewUserID("u1"),
		Email: "nodisplay@test.com",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.User.DisplayName != "" {
		t.Errorf("expected empty display name, got %q", resp.User.DisplayName)
	}
}

func TestService_Register_DuplicateEmail_CaseInsensitive(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u1"), Email: "Case@Test.COM",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u2"), Email: "case@test.com",
	})
	assertErrorIs(t, err, ErrEmailExists, "ErrEmailExists for case-insensitive duplicate")
}

func TestService_Register_MaxUsersReached(t *testing.T) {
	svc, err := NewService(ServiceConfig{MaxUsers: 1})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "first@example.com")

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u2"), Email: "second@example.com",
	})
	assertErrorIs(t, err, ErrRegistrationClosed, "registration closed after max users reached")
}

func TestService_Register_MaxUsersZero_Unlimited(t *testing.T) {
	svc, err := NewService(ServiceConfig{MaxUsers: 0})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	for i := range 5 {
		registerTestUser(t, svc, "u"+string(rune('1'+i)), "user"+string(rune('1'+i))+"@example.com")
	}
}

func TestService_Register_MaxUsersTwo_AllowsThird(t *testing.T) {
	svc, err := NewService(ServiceConfig{MaxUsers: 2})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	ctx := context.Background()

	registerTestUser(t, svc, "u1", "first@example.com")
	registerTestUser(t, svc, "u2", "second@example.com")

	_, err = svc.Register(ctx, RegisterRequest{
		ID: NewUserID("u3"), Email: "third@example.com",
	})
	assertErrorIs(t, err, ErrRegistrationClosed, "registration closed after 2 users with MaxUsers=2")
}

// TestService_Register_ConcurrentSameEmail pins the single-instance uniqueness
// guarantee. With the default MaxUsers=0 nothing serialized the
// check-then-register section, so N concurrent registrations of one address
// used to produce several users on it. The per-address stripe lock now spans
// the uniqueness check and the dispatch: exactly one succeeds and exactly one
// row holds the address.
func TestService_Register_ConcurrentSameEmail(t *testing.T) {
	t.Parallel()

	const (
		workers = 24
		email   = "same-address@example.com"
	)
	svc := newTestService(t) // MaxUsers=0: the configuration that used to race
	t.Cleanup(svc.Stop)
	ctx := context.Background()

	errs := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // barrier: every goroutine races from the same gun
			_, errs[i] = svc.Register(ctx, RegisterRequest{Email: email})
		}()
	}
	close(start)
	wg.Wait()

	var succeeded, refused int
	for i, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrEmailExists):
			refused++
		default:
			t.Errorf("worker %d: unexpected error: %v", i, err)
		}
	}
	if succeeded != 1 {
		t.Errorf("succeeded = %d, want exactly 1", succeeded)
	}
	if refused != workers-1 {
		t.Errorf("refused with ErrEmailExists = %d, want %d", refused, workers-1)
	}
	if holders := countUsersWithEmail(svc, email); holders != 1 {
		t.Errorf("rows holding %s = %d, want exactly 1", email, holders)
	}
	if got := svc.readModel.Count(); got != 1 {
		t.Errorf("read model count = %d, want 1", got)
	}
}
