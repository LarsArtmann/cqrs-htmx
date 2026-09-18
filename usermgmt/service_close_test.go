package usermgmt

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-error-family/errorfamily"
)

// TestClose_PostCloseDispatchReturnsClosedError proves Service.Close closes
// the command dispatcher: any dispatch after Close fails fast with the
// upstream ErrDispatcherClosed sentinel (Infrastructure family) instead of
// writing events to an infrastructure that is being torn down.
func TestClose_PostCloseDispatchReturnsClosedError(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{})

	reg := registerTestUser(t, svc, GenerateUserID().Get().String(), "closed@example.com")
	userID := reg.User.ID

	if err := svc.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	err := svc.ChangeDisplayName(t.Context(), userID, "Post Close")
	if err == nil {
		t.Fatal("dispatch after Close unexpectedly succeeded")
	}
	if !errors.Is(err, command.ErrDispatcherClosed) {
		t.Errorf("post-close dispatch error = %v, want one wrapping command.ErrDispatcherClosed", err)
	}

	var famErr *errorfamily.Error
	if !errors.As(err, &famErr) {
		t.Fatalf("post-close dispatch error is not an errorfamily.Error: %v", err)
	}
	if famErr.ErrorFamily() != errorfamily.FamilyInfrastructure {
		t.Errorf(
			"post-close dispatch family = %s, want %s",
			famErr.ErrorFamily(),
			errorfamily.FamilyInfrastructure,
		)
	}
}

// TestClose_IdempotentSafeToCallMultipleTimes pins the documented contract
// that Close is safe to call more than once: the upstream dispatcher
// lifecycle close is a mutex-guarded flag set, not a resource teardown that
// fails on repetition.
func TestClose_IdempotentSafeToCallMultipleTimes(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{})
	registerTestUser(t, svc, GenerateUserID().Get().String(), "idempotent@example.com")

	if err := svc.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
