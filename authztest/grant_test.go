package authztest

import (
	"context"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	fixtureResource = "orders"
	fixtureAction   = "read"
)

// fakeAuthority is a miniature authorization store with the property that
// matters here: grants live in rows that a "second process" either can or
// cannot reconstruct.
//
// It deliberately ignores the role NAMES and records the (subject, tenant)
// question instead. That keeps the fixture honest about what the suite proves —
// that the write path is durable, scoped, and reversible — without inventing a
// role model the suite has no opinion about.
type fakeAuthority struct {
	rows map[string]struct{}
	// ephemeral models the defect this suite exists to catch: a grant that
	// lives only in the writer's memory, so a fresh checker reconstructs
	// nothing.
	ephemeral bool
	// leakEveryTenant models a store that scopes a grant by subject only.
	leakEveryTenant bool
	// revokeIsNoOp models a revoke that reports success and changes nothing.
	revokeIsNoOp bool
	// grantErr, when set, is what GrantRoles returns.
	grantErr error
}

func newFakeAuthority() *fakeAuthority {
	return &fakeAuthority{rows: map[string]struct{}{}}
}

func authorityKey(subject, tenant string) string {
	return subject + "\x00" + tenant
}

func (a *fakeAuthority) GrantRoles(_ context.Context, subject, tenant string, roles []string) error {
	if a.grantErr != nil {
		return a.grantErr
	}

	if len(roles) == 0 {
		return errorfamily.NewRejection("fake.roles_empty", "grant needs at least one role")
	}

	a.rows[authorityKey(subject, tenant)] = struct{}{}

	return nil
}

func (a *fakeAuthority) RevokeRoles(_ context.Context, subject, tenant string) error {
	if a.revokeIsNoOp {
		return nil
	}

	delete(a.rows, authorityKey(subject, tenant))

	return nil
}

// Check answers the question from the rows the authority currently holds.
func (a *fakeAuthority) Check(subject, domain, resource, action string) (bool, []string, error) {
	if resource != fixtureResource || action != fixtureAction {
		return false, nil, nil
	}

	if _, ok := a.rows[authorityKey(subject, domain)]; ok {
		return true, []string{"p, " + subject + ", " + domain}, nil
	}

	// A store that scopes a grant by SUBJECT only: any tenant with a grant
	// for this subject matches. This is the leak the isolation case hunts.
	if a.leakEveryTenant && a.holdsAnyTenant(subject) {
		return true, []string{"p, " + subject + ", *"}, nil
	}

	return false, nil, nil
}

