package systemadapter_test

import (
	"errors"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestDeclarative_CredentialUsedLiftsSignCount (Lane-D plan R02, S32 proof):
// a dispatched RecordCredentialUse must lift the SignCount and assertion
// flags into the declarative UserView — the read model consumers query for
// clone detection. Without updateCredentialUsed, the view would keep the
// registration-time counter forever.
func TestDeclarative_CredentialUsedLiftsSignCount(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "cred-use@example.com", "Cred Use",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddCredentialCmd(
		userStreamID, identitymodel.WebAuthnCredential{
			CredentialCore: identitymodel.CredentialCore{
				ID:              []byte{0x01},
				PublicKey:       []byte{0x02},
				AttestationType: "none",
				SignCount:       5,
			},
		},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRecordCredentialUseCmd(
		userStreamID, identitymodel.CredentialUse{
			CredentialID: []byte{0x01},
			SignCount:    9,
			UserPresent:  true,
			UserVerified: true,
			BackupState:  true,
		},
	)))

	eventually(t, 5*time.Second, func() error {
		user, err := systemadapter.FindUserByID(ctx, sys, userStreamID.Get())
		if err != nil {
			return err
		}

		if len(user.Credentials) != 1 {
			return errors.New("expected 1 credential")
		}

		cred := user.Credentials[0]

		if cred.SignCount != 9 {
			return errors.New("SignCount not lifted (still registration-time value)")
		}

		if !cred.UserPresent || !cred.UserVerified || !cred.BackupState {
			return errors.New("assertion flags not lifted")
		}

		return nil
	})
}
