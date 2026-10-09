package systemadapter

// Harness pilot (2026-10-09): the system-level BDD harness
// (go-cqrs-lite/systemscenario, ADR-0153) replacing the hand-rolled
// dispatch→eventually blocks of the user-lifecycle sub-train with
// Given/When/Then chains. Same DomainConfig + RecommendedMemoryDeployment
// as the legacy tests — the fixture-from-production-config property.

import (
	"context"
	"fmt"
	"testing"

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

// findUser closes over the booted system for typed query assertions.
func findUser(sc *systemscenario.Scenario, ctx context.Context, userID string) func() (UserView, error) {
	return func() (UserView, error) {
		return FindUserByID(ctx, sc.System(), userID)
	}
}

func TestHarnessPilot_UserLifecycle(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "user@example.com", "Test User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeEmailCmd(userStreamID, "new@example.com"))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		checkUserFields("new@example.com", false))

	systemscenario.ThenQueryTyped(sc.Phase(), func() (UserView, error) {
		return FindUserByEmail(ctx, sc.System(), "new@example.com")
	}, func(user UserView) error {
		if user.ID != userStreamID.String() {
			return &fieldMismatch{field: "ID", want: userStreamID.String(), got: user.ID}
		}

		return nil
	})

	sc.Phase().Command(identitymodel.NewVerifyEmailCmd(userStreamID))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		checkUserFields("new@example.com", true))
}

func TestHarnessPilot_UserDisplayNameChange(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "display@example.com", "Original",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeDisplayNameCmd(userStreamID, "Updated Name"))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		func(user UserView) error {
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

	seed := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		seed, "seed@example.com", "Seed",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeDisplayNameCmd(seed, "Still Seed")).
		ThenQueryFails(func() (any, error) {
			return FindUserByID(ctx, sc.System(), id.NewStreamID().String())
		}, system.ErrNotFound).
		ThenQueryFails(func() (any, error) {
			return FindTenantByID(ctx, sc.System(), id.NewStreamID().String())
		}, system.ErrNotFound).
		ThenQueryFails(func() (any, error) {
			return FindBotByID(ctx, sc.System(), id.NewStreamID().String())
		}, system.ErrNotFound)
}

// checkUserFields asserts the fields the legacy eventually blocks checked,
// with field-named mismatch errors instead of "Email mismatch" strings.
func checkUserFields(email string, verified bool) func(UserView) error {
	return func(user UserView) error {
		if user.Email != email {
			return &fieldMismatch{field: "Email", want: email, got: user.Email}
		}

		if user.EmailVerified != verified {
			return &fieldMismatch{
				field: "EmailVerified", want: fmt.Sprint(verified), got: fmt.Sprint(user.EmailVerified),
			}
		}

		if user.CreatedAt.IsZero() {
			return &fieldMismatch{field: "CreatedAt", want: "set", got: "zero"}
		}

		return nil
	}
}

// fieldMismatch gives query checks precise, field-named errors.
type fieldMismatch struct {
	field string
	want  string
	got   string
}

func (e *fieldMismatch) Error() string {
	return e.field + ": want " + e.want + ", got " + e.got
}