// holdsAnyTenant reports whether subject holds a grant in any tenant.
func (a *fakeAuthority) holdsAnyTenant(subject string) bool {
	prefix := subject + "\x00"

	for key := range a.rows {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// freshChecker rebuilds a checker from the durable rows, the way a restarted
// process reconstructs policy from the journal.
func (a *fakeAuthority) freshChecker() *fakeAuthority {
	fresh := newFakeAuthority()
	if a.ephemeral {
		return fresh
	}

	for key := range a.rows {
		fresh.rows[key] = struct{}{}
	}

	return fresh
}

// suiteFor wires a GrantSuite over authority, including a Fresh builder that
// reconstructs from the durable rows.
func suiteFor(authority *fakeAuthority) GrantSuite {
	return GrantSuite{
		Checker:       authority,
		Governor:      authority,
		Subject:       "user-1",
		Tenant:        "acme",
		ForeignTenant: "globex",
		Roles:         []string{"staff"},
		ResourceType:  fixtureResource,
		Action:        fixtureAction,
		Fresh: func(*testing.T) cqrshtmx.PermissionChecker {
			return authority.freshChecker()
		},
	}
}

func TestGrantSuite_TenantIsolationPasses(t *testing.T) {
	t.Parallel()

	suite := suiteFor(newFakeAuthority())

	if err := suite.checkTenantIsolation(t.Context()); err != nil {
		t.Fatalf("a tenant-scoped grant must pass: %v", err)
	}
}

// TestGrantSuite_TenantIsolationDetectsLeak is the case the suite exists for:
// a store that scopes by subject instead of by subject-in-tenant.
func TestGrantSuite_TenantIsolationDetectsLeak(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	authority.leakEveryTenant = true

	err := suiteFor(authority).checkTenantIsolation(t.Context())
	if err == nil {
		t.Fatal("a grant that authorizes in another tenant must fail the isolation case")
	}

	if !strings.Contains(err.Error(), "not tenant-scoped") {
		t.Fatalf("the failure must name the defect: %v", err)
	}
}

func TestGrantSuite_RevocationPasses(t *testing.T) {
	t.Parallel()

	suite := suiteFor(newFakeAuthority())

	if err := suite.checkRevocation(t.Context()); err != nil {
		t.Fatalf("grant-then-revoke must pass: %v", err)
	}
}

// TestGrantSuite_RevocationDetectsNoOpRevoke covers the failure a plain
// "is it denied after revoke?" test cannot see: a grant that never worked in
// the first place would also be denied.
func TestGrantSuite_RevocationDetectsNoOpRevoke(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	authority.revokeIsNoOp = true

	err := suiteFor(authority).checkRevocation(t.Context())
	if err == nil {
		t.Fatal("a revoke that changes nothing must fail the revocation case")
	}

	if !strings.Contains(err.Error(), "still authorize") {
		t.Fatalf("the failure must name the defect: %v", err)
	}
}

func TestGrantSuite_DurabilityPasses(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	suite := suiteFor(authority)

	if err := suite.checkDurability(t.Context(), func() cqrshtmx.PermissionChecker {
		return authority.freshChecker()
	}); err != nil {
		t.Fatalf("a durable grant must survive a fresh checker: %v", err)
	}
}

// TestGrantSuite_DurabilityDetectsEphemeralGrant is the 2026-11-02 defect:
// the grant was written into an in-memory enforcer and vanished on restart.
func TestGrantSuite_DurabilityDetectsEphemeralGrant(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	authority.ephemeral = true

	err := suiteFor(authority).checkDurability(t.Context(), func() cqrshtmx.PermissionChecker {
		return authority.freshChecker()
	})
	if err == nil {
		t.Fatal("a grant that vanishes with the process must fail the durability case")
	}

	if !strings.Contains(err.Error(), "do not authorize") {
		t.Fatalf("the failure must name the defect: %v", err)
	}
}

// TestGrantSuite_NotEffectiveNamesTheReason proves the diagnostic contract: the
// failure has to separate "your grant is missing" from "you were refused".
func TestGrantSuite_NotEffectiveNamesTheReason(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	suite := suiteFor(authority)
	suite.ResourceType = "invoices"

	err := suite.checkTenantIsolation(t.Context())
	if err == nil {
		t.Fatal("a grant that does not cover the question must fail")
	}

	if !strings.Contains(err.Error(), string(cqrshtmx.ReasonNoMatchingPolicy)) {
		t.Fatalf("the failure must name the decision reason, got: %v", err)
	}
}

func TestGrantSuite_GrantFailureIsReported(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	authority.grantErr = errorfamily.NewTransient("fake.grant_failed", "store is down")

	err := suiteFor(authority).checkTenantIsolation(t.Context())
	if err == nil {
		t.Fatal("a failing grant must fail the case, not be swallowed")
	}

	if !strings.Contains(err.Error(), "store is down") {
		t.Fatalf("the cause must survive wrapping: %v", err)
	}
}

func TestGrantSuite_NilFreshCheckerIsReported(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()

	err := suiteFor(authority).checkDurability(t.Context(), nil)
	if err == nil {
		t.Fatal("a nil fresh checker must be an error, not a pass")
	}

	if errorfamily.Code(err) != "authztest.checker_nil" {
		t.Fatalf("code = %q, want authztest.checker_nil", errorfamily.Code(err))
	}
}

// TestGrantSuite_VerifyRequiresEveryField is the fail-closed contract: a suite
// missing a field would prove nothing, so it must be refused before any request.
func TestGrantSuite_VerifyRequiresEveryField(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		drop func(*GrantSuite)
		code string
	}{
		{"checker", func(s *GrantSuite) { s.Checker = nil }, "authztest.checker_nil"},
		{"governor", func(s *GrantSuite) { s.Governor = nil }, "authztest.governor_nil"},
		{"subject", func(s *GrantSuite) { s.Subject = "" }, "authztest.subject_empty"},
		{"tenant", func(s *GrantSuite) { s.Tenant = "" }, "authztest.tenant_empty"},
		{"foreign tenant", func(s *GrantSuite) { s.ForeignTenant = "" }, "authztest.foreign_tenant_empty"},
		{"resource type", func(s *GrantSuite) { s.ResourceType = "" }, "authztest.resource_empty"},
		{"action", func(s *GrantSuite) { s.Action = "" }, "authztest.action_empty"},
		{"roles", func(s *GrantSuite) { s.Roles = nil }, "authztest.roles_empty"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			suite := suiteFor(newFakeAuthority())
			testCase.drop(&suite)

			err := suite.Verify()
			if err == nil {
				t.Fatalf("a suite without %s must not verify", testCase.name)
			}

			if got := errorfamily.Code(err); got != testCase.code {
				t.Fatalf("code = %q, want %q", got, testCase.code)
			}
		})
	}
}

// TestGrantSuite_VerifyRejectsSelfAsForeignTenant closes the vacuous case: with
// the same tenant in both slots the isolation assertion is trivially true.
func TestGrantSuite_VerifyRejectsSelfAsForeignTenant(t *testing.T) {
	t.Parallel()

	suite := suiteFor(newFakeAuthority())
	suite.ForeignTenant = suite.Tenant

	err := suite.Verify()
	if err == nil {
		t.Fatal("ForeignTenant == Tenant makes the isolation case vacuous and must be refused")
	}

	if errorfamily.Code(err) != "authztest.foreign_tenant_not_foreign" {
		t.Fatalf("code = %q", errorfamily.Code(err))
	}
}

func TestGrantSuite_VerifyAcceptsACompleteSuite(t *testing.T) {
	t.Parallel()

	if err := suiteFor(newFakeAuthority()).Verify(); err != nil {
		t.Fatalf("a complete suite must verify: %v", err)
	}
}

// TestGrantSuite_ProveMethodsRun proves the *testing.T wrappers are wired to the
// checks — a green suite must not fail.
func TestGrantSuite_ProveMethodsRun(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()
	suite := suiteFor(authority)

	suite.ProveTenantIsolation(t)
	suite.ProveRevocation(t)
	suite.ProveDurability(t)
}

// TestGrantSuite_EmptyRolesAreNotGranted guards the fixture's own honesty: an
// empty role list must not silently count as a grant.
func TestGrantSuite_EmptyRolesAreNotGranted(t *testing.T) {
	t.Parallel()

	authority := newFakeAuthority()

	if err := authority.GrantRoles(t.Context(), "user-1", "acme", nil); err == nil {
		t.Fatal("granting no roles must be refused by the store")
	}
}
