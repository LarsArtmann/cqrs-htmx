package usermgmt

import (
	"context"
	"sort"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
)

// everyDispatchedCommand is the command set this module registers, written out
// rather than derived: a new command must appear here and in CommandPostures in
// the same change, so a forgotten posture is a test failure instead of a
// runtime denial in a consumer's job.
var everyDispatchedCommand = []string{ //nolint:gochecknoglobals // exhaustive posture table, drift-gated by the test
	"AddCredential",
	"AddMember",
	"ChangeDisplayName",
	"ChangeEmail",
	"CreateTenant",
	"DeleteBot",
	"DeleteTenant",
	"DeleteUser",
	"DisableTOTP",
	"EnableTOTP",
	"LinkExternalAccount",
	"ReactivateTenant",
	"RecordCredentialUse",
	"RegisterBot",
	"RegisterUser",
	"RemoveCredential",
	"RemoveMember",
	"SuspendTenant",
	"UnlinkExternalAccount",
	"UpdateMemberRoles",
	"VerifyEmail",
}

// TestCommandPostures_CoverEveryDispatchedCommand is the drift guard: the
// declaration table and the registration set must agree exactly, in both
// directions.
func TestCommandPostures_CoverEveryDispatchedCommand(t *testing.T) {
	t.Parallel()

	declared := make([]string, 0, len(CommandPostures()))

	for _, posture := range CommandPostures() {
		declared = append(declared, posture.Name)
	}

	sort.Strings(declared)

	want := append([]string(nil), everyDispatchedCommand...)
	sort.Strings(want)

	if strings.Join(declared, ",") != strings.Join(want, ",") {
		t.Fatalf("CommandPostures declares %v, want %v", declared, want)
	}
}

// TestCommandPostures_AreUsableAsDeclarations proves every row survives the
// gate's own boot check, so a consumer can declare the table verbatim.
func TestCommandPostures_AreUsableAsDeclarations(t *testing.T) {
	t.Parallel()

	gate, err := cqrshtmx.NewDispatchGate(allowEverySubject{})
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	for _, posture := range CommandPostures() {
		if err := gate.Declare(posture); err != nil {
			t.Fatalf("declare %s: %v", posture, err)
		}
	}

	if err := gate.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if got := len(gate.Manifest()); got != len(everyDispatchedCommand) {
		t.Fatalf("manifest has %d rows, want %d", got, len(everyDispatchedCommand))
	}
}

// TestCommandPostures_KeepTheCeremoniesPublic is the classification that matters
// most: passwordless login breaks the moment registration or credential-use
// recording requires a session.
func TestCommandPostures_KeepTheCeremoniesPublic(t *testing.T) {
	t.Parallel()

	public := map[string]bool{}

	for _, posture := range CommandPostures() {
		if posture.Posture == cqrshtmx.PosturePublic {
			public[posture.Name] = true
		}
	}

	for _, name := range []string{"RegisterUser", "RecordCredentialUse"} {
		if !public[name] {
			t.Errorf("%s must be declared public: it runs before a session exists", name)
		}
	}
}

// allowEverySubject is a permissive checker, used where the question under test
// is the wiring rather than the policy.
type allowEverySubject struct{}

func (allowEverySubject) Check(string, string, string, string) (bool, []string, error) {
	return true, nil, nil
}

// allowOnly is the discriminating checker: it admits one subject and refuses
// every other, which is how a grant path and a denial path are told apart.
type allowOnly struct{ subject string }

func (c allowOnly) Check(subject, _, _, _ string) (bool, []string, error) {
	return subject == c.subject, nil, nil
}

// newGatedService builds a service whose dispatcher carries the dispatch gate
// with the module's own posture table declared — the wiring a consumer needs.
func newGatedService(t *testing.T, checker cqrshtmx.PermissionChecker) *Service {
	t.Helper()

	gate, err := cqrshtmx.NewDispatchGate(checker)
	if err != nil {
		t.Fatalf("new gate: %v", err)
	}

	for _, posture := range CommandPostures() {
		if err := gate.Declare(posture); err != nil {
			t.Fatalf("declare %s: %v", posture, err)
		}
	}

	if err := gate.Validate(); err != nil {
		t.Fatalf("validate gate: %v", err)
	}

	svc, err := NewService(ServiceConfig{
		CommandMiddleware: []command.Middleware{gate.CommandMiddleware()},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	t.Cleanup(func() { _ = svc.Close() })

	return svc
}

// TestDispatchGate_PrivilegedBootstrapStillGrants proves the gate does not break
// the sanctioned bootstrap: an administrator subject reaches AddMember, so
// GrantRoles keeps working with a gate installed.
func TestDispatchGate_PrivilegedBootstrapStillGrants(t *testing.T) {
	t.Parallel()

	adminID := MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
	svc := newGatedService(t, allowOnly{subject: adminID.String()})

	ctx := cqrshtmx.WithUserID(context.Background(), adminID)

	err := svc.GrantRoles(ctx, ActorIDFromUser(adminID), globalDomain(), []Role{RoleSuperAdmin})
	if err != nil {
		t.Fatalf("GrantRoles with a gate installed: %v", err)
	}
}

// TestDispatchGate_UnprivilegedRevokeIsDenied proves the closing of the gap: the
// same service call, made without the permission, is refused with a recognisable
// denial even though no HTTP route was involved.
func TestDispatchGate_UnprivilegedRevokeIsDenied(t *testing.T) {
	t.Parallel()

	adminID := MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN2")
	otherID := MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN3")

	svc := newGatedService(t, allowOnly{subject: adminID.String()})

	ctx := cqrshtmx.WithUserID(context.Background(), otherID)

	err := svc.RevokeRoles(ctx, ActorIDFromUser(otherID), globalDomain())
	if err == nil {
		t.Fatal("RevokeRoles succeeded for an unprivileged actor")
	}

	if !cqrshtmx.IsForbidden(err) {
		t.Fatalf("IsForbidden = false for %v", err)
	}

	if !strings.Contains(err.Error(), "command/RemoveMember") {
		t.Errorf("denial should name the dispatch object, got: %v", err)
	}
}

// TestDispatchGate_NoIdentityIsUnauthorized proves a caller with no identity at
// all — a scheduler, a CLI, a misconfigured bootstrap — is refused as
// unauthenticated rather than allowed by default.
func TestDispatchGate_NoIdentityIsUnauthorized(t *testing.T) {
	t.Parallel()

	svc := newGatedService(t, allowOnly{subject: "nobody"})

	err := svc.RevokeRoles(context.Background(),
		ActorIDFromUser(MustParseUserID("01HK154ANGZHV2ZW0X3SKSNEN4")), globalDomain())
	if err == nil {
		t.Fatal("RevokeRoles succeeded with no subject in the context")
	}

	if !cqrshtmx.IsUnauthorized(err) && !cqrshtmx.IsForbidden(err) {
		t.Fatalf("anonymous dispatch produced an unclassified error: %v", err)
	}
}

// TestDispatchGate_PublicRegistrationStaysReachable proves the public posture is
// load-bearing: registration with no session still reaches the decider.
func TestDispatchGate_PublicRegistrationStaysReachable(t *testing.T) {
	t.Parallel()

	svc := newGatedService(t, allowOnly{subject: "nobody"})

	_, err := svc.Register(context.Background(), RegisterRequest{
		Email:       "public@example.com",
		DisplayName: "Public",
	})
	if cqrshtmx.IsUnauthorized(err) || cqrshtmx.IsForbidden(err) {
		t.Fatalf("public registration was refused by the gate: %v", err)
	}
}
