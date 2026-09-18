package usermgmt

import (
	"errors"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

// TestCommandValidation_WiredViaCommandMiddleware proves the opt-in
// syntactic-validation seam: middleware.CommandValidation(ValidateCommand)
// rejects malformed commands before the domain handler runs (uniform
// Rejection-family 400 with the request-layer message + sentinel), while
// well-formed commands pass through and execute.
func TestCommandValidation_WiredViaCommandMiddleware(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{
		CommandMiddleware: []command.Middleware{
			middleware.CommandValidation(ValidateCommand),
		},
	})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}

	// Malformed email: rejected before the handler, with the request-layer
	// message and the ErrValidation sentinel.
	err = svc.dispatcher.Dispatch(t.Context(), NewRegisterUserCmd(aggID, "not-an-email", "", nil))
	if err == nil {
		t.Fatal("dispatch with invalid email unexpectedly succeeded")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("validation error = %v, want ErrValidation sentinel", err)
	}
	if !strings.Contains(err.Error(), "invalid email") {
		t.Errorf("validation error message = %q, want the request-layer %q wording", err.Error(), "invalid email")
	}
	if _, ok := svc.readModel.FindByEmail("not-an-email"); ok {
		t.Error("invalid command reached the domain (read model mutated)")
	}

	// Oversized display name: same treatment.
	long := strings.Repeat("x", maxDisplayNameLength+1)
	if err := svc.dispatcher.Dispatch(t.Context(), NewRegisterUserCmd(aggID, "valid@example.com", long, nil)); !errors.Is(err, ErrValidation) {
		t.Errorf("oversized display name error = %v, want ErrValidation", err)
	}

	// Well-formed command passes through and registers.
	if err := svc.dispatcher.Dispatch(t.Context(), NewRegisterUserCmd(aggID, "valid@example.com", "Valid Name", nil)); err != nil {
		t.Fatalf("valid dispatch: %v", err)
	}
	if _, ok := svc.readModel.FindByEmail("valid@example.com"); !ok {
		t.Error("valid registration not visible in read model")
	}
}

// TestValidateCommand_CommandsWithoutSyntacticRulesPassThrough pins that
// commands carrying only opaque/binary payloads (delete reason, credentials)
// are not rejected: their invariants are semantic and live in the domain
// decide functions, which remain authoritative.
func TestValidateCommand_CommandsWithoutSyntacticRulesPassThrough(t *testing.T) {
	t.Parallel()

	if err := ValidateCommand(NewDeleteUserCmd(aggIDFromUserID(t, GenerateUserID()), "any reason")); err != nil {
		t.Errorf("DeleteUserCmd rejected: %v", err)
	}
	if err := ValidateCommand(NewVerifyEmailCmd(aggIDFromUserID(t, GenerateUserID()))); err != nil {
		t.Errorf("VerifyEmailCmd rejected: %v", err)
	}
}
