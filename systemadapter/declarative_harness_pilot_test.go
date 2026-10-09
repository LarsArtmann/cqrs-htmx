package systemadapter

// Harness pilot (2026-10-09): the system-level BDD harness
// (go-cqrs-lite/systemscenario, ADR-0153) replacing the hand-rolled
// dispatch→eventually blocks of the user-lifecycle sub-train with
// Given/When/Then chains. Same DomainConfig + RecommendedMemoryDeployment
// as the legacy tests — the fixture-from-production-config property.

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
)

// newHarnessScenario boots the declarative system under the harness.
func newHarnessScenario(t *testing.T) (*systemscenario.Scenario, context.Context) {
	t.Helper()

	ctx := context.Background()
	sc := systemscenario.System(t, ctx, DomainConfig(), RecommendedMemoryDeployment())

	return sc, ctx
}

// findUser closes over the booted system for ThenQuery lookups.
func findUser(sc *systemscenario.Scenario, ctx context.Context, userID string) func() (any, error) {
	return func() (any, error) {
		return FindUserByID(ctx, sc.System(), userID)
	}
}

func TestHarnessPilot_UserLifecycle(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "user@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeEmailCmd(userStreamID, "new@example.com")).
		ThenQueryFunc(findUser(sc, ctx, userStreamID.String()), func(got any) error {
			user, ok := got.(UserView)
			if !ok {
				t.Fatalf("FindUserByID returned %T, want UserView", got)
			}

			if user.Email != "new@example.com" {
				return &fieldMismatch{field: "Email", want: "new@example.com", got: user.Email}
			}

			if user.EmailVerified {
				return &fieldMismatch{field: "EmailVerified", want: "false", got: "true"}
			}

			if user.CreatedAt.IsZero() {
				return &fieldMismatch{field: "CreatedAt", want: "set", got: "zero"}
			}

			return nil
		}).
		ThenQueryFunc(func() (any, error) {
			return FindUserByEmail(ctx, sc.System(), "new@example.com")
		}, func(got any) error {
			user, ok := got.(UserView)
			if !ok {
				t.Fatalf("FindUserByEmail returned %T, want UserView", got)
			}

			if user.ID != userStreamID.String() {
				return &fieldMismatch{field: "ID", want: userStreamID.String(), got: user.ID}
			}

			return nil
		}).
		Command(identitymodel.NewVerifyEmailCmd(userStreamID)).
		ThenQueryFunc(findUser(sc, ctx, userStreamID.String()), func(got any) error {
			user, ok := got.(UserView)
			if !ok {
				t.Fatalf("FindUserByID returned %T, want UserView", got)
			}

			if !user.EmailVerified {
				return &fieldMismatch{field: "EmailVerified", want: "true", got: "false"}
			}

			return nil
		})
}

func TestHarnessPilot_UserDisplayNameChange(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "display@example.com", "Original",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeDisplayNameCmd(userStreamID, "Updated Name")).
		ThenQueryFunc(findUser(sc, ctx, userStreamID.String()), func(got any) error {
			user, ok := got.(UserView)
			if !ok {
				t.Fatalf("FindUserByID returned %T, want UserView", got)
			}

			if user.DisplayName != "Updated Name" {
				return &fieldMismatch{field: "DisplayName", want: "Updated Name", got: user.DisplayName}
			}

			return nil
		})
}

// TestHarnessPilot_MissingLookups migrates the negative lookups: the
// harness's ThenQueryFails asserts the classified ErrNotFound immediately
// (ThenQuery would poll for 5s on a legitimately-missing row).
func TestHarnessPilot_MissingLookups(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	missing := id.NewStreamID().String()

	seed := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		seed, "seed@example.com", "Seed",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeDisplayNameCmd(seed, "Still Seed")).
		ThenQueryFails(func() (any, error) {
			return FindUserByID(ctx, sc.System(), missing)
		}, system.ErrNotFound).
		ThenQueryFails(func() (any, error) {
			return FindTenantByID(ctx, sc.System(), missing)
		}, system.ErrNotFound).
		ThenQueryFails(func() (any, error) {
			return FindBotByID(ctx, sc.System(), missing)
		}, system.ErrNotFound)
}

// fieldMismatch gives ThenQueryFunc checks precise, field-named errors
// instead of the legacy string-only "Email mismatch" messages.
type fieldMismatch struct {
	field string
	want  string
	got   string
}

func (e *fieldMismatch) Error() string {
	return e.field + ": want " + e.want + ", got " + e.got
}
