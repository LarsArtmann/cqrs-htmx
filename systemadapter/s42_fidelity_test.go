package systemadapter_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	usermgmt "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// S42 fidelity proofs: the declarative Enforce ladder must answer exactly
// what the legacy Casbin path answers (parity, not invention), and the audit
// Action vocabulary must be the usermgmt.AuditAction* vocabulary.

// legacyVerdict answers Enforce through the LEGACY path: a default-model
// Casbin enforcer with the role granted in the domain, exactly how the
// usermgmt CasbinProjection seeds grants.
func legacyVerdict(t *testing.T, subject string, role identitymodel.Role, domain, action string) bool {
	t.Helper()

	a, err := identitymodel.NewAuthz()
	if err != nil {
		t.Fatalf("legacy NewAuthz: %v", err)
	}

	if err := a.AddGroupPolicy(identitymodel.GroupPolicy{
		Subject: subject,
		Role:    role,
		Domain:  domain,
	}); err != nil {
		t.Fatalf("legacy AddGroupPolicy(%s): %v", role, err)
	}

	allowed, err := a.Enforce(subject, domain, "resource", identitymodel.Action(action))
	if err != nil {
		t.Fatalf("legacy Enforce(%s): %v", role, err)
	}

	return allowed
}

// TestDeclarative_EnforceParityWithLegacyCasbin is the S42a diff table as a
// live parity test: for every role and a spread of actions, the declarative
// verdict must EQUAL the legacy Casbin verdict. The old ladder invented
// viewer→read / user→write / owner≡admin grants legacy never made — any
// reintroduction of that widening fails here.
func TestDeclarative_EnforceParityWithLegacyCasbin(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)

	tenantID := identitymodel.NewTenantID("tenant-parity")
	roles := []identitymodel.Role{
		identitymodel.RoleSuperAdmin,
		identitymodel.RoleAdmin,
		identitymodel.RoleUser,
		identitymodel.RoleViewer,
		identitymodel.RoleOwner,
	}
	actions := []string{"view", "write", "manage"}

	for _, role := range roles {
		streamID := id.NewStreamID()
		actorID := identitymodel.NewActorID(identitymodel.ActorUser, streamID.Get())

		must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
			streamID, string(role)+"@parity.example.com", "Parity "+string(role),
			[]identitymodel.Role{identitymodel.RoleUser},
		)))
		must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewAddMemberCmd(
			actorID, tenantID, []identitymodel.Role{role},
		)))

		for _, action := range actions {
			want := legacyVerdict(t, streamID.Get(), role, tenantID.Get(), action)

			eventually(t, 5*time.Second, func() error {
				allowed, err := systemadapter.Enforce(ctx, sys, streamID.Get(), tenantID.Get(), action)
				if err != nil {
					return err
				}

				if allowed != want {
					return errors.New("declarative Enforce diverged from legacy Casbin: " +
						"role=" + string(role) + " action=" + action +
						" declarative=" + verdictString(allowed) + " legacy=" + verdictString(want))
				}

				return nil
			})
		}
	}
}

func verdictString(allowed bool) string {
	if allowed {
		return "allow"
	}

	return "deny"
}

// TestDeclarative_AuditVocabularyParity pins S42b: audit entries carry the
// usermgmt.AuditAction* vocabulary, never the invented change_*/enable_* verbs.
func TestDeclarative_AuditVocabularyParity(t *testing.T) {
	ctx, sys := startDeclarativeSystem(t)

	userStreamID := id.NewStreamID()
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewRegisterUserCmd(
		userStreamID, "vocab@example.com", "Vocab User",
		[]identitymodel.Role{identitymodel.RoleUser},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeEmailCmd(
		userStreamID, "vocab2@example.com",
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewChangeDisplayNameCmd(
		userStreamID, "Vocab User 2",
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewVerifyEmailCmd(userStreamID)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewEnableTOTPCmd(
		userStreamID, []byte{0x01, 0x02},
	)))
	must(t, sys.CommandDispatcher().Dispatch(ctx, identitymodel.NewDisableTOTPCmd(userStreamID)))

	required := []string{
		usermgmt.AuditActionRegister,
		usermgmt.AuditActionEmailChanged,
		usermgmt.AuditActionDisplayNameChanged,
		usermgmt.AuditActionEmailVerified,
		usermgmt.AuditActionTOTPEnabled,
		usermgmt.AuditActionTOTPDisabled,
	}
	banned := []string{
		"change_email", "change_display_name", "verify_email",
		"enable_totp", "disable_totp", "update_roles",
		"add_credential", "remove_credential", "delete_user",
	}

	eventually(t, 5*time.Second, func() error {
		entries, err := systemadapter.AuditEntriesFor(ctx, sys, userStreamID.Get())
		if err != nil {
			return err
		}

		seen := make(map[string]bool, len(entries))
		for _, e := range entries {
			seen[e.Action] = true

			if slices.Contains(banned, e.Action) {
				return errors.New("audit entry carries an invented action verb: " + e.Action)
			}
		}

		for _, want := range required {
			if !seen[want] {
				return errors.New("audit vocabulary is missing the legacy action " + want)
			}
		}

		return nil
	})
}
