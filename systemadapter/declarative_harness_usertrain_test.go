package systemadapter

// User-train harness migration (2026-10-09, adoption wave T10-T12): the
// Credentials/TOTP/ExternalAccounts/Delete/AllUsers/AuditLog/lookup tests
// from declarative_test.go rewritten as systemscenario Given/When/Then
// chains. Same DomainConfig + RecommendedMemoryDeployment as production —
// per the migrate-and-delete ruling, the legacy twins are deleted once this
// train is green (declarative_test.go keeps the tenant/bot/membership/authz/
// sqlite/equivalence trains, tracked for later waves).

import (
	"errors"
	"strconv"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

// awaitNotFound turns "the row must eventually vanish" into a ThenQueryFunc
// probe: ErrNotFound counts as success, a readable row keeps polling.
func awaitNotFound(fn func() error) func() (any, error) {
	return func() (any, error) {
		if err := fn(); err != nil {
			if errors.Is(err, system.ErrNotFound) {
				return nil, nil
			}

			return nil, err
		}

		return errors.New("row is still readable"), nil
	}
}

// checkCredentialCount asserts the user carries exactly n credentials.
func checkCredentialCount(n int) func(UserView) error {
	return func(user UserView) error {
		if len(user.Credentials) != n {
			return &fieldMismatchError{
				field: "len(Credentials)",
				want:  strconv.Itoa(n),
				got:   strconv.Itoa(len(user.Credentials)),
			}
		}

		return nil
	}
}

func TestHarnessUserTrain_Credentials(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()
	credID := []byte{0xAA, 0xBB, 0xCC}

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "cred@example.com", "Cred User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewAddCredentialCmd(
		userStreamID, identitymodel.WebAuthnCredential{
			ID:              credID,
			PublicKey:       []byte{0x01, 0x02, 0x03},
			AttestationType: "none",
			Transports:      []string{"internal"},
			Name:            "My Passkey",
		},
	))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		func(user UserView) error {
			if err := checkCredentialCount(1)(user); err != nil {
				return err
			}

			cred := user.Credentials[0]
			if cred.AttestationType != "none" {
				return &fieldMismatchError{
					field: "Credentials[0].AttestationType",
					want:  "none",
					got:   cred.AttestationType,
				}
			}

			if cred.Name != "My Passkey" {
				return &fieldMismatchError{field: "Credentials[0].Name", want: "My Passkey", got: cred.Name}
			}

			return nil
		})

	sc.Phase().Command(identitymodel.NewRemoveCredentialCmd(userStreamID, credID))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()), checkCredentialCount(0))
}

func TestHarnessUserTrain_TOTP(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "totp@example.com", "TOTP User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewEnableTOTPCmd(userStreamID, []byte("secret-key")))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		func(user UserView) error {
			if !user.TOTPEnabled {
				return &fieldMismatchError{field: "TOTPEnabled", want: "true", got: "false"}
			}

			return nil
		})

	sc.Phase().Command(identitymodel.NewDisableTOTPCmd(userStreamID))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		func(user UserView) error {
			if user.TOTPEnabled {
				return &fieldMismatchError{field: "TOTPEnabled", want: "false", got: "true"}
			}

			return nil
		})
}

func TestHarnessUserTrain_ExternalAccounts(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	// A credential first so unlinking the external account cannot violate
	// the last-auth-method invariant (same setup as the legacy test).
	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "ext@example.com", "Ext User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).Command(identitymodel.NewAddCredentialCmd(
		userStreamID, identitymodel.WebAuthnCredential{
			ID:              []byte{0x01},
			PublicKey:       []byte{0x02},
			AttestationType: "none",
		},
	)).When(identitymodel.NewLinkExternalAccountCmd(
		userStreamID, "github", "gh-123", "ext@github.com", "Ext User",
	))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		func(user UserView) error {
			if len(user.ExternalAccounts) != 1 {
				return &fieldMismatchError{
					field: "len(ExternalAccounts)",
					want:  "1",
					got:   strconv.Itoa(len(user.ExternalAccounts)),
				}
			}

			if user.ExternalAccounts[0].Provider != "github" {
				return &fieldMismatchError{
					field: "ExternalAccounts[0].Provider",
					want:  "github",
					got:   user.ExternalAccounts[0].Provider,
				}
			}

			return nil
		})

	systemscenario.ThenQueryTyped(sc.Phase(), func() (UserView, error) {
		return FindUserByExternalAccount(ctx, sc.System(), "github", "gh-123")
	}, func(byExt UserView) error {
		if byExt.ID != userStreamID.String() {
			return &fieldMismatchError{field: "external-account lookup ID", want: userStreamID.String(), got: byExt.ID}
		}

		return nil
	})

	sc.Phase().Command(identitymodel.NewUnlinkExternalAccountCmd(userStreamID, "github", "gh-123"))

	systemscenario.ThenQueryTyped(sc.Phase(), findUser(sc, ctx, userStreamID.String()),
		func(user UserView) error {
			if len(user.ExternalAccounts) != 0 {
				return &fieldMismatchError{
					field: "len(ExternalAccounts) after unlink",
					want:  "0",
					got:   strconv.Itoa(len(user.ExternalAccounts)),
				}
			}

			return nil
		})

	// Regression (2026-09-09): the link index must drop the (provider,
	// subject) row on unlink — the lookup reports ErrNotFound once unlinked.
	sc.Phase().ThenQueryFunc(awaitNotFound(func() error {
		_, err := FindUserByExternalAccount(ctx, sc.System(), "github", "gh-123")
		return err
	}), func(got any) error { return nil })

	// A freed identity can re-link to a different user; the lookup follows
	// the new owner only.
	secondStreamID := id.NewStreamID()
	sc.Phase().Command(identitymodel.NewRegisterUserCmd(
		secondStreamID, "second@example.com", "Second User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).Command(identitymodel.NewLinkExternalAccountCmd(
		secondStreamID, "github", "gh-123", "ext@github.com", "Ext User",
	))

	systemscenario.ThenQueryTyped(sc.Phase(), func() (UserView, error) {
		return FindUserByExternalAccount(ctx, sc.System(), "github", "gh-123")
	}, func(byExt UserView) error {
		if byExt.ID != secondStreamID.String() {
			return &fieldMismatchError{
				field: "re-link lookup ID (must follow new owner)",
				want:  secondStreamID.String(),
				got:   byExt.ID,
			}
		}

		return nil
	})
}

func TestHarnessUserTrain_UserDelete(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "delete@example.com", "Delete Me",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewDeleteUserCmd(userStreamID, "test"))

	sc.Phase().ThenQueryFunc(awaitNotFound(func() error {
		_, err := FindUserByID(ctx, sc.System(), userStreamID.String())
		return err
	}), func(got any) error { return nil })

	sc.Phase().ThenQueryFunc(func() (any, error) {
		return AllUsers(ctx, sc.System())
	}, func(got any) error {
		all, ok := got.([]UserView)
		if !ok {
			return errors.New("AllUsers: unexpected result type")
		}

		if len(all) != 0 {
			return &fieldMismatchError{field: "len(AllUsers) after delete", want: "0", got: strconv.Itoa(len(all))}
		}

		return nil
	})
}

func TestHarnessUserTrain_AllUsers(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	uids := []id.StreamID{id.NewStreamID(), id.NewStreamID(), id.NewStreamID()}
	emails := []string{"first@example.com", "second@example.com", "third@example.com"}

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		uids[0], emails[0], "User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).Command(identitymodel.NewRegisterUserCmd(
		uids[1], emails[1], "User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewRegisterUserCmd(
		uids[2], emails[2], "User",
		[]identitymodel.Role{identitymodel.RoleUser},
	))

	sc.Phase().ThenQueryFunc(func() (any, error) {
		return AllUsers(ctx, sc.System())
	}, func(got any) error {
		all, ok := got.([]UserView)
		if !ok {
			return errors.New("AllUsers: unexpected result type")
		}

		if len(all) != 3 {
			return &fieldMismatchError{field: "len(AllUsers)", want: "3", got: strconv.Itoa(len(all))}
		}

		return nil
	})
}

func TestHarnessUserTrain_AuditLog(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	userStreamID := id.NewStreamID()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		userStreamID, "audit@example.com", "Audit User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeEmailCmd(userStreamID, "changed@example.com"))

	sc.Phase().ThenQueryFunc(func() (any, error) {
		return AuditEntries(ctx, sc.System())
	}, func(got any) error {
		entries, ok := got.([]AuditEntryView)
		if !ok {
			return errors.New("AuditEntries: unexpected result type")
		}

		if len(entries) < 2 {
			return &fieldMismatchError{field: "len(AuditEntries)", want: ">=2", got: strconv.Itoa(len(entries))}
		}

		return nil
	})

	sc.Phase().ThenQueryFunc(func() (any, error) {
		return AuditEntriesFor(ctx, sc.System(), userStreamID.String())
	}, func(got any) error {
		forUser, ok := got.([]AuditEntryView)
		if !ok {
			return errors.New("AuditEntriesFor: unexpected result type")
		}

		foundRegister, foundChangeEmail := false, false
		for _, entry := range forUser {
			switch entry.EventType {
			case "UserRegistered":
				foundRegister = true
			case "EmailChanged":
				foundChangeEmail = true
			}

			if entry.OccurredAt.IsZero() {
				return &fieldMismatchError{field: "AuditEntry.OccurredAt", want: "set", got: "zero"}
			}
		}

		if !foundRegister {
			return &fieldMismatchError{
				field: "AuditEntriesFor event types",
				want:  "UserRegistered present",
				got:   "missing",
			}
		}

		if !foundChangeEmail {
			return &fieldMismatchError{
				field: "AuditEntriesFor event types",
				want:  "EmailChanged present",
				got:   "missing",
			}
		}

		return nil
	})

	sc.Phase().ThenQueryFunc(func() (any, error) {
		return RecentAuditEntries(ctx, sc.System(), 1)
	}, func(got any) error {
		recent, ok := got.([]AuditEntryView)
		if !ok {
			return errors.New("RecentAuditEntries: unexpected result type")
		}

		if len(recent) != 1 {
			return &fieldMismatchError{field: "len(RecentAuditEntries(1))", want: "1", got: strconv.Itoa(len(recent))}
		}

		return nil
	})
}

// TestHarnessUserTrain_MissingLookups migrates the FULL negative-lookup set:
// every by-key lookup reports system.ErrNotFound; the scan-style membership
// lookup returns empty-with-nil-error, never a zero-value row.
func TestHarnessUserTrain_MissingLookups(t *testing.T) {
	sc, ctx := newHarnessScenario(t)

	seed := id.NewStreamID()
	missing := id.NewStreamID().String()

	sc.Given().Command(identitymodel.NewRegisterUserCmd(
		seed, "seed@example.com", "Seed",
		[]identitymodel.Role{identitymodel.RoleUser},
	)).When(identitymodel.NewChangeDisplayNameCmd(seed, "Still Seed"))

	phase := sc.Phase()
	missingLookup := func(fn func() (any, error)) *systemscenario.WhenPhase {
		return phase.ThenQueryFails(fn, system.ErrNotFound)
	}

	missingLookup(func() (any, error) { return FindUserByID(ctx, sc.System(), missing) })
	missingLookup(func() (any, error) { return FindUserByEmail(ctx, sc.System(), "missing@example.com") })
	missingLookup(func() (any, error) {
		return FindUserByExternalAccount(ctx, sc.System(), "github", "missing")
	})
	missingLookup(func() (any, error) { return FindTenantByID(ctx, sc.System(), missing) })
	missingLookup(func() (any, error) { return FindTenantByName(ctx, sc.System(), "missing-tenant") })
	missingLookup(func() (any, error) { return FindBotByID(ctx, sc.System(), missing) })
	missingLookup(func() (any, error) { return FindBotByTokenHash(ctx, sc.System(), "deadbeef") })
	missingLookup(func() (any, error) { return FindMembershipByID(ctx, sc.System(), missing) })
	missingLookup(func() (any, error) { return FindPolicyByStreamID(ctx, sc.System(), missing) })

	phase.ThenQueryFunc(func() (any, error) {
		return FindMembershipsByActor(ctx, sc.System(), missing)
	}, func(got any) error {
		memberships, ok := got.([]MembershipView)
		if !ok {
			return errors.New("FindMembershipsByActor: unexpected result type")
		}

		if len(memberships) != 0 {
			return &fieldMismatchError{
				field: "len(FindMembershipsByActor(missing))",
				want:  "0",
				got:   strconv.Itoa(len(memberships)),
			}
		}

		return nil
	})
}
